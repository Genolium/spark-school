package handlers

import (
	"math"
	"strings"

	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

// The essay audit has a fixed price that is not configured in the admin panel.
const minAuditPayment = 1500.00

// maxUnderpaymentDiscountPercent caps how much of a tier price a promo code
// can remove before the fraud floor stops following it.
const maxUnderpaymentDiscountPercent = 90

func roundAmount(v float64) float64 {
	return math.Round(v*100) / 100
}

func normalizePaidTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "basic":
		return "basic"
	case "vip":
		return "vip"
	case "accelerator":
		return "accelerator"
	case "audit", "audit_essay", "essay_audit":
		return "audit_essay"
	default:
		return ""
	}
}

// maxActivePromoDiscount returns the largest discount percent among active
// promo codes, clamped to [0, maxUnderpaymentDiscountPercent].
func maxActivePromoDiscount(db *gorm.DB) float64 {
	if db == nil {
		return 0
	}
	var maxPct int
	if err := db.Model(&models.PromoCode{}).
		Where("is_active = ?", true).
		Select("COALESCE(MAX(discount_percent), 0)").
		Scan(&maxPct).Error; err != nil {
		return 0
	}
	if maxPct < 0 {
		maxPct = 0
	}
	if maxPct > maxUnderpaymentDiscountPercent {
		maxPct = maxUnderpaymentDiscountPercent
	}
	return float64(maxPct)
}

// minimumForTier returns the lowest amount accepted as a genuine payment for
// the tier: the current configured price with the largest active promo
// discount applied. It follows the admin prices, so a price change cannot
// leave a full-price payment below the floor.
func minimumForTier(db *gorm.DB, tier string) float64 {
	normalized := normalizePaidTier(tier)
	if normalized == "audit_essay" {
		return minAuditPayment
	}
	pricing := models.DefaultCoursePricing
	if db != nil {
		pricing = models.GetCoursePricing(db)
	}
	price := pricing.AcceleratorPrice
	switch normalized {
	case "vip":
		price = pricing.VIPPrice
	case "basic":
		price = pricing.BasicPrice
	}
	floor := price * (100 - maxActivePromoDiscount(db)) / 100
	// One rouble of slack absorbs rounding differences between providers.
	return math.Max(roundAmount(floor)-1, 0)
}

// minimumCoursePayment is the floor for the cheapest course tier. It is used
// before the tier of an incoming payment is known.
func minimumCoursePayment(db *gorm.DB) float64 {
	return math.Min(minimumForTier(db, "basic"), math.Min(minimumForTier(db, "accelerator"), minimumForTier(db, "vip")))
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func notifyAdmin(cfg *config.Config, notify func(int64, string) error, message string) {
	if notify == nil || cfg == nil {
		return
	}
	adminID := cfg.AdminTelegramID
	if adminID == 0 {
		return
	}
	_ = notify(adminID, message)
}

func (h *YooKassaHandler) alertAdmin(message string) {
	notifyAdmin(h.cfg, h.notifyFunc, message)
}

func (h *PaymentWebhookHandler) alertAdmin(message string) {
	notifyAdmin(h.cfg, h.notifyFunc, message)
}

func sanitizeToken(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) > max {
		value = value[:max]
	}
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return ""
		}
	}
	return value
}
