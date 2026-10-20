package database

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.PaymentReceipt{},
		&models.PaymentTransaction{},
		&models.CallBooking{},
		&models.StudentSubmission{},
		&models.AnalyticsEvent{},
	)
	if err != nil {
		t.Fatalf("failed to automigrate models: %v", err)
	}

	return db
}

func TestReconcileHistoricalSales_PureUser(t *testing.T) {
	db := setupTestDB(t)

	// User 1: Has paid_amount = 6900, but no receipt and no transaction records
	user1 := models.User{
		TelegramID: 111111,
		Username:   "student_one",
		Tier:       "accelerator",
		PaidAmount: 6900.0,
		Role:       "user",
	}
	db.Create(&user1)

	ReconcileHistoricalSales(db)

	var txs []models.PaymentTransaction
	db.Where("telegram_id = ?", user1.TelegramID).Find(&txs)
	if len(txs) != 1 {
		t.Fatalf("expected 1 transaction for student 1, got %d", len(txs))
	}
	if txs[0].Amount != 6900.0 {
		t.Errorf("expected transaction amount 6900.0, got %f", txs[0].Amount)
	}
	if txs[0].Status != "SUCCESS" {
		t.Errorf("expected transaction status SUCCESS, got %s", txs[0].Status)
	}
}

func TestReconcileHistoricalSales_UpgradePreservesInitialPurchase(t *testing.T) {
	db := setupTestDB(t)

	// User 2: Paid 6900 initially (recorded only in PaidAmount), then upgraded to VIP with 8000 via receipt
	// Total PaidAmount = 14900, Receipt = 8000
	user2 := models.User{
		TelegramID: 222222,
		Username:   "student_upgrade",
		Tier:       "vip",
		PaidAmount: 14900.0,
		Role:       "student",
	}
	db.Create(&user2)

	receipt := models.PaymentReceipt{
		TelegramID: user2.TelegramID,
		Username:   user2.Username,
		Amount:     8000.0,
		Tier:       "vip",
		Status:     "approved",
		CreatedAt:  time.Now(),
	}
	db.Create(&receipt)

	ReconcileHistoricalSales(db)

	var txs []models.PaymentTransaction
	db.Where("telegram_id = ?", user2.TelegramID).Order("created_at asc").Find(&txs)
	if len(txs) != 2 {
		t.Fatalf("expected 2 transactions (receipt 8000 + reconciled initial 6900), got %d", len(txs))
	}

	var totalAmount float64
	for _, tx := range txs {
		totalAmount += tx.Amount
	}
	if totalAmount != 14900.0 {
		t.Errorf("expected total transaction amount 14900.0, got %f", totalAmount)
	}

	// Running ReconcileHistoricalSales a second time should be idempotent
	ReconcileHistoricalSales(db)
	var txsAfter []models.PaymentTransaction
	db.Where("telegram_id = ?", user2.TelegramID).Find(&txsAfter)
	if len(txsAfter) != 2 {
		t.Fatalf("expected idempotent reconciliation: got %d transactions", len(txsAfter))
	}
}
