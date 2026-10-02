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

	// Run Auto-Migrations
	err = db.AutoMigrate(
		&models.User{},
		&models.Referral{},
		&models.AffiliateProfile{},
		&models.ReferralTransaction{},
		&models.PayoutRequest{},
		&models.PromoCode{},
		&models.PaymentReceipt{},
	)
	if err != nil {
		log.Printf("[WARN] Notice during database auto-migration: %v", err)
	}

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
