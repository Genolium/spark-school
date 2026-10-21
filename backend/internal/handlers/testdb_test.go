package handlers

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupHandlerDB creates an isolated in-memory database for handler tests.
// PostgreSQL-specific concurrency guarantees are covered separately by the
// payments integration suite (see internal/payments/postgres_test.go).
func setupHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:handler_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.PaymentReceipt{},
		&models.PaymentTransaction{},
		&models.PaymentOrder{},
		&models.PromoCode{},
		&models.AffiliateProfile{},
		&models.Referral{},
		&models.ReferralTransaction{},
		&models.OutboxEvent{},
		&models.AnalyticsEvent{},
		&models.ExperimentAssignment{},
		&models.CallBooking{},
		&models.StudentSubmission{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_experiment_visitor_unique ON experiment_assignments(experiment, visitor_id) WHERE visitor_id <> ''").Error; err != nil {
		t.Fatalf("failed to create experiment index: %v", err)
	}
	return db
}
