package handlers

import (
	"math"
	"strings"

	"github.com/spark-school/backend/internal/config"
)

// Minimum course prices used for fraud checks. Basic with a 5% promo is
// 3 790.50 ₽, so 3 500 ₽ remains a safe floor for tier purchases.
const (
	minBasicPayment       = 3500.00
	minAcceleratorPayment = 3500.00
	minVIPPayment         = 13000.00
	minAuditPayment       = 1500.00
)

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

func minimumForTier(tier string) float64 {
	switch normalizePaidTier(tier) {
	case "vip":
		return minVIPPayment
	case "audit_essay":
		return minAuditPayment
	case "basic":
		return minBasicPayment
	default:
		return minAcceleratorPayment
	}
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
