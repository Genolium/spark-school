package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Client-emitted funnel steps. Server-only steps (payment_confirmed, access
// delivered, completed work) are deliberately excluded so the browser cannot
// fabricate a paid conversion.
var allowedClientSteps = map[string]struct{}{
	"landing_view":        {},
	"demo_lesson_view":    {},
	"diagnostic_start":    {},
	"diagnostic_complete": {},
	"pricing_select_tier": {},
	"bot_handoff":         {},
}

var serverOnlySteps = map[string]struct{}{
	"payment_confirmed":    {},
	"bot_lead":             {},
	"access_delivered":     {},
	"first_task_completed": {},
}

var funnelOrder = []string{
	"landing_view",
	"diagnostic_start",
	"diagnostic_complete",
	"diagnostic_plan_created",
	"pricing_select_tier",
	"bot_handoff",
	"bot_lead",
	"checkout_started",
	"order_created",
	"payment_confirmed",
	"access_delivered",
}

// piiMetadataKeys are dropped before persistence. Analytics must not become a
// second, unmanaged store of personal data.
var allowedMetadataKeys = map[string]struct{}{
	"utm_source": {}, "utm_medium": {}, "utm_campaign": {}, "utm_content": {},
	"source": {}, "tier": {}, "price": {}, "level": {}, "score": {},
	"universityyear": {}, "tab": {}, "method": {},
}

type analyticsTrackRequest struct {
	Step             string                 `json:"step" binding:"required"`
	Timestamp        string                 `json:"timestamp"`
	VisitorID        string                 `json:"visitor_id"`
	AttributionToken string                 `json:"attribution_token"`
	Source           string                 `json:"source"`
	Campaign         string                 `json:"campaign"`
	Tier             string                 `json:"tier"`
	PromoCode        string                 `json:"promo_code"`
	Metadata         map[string]interface{} `json:"metadata"`
	Experiments      map[string]string      `json:"experiments"`
}

// Server-side experiment definitions. The browser never chooses a variant:
// the server derives it deterministically from the anonymous visitor id.
var experimentVariants = map[string][]string{
	"hero_cta_copy":    {"control", "benefit", "diagnostic"},
	"hero_headline":    {"control", "benefit", "urgency"},
	"pricing_cta_copy": {"control", "action"},
	"pricing_price":    {"control", "lower"},
}

func assignVariant(visitorID, experiment string) string {
	variants, ok := experimentVariants[experiment]
	if !ok || len(variants) == 0 {
		return ""
	}
	h := fnv32a(visitorID + ":" + experiment)
	return variants[int(h%uint32(len(variants)))]
}

const priceExperimentLower = 9990.0

// acceleratorPriceForVariant keeps display and checkout on the same price.
// Only the accelerator tier is part of the price test.
func acceleratorPriceForVariant(base float64, variant string) float64 {
	if variant == "lower" {
		return priceExperimentLower
	}
	return base
}

// fnv32a matches the FNV-1a implementation used by the frontend so the first
// paint shows the same bucket the server stores.
func fnv32a(input string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(input); i++ {
		hash ^= uint32(input[i])
		hash *= 16777619
	}
	return hash
}

// TrackAnalytics ingests a whitelisted client funnel event.
func TrackAnalytics(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req analyticsTrackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid analytics payload"})
			return
		}

		step := strings.TrimSpace(strings.ToLower(req.Step))
		if _, ok := allowedClientSteps[step]; !ok {
			if _, serverOnly := serverOnlySteps[step]; serverOnly {
				c.JSON(http.StatusForbidden, gin.H{"error": "this event is recorded by the server only"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown funnel step"})
			return
		}

		visitorID := sanitizeToken(req.VisitorID, 64)
		attributionToken := sanitizeToken(req.AttributionToken, 64)
		source := sanitizeToken(req.Source, 64)
		campaign := sanitizeToken(req.Campaign, 80)
		tier := normalizePaidTier(req.Tier)
		promo := sanitizeToken(strings.ToUpper(req.PromoCode), 32)

		metadata := make(map[string]interface{}, len(req.Metadata))
		for key, value := range req.Metadata {
			lower := strings.ToLower(key)
			if _, allowed := allowedMetadataKeys[lower]; !allowed {
				continue
			}
			switch value.(type) {
			case string, float64, bool, int, int64:
				metadata[key] = value
			}
		}
		metaBytes, err := json.Marshal(metadata)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metadata"})
			return
		}
		if len(metaBytes) > 4096 {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "metadata is too large"})
			return
		}
		if source == "" {
			if v, ok := metadata["utm_source"].(string); ok {
				source = sanitizeToken(v, 64)
			}
		}
		if campaign == "" {
			if v, ok := metadata["utm_campaign"].(string); ok {
				campaign = sanitizeToken(v, 80)
			}
		}

		event := models.AnalyticsEvent{
			SchemaVersion:    1,
			VisitorID:        visitorID,
			AttributionToken: attributionToken,
			Step:             step,
			Source:           source,
			Campaign:         campaign,
			Tier:             tier,
			PromoCode:        promo,
			Metadata:         string(metaBytes),
			CreatedAt:        time.Now(),
		}
		if visitorID != "" {
			for name := range experimentVariants {
				variant := assignVariant(visitorID, name)
				if variant == "" {
					continue
				}
				_ = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ExperimentAssignment{
					Experiment:       name,
					Variant:          variant,
					VisitorID:        visitorID,
					AttributionToken: attributionToken,
					CreatedAt:        time.Now(),
				}).Error
			}
		}

		if err := db.Create(&event).Error; err != nil {
			log.Printf("[Analytics] failed to persist step %q: %v", step, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics event was not recorded"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// AssignExperiments returns and persists the server-side A/B buckets for a
// visitor. The client uses it only for display; the database is canonical.
func AssignExperiments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			VisitorID        string `json:"visitor_id"`
			AttributionToken string `json:"attribution_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment payload"})
			return
		}
		visitorID := sanitizeToken(req.VisitorID, 64)
		if visitorID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "visitor_id is required"})
			return
		}
		assignments := make(map[string]string, len(experimentVariants))
		for name := range experimentVariants {
			variant := assignVariant(visitorID, name)
			assignments[name] = variant
			_ = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ExperimentAssignment{
				Experiment:       name,
				Variant:          variant,
				VisitorID:        visitorID,
				AttributionToken: sanitizeToken(req.AttributionToken, 64),
				CreatedAt:        time.Now(),
			}).Error
		}
		c.JSON(http.StatusOK, gin.H{"experiments": assignments})
	}
}

type funnelStepResponse struct {
	Step                   string  `json:"step"`
	Count                  int64   `json:"count"`
	ConversionFromPrevious float64 `json:"conversion_from_previous"`
}

// GetFunnel returns an aggregated funnel. Financial figures come only from
// payment_transactions; event counts come only from analytics_events.
func GetFunnel(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
			return
		}

		now := time.Now()
		from := now.AddDate(0, 0, -29)
		to := now.Add(24 * time.Hour)
		if raw := c.Query("from"); raw != "" {
			if parsed, err := time.Parse("2006-01-02", raw); err == nil {
				from = parsed
			}
		}
		if raw := c.Query("to"); raw != "" {
			if parsed, err := time.Parse("2006-01-02", raw); err == nil {
				to = parsed.Add(24 * time.Hour)
			}
		}
		source := sanitizeToken(c.Query("source"), 64)
		campaign := sanitizeToken(c.Query("campaign"), 80)
		tier := normalizePaidTier(c.Query("tier"))
		promo := sanitizeToken(strings.ToUpper(c.Query("promo")), 32)

		// A step is only meaningful for visitors who entered the funnel through
		// its first step inside this window. Payments imported from before
		// client-side tracking (and manual grants) otherwise push
		// `payment_confirmed` above `landing_view` and make conversion look
		// broken; they are reported separately as untracked money.
		applyEventFilters := func(q *gorm.DB) *gorm.DB {
			if source != "" {
				q = q.Where("source = ?", source)
			}
			if campaign != "" {
				q = q.Where("campaign = ?", campaign)
			}
			if tier != "" {
				q = q.Where("tier = ?", tier)
			}
			if promo != "" {
				q = q.Where("promo_code = ?", promo)
			}
			return q
		}

		cohort := applyEventFilters(
			db.Model(&models.AnalyticsEvent{}).
				Select("visitor_id").
				Where("step = ? AND visitor_id IS NOT NULL AND visitor_id <> ''", "landing_view").
				Where("created_at >= ? AND created_at < ?", from, to),
		).Group("visitor_id")

		eventQuery := applyEventFilters(
			db.Model(&models.AnalyticsEvent{}).
				Where("visitor_id IN (?)", cohort).
				Where("created_at >= ? AND created_at < ?", from, to),
		)

		type row struct {
			Step  string
			Count int64
		}
		var rows []row
		if err := eventQuery.
			Select("step, COUNT(DISTINCT visitor_id) AS count").
			Group("step").Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "funnel query failed"})
			return
		}
		counts := make(map[string]int64, len(rows))
		for _, r := range rows {
			counts[r.Step] = r.Count
		}

		steps := make([]funnelStepResponse, 0, len(funnelOrder))
		var previous int64
		for i, step := range funnelOrder {
			count := counts[step]
			conversion := 0.0
			if i > 0 && previous > 0 {
				conversion = float64(count) / float64(previous) * 100
			}
			steps = append(steps, funnelStepResponse{Step: step, Count: count, ConversionFromPrevious: roundAmount(conversion)})
			previous = count
		}

		newPaymentQuery := func() *gorm.DB {
			q := db.Model(&models.PaymentTransaction{}).
				Where("status = 'SUCCESS' AND created_at >= ? AND created_at < ?", from, to)
			if source != "" {
				q = q.Where("source = ?", source)
			}
			if tier != "" {
				q = q.Where("tier = ?", tier)
			}
			if promo != "" {
				q = q.Where("promo_code = ?", promo)
			}
			return q
		}

		var paymentCount int64
		var revenue float64
		if err := newPaymentQuery().Count(&paymentCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "payment aggregation failed"})
			return
		}
		var sum struct{ Total float64 }
		if err := newPaymentQuery().Select("COALESCE(SUM(amount), 0) AS total").Scan(&sum).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "revenue aggregation failed"})
			return
		}
		revenue = roundAmount(sum.Total)

		var trackedCount int64
		var trackedSum struct{ Total float64 }
		if err := newPaymentQuery().Where("visitor_id IN (?)", cohort).Count(&trackedCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "tracked payment aggregation failed"})
			return
		}
		if err := newPaymentQuery().Where("visitor_id IN (?)", cohort).
			Select("COALESCE(SUM(amount), 0) AS total").Scan(&trackedSum).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "tracked revenue aggregation failed"})
			return
		}
		trackedRevenue := roundAmount(trackedSum.Total)
		untrackedRevenue := roundAmount(revenue - trackedRevenue)
		untrackedCount := paymentCount - trackedCount

		average := 0.0
		if paymentCount > 0 {
			average = roundAmount(revenue / float64(paymentCount))
		}

		c.JSON(http.StatusOK, gin.H{
			"filters": gin.H{
				"from": from.Format("2006-01-02"), "to": to.Add(-time.Nanosecond).Format("2006-01-02"),
				"source": source, "campaign": campaign, "tier": tier, "promo": promo,
			},
			"steps": steps,
			"payments": gin.H{
				"count":             paymentCount,
				"revenue":           revenue,
				"average":           average,
				"tracked_count":     trackedCount,
				"tracked_revenue":   trackedRevenue,
				"untracked_count":   untrackedCount,
				"untracked_revenue": untrackedRevenue,
				"source":            "payment_transactions",
			},
		})
	}
}

type experimentRow struct {
	Experiment        string  `json:"experiment"`
	Variant           string  `json:"variant"`
	Visitors          int64   `json:"visitors"`
	BotLeads          int64   `json:"bot_leads"`
	Orders            int64   `json:"orders"`
	Payers            int64   `json:"payers"`
	Revenue           float64 `json:"revenue"`
	RevenueSquared    float64 `json:"-"`
	RevenuePerVisitor float64 `json:"revenue_per_visitor"`
	Conversion        float64 `json:"conversion"`
	UpliftPercent     float64 `json:"uplift_percent"`
	PValue            float64 `json:"p_value"`
	Significant       bool    `json:"significant"`
	Decision          string  `json:"decision"`
	MinSampleReached  bool    `json:"min_sample_reached"`
	RequiredSample    int64   `json:"required_sample_per_variant"`
}

// GetExperiments reports A/B results over a fixed decision window. The metric
// is revenue per visitor in payment_transactions, not clicks or page events:
// a cheaper price can raise conversion while lowering revenue, and the report
// must make that trade-off visible.
func GetExperiments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
			return
		}

		windowDays := 14
		if raw := c.Query("days"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				windowDays = parsed
			}
		}
		if windowDays < 1 {
			windowDays = 1
		}
		if windowDays > 90 {
			windowDays = 90
		}
		now := time.Now()
		from := now.AddDate(0, 0, -windowDays)
		to := now
		if raw := c.Query("from"); raw != "" {
			if parsed, err := time.Parse("2006-01-02", raw); err == nil {
				from = parsed
			}
		}
		if raw := c.Query("to"); raw != "" {
			if parsed, err := time.Parse("2006-01-02", raw); err == nil {
				to = parsed.Add(24 * time.Hour)
			}
		}

		var rows []experimentRow
		err := db.Raw(`
			WITH assignments AS (
				SELECT DISTINCT experiment, variant, visitor_id
				FROM experiment_assignments
				WHERE visitor_id IS NOT NULL AND visitor_id <> ''
				  AND created_at >= ? AND created_at < ?
			),
			events AS (
				SELECT visitor_id,
				       BOOL_OR(step = 'bot_lead') AS has_lead,
				       BOOL_OR(step = 'order_created') AS has_order
				FROM analytics_events
				WHERE visitor_id IS NOT NULL AND visitor_id <> ''
				  AND created_at >= ? AND created_at < ?
				GROUP BY visitor_id
			),
			payments AS (
				SELECT visitor_id, COUNT(*) AS payments, SUM(amount) AS revenue
				FROM payment_transactions
				WHERE status = 'SUCCESS' AND visitor_id IS NOT NULL AND visitor_id <> ''
				  AND created_at >= ? AND created_at < ?
				GROUP BY visitor_id
			)
			SELECT a.experiment, a.variant,
			       COUNT(*) AS visitors,
			       COUNT(*) FILTER (WHERE COALESCE(e.has_lead, FALSE)) AS bot_leads,
			       COUNT(*) FILTER (WHERE COALESCE(e.has_order, FALSE)) AS orders,
			       COUNT(*) FILTER (WHERE COALESCE(p.payments, 0) > 0) AS payers,
			       COALESCE(SUM(p.revenue), 0) AS revenue,
			       COALESCE(SUM(p.revenue * p.revenue), 0) AS revenue_squared
			FROM assignments a
			LEFT JOIN events e ON e.visitor_id = a.visitor_id
			LEFT JOIN payments p ON p.visitor_id = a.visitor_id
			GROUP BY a.experiment, a.variant
			ORDER BY a.experiment, a.variant`, from, to, from, to, from, to).Scan(&rows).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "experiment query failed"})
			return
		}
		for i := range rows {
			if rows[i].Visitors > 0 {
				rows[i].Conversion = roundAmount(float64(rows[i].Payers) / float64(rows[i].Visitors) * 100)
				rows[i].RevenuePerVisitor = roundAmount(rows[i].Revenue / float64(rows[i].Visitors))
			}
			rows[i].Revenue = roundAmount(rows[i].Revenue)
		}
		applyExperimentStatistics(rows)
		c.JSON(http.StatusOK, gin.H{
			"experiments":            rows,
			"min_sample_per_variant": experimentMinSample,
			"decision_metric":        "revenue_per_visitor",
			"window_days":            windowDays,
			"from":                   from.Format("2006-01-02"),
			"to":                     to.Add(-time.Nanosecond).Format("2006-01-02"),
		})
	}
}

const experimentMinSample int64 = 100

// applyExperimentStatistics compares each non-control variant with control on
// revenue per visitor. A cheaper tier can win on conversion and still lose on
// money, so the decision metric is rubles per visitor, not payer count.
// The report needs at least 100 visitors in both arms before it calls a winner.
func applyExperimentStatistics(rows []experimentRow) {
	byExperiment := make(map[string][]int)
	for i, row := range rows {
		byExperiment[row.Experiment] = append(byExperiment[row.Experiment], i)
	}
	for _, indexes := range byExperiment {
		controlIdx := -1
		for _, idx := range indexes {
			if rows[idx].Variant == "control" {
				controlIdx = idx
				break
			}
		}
		if controlIdx < 0 {
			continue
		}
		control := rows[controlIdx]
		for _, idx := range indexes {
			row := &rows[idx]
			row.MinSampleReached = control.Visitors >= experimentMinSample && row.Visitors >= experimentMinSample
			row.RequiredSample = experimentMinSample
			if row.Variant == "control" {
				row.Decision = "reference"
				continue
			}
			if control.Visitors == 0 || row.Visitors == 0 {
				row.Decision = "collect"
				continue
			}
			baseRPV := control.Revenue / float64(control.Visitors)
			variantRPV := row.Revenue / float64(row.Visitors)
			if baseRPV > 0 {
				row.UpliftPercent = roundAmount((variantRPV - baseRPV) / baseRPV * 100)
			}
			row.PValue = roundAmount(revenuePerVisitorPValue(control, *row)*10000) / 10000
			row.Significant = row.MinSampleReached && row.PValue < 0.05
			switch {
			case !row.MinSampleReached:
				row.Decision = "collect"
			case row.Significant && row.UpliftPercent > 0:
				row.Decision = "switch"
			case row.Significant && row.UpliftPercent < 0:
				row.Decision = "keep_control"
			default:
				row.Decision = "no_difference"
			}
		}
	}
}

// revenuePerVisitorPValue runs a two-sample z-test on revenue per visitor.
// With at least 100 visitors per arm the normal approximation is adequate and
// avoids pulling a t-distribution implementation into the handlers package.
func revenuePerVisitorPValue(control, variant experimentRow) float64 {
	controlMean := control.Revenue / float64(control.Visitors)
	variantMean := variant.Revenue / float64(variant.Visitors)
	controlVar := revenueSampleVariance(control, controlMean)
	variantVar := revenueSampleVariance(variant, variantMean)
	se := math.Sqrt(controlVar/float64(control.Visitors) + variantVar/float64(variant.Visitors))
	if se == 0 {
		if controlMean == variantMean {
			return 1
		}
		return 0
	}
	z := math.Abs(variantMean-controlMean) / se
	return math.Erfc(z / math.Sqrt2)
}

func revenueSampleVariance(row experimentRow, mean float64) float64 {
	if row.Visitors < 2 {
		return 0
	}
	n := float64(row.Visitors)
	variance := (row.RevenueSquared - n*mean*mean) / (n - 1)
	if variance < 0 {
		return 0
	}
	return variance
}
