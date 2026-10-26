package bot

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/payments"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestBot(t *testing.T) (*BotService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:mem_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open sqlite in-memory db: %v", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.StudentSubmission{},
		&models.PaymentTransaction{},
		&models.PaymentReceipt{},
		&models.AnalyticsEvent{},
		&models.PromoCode{},
		&models.AffiliateProfile{},
		&models.Referral{},
		&models.ReferralTransaction{},
		&models.OutboxEvent{},
		&models.PaymentOrder{},
		&models.Diagnostic{},
		&models.CampaignLinkClick{},
	)
	if err != nil {
		t.Fatalf("failed to automigrate models: %v", err)
	}

	affService := services.NewAffiliateService(db)
	botService := &BotService{
		db:               db,
		cfg:              &config.Config{AdminTelegramID: 999999},
		affiliateService: affService,
		paymentsService:  payments.NewService(db, affService),
	}

	return botService, db
}

func TestCuratorTools_GetSafeInviteLink(t *testing.T) {
	bot1 := &BotService{
		cfg: &config.Config{
			TelegramInviteLink: "https://t.me/+AbCdEfGhIjKlMnOp",
		},
	}
	link := bot1.GetSafeInviteLink(context.Background())
	if link != "https://t.me/+AbCdEfGhIjKlMnOp" {
		t.Errorf("expected static invite link, got %q", link)
	}

	bot2 := &BotService{
		cfg: &config.Config{},
	}
	linkEmpty := bot2.GetSafeInviteLink(context.Background())
	if linkEmpty != "" {
		t.Errorf("expected empty string when no link or channel configured, got %q", linkEmpty)
	}
}

func TestCuratorTools_SubmitWork_AuditCredits(t *testing.T) {
	bot, db := setupTestBot(t)

	// User on Basic tier with 1 audit credit
	user := models.User{
		TelegramID:     1006,
		Username:       "basic_audit_user",
		Tier:           "basic",
		HasAccess:      true,
		AuditCredits:   1,
		HasAuditUpsell: true,
	}
	db.Create(&user)

	// 1. Submit essay -> should succeed and consume credit
	bot.SubmitWork(tu.ID(user.TelegramID), &telego.User{ID: user.TelegramID, Username: user.Username}, "/submit_work essay https://docs.google.com/essay_draft_1")

	var subs []models.StudentSubmission
	db.Where("telegram_id = ?", user.TelegramID).Find(&subs)
	if len(subs) != 1 {
		t.Fatalf("expected 1 submission created, got %d", len(subs))
	}
	if subs[0].WorkType != "essay" {
		t.Errorf("expected work_type essay, got %s", subs[0].WorkType)
	}

	// Verify audit credit decremented
	var updatedUser models.User
	db.Where("telegram_id = ?", user.TelegramID).First(&updatedUser)
	if updatedUser.AuditCredits != 0 {
		t.Errorf("expected audit_credits to be 0 after submission, got %d", updatedUser.AuditCredits)
	}
	if updatedUser.HasAuditUpsell != false {
		t.Errorf("expected has_audit_upsell to be false after consuming single credit")
	}

	// 2. Submit second essay with 0 credits -> should be rejected
	bot.SubmitWork(tu.ID(user.TelegramID), &telego.User{ID: user.TelegramID, Username: user.Username}, "/submit_work essay https://docs.google.com/essay_draft_2")

	var subsAfter []models.StudentSubmission
	db.Where("telegram_id = ?", user.TelegramID).Find(&subsAfter)
	if len(subsAfter) != 1 {
		t.Fatalf("expected second submission to be rejected when credits=0, got %d total", len(subsAfter))
	}

	// 3. User with Basic tier and 1 credit submitting 'resume' -> rejected (Basic credit is essay-only)
	userResume := models.User{
		TelegramID:     1007,
		Username:       "resume_unauthorized",
		Tier:           "basic",
		HasAccess:      true,
		AuditCredits:   1,
		HasAuditUpsell: true,
	}
	db.Create(&userResume)

	bot.SubmitWork(tu.ID(userResume.TelegramID), &telego.User{ID: userResume.TelegramID, Username: userResume.Username}, "/submit_work resume https://docs.google.com/resume_draft")
	var resumeSubs []models.StudentSubmission
	db.Where("telegram_id = ?", userResume.TelegramID).Find(&resumeSubs)
	if len(resumeSubs) != 0 {
		t.Fatalf("expected resume submission on basic tier to be rejected, got %d", len(resumeSubs))
	}
}

func TestCuratorTools_ReceiptApproval_PaymentTransaction(t *testing.T) {
	bot, db := setupTestBot(t)

	student := models.User{
		TelegramID: 1008,
		Username:   "receipt_student",
		Tier:       "accelerator",
		HasAccess:  false,
		PaidAmount: 0.0,
	}
	db.Create(&student)

	receipt := models.PaymentReceipt{
		TelegramID: student.TelegramID,
		Username:   student.Username,
		Amount:     6900.0,
		Tier:       "accelerator",
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	db.Create(&receipt)

	query := &telego.CallbackQuery{
		ID:   "cb_approve_808",
		From: telego.User{ID: 999999}, // Admin
	}

	bot.processReceiptApproval(query, fmt.Sprintf("%d", receipt.ID))

	// Verify PaymentTransaction created in real-time
	var tx models.PaymentTransaction
	receiptPID := fmt.Sprintf("RECEIPT_%d", receipt.ID)
	err := db.Where("payment_id = ?", receiptPID).First(&tx).Error
	if err != nil {
		t.Fatalf("expected real-time PaymentTransaction %s to be created: %v", receiptPID, err)
	}
	if tx.Amount != 6900.0 {
		t.Errorf("expected transaction amount 6900.0, got %f", tx.Amount)
	}
	if tx.Status != "SUCCESS" {
		t.Errorf("expected status SUCCESS, got %s", tx.Status)
	}

	// Verify Student updated
	var updatedStudent models.User
	db.Where("telegram_id = ?", student.TelegramID).First(&updatedStudent)
	if !updatedStudent.HasAccess {
		t.Errorf("expected student to have access granted")
	}
	if updatedStudent.PaidAmount != 6900.0 {
		t.Errorf("expected paid_amount 6900.0, got %f", updatedStudent.PaidAmount)
	}

	// Verify payment_confirmed analytics event recorded
	var event models.AnalyticsEvent
	err = db.Where("step = 'payment_confirmed'").First(&event).Error
	if err != nil {
		t.Errorf("expected payment_confirmed analytics event to be recorded: %v", err)
	}
}
