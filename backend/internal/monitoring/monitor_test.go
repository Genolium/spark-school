package monitoring

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/database"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/security"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testMonitorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:monitor_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.OutboxEvent{}, &models.PaymentTransaction{}, &models.AnalyticsEvent{}, &models.BackupHeartbeat{}, &models.User{}, &models.PaymentReceipt{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMonitor_AlertsAndCooldown(t *testing.T) {
	db := testMonitorDB(t)
	now := time.Now()
	db.Create(&models.OutboxEvent{EventType: "telegram_notification", Payload: "{}", Status: "failed", CreatedAt: now})
	db.Create(&models.PaymentTransaction{PaymentID: "missing-event", Status: "SUCCESS", Amount: 6900, AmountKopecks: 690000, CreatedAt: now.Add(-20 * time.Minute)})
	db.Create(&models.BackupHeartbeat{ID: 1, LastSuccessAt: now.Add(-30 * time.Hour)})
	metrics := security.NewMetrics()
	metrics.RecordWebhook5xx()
	metrics.RecordWebhook5xx()

	var messages []string
	monitor := New(db, 42, func(_ int64, message string) error {
		messages = append(messages, message)
		return nil
	}, metrics, func(context.Context) error { return fmt.Errorf("database not ready") })
	monitor.cooldown = time.Hour

	alerts := monitor.CheckOnce(context.Background())
	if len(alerts) != 5 {
		t.Fatalf("expected 5 alerts (outbox, payment, backup, readiness, webhook), got %d: %v", len(alerts), alerts)
	}
	if len(messages) != 5 {
		t.Fatalf("expected 5 Telegram messages, got %d", len(messages))
	}

	// Same broken state must not spam during the cooldown.
	if second := monitor.CheckOnce(context.Background()); len(second) != 0 {
		t.Fatalf("expected cooldown to suppress alerts, got %v", second)
	}
}

func TestMonitor_TelegramDeliveryDisabled(t *testing.T) {
	db := testMonitorDB(t)
	if err := db.Create(&models.OutboxEvent{EventType: "telegram_notification", Status: "failed", MaxRetries: 5}).Error; err != nil {
		t.Fatal(err)
	}

	calls := 0
	monitor := New(db, 0, func(int64, string) error {
		calls++
		return nil
	}, security.NewMetrics(), func(context.Context) error { return nil })

	alerts := monitor.CheckOnce(context.Background())
	if len(alerts) == 0 {
		t.Fatal("expected alerts to still be produced for logs")
	}
	if calls != 0 {
		t.Fatalf("monitoring must not send Telegram messages when disabled, got %d calls", calls)
	}
}

func TestMonitor_LegacyReconciledReceiptIsNotReportedAsUnconfirmed(t *testing.T) {
	db := testMonitorDB(t)
	now := time.Now()

	user := models.User{TelegramID: 555, Username: "legacy_buyer", Tier: "accelerator", PaidAmount: 6900, Role: "student"}
	db.Create(&user)
	// A receipt approved by the legacy code path: the financial operation is
	// reconciled from `payment_receipts`, so the funnel event has to be
	// created together with it.
	db.Create(&models.PaymentReceipt{
		TelegramID: user.TelegramID,
		Username:   user.Username,
		Amount:     6900,
		Tier:       "accelerator",
		Status:     "approved",
		CreatedAt:  now.Add(-time.Hour),
	})
	db.Create(&models.BackupHeartbeat{ID: 1, LastSuccessAt: now.Add(-time.Hour)})

	database.ReconcileHistoricalSales(db)

	monitor := New(db, 42, nil, security.NewMetrics(), func(context.Context) error { return nil })
	for _, alert := range monitor.CheckOnce(context.Background()) {
		if strings.Contains(alert, "payment_confirmed") {
			t.Fatalf("reconciled receipt must not be reported as unconfirmed: %s", alert)
		}
	}
}

func TestMonitor_HealthyStateHasNoAlerts(t *testing.T) {
	db := testMonitorDB(t)
	now := time.Now()
	db.Create(&models.PaymentTransaction{PaymentID: "ok", Status: "SUCCESS", Amount: 6900, AmountKopecks: 690000, CreatedAt: now})
	db.Create(&models.AnalyticsEvent{Step: "payment_confirmed", PaymentID: "ok", CreatedAt: now})
	db.Create(&models.BackupHeartbeat{ID: 1, LastSuccessAt: now.Add(-time.Hour)})
	monitor := New(db, 42, nil, security.NewMetrics(), func(context.Context) error { return nil })
	if alerts := monitor.CheckOnce(context.Background()); len(alerts) != 0 {
		t.Fatalf("expected no alerts, got %v", alerts)
	}
}
