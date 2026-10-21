package monitoring

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
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
	if err := db.AutoMigrate(&models.OutboxEvent{}, &models.PaymentTransaction{}, &models.AnalyticsEvent{}, &models.BackupHeartbeat{}); err != nil {
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
