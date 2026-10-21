package bot

import (
	"context"
	"fmt"
	"sync"
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
		&models.CallBooking{},
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
	)
	if err != nil {
		t.Fatalf("failed to automigrate models: %v", err)
	}

	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_slot_key ON call_bookings (slot_key) WHERE status = 'confirmed' AND cancelled_at IS NULL;")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_user_booking ON call_bookings (telegram_id) WHERE status = 'confirmed' AND cancelled_at IS NULL;")

	affService := services.NewAffiliateService(db)
	botService := &BotService{
		db:               db,
		cfg:              &config.Config{AdminTelegramID: 999999},
		affiliateService: affService,
		paymentsService:  payments.NewService(db, affService),
	}

	return botService, db
}

func TestCuratorTools_UpcomingSlots(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	slots := getUpcomingSlots(now)

	if len(slots) == 0 {
		t.Fatalf("expected at least 1 upcoming slot, got 0")
	}

	for i, sl := range slots {
		if sl.Key == "" {
			t.Errorf("slot %d: empty key", i)
		}
		if sl.Display == "" {
			t.Errorf("slot %d: empty display", i)
		}
		if !sl.Time.After(now.Add(2 * time.Hour)) {
			t.Errorf("slot %d: time %v should be at least 2h after %v", i, sl.Time, now)
		}
	}
}

func TestCuratorTools_InterviewQuestions(t *testing.T) {
	if len(interviewQuestions) == 0 {
		t.Fatalf("expected interview questions list to be populated")
	}

	for i, q := range interviewQuestions {
		if q.Question == "" {
			t.Errorf("question %d: empty question text", i)
		}
		if q.Context == "" {
			t.Errorf("question %d: empty context", i)
		}
		if q.Tip == "" {
			t.Errorf("question %d: empty tip", i)
		}
	}
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

func TestCuratorTools_ConcurrentBooking_RaceCondition(t *testing.T) {
	bot, db := setupTestBot(t)

	user1 := models.User{TelegramID: 1001, Username: "alex_acc", Tier: "accelerator", HasAccess: true}
	user2 := models.User{TelegramID: 1002, Username: "misha_vip", Tier: "vip", HasAccess: true}
	db.Create(&user1)
	db.Create(&user2)

	slots := getUpcomingSlots(time.Now())
	if len(slots) == 0 {
		t.Fatalf("no upcoming slots found")
	}
	targetSlot := slots[0].Key

	var wg sync.WaitGroup
	workers := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var u models.User
			if idx%2 == 0 {
				u = user1
			} else {
				u = user2
			}
			bot.BookCallDialog(tu.ID(u.TelegramID), &telego.User{ID: u.TelegramID, Username: u.Username}, targetSlot)
		}(i)
	}

	wg.Wait()

	var bookings []models.CallBooking
	db.Where("slot_key = ? AND status = ? AND cancelled_at IS NULL", targetSlot, "confirmed").Find(&bookings)
	if len(bookings) != 1 {
		t.Fatalf("expected exactly 1 confirmed booking for slot %s, got %d", targetSlot, len(bookings))
	}
}

func TestCuratorTools_UserActiveBookingConflict(t *testing.T) {
	bot, db := setupTestBot(t)

	user := models.User{TelegramID: 1003, Username: "conflict_user", Tier: "accelerator", HasAccess: true}
	db.Create(&user)

	slots := getUpcomingSlots(time.Now())
	if len(slots) < 2 {
		t.Fatalf("need at least 2 upcoming slots")
	}

	slotA := slots[0].Key
	slotB := slots[1].Key

	// First booking must succeed
	bot.BookCallDialog(tu.ID(user.TelegramID), &telego.User{ID: user.TelegramID, Username: user.Username}, slotA)

	var bookingsA []models.CallBooking
	db.Where("telegram_id = ? AND slot_key = ? AND status = 'confirmed'", user.TelegramID, slotA).Find(&bookingsA)
	if len(bookingsA) != 1 {
		t.Fatalf("expected 1 confirmed booking for slot A, got %d", len(bookingsA))
	}

	// Second booking attempt on different slot must be blocked
	bot.BookCallDialog(tu.ID(user.TelegramID), &telego.User{ID: user.TelegramID, Username: user.Username}, slotB)

	var allUserBookings []models.CallBooking
	db.Where("telegram_id = ? AND status = 'confirmed' AND cancelled_at IS NULL", user.TelegramID).Find(&allUserBookings)
	if len(allUserBookings) != 1 {
		t.Fatalf("expected user to maintain only 1 active booking, got %d", len(allUserBookings))
	}
	if allUserBookings[0].SlotKey != slotA {
		t.Errorf("expected active booking to remain on slot A, got %s", allUserBookings[0].SlotKey)
	}
}

func TestCuratorTools_CancelBooking_FreesSlot(t *testing.T) {
	bot, db := setupTestBot(t)

	user1 := models.User{TelegramID: 1004, Username: "canceller", Tier: "accelerator", HasAccess: true}
	user2 := models.User{TelegramID: 1005, Username: "successor", Tier: "vip", HasAccess: true}
	db.Create(&user1)
	db.Create(&user2)

	slots := getUpcomingSlots(time.Now())
	targetSlot := slots[0].Key

	// User 1 books slot
	bot.BookCallDialog(tu.ID(user1.TelegramID), &telego.User{ID: user1.TelegramID, Username: user1.Username}, targetSlot)

	// User 2 tries to book same slot -> should fail
	bot.BookCallDialog(tu.ID(user2.TelegramID), &telego.User{ID: user2.TelegramID, Username: user2.Username}, targetSlot)
	var u2Bookings []models.CallBooking
	db.Where("telegram_id = ? AND status = 'confirmed'", user2.TelegramID).Find(&u2Bookings)
	if len(u2Bookings) != 0 {
		t.Fatalf("expected user 2 booking to be rejected while slot is taken")
	}

	// User 1 cancels booking
	bot.CancelBooking(tu.ID(user1.TelegramID), &telego.User{ID: user1.TelegramID, Username: user1.Username}, 0)

	var cancelled []models.CallBooking
	db.Where("telegram_id = ? AND status = 'cancelled'", user1.TelegramID).Find(&cancelled)
	if len(cancelled) != 1 {
		t.Fatalf("expected user 1 booking to be cancelled")
	}

	// User 2 now books the freed slot -> should succeed!
	bot.BookCallDialog(tu.ID(user2.TelegramID), &telego.User{ID: user2.TelegramID, Username: user2.Username}, targetSlot)
	var u2Confirmed []models.CallBooking
	db.Where("telegram_id = ? AND status = 'confirmed' AND cancelled_at IS NULL", user2.TelegramID).Find(&u2Confirmed)
	if len(u2Confirmed) != 1 {
		t.Fatalf("expected user 2 to successfully book freed slot, got %d", len(u2Confirmed))
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
