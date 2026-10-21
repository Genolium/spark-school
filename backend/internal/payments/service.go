// Package payments contains the single transactional entry point for
// confirming a payment and granting the corresponding entitlements.
package payments

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/money"
	"github.com/spark-school/backend/internal/outbox"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrInvalidTransition means the payment source is not in a pending state.
	ErrInvalidTransition = errors.New("payment is not in a pending state")
	// ErrNoRecipient means the payment cannot be attributed to a Telegram user.
	ErrNoRecipient = errors.New("payment has no Telegram recipient")
	// ErrInvalidAmount means the settled amount is not usable for a purchase.
	ErrInvalidAmount = errors.New("invalid payment amount")
)

// ConfirmInput describes one payment confirmation attempt. PaymentID is the
// global idempotency key; the same key must be used by every retry of the same
// provider callback.
type ConfirmInput struct {
	Source           string
	PaymentID        string
	OrderID          string
	ReceiptID        uint
	TelegramID       int64
	Username         string
	FirstName        string
	Tier             string
	Amount           float64
	AmountKopecks    int64
	Currency         string
	PayMethod        string
	PromoCode        string
	ProductName      string
	RawCallback      string
	VisitorID        string
	AttributionToken string
	AuditCredits     int
	CreatedAt        time.Time
}

// Result is returned for both the first confirmation and safe replays.
type Result struct {
	PaymentID        string
	TransactionID    uint
	UserID           uint
	TelegramID       int64
	Tier             string
	Amount           float64
	AmountKopecks    int64
	AuditCredits     int
	AlreadyProcessed bool
	AccessGranted    bool
}

// Service confirms payments atomically.
type Service struct {
	db  *gorm.DB
	aff *services.AffiliateService
}

func NewService(db *gorm.DB, aff *services.AffiliateService) *Service {
	return &Service{db: db, aff: aff}
}

// Confirm performs the whole confirmation in one PostgreSQL transaction:
// row lock -> state check -> idempotent transaction insert -> entitlements ->
// paid amount -> promo usage -> affiliate commissions -> analytics -> outbox.
func (s *Service) Confirm(in ConfirmInput) (*Result, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	var result *Result
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		result, err = s.confirmTx(tx, in)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ConfirmTx runs the confirmation inside an existing transaction so callers
// such as the admin manual-activation endpoint can combine it with their own
// row lock without nesting transactions.
func (s *Service) ConfirmTx(tx *gorm.DB, in ConfirmInput) (*Result, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction is nil")
	}
	return s.confirmTx(tx, in)
}

func (s *Service) confirmTx(tx *gorm.DB, in ConfirmInput) (*Result, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database is not configured")
	}

	in.Source = strings.ToLower(strings.TrimSpace(in.Source))
	tierProvided := strings.TrimSpace(in.Tier) != ""
	amountProvided := in.Amount != 0
	in.Tier = normalizeTier(in.Tier)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Currency == "" {
		in.Currency = "RUB"
	}
	in.Amount = roundMoney(in.Amount)
	if in.AmountKopecks <= 0 {
		in.AmountKopecks = int64(money.FromRubles(in.Amount))
	}
	if in.PaymentID == "" {
		return nil, fmt.Errorf("payment id is required")
	}
	if in.Amount < 0 {
		return nil, ErrInvalidAmount
	}
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.AuditCredits <= 0 && isAuditTier(in.Tier) {
		in.AuditCredits = 1
	}

	result := &Result{
		PaymentID:     in.PaymentID,
		TelegramID:    in.TelegramID,
		Tier:          in.Tier,
		Amount:        in.Amount,
		AmountKopecks: in.AmountKopecks,
		AuditCredits:  in.AuditCredits,
	}
	if err := s.applyConfirmation(tx, in, result, tierProvided, amountProvided); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) applyConfirmation(tx *gorm.DB, in ConfirmInput, result *Result, tierProvided, amountProvided bool) error {
	// 1. Lock and validate the payment source.
	if in.ReceiptID != 0 {
		var receipt models.PaymentReceipt
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&receipt, in.ReceiptID).Error; err != nil {
			return err
		}
		if receipt.Status == "approved" {
			return s.loadExisting(tx, in.PaymentID, result)
		}
		if receipt.Status != "pending" {
			return fmt.Errorf("%w: receipt status %s", ErrInvalidTransition, receipt.Status)
		}
		now := time.Now()
		res := tx.Model(&models.PaymentReceipt{}).
			Where("id = ? AND status = ?", receipt.ID, "pending").
			Updates(map[string]interface{}{
				"status":       "approved",
				"confirmed_at": &now,
				"updated_at":   &now,
				"promo_code":   firstNonEmpty(receipt.PromoCode, in.PromoCode),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrInvalidTransition
		}
		if in.TelegramID == 0 {
			in.TelegramID = receipt.TelegramID
			result.TelegramID = receipt.TelegramID
		}
		if in.Username == "" {
			in.Username = receipt.Username
		}
		if !tierProvided {
			in.Tier = normalizeTier(receipt.Tier)
			result.Tier = in.Tier
		}
		if !amountProvided {
			in.Amount = roundMoney(receipt.Amount)
			result.Amount = in.Amount
		}
		if in.PromoCode == "" {
			in.PromoCode = receipt.PromoCode
		}
		if in.OrderID == "" {
			in.OrderID = fmt.Sprintf("RECEIPT_%d", receipt.ID)
		}
		if in.Source == "" {
			in.Source = "sbp_receipt"
		}
		if in.PayMethod == "" {
			in.PayMethod = "sbp_receipt"
		}
		if in.ProductName == "" {
			in.ProductName = "Оплата участия в проекте «так называемый SPARK»"
		}
	}

	// 2. Insert the financial operation with a unique idempotency key.
	transaction := models.PaymentTransaction{
		PaymentID:        in.PaymentID,
		OrderID:          in.OrderID,
		TelegramID:       in.TelegramID,
		Username:         in.Username,
		Tier:             in.Tier,
		Amount:           in.Amount,
		AmountKopecks:    in.AmountKopecks,
		OriginalAmount:   in.Amount,
		Status:           "SUCCESS",
		Currency:         in.Currency,
		PayMethod:        in.PayMethod,
		ProductName:      in.ProductName,
		PromoCode:        in.PromoCode,
		RawCallback:      in.RawCallback,
		Source:           in.Source,
		VisitorID:        in.VisitorID,
		AttributionToken: in.AttributionToken,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        time.Now(),
	}

	if in.TelegramID <= 0 {
		// A real provider settlement without a recipient is still recorded
		// for manual reconciliation, but no access can be granted.
		transaction.Status = "FLAGGED_NO_RECIPIENT"
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "payment_id"}},
			DoNothing: true,
		}).Create(&transaction)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return s.loadExisting(tx, in.PaymentID, result)
		}
		result.TransactionID = transaction.ID
		result.AlreadyProcessed = false
		return outbox.EnqueuePaymentNoticeTx(tx, outbox.PaymentNoticePayload{
			Kind:      "admin",
			PaymentID: in.PaymentID,
			Amount:    in.Amount,
			Tier:      in.Tier,
			Username:  in.Username,
			Source:    in.Source,
		})
	}

	res := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "payment_id"}},
		DoNothing: true,
	}).Create(&transaction)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return s.loadExisting(tx, in.PaymentID, result)
	}
	result.TransactionID = transaction.ID

	// 3. Lock the user row. Create the user if this is their first purchase.
	var user models.User
	wasNew := false
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("telegram_id = ?", in.TelegramID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		wasNew = true
		firstName := strings.TrimSpace(in.FirstName)
		if firstName == "" {
			firstName = "Студент"
		}
		newUser := models.User{
			TelegramID:       in.TelegramID,
			Username:         in.Username,
			FirstName:        firstName,
			Role:             "student",
			AttributionToken: in.AttributionToken,
			VisitorID:        in.VisitorID,
		}
		createRes := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "telegram_id"}},
			DoNothing: true,
		}).Create(&newUser)
		if createRes.Error != nil {
			return createRes.Error
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("telegram_id = ?", in.TelegramID).First(&user).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	result.UserID = user.ID
	if in.VisitorID == "" {
		in.VisitorID = user.VisitorID
	}
	if in.AttributionToken == "" {
		in.AttributionToken = user.AttributionToken
	}
	if in.VisitorID != "" || in.AttributionToken != "" {
		_ = tx.Model(&models.PaymentTransaction{}).Where("payment_id = ?", in.PaymentID).Updates(map[string]interface{}{
			"visitor_id":        in.VisitorID,
			"attribution_token": in.AttributionToken,
		}).Error
	}

	// 4. Grant exactly the purchased entitlement.
	now := time.Now()
	user.HasAccess = true
	if user.AccessGrantedAt == nil {
		user.AccessGrantedAt = &now
	}
	if user.AnonymizedAt == nil {
		if in.Username != "" && user.Username == "" {
			user.Username = in.Username
		}
		if in.AttributionToken != "" && user.AttributionToken == "" {
			user.AttributionToken = in.AttributionToken
		}
		if in.VisitorID != "" && user.VisitorID == "" {
			user.VisitorID = in.VisitorID
		}
	}
	if in.PromoCode != "" {
		user.PromoCode = strings.ToUpper(in.PromoCode)
	}

	if isAuditTier(in.Tier) {
		credits := in.AuditCredits
		if credits <= 0 {
			credits = 1
		}
		user.HasAuditUpsell = true
		user.AuditCredits += credits
		result.AuditCredits = credits
	} else if wasNew || rank(in.Tier) >= rank(user.Tier) {
		user.Tier = in.Tier
	}
	paidKopecks := money.Kopecks(user.PaidAmountKopecks)
	if paidKopecks == 0 {
		paidKopecks = money.FromRubles(user.PaidAmount)
	}
	paidKopecks += money.Kopecks(in.AmountKopecks)
	user.PaidAmountKopecks = int64(paidKopecks)
	user.PaidAmount = paidKopecks.ToRubles()
	if err := tx.Save(&user).Error; err != nil {
		return err
	}
	result.AccessGranted = true

	// 5. Promo usage is incremented exactly once, in the same transaction.
	if in.PromoCode != "" {
		if err := tx.Model(&models.PromoCode{}).
			Where("UPPER(code) = ? AND is_active = true", strings.ToUpper(in.PromoCode)).
			UpdateColumn("uses_count", gorm.Expr("uses_count + 1")).Error; err != nil {
			return err
		}
	}

	// 6. Affiliate commissions, keyed by the payment id.
	if s.aff != nil {
		if err := s.aff.ProcessCoursePurchaseForPaymentTx(tx, user.ID, in.PaymentID, in.Amount); err != nil {
			return err
		}
	}

	// 7. Server-side funnel event, tied to the same idempotency key.
	metadata, _ := json.Marshal(map[string]interface{}{
		"payment_id":        in.PaymentID,
		"order_id":          in.OrderID,
		"visitor_id":        firstNonEmpty(in.VisitorID, user.VisitorID),
		"attribution_token": firstNonEmpty(in.AttributionToken, user.AttributionToken),
		"telegram_id":       in.TelegramID,
		"tier":              in.Tier,
		"amount":            in.Amount,
		"source":            in.Source,
		"promo_code":        in.PromoCode,
	})
	if err := tx.Create(&models.AnalyticsEvent{
		VisitorID:        firstNonEmpty(in.VisitorID, user.VisitorID),
		AttributionToken: firstNonEmpty(in.AttributionToken, user.AttributionToken),
		Step:             "payment_confirmed",
		Source:           in.Source,
		Tier:             in.Tier,
		PromoCode:        in.PromoCode,
		PaymentID:        in.PaymentID,
		Metadata:         string(metadata),
		CreatedAt:        time.Now(),
	}).Error; err != nil {
		return err
	}

	// 8. Mark the checkout order as paid (if this payment came from one).
	if in.OrderID != "" {
		if err := tx.Model(&models.PaymentOrder{}).
			Where("order_id = ? AND status = 'pending'", in.OrderID).
			Updates(map[string]interface{}{
				"status":              "paid",
				"provider_payment_id": in.PaymentID,
				"updated_at":          time.Now(),
			}).Error; err != nil {
			return err
		}
	}

	// 9. Queue user and admin notifications in the same transaction.
	kind := "access"
	if isAuditTier(in.Tier) {
		kind = "audit"
	} else if user.PaidAmount > in.Amount {
		kind = "upgrade"
	}
	if err := outbox.EnqueuePaymentNoticeTx(tx, outbox.PaymentNoticePayload{
		Kind:       kind,
		TelegramID: in.TelegramID,
		PaymentID:  in.PaymentID,
		Tier:       in.Tier,
		Amount:     in.Amount,
		Username:   in.Username,
		PromoCode:  in.PromoCode,
		Source:     in.Source,
	}); err != nil {
		return err
	}
	return outbox.EnqueuePaymentNoticeTx(tx, outbox.PaymentNoticePayload{
		Kind:       "admin",
		TelegramID: in.TelegramID,
		PaymentID:  in.PaymentID,
		Tier:       in.Tier,
		Amount:     in.Amount,
		Username:   in.Username,
		PromoCode:  in.PromoCode,
		Source:     in.Source,
	})
}

// RequeueAccessNotification schedules a repeated delivery for an already
// confirmed payment without touching money or entitlements.
func (s *Service) RequeueAccessNotification(paymentID string) error {
	if s.db == nil {
		return fmt.Errorf("database is not configured")
	}
	var pt models.PaymentTransaction
	if err := s.db.Where("payment_id = ?", paymentID).First(&pt).Error; err != nil {
		return err
	}
	if pt.Status != "SUCCESS" {
		return fmt.Errorf("payment %s has status %s", paymentID, pt.Status)
	}
	kind := "access"
	if isAuditTier(pt.Tier) {
		kind = "audit"
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return outbox.EnqueuePaymentNoticeTx(tx, outbox.PaymentNoticePayload{
			Kind:       kind,
			TelegramID: pt.TelegramID,
			PaymentID:  pt.PaymentID,
			Tier:       pt.Tier,
			Amount:     pt.Amount,
			Username:   pt.Username,
			PromoCode:  pt.PromoCode,
			Source:     pt.Source,
		})
	})
}

func (s *Service) loadExisting(tx *gorm.DB, paymentID string, result *Result) error {
	var existing models.PaymentTransaction
	if err := tx.Where("payment_id = ?", paymentID).First(&existing).Error; err != nil {
		return err
	}
	result.AlreadyProcessed = true
	result.TransactionID = existing.ID
	result.Tier = existing.Tier
	result.Amount = existing.Amount
	result.TelegramID = existing.TelegramID
	if existing.TelegramID > 0 {
		var user models.User
		if err := tx.Where("telegram_id = ?", existing.TelegramID).First(&user).Error; err == nil {
			result.UserID = user.ID
		}
	}
	return nil
}

func normalizeTier(tier string) string {
	t := strings.ToLower(strings.TrimSpace(tier))
	switch t {
	case "basic", "accelerator", "vip":
		return t
	case "audit", "audit_essay", "essay_audit":
		return "audit_essay"
	default:
		return "accelerator"
	}
}

func isAuditTier(tier string) bool {
	return tier == "audit_essay"
}

func rank(tier string) int {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "basic":
		return 1
	case "accelerator":
		return 2
	case "vip":
		return 3
	default:
		return 0
	}
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
