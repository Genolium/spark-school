package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AffiliateService struct {
	db *gorm.DB
}

func NewAffiliateService(db *gorm.DB) *AffiliateService {
	return &AffiliateService{db: db}
}

// TrackReferral links the new student to the referring affiliate if not already linked.
func (s *AffiliateService) TrackReferral(userID uint, refCode string) error {
	if refCode == "" {
		return nil
	}

	var existing models.Referral
	if err := s.db.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		// Already has a referrer, connection is permanent
		return nil
	}

	var profile models.AffiliateProfile
	if err := s.db.Where("referral_code = ?", refCode).First(&profile).Error; err != nil {
		return fmt.Errorf("invalid referral code: %s", refCode)
	}

	if profile.UserID == userID {
		return errors.New("cannot refer yourself")
	}

	// Check if the referrer has a parent (Level 2)
	var referrer models.Referral
	var parentID *uint
	if err := s.db.Where("user_id = ?", profile.UserID).First(&referrer).Error; err == nil {
		parentID = referrer.ReferredByID
	}

	directReferrerID := profile.UserID
	ref := models.Referral{
		UserID:           userID,
		ReferredByID:     &directReferrerID,
		ParentReferrerID: parentID,
		CreatedAt:        time.Now(),
	}

	return s.db.Create(&ref).Error
}

// ProcessCoursePurchaseTx calculates 15% and 5% commissions within an existing database transaction.
func (s *AffiliateService) ProcessCoursePurchaseTx(tx *gorm.DB, buyerID uint, coursePrice float64) error {
	return s.ProcessCoursePurchaseForPaymentTx(tx, buyerID, "", coursePrice)
}

// ProcessCoursePurchaseForPaymentTx distributes commission at most once per
// payment. The uniqueness of (payment_id, level) at the database level is the
// final guard against two concurrent confirmations of the same purchase.
func (s *AffiliateService) ProcessCoursePurchaseForPaymentTx(tx *gorm.DB, buyerID uint, paymentID string, coursePrice float64) error {
	if tx == nil {
		return nil
	}

	if paymentID != "" {
		var count int64
		if err := tx.Model(&models.ReferralTransaction{}).Where("payment_id = ?", paymentID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
	} else {
		// Legacy callers without a payment key keep the old per-buyer guard.
		var existingTx models.ReferralTransaction
		if err := tx.Where("buyer_id = ?", buyerID).First(&existingTx).Error; err == nil {
			return nil
		}
	}

	var ref models.Referral
	if err := tx.Where("user_id = ?", buyerID).First(&ref).Error; err != nil {
		// Organic purchase without referrer
		return nil
	}

	// Level 1: 15% Commission (1 035 ₽ on 6 900 ₽ base)
	if ref.ReferredByID != nil {
		l1Amount := coursePrice * 0.15
		tx1 := models.ReferralTransaction{
			AffiliateID: *ref.ReferredByID,
			BuyerID:     buyerID,
			Level:       1,
			PaymentID:   paymentID,
			Amount:      l1Amount,
			CreatedAt:   time.Now(),
		}
		if err := tx.Create(&tx1).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.AffiliateProfile{}).
			Where("user_id = ?", *ref.ReferredByID).
			Updates(map[string]interface{}{
				"current_balance": gorm.Expr("current_balance + ?", l1Amount),
				"total_earned":    gorm.Expr("total_earned + ?", l1Amount),
			}).Error; err != nil {
			return err
		}
	}

	// Level 2: 5% Commission (345 ₽ on 6 900 ₽ base)
	if ref.ParentReferrerID != nil {
		l2Amount := coursePrice * 0.05
		tx2 := models.ReferralTransaction{
			AffiliateID: *ref.ParentReferrerID,
			BuyerID:     buyerID,
			Level:       2,
			PaymentID:   paymentID,
			Amount:      l2Amount,
			CreatedAt:   time.Now(),
		}
		if err := tx.Create(&tx2).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.AffiliateProfile{}).
			Where("user_id = ?", *ref.ParentReferrerID).
			Updates(map[string]interface{}{
				"current_balance": gorm.Expr("current_balance + ?", l2Amount),
				"total_earned":    gorm.Expr("total_earned + ?", l2Amount),
			}).Error; err != nil {
			return err
		}
	}

	return nil
}

// ProcessCoursePurchase calculates 15% and 5% commissions upon student enrollment.
func (s *AffiliateService) ProcessCoursePurchase(buyerID uint, coursePrice float64) error {
	if s.db == nil {
		return nil
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.ProcessCoursePurchaseTx(tx, buyerID, coursePrice)
	})
}

// RequestPayout creates a withdrawal request and freezes the funds on balance.
func (s *AffiliateService) RequestPayout(userID uint, amount float64, details string) (*models.PayoutRequest, error) {
	if amount < 1000 {
		return nil, errors.New("минимальная сумма вывода составляет 1 000 ₽")
	}

	var payout models.PayoutRequest
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var profile models.AffiliateProfile
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&profile).Error; err != nil {
			return errors.New("affiliate profile not found")
		}

		if profile.CurrentBalance < amount {
			return errors.New("недостаточно средств на балансе для вывода")
		}

		// Conditional debit: the database itself guarantees the balance can
		// never go negative even if two requests race past the read above.
		res := tx.Model(&models.AffiliateProfile{}).
			Where("user_id = ? AND current_balance >= ?", userID, amount).
			Update("current_balance", gorm.Expr("current_balance - ?", amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return errors.New("недостаточно средств на балансе для вывода")
		}

		payout = models.PayoutRequest{
			UserID:         userID,
			Amount:         amount,
			PaymentDetails: details,
			Status:         "pending",
			CreatedAt:      time.Now(),
		}

		return tx.Create(&payout).Error
	})

	return &payout, err
}

// ApprovePayout marks payout as paid and increments total_withdrawn.
func (s *AffiliateService) ApprovePayout(payoutID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var payout models.PayoutRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", payoutID).First(&payout).Error; err != nil {
			return err
		}

		if payout.Status != "pending" {
			return fmt.Errorf("payout is already in status: %s", payout.Status)
		}

		now := time.Now()
		res := tx.Model(&models.PayoutRequest{}).
			Where("id = ? AND status = ?", payout.ID, "pending").
			Updates(map[string]interface{}{"status": "paid", "resolved_at": &now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("payout is no longer pending")
		}

		return tx.Model(&models.AffiliateProfile{}).
			Where("user_id = ?", payout.UserID).
			Update("total_withdrawn", gorm.Expr("total_withdrawn + ?", payout.Amount)).Error
	})
}

// RejectPayout refunds frozen funds back to current_balance.
func (s *AffiliateService) RejectPayout(payoutID uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var payout models.PayoutRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", payoutID).First(&payout).Error; err != nil {
			return err
		}

		if payout.Status != "pending" {
			return fmt.Errorf("payout is already in status: %s", payout.Status)
		}

		now := time.Now()
		res := tx.Model(&models.PayoutRequest{}).
			Where("id = ? AND status = ?", payout.ID, "pending").
			Updates(map[string]interface{}{"status": "rejected", "resolved_at": &now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("payout is no longer pending")
		}

		// Refund balance
		return tx.Model(&models.AffiliateProfile{}).
			Where("user_id = ?", payout.UserID).
			Update("current_balance", gorm.Expr("current_balance + ?", payout.Amount)).Error
	})
}
