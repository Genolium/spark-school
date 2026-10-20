package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init establishes connection with PostgreSQL and runs migrations.
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

	// Run Production Database Migrations via Goose
	if err := RunMigrations(sqlDB); err != nil {
		log.Printf("[WARN] Goose migration failed (falling back to GORM AutoMigrate): %v", err)
	}

	// Direct DDL to guarantee critical tables exist immediately
	db.Exec(`
		CREATE TABLE IF NOT EXISTS drip_leads (
			id BIGSERIAL PRIMARY KEY,
			telegram_id BIGINT UNIQUE NOT NULL,
			username VARCHAR(64),
			first_name VARCHAR(128),
			stage INT DEFAULT 0,
			calculated_score INT DEFAULT 0,
			next_trigger_at TIMESTAMP WITH TIME ZONE NOT NULL,
			is_completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_drip_leads_next_trigger_at ON drip_leads(next_trigger_at);
		CREATE INDEX IF NOT EXISTS idx_drip_leads_is_completed ON drip_leads(is_completed);

		CREATE TABLE IF NOT EXISTS payment_transactions (
			id BIGSERIAL PRIMARY KEY,
			payment_id VARCHAR(64) UNIQUE NOT NULL,
			order_id VARCHAR(100),
			telegram_id BIGINT,
			username VARCHAR(64),
			tier VARCHAR(32) DEFAULT 'accelerator',
			amount NUMERIC(10, 2) NOT NULL,
			original_amount NUMERIC(10, 2),
			fee NUMERIC(10, 2),
			status VARCHAR(32) NOT NULL,
			currency VARCHAR(10) DEFAULT 'RUB',
			pay_method VARCHAR(32),
			product_name VARCHAR(128),
			promo_code VARCHAR(32),
			customer_email VARCHAR(128),
			customer_phone VARCHAR(64),
			raw_callback TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_payment_transactions_telegram_id ON payment_transactions(telegram_id);
		CREATE INDEX IF NOT EXISTS idx_payment_transactions_order_id ON payment_transactions(order_id);
	`)

	// Run GORM Auto-Migrations per-model so single constraint warnings don't block subsequent models
	modelsToMigrate := []any{
		&models.User{},
		&models.Referral{},
		&models.AffiliateProfile{},
		&models.ReferralTransaction{},
		&models.PayoutRequest{},
		&models.PromoCode{},
		&models.PaymentReceipt{},
		&models.OutboxEvent{},
		&models.PaymentTransaction{},
		&models.DripLead{},
		&models.CoursePricing{},
		&models.FiscalReceipt{},
		&models.StudentSubmission{},
		&models.StudentProgress{},
		&models.CallBooking{},
		&models.ProcessedTelegramUpdate{},
		&models.AnalyticsEvent{},
	}
	for _, m := range modelsToMigrate {
		if mErr := db.AutoMigrate(m); mErr != nil {
			log.Printf("[Database AutoMigrate] Note on %T: %v", m, mErr)
		}
	}

	// Backfill existing active students with defaults if empty
	db.Exec("UPDATE users SET tier = 'accelerator' WHERE tier IS NULL OR tier = ''")

	// Ensure basic_price and basic_old_price columns exist in course_pricings
	db.Exec("ALTER TABLE IF EXISTS course_pricings ADD COLUMN IF NOT EXISTS basic_price NUMERIC(10,2) DEFAULT 3990.00;")
	db.Exec("ALTER TABLE IF EXISTS course_pricings ADD COLUMN IF NOT EXISTS basic_old_price NUMERIC(10,2) DEFAULT 5900.00;")
	db.Exec("UPDATE course_pricings SET basic_price = 3990.00 WHERE basic_price IS NULL OR basic_price <= 0;")
	db.Exec("UPDATE course_pricings SET basic_old_price = 5900.00 WHERE basic_old_price IS NULL OR basic_old_price <= 0;")

	// Ensure upsell audit columns exist in users table
	db.Exec("ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS upsell_audit_sent BOOLEAN DEFAULT FALSE;")
	db.Exec("ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS upsell_audit_sent_at TIMESTAMP WITH TIME ZONE;")
	db.Exec("ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS has_audit_upsell BOOLEAN DEFAULT FALSE;")
	db.Exec("ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS audit_credits INT DEFAULT 0;")
	db.Exec("UPDATE users SET audit_credits = 1 WHERE has_audit_upsell = true AND audit_credits = 0;")

	// Enforce DB-level unique partial indexes to physically prevent double booking and multi-booking race conditions
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_slot_key ON call_bookings (slot_key) WHERE status = 'confirmed' AND cancelled_at IS NULL;")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_user_booking ON call_bookings (telegram_id) WHERE status = 'confirmed' AND cancelled_at IS NULL;")

	// Ensure receipt_url and receipt_number columns exist in users table
	db.Exec("ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_url VARCHAR(512);")
	db.Exec("ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_number VARCHAR(64);")

	// Backfill users from payment_receipts if missing in users table
	db.Exec(`
		INSERT INTO users (telegram_id, username, first_name, role, has_access, tier, paid_amount, created_at)
		SELECT DISTINCT pr.telegram_id, pr.username, COALESCE(NULLIF(pr.username, ''), 'Студент'), 'student', 
		       (pr.status = 'approved'), pr.tier, 
		       CASE WHEN pr.status = 'approved' THEN pr.amount ELSE 0 END, 
		       pr.created_at
		FROM payment_receipts pr
		WHERE pr.telegram_id > 0 
		  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.telegram_id = pr.telegram_id)
	`)

	// Perform robust historical reconciliation into payment_transactions
	ReconcileHistoricalSales(db)

	// Clean up legacy LMS tables if they exist from prior schema
	_ = db.Migrator().DropTable("user_progress", "lessons", "modules")

	DB = db
	SeedDefaultData(db, cfg)
	return db
}

// SeedDefaultData seeds the admin user if defined.
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
			pt := models.PaymentTransaction{
				PaymentID:  receiptPID,
				TelegramID: pr.TelegramID,
				Username:   pr.Username,
				Amount:     pr.Amount,
				Currency:   "RUB",
				Tier:       tier,
				Status:     "SUCCESS",
				PayMethod:  "sbp_receipt",
				CreatedAt:  pr.CreatedAt,
			}
			db.Create(&pt)
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
					PaymentID:  histPID,
					TelegramID: u.TelegramID,
					Username:   u.Username,
					Amount:     diff,
					Currency:   "RUB",
					Tier:       tier,
					Status:     "SUCCESS",
					PayMethod:  "historical_access",
					PromoCode:  u.PromoCode,
					CreatedAt:  txTime,
				}
				db.Create(&pt)
			}
		}
	}
}
