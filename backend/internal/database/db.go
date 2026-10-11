package database

import (
	"log"
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

	// Run GORM Auto-Migrations fallback to ensure all struct fields match
	err = db.AutoMigrate(
		&models.User{},
		&models.Referral{},
		&models.AffiliateProfile{},
		&models.ReferralTransaction{},
		&models.PayoutRequest{},
		&models.PromoCode{},
		&models.PaymentReceipt{},
		&models.OutboxEvent{},
		&models.PaymentTransaction{},
	)
	if err != nil {
		log.Printf("[WARN] Notice during database auto-migration: %v", err)
	}

	// Backfill existing active students with defaults if empty
	db.Exec("UPDATE users SET tier = 'accelerator' WHERE tier IS NULL OR tier = ''")
	db.Exec("UPDATE users SET paid_amount = 6900.00 WHERE has_access = true AND (paid_amount IS NULL OR paid_amount = 0)")

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

	log.Println("Seeding complete.")
}
