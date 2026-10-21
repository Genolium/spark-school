package bot

import (
	"os"
	"testing"
	"time"

	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func recoveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	prepareSchema(t, dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.AnalyticsEvent{}, &models.OrderIntent{},
		&models.OutboxEvent{}, &models.PaymentTransaction{}, &models.DripLead{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPostgres_Recovery_AbandonedOrderTwoReminders(t *testing.T) {
	db := recoveryTestDB(t)
	user := basicStudent(t, db, 31001, 0)
	now := time.Now()
	intent := models.OrderIntent{
		UserID: user.ID, TelegramID: user.TelegramID, Tier: "accelerator",
		Amount: 6900, Status: "pending", CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now,
	}
	if err := db.Create(&intent).Error; err != nil {
		t.Fatal(err)
	}
	service := &BotService{db: db, cfg: &config.Config{}}

	if err := service.processAbandonedOrders(now); err != nil {
		t.Fatal(err)
	}
	if err := service.processAbandonedOrders(now); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 1 {
		t.Fatalf("expected one reminder within 20h window, got %d", count)
	}

	// Move the first reminder back in time, the second stage becomes due.
	db.Model(&models.OrderIntent{}).Where("id = ?", intent.ID).
		Updates(map[string]interface{}{"last_reminder_at": now.Add(-21 * time.Hour)})
	if err := service.processAbandonedOrders(now); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 2 {
		t.Fatalf("expected second reminder, got %d", count)
	}
	// Never more than two.
	db.Model(&models.OrderIntent{}).Where("id = ?", intent.ID).
		Updates(map[string]interface{}{"last_reminder_at": now.Add(-21 * time.Hour)})
	if err := service.processAbandonedOrders(now); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 2 {
		t.Fatalf("reminder cap exceeded: %d", count)
	}
}

func TestPostgres_Recovery_OptOutAndPaidIntent(t *testing.T) {
	db := recoveryTestDB(t)
	now := time.Now()
	optedOut := basicStudent(t, db, 32001, 0)
	db.Model(&models.User{}).Where("id = ?", optedOut.ID).Update("reminders_opt_out", true)
	db.Create(&models.OrderIntent{UserID: optedOut.ID, TelegramID: optedOut.TelegramID, Tier: "basic", Amount: 3990, Status: "pending", CreatedAt: now.Add(-3 * time.Hour)})

	paid := basicStudent(t, db, 32002, 0)
	db.Create(&models.OrderIntent{UserID: paid.ID, TelegramID: paid.TelegramID, Tier: "basic", Amount: 3990, Status: "pending", CreatedAt: now.Add(-3 * time.Hour)})
	db.Create(&models.PaymentTransaction{PaymentID: "recovery-paid", TelegramID: paid.TelegramID, Tier: "basic", Amount: 3990, Status: "SUCCESS", Currency: "RUB", CreatedAt: now.Add(-time.Hour)})

	service := &BotService{db: db, cfg: &config.Config{}}
	if err := service.processAbandonedOrders(now); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 0 {
		t.Fatalf("expected no reminders for opted-out/paid intents, got %d", count)
	}
	var optedIntent models.OrderIntent
	db.Where("telegram_id = ?", optedOut.TelegramID).First(&optedIntent)
	if optedIntent.Status != "cancelled" {
		t.Fatalf("opted-out intent should be cancelled, got %s", optedIntent.Status)
	}
	var paidIntent models.OrderIntent
	db.Where("telegram_id = ?", paid.TelegramID).First(&paidIntent)
	if paidIntent.Status != "paid" {
		t.Fatalf("paid intent should be closed, got %s", paidIntent.Status)
	}
}

func TestPostgres_Recovery_AbandonedAndCompletedDiagnostic(t *testing.T) {
	db := recoveryTestDB(t)
	now := time.Now()

	started := basicStudent(t, db, 33001, 0)
	db.Model(&models.User{}).Where("id = ?", started.ID).Update("attribution_token", "a_started")
	db.Create(&models.AnalyticsEvent{VisitorID: "v-started", AttributionToken: "a_started", Step: "diagnostic_start", CreatedAt: now.Add(-2 * time.Hour)})

	completed := basicStudent(t, db, 33002, 0)
	db.Model(&models.User{}).Where("id = ?", completed.ID).Update("attribution_token", "a_completed")
	db.Create(&models.AnalyticsEvent{VisitorID: "v-completed", AttributionToken: "a_completed", Step: "diagnostic_complete", CreatedAt: now.Add(-7 * time.Hour)})

	service := &BotService{db: db, cfg: &config.Config{}}
	if err := service.processDiagnosticRecovery(now); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 2 {
		t.Fatalf("expected one started and one completed reminder, got %d", count)
	}
	// Repeat must not duplicate.
	if err := service.processDiagnosticRecovery(now); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 2 {
		t.Fatalf("diagnostic reminders duplicated: %d", count)
	}
	var startedUser, completedUser models.User
	db.First(&startedUser, started.ID)
	db.First(&completedUser, completed.ID)
	if !startedUser.DiagnosticStartedReminderSent || !completedUser.DiagnosticCompletedReminderSent {
		t.Fatal("diagnostic reminder flags were not persisted")
	}
}
