package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
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
	"pricing_cta_copy": {"control", "action"},
}

func assignVariant(visitorID, experiment string) string {
	variants, ok := experimentVariants[experiment]
	if !ok || len(variants) == 0 {
		return ""
	}
	h := fnv32a(visitorID + ":" + experiment)
	return variants[int(h%uint32(len(variants)))]
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

		eventQuery := db.Model(&models.AnalyticsEvent{}).Where("created_at >= ? AND created_at < ?", from, to)
		if source != "" {
			eventQuery = eventQuery.Where("source = ?", source)
		}
		if campaign != "" {
			eventQuery = eventQuery.Where("campaign = ?", campaign)
		}
		if tier != "" {
			eventQuery = eventQuery.Where("tier = ?", tier)
		}
		if promo != "" {
			eventQuery = eventQuery.Where("promo_code = ?", promo)
		}

		type row struct {
			Step  string
			Count int64
		}
		var rows []row
		if err := eventQuery.
			Select("step, COUNT(DISTINCT COALESCE(NULLIF(visitor_id, ''), 'event:' || id::text)) AS count").
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

		txQuery := db.Model(&models.PaymentTransaction{}).
			Where("status = 'SUCCESS' AND created_at >= ? AND created_at < ?", from, to)
		if source != "" {
			txQuery = txQuery.Where("source = ?", source)
		}
		if tier != "" {
			txQuery = txQuery.Where("tier = ?", tier)
		}
		if promo != "" {
			txQuery = txQuery.Where("promo_code = ?", promo)
		}
		var paymentCount int64
		var revenue float64
		if err := txQuery.Count(&paymentCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "payment aggregation failed"})
			return
		}
		var sum struct{ Total float64 }
		if err := txQuery.Select("COALESCE(SUM(amount), 0) AS total").Scan(&sum).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "revenue aggregation failed"})
			return
		}
		revenue = roundAmount(sum.Total)

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
				"count":   paymentCount,
				"revenue": revenue,
				"average": average,
				"source":  "payment_transactions",
			},
		})
	}
}

type experimentRow struct {
	Experiment       string  `json:"experiment"`
	Variant          string  `json:"variant"`
	Visitors         int64   `json:"visitors"`
	BotLeads         int64   `json:"bot_leads"`
	Orders           int64   `json:"orders"`
	Payers           int64   `json:"payers"`
	Revenue          float64 `json:"revenue"`
	Conversion       float64 `json:"conversion"`
	UpliftPercent    float64 `json:"uplift_percent"`
	PValue           float64 `json:"p_value"`
	Significant      bool    `json:"significant"`
	MinSampleReached bool    `json:"min_sample_reached"`
	RequiredSample   int64   `json:"required_sample_per_variant"`
}

// GetExperiments reports A/B results. The success metric is a confirmed
// payment in payment_transactions, not a click or an event.
func GetExperiments(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
			return
		}
		var rows []experimentRow
		err := db.Raw(`
			WITH assignments AS (
				SELECT DISTINCT experiment, variant, visitor_id
				FROM experiment_assignments
				WHERE visitor_id IS NOT NULL AND visitor_id <> ''
			),
			events AS (
				SELECT visitor_id,
				       BOOL_OR(step = 'bot_lead') AS has_lead,
				       BOOL_OR(step = 'order_created') AS has_order
				FROM analytics_events
				WHERE visitor_id IS NOT NULL AND visitor_id <> ''
				GROUP BY visitor_id
			),
			payments AS (
				SELECT visitor_id, COUNT(*) AS payments, SUM(amount) AS revenue
				FROM payment_transactions
				WHERE status = 'SUCCESS' AND visitor_id IS NOT NULL AND visitor_id <> ''
				GROUP BY visitor_id
			)
			SELECT a.experiment, a.variant,
			       COUNT(*) AS visitors,
			       COUNT(*) FILTER (WHERE COALESCE(e.has_lead, FALSE)) AS bot_leads,
			       COUNT(*) FILTER (WHERE COALESCE(e.has_order, FALSE)) AS orders,
			       COUNT(*) FILTER (WHERE COALESCE(p.payments, 0) > 0) AS payers,
			       COALESCE(SUM(p.revenue), 0) AS revenue
			FROM assignments a
			LEFT JOIN events e ON e.visitor_id = a.visitor_id
			LEFT JOIN payments p ON p.visitor_id = a.visitor_id
			GROUP BY a.experiment, a.variant
			ORDER BY a.experiment, a.variant`).Scan(&rows).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "experiment query failed"})
			return
		}
		for i := range rows {
			if rows[i].Visitors > 0 {
				rows[i].Conversion = roundAmount(float64(rows[i].Payers) / float64(rows[i].Visitors) * 100)
			}
			rows[i].Revenue = roundAmount(rows[i].Revenue)
		}
		applyExperimentStatistics(rows)
		c.JSON(http.StatusOK, gin.H{"experiments": rows, "min_sample_per_variant": experimentMinSample})
	}
}

const experimentMinSample int64 = 100

// applyExperimentStatistics compares each non-control variant with control using
// a two-proportion z-test. With a small sample the result is reported but not
// marked significant, so a decision is not made on the first few payments.
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
			row.RequiredSample = requiredSamplePerVariant(control)
			if row.Variant == "control" || control.Visitors == 0 || row.Visitors == 0 {
				continue
			}
			baseRate := float64(control.Payers) / float64(control.Visitors)
			variantRate := float64(row.Payers) / float64(row.Visitors)
			if baseRate > 0 {
				row.UpliftPercent = roundAmount((variantRate - baseRate) / baseRate * 100)
			}
			row.PValue = roundAmount(twoProportionPValue(control.Payers, control.Visitors, row.Payers, row.Visitors)*10000) / 10000
			row.Significant = row.MinSampleReached && row.PValue < 0.05
		}
	}
}

func twoProportionPValue(x1, n1, x2, n2 int64) float64 {
	if n1 == 0 || n2 == 0 {
		return 1
	}
	p1 := float64(x1) / float64(n1)
	p2 := float64(x2) / float64(n2)
	pooled := float64(x1+x2) / float64(n1+n2)
	se := math.Sqrt(pooled * (1 - pooled) * (1/float64(n1) + 1/float64(n2)))
	if se == 0 {
		return 1
	}
	z := math.Abs(p1-p2) / se
	return math.Erfc(z / math.Sqrt2)
}

func requiredSamplePerVariant(control experimentRow) int64 {
	p1 := 0.05
	if control.Visitors > 0 && control.Payers > 0 {
		p1 = float64(control.Payers) / float64(control.Visitors)
	}
	if p1 < 0.01 {
		p1 = 0.01
	}
	if p1 > 0.9 {
		p1 = 0.9
	}
	p2 := math.Min(0.99, p1+0.01)
	pooled := (p1 + p2) / 2
	numerator := math.Pow(1.96*math.Sqrt(2*pooled*(1-pooled))+0.84*math.Sqrt(p1*(1-p1)+p2*(1-p2)), 2)
	sample := int64(math.Ceil(numerator / math.Pow(p2-p1, 2)))
	if sample < experimentMinSample {
		sample = experimentMinSample
	}
	if sample > 100000 {
		sample = 100000
	}
	return sample
}
