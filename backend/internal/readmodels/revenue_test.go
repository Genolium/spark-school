package readmodels

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgres_RevenueRollup(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	prepareSchema(t, dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.PaymentTransaction{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE admin_revenue_daily (
		day DATE NOT NULL, tier VARCHAR(32) NOT NULL, source VARCHAR(64) NOT NULL DEFAULT '',
		promo_code VARCHAR(32) NOT NULL DEFAULT '', purchases BIGINT NOT NULL DEFAULT 0,
		revenue_kopecks BIGINT NOT NULL DEFAULT 0, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (day, tier, source, promo_code))`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	db.Create(&models.PaymentTransaction{PaymentID: "r1", Tier: "vip", Amount: 14900, AmountKopecks: 1490000, Status: "SUCCESS", Source: "tribute", PromoCode: "SPARK5", CreatedAt: now})
	db.Create(&models.PaymentTransaction{PaymentID: "r2", Tier: "vip", Amount: 14900, AmountKopecks: 1490000, Status: "SUCCESS", Source: "tribute", PromoCode: "SPARK5", CreatedAt: now})
	db.Create(&models.PaymentTransaction{PaymentID: "r3", Tier: "basic", Amount: 3990, AmountKopecks: 399000, Status: "FAIL", CreatedAt: now})
	// Legacy reconciliation rows only had rubles; the rollup must still count them.
	db.Create(&models.PaymentTransaction{PaymentID: "legacy", Tier: "vip", Amount: 6900, AmountKopecks: 0, Status: "SUCCESS", Source: "tribute", PromoCode: "SPARK5", CreatedAt: now})

	worker := NewRevenueWorker(db)
	if err := worker.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	var rows []RevenueDaily
	db.Table("admin_revenue_daily").Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("expected one rollup row, got %d", len(rows))
	}
	if rows[0].Purchases != 3 || rows[0].RevenueKopecks != 3670000 || rows[0].Tier != "vip" {
		t.Fatalf("unexpected rollup: %+v", rows[0])
	}
}
