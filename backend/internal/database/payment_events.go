package database

import (
	"encoding/json"
	"time"

	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

// EnsurePaymentConfirmedEvent mirrors a financial operation into the funnel as
// a `payment_confirmed` event.
//
// Payments confirmed through payments.Service create the transaction and the
// funnel event in one transaction, so they never need this helper. Legacy
// receipts reconciled from `payment_receipts` do not have that guarantee, and
// the monitoring check joins `payment_transactions` with `analytics_events`:
// an operation without the event is reported as "payment without
// payment_confirmed".
//
// The helper is idempotent: an existing event for the same payment id is kept
// as is, so it is safe to call on every reconciliation pass.
func EnsurePaymentConfirmedEvent(db *gorm.DB, tx models.PaymentTransaction) {
	if db == nil || tx.PaymentID == "" {
		return
	}

	var count int64
	if err := db.Model(&models.AnalyticsEvent{}).
		Where("step = ? AND payment_id = ?", "payment_confirmed", tx.PaymentID).
		Count(&count).Error; err != nil || count > 0 {
		return
	}

	metadata, _ := json.Marshal(map[string]interface{}{
		"payment_id":  tx.PaymentID,
		"order_id":    tx.OrderID,
		"telegram_id": tx.TelegramID,
		"tier":        tx.Tier,
		"amount":      tx.Amount,
		"source":      tx.Source,
		"pay_method":  tx.PayMethod,
		"promo_code":  tx.PromoCode,
		"backfilled":  true,
	})

	createdAt := tx.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_ = db.Create(&models.AnalyticsEvent{
		VisitorID:        tx.VisitorID,
		AttributionToken: tx.AttributionToken,
		Step:             "payment_confirmed",
		Source:           tx.Source,
		Tier:             tx.Tier,
		PromoCode:        tx.PromoCode,
		PaymentID:        tx.PaymentID,
		Metadata:         string(metadata),
		CreatedAt:        createdAt,
	}).Error
}
