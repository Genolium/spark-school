package payments

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// These tests require a real PostgreSQL instance. They are skipped unless
// TEST_DATABASE_URL is set, e.g.:
//
//	TEST_DATABASE_URL='postgres://spark@127.0.0.1:55432/spark_test?sslmode=disable' go test -race ./internal/payments
//
// SQLite is deliberately not used here: row locks, ON CONFLICT and partial
// unique indexes must be verified by the database that runs in production.

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	prepareSchema(t, dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.AffiliateProfile{},
		&models.Referral{},
		&models.ReferralTransaction{},
		&models.PaymentReceipt{},
		&models.PaymentTransaction{},
		&models.PaymentOrder{},
		&models.PromoCode{},
		&models.OutboxEvent{},
		&models.AnalyticsEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_payment_level ON referral_transactions(payment_id, level) WHERE payment_id <> ''").Error; err != nil {
		t.Fatalf("referral unique index: %v", err)
	}
	return db
}

func newService(db *gorm.DB) *Service {
	return NewService(db, services.NewAffiliateService(db))
}

func pendingReceipt(t *testing.T, db *gorm.DB, telegramID int64, tier string, amount float64) models.PaymentReceipt {
	t.Helper()
	receipt := models.PaymentReceipt{
		FileHash:   fmt.Sprintf("hash-%d-%d", telegramID, time.Now().UnixNano()),
		TelegramID: telegramID,
		Username:   "student",
		Tier:       tier,
		Amount:     amount,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	if err := db.Create(&receipt).Error; err != nil {
		t.Fatalf("create receipt: %v", err)
	}
	return receipt
}

func seedUser(t *testing.T, db *gorm.DB, telegramID int64) models.User {
	t.Helper()
	u := models.User{TelegramID: telegramID, Username: "seed", FirstName: "Seed", Role: "student"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func receiptInput(r models.PaymentReceipt) ConfirmInput {
	return ConfirmInput{
		Source:     "sbp_receipt",
		PaymentID:  fmt.Sprintf("RECEIPT_%d", r.ID),
		ReceiptID:  r.ID,
		TelegramID: r.TelegramID,
		Tier:       r.Tier,
		Amount:     r.Amount,
		Currency:   "RUB",
		PayMethod:  "sbp_receipt",
		CreatedAt:  r.CreatedAt,
	}
}

func TestPostgres_ParallelReceiptApproval(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 1001)
	receipt := pendingReceipt(t, db, 1001, "accelerator", 6900)

	var wg sync.WaitGroup
	errs := make([]error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = svc.Confirm(receiptInput(receipt))
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("confirmation %d failed: %v", i, err)
		}
	}

	var tx models.PaymentTransaction
	if err := db.Where("payment_id = ?", fmt.Sprintf("RECEIPT_%d", receipt.ID)).First(&tx).Error; err != nil {
		t.Fatal(err)
	}
	if tx.AmountKopecks != 690000 {
		t.Fatalf("transaction must store integer kopecks, got %d", tx.AmountKopecks)
	}
	var user models.User
	db.Where("telegram_id = ?", 1001).First(&user)
	if user.PaidAmountKopecks != 690000 {
		t.Fatalf("expected 690000 kopecks, got %d", user.PaidAmountKopecks)
	}
	if user.PaidAmount != 6900 || !user.HasAccess || user.Tier != "accelerator" {
		t.Fatalf("unexpected user after parallel approval: %+v", user)
	}
	var notices int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'payment_notice'").Count(&notices)
	if notices != 2 {
		t.Fatalf("expected one user and one admin notice, got %d", notices)
	}
	var analytics int64
	db.Model(&models.AnalyticsEvent{}).Where("step = 'payment_confirmed'").Count(&analytics)
	if analytics != 1 {
		t.Fatalf("expected exactly one payment_confirmed event, got %d", analytics)
	}
}

func TestPostgres_RepeatedGatewayPayment(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 2001)

	in := ConfirmInput{
		Source:     "yookassa",
		PaymentID:  "yk-payment-1",
		OrderID:    "order-1",
		TelegramID: 2001,
		Tier:       "accelerator",
		Amount:     6900,
		Currency:   "RUB",
		PayMethod:  "YOOKASSA",
		CreatedAt:  time.Now(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Confirm(in); err != nil {
				t.Errorf("concurrent confirm: %v", err)
			}
		}()
	}
	wg.Wait()

	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", in.PaymentID).Count(&txCount)
	if txCount != 1 {
		t.Fatalf("expected one transaction, got %d", txCount)
	}
	var user models.User
	db.Where("telegram_id = ?", 2001).First(&user)
	if user.PaidAmount != 6900 {
		t.Fatalf("paid amount changed on replay: %f", user.PaidAmount)
	}
}

func TestPostgres_InitialPurchaseAndUpgrade(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 3001)

	first, err := svc.Confirm(ConfirmInput{Source: "tribute", PaymentID: "t-1", TelegramID: 3001, Tier: "basic", Amount: 3990, Currency: "RUB", CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("basic purchase: %v", err)
	}
	upgrade, err := svc.Confirm(ConfirmInput{Source: "tribute", PaymentID: "t-2", TelegramID: 3001, Tier: "vip", Amount: 11000, Currency: "RUB", CreatedAt: time.Now()})
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if first.AlreadyProcessed || upgrade.AlreadyProcessed {
		t.Fatal("new payments must not be treated as replays")
	}
	var user models.User
	db.Where("telegram_id = ?", 3001).First(&user)
	if user.Tier != "vip" {
		t.Fatalf("expected vip after upgrade, got %s", user.Tier)
	}
	if user.PaidAmountKopecks != 1499000 {
		t.Fatalf("expected 1499000 kopecks, got %d", user.PaidAmountKopecks)
	}
	if user.PaidAmount != 14990 {
		t.Fatalf("expected cumulative paid amount 14990, got %f", user.PaidAmount)
	}
	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("telegram_id = ?", 3001).Count(&txCount)
	if txCount != 2 {
		t.Fatalf("expected two financial operations, got %d", txCount)
	}
}

func TestPostgres_TwoEssayAudits(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 4001)

	for i := 0; i < 2; i++ {
		if _, err := svc.Confirm(ConfirmInput{
			Source: "sbp_receipt", PaymentID: fmt.Sprintf("audit-%d", i), TelegramID: 4001,
			Tier: "audit_essay", Amount: 1900, Currency: "RUB", AuditCredits: 1, CreatedAt: time.Now(),
		}); err != nil {
			t.Fatalf("audit %d: %v", i, err)
		}
	}
	var user models.User
	db.Where("telegram_id = ?", 4001).First(&user)
	if user.AuditCredits != 2 {
		t.Fatalf("expected two audit credits, got %d", user.AuditCredits)
	}
	if user.PaidAmountKopecks != 380000 {
		t.Fatalf("expected 380000 kopecks for two audits, got %d", user.PaidAmountKopecks)
	}
	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("telegram_id = ?", 4001).Count(&txCount)
	if txCount != 2 {
		t.Fatalf("expected two audit payments, got %d", txCount)
	}
}

func TestPostgres_RollbackWhenOutboxFails(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 5001)
	receipt := pendingReceipt(t, db, 5001, "accelerator", 6900)

	if err := db.Exec("ALTER TABLE outbox_events RENAME TO outbox_events_broken").Error; err != nil {
		t.Fatalf("rename outbox: %v", err)
	}
	if _, err := svc.Confirm(receiptInput(receipt)); err == nil {
		t.Fatal("expected confirmation to fail when outbox insert fails")
	}
	if err := db.Exec("ALTER TABLE outbox_events_broken RENAME TO outbox_events").Error; err != nil {
		t.Fatalf("restore outbox: %v", err)
	}

	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", fmt.Sprintf("RECEIPT_%d", receipt.ID)).Count(&txCount)
	if txCount != 0 {
		t.Fatalf("financial operation must be rolled back, got %d", txCount)
	}
	var reloaded models.PaymentReceipt
	db.First(&reloaded, receipt.ID)
	if reloaded.Status != "pending" {
		t.Fatalf("receipt must remain pending after rollback, got %s", reloaded.Status)
	}
	var user models.User
	db.Where("telegram_id = ?", 5001).First(&user)
	if user.HasAccess || user.PaidAmount != 0 {
		t.Fatalf("rights must not be granted on rollback: %+v", user)
	}

	// A retry after the transient failure must succeed exactly once.
	if _, err := svc.Confirm(receiptInput(receipt)); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
	db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", fmt.Sprintf("RECEIPT_%d", receipt.ID)).Count(&txCount)
	if txCount != 1 {
		t.Fatalf("expected one transaction after retry, got %d", txCount)
	}
}

func TestPostgres_RepeatedDeliveryDoesNotChangeMoney(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 6001)
	receipt := pendingReceipt(t, db, 6001, "accelerator", 6900)
	if _, err := svc.Confirm(receiptInput(receipt)); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := svc.RequeueAccessNotification(fmt.Sprintf("RECEIPT_%d", receipt.ID)); err != nil {
			t.Fatalf("requeue: %v", err)
		}
	}
	var user models.User
	db.Where("telegram_id = ?", 6001).First(&user)
	if user.PaidAmount != 6900 {
		t.Fatalf("repeated delivery changed money: %f", user.PaidAmount)
	}
	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", fmt.Sprintf("RECEIPT_%d", receipt.ID)).Count(&txCount)
	if txCount != 1 {
		t.Fatalf("repeated delivery created %d transactions", txCount)
	}
	var notices int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'payment_notice'").Count(&notices)
	if notices != 5 { // 2 original + 3 requeued
		t.Fatalf("expected 5 notices, got %d", notices)
	}
}

func TestPostgres_MissingRecipient(t *testing.T) {
	db := testDB(t)
	svc := newService(db)

	res, err := svc.Confirm(ConfirmInput{
		Source: "yookassa", PaymentID: "no-recipient", TelegramID: 0,
		Tier: "accelerator", Amount: 6900, Currency: "RUB", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("confirm without recipient: %v", err)
	}
	if res.AccessGranted {
		t.Fatal("access must not be granted without a recipient")
	}
	var tx models.PaymentTransaction
	if err := db.Where("payment_id = ?", "no-recipient").First(&tx).Error; err != nil {
		t.Fatalf("flagged transaction must be recorded: %v", err)
	}
	if tx.Status != "FLAGGED_NO_RECIPIENT" {
		t.Fatalf("unexpected status: %s", tx.Status)
	}
	var users int64
	db.Model(&models.User{}).Count(&users)
	if users != 0 {
		t.Fatalf("no user should be created without telegram id, got %d", users)
	}
}

func TestPostgres_UnknownTelegramUserIsCreated(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	res, err := svc.Confirm(ConfirmInput{
		Source: "antilopay", PaymentID: "new-user", TelegramID: 7001,
		Username: "newbie", Tier: "basic", Amount: 3990, Currency: "RUB", CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if res.UserID == 0 {
		t.Fatal("expected created user")
	}
	var user models.User
	if err := db.Where("telegram_id = ?", 7001).First(&user).Error; err != nil {
		t.Fatalf("user not created: %v", err)
	}
	if !user.HasAccess || user.Tier != "basic" || user.PaidAmount != 3990 {
		t.Fatalf("unexpected user: %+v", user)
	}
	if user.OnboardingStatus != "pending_intro" {
		t.Fatalf("new purchase must start onboarding, got %q", user.OnboardingStatus)
	}
}

func TestPostgres_PromoUsageCountedOnce(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	seedUser(t, db, 8001)
	promo := models.PromoCode{Code: "SPARK5", DiscountPercent: 5, IsActive: true}
	if err := db.Create(&promo).Error; err != nil {
		t.Fatalf("promo: %v", err)
	}

	in := ConfirmInput{Source: "tribute", PaymentID: "promo-1", TelegramID: 8001, Tier: "accelerator", Amount: 6555, Currency: "RUB", PromoCode: "SPARK5", CreatedAt: time.Now()}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Confirm(in); err != nil {
				t.Errorf("confirm: %v", err)
			}
		}()
	}
	wg.Wait()
	var reloaded models.PromoCode
	db.Where("code = ?", "SPARK5").First(&reloaded)
	if reloaded.UsesCount != 1 {
		t.Fatalf("promo usage must be one, got %d", reloaded.UsesCount)
	}
}

func TestPostgres_AffiliateCommissionOncePerPayment(t *testing.T) {
	db := testDB(t)
	svc := newService(db)
	referrer := seedUser(t, db, 9001)
	buyer := seedUser(t, db, 9002)
	db.Create(&models.AffiliateProfile{UserID: referrer.ID, ReferralCode: "REF9001"})
	db.Create(&models.Referral{UserID: buyer.ID, ReferredByID: &referrer.ID})

	in := ConfirmInput{Source: "tribute", PaymentID: "aff-1", TelegramID: 9002, Tier: "accelerator", Amount: 6900, Currency: "RUB", CreatedAt: time.Now()}
	if _, err := svc.Confirm(in); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := svc.Confirm(in); err != nil {
		t.Fatalf("replay: %v", err)
	}
	var commissions int64
	db.Model(&models.ReferralTransaction{}).Where("payment_id = ?", "aff-1").Count(&commissions)
	if commissions != 1 {
		t.Fatalf("expected one level-1 commission, got %d", commissions)
	}
	var profile models.AffiliateProfile
	db.Where("user_id = ?", referrer.ID).First(&profile)
	if profile.CurrentBalance != 1035 {
		t.Fatalf("expected 1035 commission, got %f", profile.CurrentBalance)
	}
	var commission models.ReferralTransaction
	db.Where("payment_id = ?", "aff-1").First(&commission)
	if commission.AmountKopecks != 103500 {
		t.Fatalf("expected integer commission 103500, got %d", commission.AmountKopecks)
	}
}
