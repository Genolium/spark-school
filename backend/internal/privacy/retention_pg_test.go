package privacy

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// privacyPGDB prepares an isolated PostgreSQL schema for the retention tests.
// It skips when TEST_DATABASE_URL is not set, matching the other PostgreSQL
// integration suites in this repository.
func privacyPGDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	lockPath := filepath.Join(os.TempDir(), "spark-pg-integration.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.AnalyticsEvent{}, &models.ExperimentAssignment{},
		&models.DripLead{}, &models.PaymentTransaction{}, &models.CampaignLink{}, &models.CampaignLinkClick{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCleaner_Retention_Postgres(t *testing.T) {
	db := privacyPGDB(t)
	old := time.Now().AddDate(0, 0, -200)
	recent := time.Now()

	link := models.CampaignLink{Slug: "pg-retention", Title: "pg-retention", TargetType: "landing", IsActive: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		visitor string
		created time.Time
	}{{"v-old", old}, {"v-new", recent}} {
		if err := db.Create(&models.AnalyticsEvent{VisitorID: row.visitor, Step: "landing_view", CreatedAt: row.created}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.ExperimentAssignment{Experiment: "hero_cta_copy", Variant: "control", VisitorID: row.visitor, CreatedAt: row.created}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.CampaignLinkClick{LinkID: link.ID, VisitorID: row.visitor, CreatedAt: row.created}).Error; err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := NewCleaner(db, 180).RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 3 {
		t.Fatalf("expected 3 deleted rows, got %d", deleted)
	}

	var oldClicks int64
	db.Model(&models.CampaignLinkClick{}).Where("visitor_id = ?", "v-old").Count(&oldClicks)
	if oldClicks != 0 {
		t.Fatalf("old campaign click must be deleted, got %d", oldClicks)
	}
	var recentClicks int64
	db.Model(&models.CampaignLinkClick{}).Where("visitor_id = ?", "v-new").Count(&recentClicks)
	if recentClicks != 1 {
		t.Fatalf("recent campaign click must be kept, got %d", recentClicks)
	}
}

func TestEraseUser_CampaignClicks_Postgres(t *testing.T) {
	db := privacyPGDB(t)
	now := time.Now()

	user := models.User{
		TelegramID: 777002, Username: "pgstudent", FirstName: "Иван",
		VisitorID: "visitor-pg-erase", AttributionToken: "a_pg_erase", HasAccess: true,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	link := models.CampaignLink{Slug: "pg-erase", Title: "pg-erase", TargetType: "landing", IsActive: true}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}
	for _, click := range []models.CampaignLinkClick{
		{LinkID: link.ID, VisitorID: "visitor-pg-erase", CreatedAt: now},
		{LinkID: link.ID, AttributionToken: "a_pg_erase", CreatedAt: now},
		{LinkID: link.ID, VisitorID: "other", CreatedAt: now},
	} {
		if err := db.Create(&click).Error; err != nil {
			t.Fatal(err)
		}
	}
	payment := models.PaymentTransaction{
		PaymentID: "pg-keep-me", TelegramID: user.TelegramID, Amount: 6900,
		AmountKopecks: 690000, Status: "SUCCESS", CreatedAt: now,
	}
	if err := db.Create(&payment).Error; err != nil {
		t.Fatal(err)
	}

	if err := EraseUser(db, user.ID); err != nil {
		t.Fatal(err)
	}

	var userClicks int64
	db.Model(&models.CampaignLinkClick{}).
		Where("visitor_id = ? OR attribution_token = ?", "visitor-pg-erase", "a_pg_erase").
		Count(&userClicks)
	if userClicks != 0 {
		t.Fatalf("user campaign clicks must be deleted, got %d", userClicks)
	}
	var otherClicks int64
	db.Model(&models.CampaignLinkClick{}).Where("visitor_id = ?", "other").Count(&otherClicks)
	if otherClicks != 1 {
		t.Fatalf("unrelated campaign click must be preserved, got %d", otherClicks)
	}
	var payments int64
	db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", "pg-keep-me").Count(&payments)
	if payments != 1 {
		t.Fatal("financial transaction must be preserved")
	}
}
