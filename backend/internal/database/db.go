package database

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/money"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// RequiredSchemaVersion is the migration version a process needs before it is
// considered ready to serve traffic.
const RequiredSchemaVersion int64 = 27

// MigrationVersion returns the latest applied Goose version.
func MigrationVersion(db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}
	var version int64
	if err := db.Raw("SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version").Scan(&version).Error; err != nil {
		return 0, err
	}
	return version, nil
}

// Ready verifies that PostgreSQL is reachable and the schema is fully migrated.
func Ready(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return 0, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("postgres ping failed: %w", err)
	}
	version, err := MigrationVersion(db)
	if err != nil {
		return 0, fmt.Errorf("migration version check failed: %w", err)
	}
	if version < RequiredSchemaVersion {
		return version, fmt.Errorf("schema version %d is older than required %d", version, RequiredSchemaVersion)
	}
	return version, nil
}

// ValidateSchema fails fast when a critical column expected by the code is
// missing. This catches schema drift that CREATE TABLE IF NOT EXISTS cannot
// repair, with a message that names the exact table and column.
func ValidateSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	checks := []struct {
		table   string
		columns []string
	}{
		{"users", []string{"telegram_id", "has_access", "tier", "paid_amount", "promo_code"}},
		{"payment_transactions", []string{"payment_id", "amount", "status", "promo_code", "visitor_id", "source"}},
		{"payment_receipts", []string{"amount", "status", "promo_code"}},
		{"payment_orders", []string{"order_id", "amount", "status", "promo_code", "visitor_id"}},
		{"course_pricings", []string{"basic_price", "basic_old_price", "accelerator_price", "accelerator_old_price", "vip_price", "vip_old_price", "bank_name"}},
		{"outbox_events", []string{"event_type", "payload", "status", "last_error"}},
		{"experiment_assignments", []string{"experiment", "variant", "visitor_id"}},
		{"bot_sessions", []string{"telegram_id", "state", "price_experiment", "vip_price_experiment"}},
		{"campaign_links", []string{"slug", "title", "target_type", "target_value", "is_active", "created_by"}},
		{"campaign_link_clicks", []string{"link_id", "visitor_id", "attribution_token", "ip_hash"}},
	}
	for _, check := range checks {
		for _, column := range check.columns {
			var exists bool
			if err := db.Raw(
				`SELECT EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_schema = current_schema()
					  AND table_name = ? AND column_name = ?
				)`, check.table, column,
			).Scan(&exists).Error; err != nil {
				return fmt.Errorf("schema check %s.%s failed: %w", check.table, column, err)
			}
			if !exists {
				return fmt.Errorf("required column %s.%s is missing", check.table, column)
			}
		}
	}
	return nil
}

// Init connects to PostgreSQL, applies the canonical Goose migration chain and
// seeds idempotent reference data. Schema changes are never performed here:
// if a migration fails, the process stops instead of running on a partial
// schema (the previous behaviour was fail-open).
func Init(cfg *config.Config) *gorm.DB {
	var err error
	var db *gorm.DB

	// Retry connection loop for Dockerized environments
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			break
		}
		log.Printf("Connecting to database... attempt %d/10: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to retrieve sql.DB instance: %v", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Single canonical migration chain (embedded from internal/database/migrations).
	if err := RunMigrations(sqlDB); err != nil {
		log.Fatalf("Database migrations failed, refusing to start: %v", err)
	}
	if err := ValidateSchema(db); err != nil {
		log.Fatalf("Database schema validation failed, refusing to start: %v", err)
	}
	if err := EnsureAnalyticsPartitions(db, time.Now()); err != nil {
		log.Fatalf("Failed to prepare analytics partitions: %v", err)
	}

	DB = db
	ReconcileHistoricalSalesOnce(db)
	SeedDefaultData(db, cfg)
	return db
}

// SeedDefaultData seeds the admin user, pricing and promo codes. Every insert
// is idempotent; it does not change schema.
func SeedDefaultData(db *gorm.DB, cfg *config.Config) {
	// Seed Admin user if defined
	if cfg.AdminTelegramID != 0 {
		var admin models.User
		if err := db.Where("telegram_id = ?", cfg.AdminTelegramID).First(&admin).Error; err != nil {
			now := time.Now()
			admin = models.User{
				TelegramID:      cfg.AdminTelegramID,
				Username:        cfg.AdminUsername,
				FirstName:       "Илья",
				LastName:        "Васюнин",
				Role:            "admin",
				HasAccess:       true,
				AccessGrantedAt: &now,
			}
			db.Create(&admin)
			db.Create(&models.AffiliateProfile{
				UserID:       admin.ID,
				ReferralCode: "admin",
			})
		}
	}

	// Seed default pricing if not exists
	models.GetCoursePricing(db)

	// Seed default promo codes if not exist
	defaultPromos := []models.PromoCode{
		{Code: "SPARK5", DiscountPercent: 5, OwnerUsername: "admin", RewardAmount: 1000.0, IsActive: true},
		{Code: "START5", DiscountPercent: 5, OwnerUsername: "admin", RewardAmount: 1000.0, IsActive: true},
		{Code: "IL5", DiscountPercent: 5, OwnerUsername: "ilyan_vas", RewardAmount: 1000.0, IsActive: true},
		{Code: "ILYA5", DiscountPercent: 5, OwnerUsername: "ilyan_vas", RewardAmount: 1000.0, IsActive: true},
		{Code: "SMTH_FAMILIAR", DiscountPercent: 5, OwnerUsername: "admin", RewardAmount: 1000.0, IsActive: true},
		{Code: "VOENMEH404", DiscountPercent: 5, OwnerUsername: "admin", RewardAmount: 1000.0, IsActive: true},
		{Code: "DET_ACADEMY", DiscountPercent: 5, OwnerUsername: "admin", RewardAmount: 1000.0, IsActive: true},
		{Code: "JUGA", DiscountPercent: 5, OwnerUsername: "admin", RewardAmount: 1000.0, IsActive: true},
	}
	for _, p := range defaultPromos {
		var existing models.PromoCode
		if err := db.Where("UPPER(code) = ?", strings.ToUpper(p.Code)).First(&existing).Error; err != nil {
			if cfg.AdminTelegramID != 0 {
				p.OwnerTelegramID = cfg.AdminTelegramID
			}
			db.Create(&p)
		}
	}

	log.Println("Seeding complete.")
}

// ReconcileHistoricalSalesOnce runs the legacy revenue reconciliation at most
// once per installation. The marker lives in app_meta (created by migration
// 00004); if the process crashes mid-run, the next start retries because all
// inserts are idempotent.
func ReconcileHistoricalSalesOnce(db *gorm.DB) {
	if db == nil {
		return
	}
	const key = "historical_reconciliation_v1"
	var value string
	err := db.Raw("SELECT value FROM app_meta WHERE key = ?", key).Scan(&value).Error
	if err == nil && value == "done" {
		return
	}
	// Claim the marker without blocking concurrent starts.
	_ = db.Exec("INSERT INTO app_meta (key, value, updated_at) VALUES (?, ?, NOW()) ON CONFLICT (key) DO NOTHING", key, "running").Error
	ReconcileHistoricalSales(db)
	_ = db.Exec("UPDATE app_meta SET value = ?, updated_at = NOW() WHERE key = ?", "done", key).Error
}

// ReconcileHistoricalSales reconciles and backfills historical sales into payment_transactions
// using strict mathematical difference logic to prevent losing initial purchases when upgrades exist.
func ReconcileHistoricalSales(db *gorm.DB) {
	if db == nil {
		return
	}

	// 1. Backfill all approved receipts first
	var receipts []models.PaymentReceipt
	db.Where("status = 'approved' AND amount > 0").Find(&receipts)
	for _, pr := range receipts {
		receiptPID := fmt.Sprintf("RECEIPT_%d", pr.ID)
		var count int64
		db.Model(&models.PaymentTransaction{}).Where("payment_id = ?", receiptPID).Count(&count)
		if count == 0 {
			tier := strings.ToLower(strings.TrimSpace(pr.Tier))
			if tier == "" {
				tier = "accelerator"
			}
			// The receipt itself carries no visitor attribution, but the buyer
			// does: linking them keeps the funnel connected to the payment.
			var owner models.User
			visitorID := ""
			attributionToken := ""
			if err := db.Where("telegram_id = ?", pr.TelegramID).First(&owner).Error; err == nil {
				visitorID = owner.VisitorID
				attributionToken = owner.AttributionToken
			}
			pt := models.PaymentTransaction{
				PaymentID:        receiptPID,
				TelegramID:       pr.TelegramID,
				Username:         pr.Username,
				Amount:           pr.Amount,
				AmountKopecks:    int64(money.FromRubles(pr.Amount)),
				Currency:         "RUB",
				Tier:             tier,
				Status:           "SUCCESS",
				PayMethod:        "sbp_receipt",
				Source:           "sbp_receipt",
				PromoCode:        pr.PromoCode,
				VisitorID:        visitorID,
				AttributionToken: attributionToken,
				CreatedAt:        pr.CreatedAt,
			}
			if err := db.Create(&pt).Error; err == nil {
				// A financial operation without a funnel event is reported by
				// the monitoring check as an unconfirmed payment.
				EnsurePaymentConfirmedEvent(db, pt)
			}
		}
	}

	// 2. For every user with paid_amount > 0 (even if access was subsequently revoked):
	var users []models.User
	db.Where("paid_amount > 0 AND role != 'admin'").Find(&users)
	for _, u := range users {
		type SumResult struct {
			Total float64
		}
		var sumRes SumResult
		db.Model(&models.PaymentTransaction{}).
			Select("COALESCE(SUM(amount), 0) as total").
			Where("telegram_id = ? AND status = 'SUCCESS'", u.TelegramID).
			Scan(&sumRes)

		diff := u.PaidAmount - sumRes.Total
		if diff > 0.01 {
			histPID := fmt.Sprintf("HIST_USER_%d", u.ID)
			var existing models.PaymentTransaction
			if err := db.Where("payment_id = ?", histPID).First(&existing).Error; err != nil {
				txTime := u.CreatedAt
				if u.AccessGrantedAt != nil {
					txTime = *u.AccessGrantedAt
				}
				tier := strings.ToLower(strings.TrimSpace(u.Tier))
				if tier == "" {
					tier = "accelerator"
				}
				pt := models.PaymentTransaction{
					PaymentID:     histPID,
					TelegramID:    u.TelegramID,
					Username:      u.Username,
					Amount:        diff,
					AmountKopecks: int64(money.FromRubles(diff)),
					Currency:      "RUB",
					Tier:          tier,
					Status:        "SUCCESS",
					PayMethod:     "historical_access",
					Source:        "historical",
					PromoCode:     u.PromoCode,
					CreatedAt:     txTime,
				}
				db.Create(&pt)
			}
		}
	}
}
