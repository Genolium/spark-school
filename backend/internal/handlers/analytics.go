package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
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
	"pricing_select_tier",
	"bot_handoff",
	"bot_lead",
	"order_created",
	"payment_confirmed",
}

// piiMetadataKeys are dropped before persistence. Analytics must not become a
// second, unmanaged store of personal data.
var piiMetadataKeys = map[string]struct{}{
	"username": {}, "email": {}, "phone": {}, "name": {}, "first_name": {},
	"last_name": {}, "telegram_username": {}, "address": {},
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
			if _, isPII := piiMetadataKeys[lower]; isPII {
				continue
			}
			metadata[key] = value
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
		if err := db.Create(&event).Error; err != nil {
			log.Printf("[Analytics] failed to persist step %q: %v", step, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "analytics event was not recorded"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
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
