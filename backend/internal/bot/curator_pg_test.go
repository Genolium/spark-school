package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func curatorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public;").Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.StudentSubmission{},
		&models.OutboxEvent{},
		&models.CallBooking{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_submission_version ON student_submissions(telegram_id, work_type, version)").Error; err != nil {
		t.Fatalf("unique submission version: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_slot_key ON call_bookings(slot_key) WHERE status = 'confirmed' AND cancelled_at IS NULL").Error; err != nil {
		t.Fatalf("active slot index: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_user_booking ON call_bookings(telegram_id) WHERE status = 'confirmed' AND cancelled_at IS NULL").Error; err != nil {
		t.Fatalf("active user booking index: %v", err)
	}
	return db
}

func basicStudent(t *testing.T, db *gorm.DB, id int64, credits int) models.User {
	t.Helper()
	u := models.User{
		TelegramID: id, Username: "basic", FirstName: "Basic", Role: "student",
		HasAccess: true, Tier: "basic", AuditCredits: credits, HasAuditUpsell: credits > 0,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	return u
}

func TestPostgres_SubmitWork_LastCreditSpentOnce(t *testing.T) {
	db := curatorTestDB(t)
	user := basicStudent(t, db, 11001, 1)
	service := &BotService{db: db, cfg: &config.Config{}}

	var wg sync.WaitGroup
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs <- service.SubmitWork(tuID(user.TelegramID), &telego.User{ID: user.TelegramID, Username: "basic"},
				fmt.Sprintf("/submit_work essay https://docs.google.com/essay_%d", idx))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	var count int64
	db.Model(&models.StudentSubmission{}).Where("telegram_id = ?", user.TelegramID).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly one submission for one credit, got %d", count)
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if reloaded.AuditCredits != 0 {
		t.Fatalf("expected zero credits, got %d", reloaded.AuditCredits)
	}
	if reloaded.HasAuditUpsell {
		t.Fatal("upsell flag must be cleared when credits reach zero")
	}
}

func TestPostgres_SubmitWork_TwoAuditsAndDuplicateRetry(t *testing.T) {
	db := curatorTestDB(t)
	user := basicStudent(t, db, 12001, 2)
	service := &BotService{db: db, cfg: &config.Config{}}
	from := &telego.User{ID: user.TelegramID, Username: "basic"}

	if err := service.SubmitWork(tuID(user.TelegramID), from, "/submit_work essay https://docs.google.com/one"); err != nil {
		t.Fatal(err)
	}
	// Same command redelivered: must not spend a second credit.
	if err := service.SubmitWork(tuID(user.TelegramID), from, "/submit_work essay https://docs.google.com/one"); err != nil {
		t.Fatal(err)
	}
	if err := service.SubmitWork(tuID(user.TelegramID), from, "/submit_work essay https://docs.google.com/two"); err != nil {
		t.Fatal(err)
	}

	var count int64
	db.Model(&models.StudentSubmission{}).Where("telegram_id = ?", user.TelegramID).Count(&count)
	if count != 2 {
		t.Fatalf("expected two distinct submissions, got %d", count)
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if reloaded.AuditCredits != 0 {
		t.Fatalf("expected both credits spent, got %d", reloaded.AuditCredits)
	}
	var versions []int
	db.Model(&models.StudentSubmission{}).Where("telegram_id = ?", user.TelegramID).Pluck("version", &versions)
	if len(versions) != 2 || versions[0] == versions[1] {
		t.Fatalf("expected unique versions, got %v", versions)
	}
}

func TestPostgres_SubmissionReview_OnceAndEscaped(t *testing.T) {
	db := curatorTestDB(t)
	user := basicStudent(t, db, 13001, 1)
	service := &BotService{db: db, cfg: &config.Config{AdminTelegramID: 999}}
	from := &telego.User{ID: user.TelegramID, Username: "basic"}
	if err := service.SubmitWork(tuID(user.TelegramID), from, "/submit_work essay https://docs.google.com/x"); err != nil {
		t.Fatal(err)
	}
	var sub models.StudentSubmission
	if err := db.Where("telegram_id = ?", user.TelegramID).First(&sub).Error; err != nil {
		t.Fatal(err)
	}

	admin := &telego.User{ID: 999, Username: "curator"}
	notes := "Хорошо <script>alert(1)</script>"
	if err := service.ProcessSubmissionReview(tuID(999), admin, sub.ID, notes, true); err != nil {
		t.Fatalf("review: %v", err)
	}
	// Repeated callback must be acknowledged without a second transition.
	if err := service.ProcessSubmissionReview(tuID(999), admin, sub.ID, notes, false); err != nil {
		t.Fatalf("repeated review: %v", err)
	}

	var reloaded models.StudentSubmission
	db.First(&reloaded, sub.ID)
	if reloaded.Status != "reviewed" {
		t.Fatalf("repeated callback changed status to %s", reloaded.Status)
	}
	var events []models.OutboxEvent
	db.Where("event_type = ?", "telegram_notification").Find(&events)
	if len(events) != 2 { // submission confirmation + admin alert from SubmitWork, plus review notification
		// SubmitWork enqueued 2 (student + admin), review adds 1 => 3.
		if len(events) != 3 {
			t.Fatalf("expected 3 notifications, got %d", len(events))
		}
	}
	var reviewNotice string
	for _, event := range events {
		var payload struct {
			TelegramID int64  `json:"telegram_id"`
			Message    string `json:"message"`
		}
		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.TelegramID == user.TelegramID && strings.Contains(payload.Message, "проверена куратором") {
			reviewNotice = payload.Message
		}
	}
	if reviewNotice == "" {
		t.Fatal("review notification was not enqueued")
	}
	if strings.Contains(reviewNotice, "<script>") || !strings.Contains(reviewNotice, "&lt;script&gt;") {
		t.Fatalf("curator notes were not HTML-escaped: %s", reviewNotice)
	}
}

// tuID is a small helper to avoid importing telegoutil under an alias in this file.
func tuID(id int64) telego.ChatID { return telego.ChatID{ID: id} }

var _ = time.Now

func acceleratorStudent(t *testing.T, db *gorm.DB, id int64) models.User {
	t.Helper()
	u := models.User{TelegramID: id, Username: fmt.Sprintf("acc%d", id), FirstName: "Acc", Role: "student", HasAccess: true, Tier: "accelerator"}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("create accelerator: %v", err)
	}
	return u
}

func TestPostgres_Booking_SameSlotOnce(t *testing.T) {
	db := curatorTestDB(t)
	a := acceleratorStudent(t, db, 21001)
	b := acceleratorStudent(t, db, 21002)
	service := &BotService{db: db, cfg: &config.Config{}}
	slot := getUpcomingSlots(time.Now())[0]

	var wg sync.WaitGroup
	for _, user := range []models.User{a, b} {
		wg.Add(1)
		go func(u models.User) {
			defer wg.Done()
			service.BookCallDialog(tuID(u.TelegramID), &telego.User{ID: u.TelegramID, Username: u.Username}, slot.Key)
		}(user)
	}
	wg.Wait()

	var count int64
	db.Model(&models.CallBooking{}).Where("slot_key = ? AND status = 'confirmed' AND cancelled_at IS NULL", slot.Key).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly one booking for the slot, got %d", count)
	}
}

func TestPostgres_Booking_OneActivePerUser(t *testing.T) {
	db := curatorTestDB(t)
	u := acceleratorStudent(t, db, 22001)
	service := &BotService{db: db, cfg: &config.Config{}}
	slots := getUpcomingSlots(time.Now())
	if len(slots) < 2 {
		t.Skip("not enough upcoming slots")
	}

	var wg sync.WaitGroup
	for _, slot := range slots[:2] {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			service.BookCallDialog(tuID(u.TelegramID), &telego.User{ID: u.TelegramID, Username: u.Username}, key)
		}(slot.Key)
	}
	wg.Wait()

	var count int64
	db.Model(&models.CallBooking{}).Where("telegram_id = ? AND status = 'confirmed' AND cancelled_at IS NULL", u.TelegramID).Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly one active booking per user, got %d", count)
	}
}

func TestPostgres_Booking_RepeatedCallbackIsIdempotent(t *testing.T) {
	db := curatorTestDB(t)
	u := acceleratorStudent(t, db, 23001)
	service := &BotService{db: db, cfg: &config.Config{}}
	slot := getUpcomingSlots(time.Now())[0]
	from := &telego.User{ID: u.TelegramID, Username: u.Username}

	service.BookCallDialog(tuID(u.TelegramID), from, slot.Key)
	service.BookCallDialog(tuID(u.TelegramID), from, slot.Key)

	var count int64
	db.Model(&models.CallBooking{}).Where("telegram_id = ?", u.TelegramID).Count(&count)
	if count != 1 {
		t.Fatalf("repeated booking callback created %d rows", count)
	}
}

func TestPostgres_Booking_CancelIsConditional(t *testing.T) {
	db := curatorTestDB(t)
	u := acceleratorStudent(t, db, 24001)
	service := &BotService{db: db, cfg: &config.Config{}}
	slot := getUpcomingSlots(time.Now())[0]
	from := &telego.User{ID: u.TelegramID, Username: u.Username}
	service.BookCallDialog(tuID(u.TelegramID), from, slot.Key)

	var booking models.CallBooking
	db.Where("telegram_id = ?", u.TelegramID).First(&booking)
	service.CancelBooking(tuID(u.TelegramID), from, booking.ID)
	service.CancelBooking(tuID(u.TelegramID), from, booking.ID)

	var cancelled int64
	db.Model(&models.CallBooking{}).Where("id = ? AND status = 'cancelled'", booking.ID).Count(&cancelled)
	if cancelled != 1 {
		t.Fatalf("expected cancelled booking, got %d", cancelled)
	}
	// The slot must be reusable after cancellation.
	other := acceleratorStudent(t, db, 24002)
	service.BookCallDialog(tuID(other.TelegramID), &telego.User{ID: other.TelegramID, Username: other.Username}, slot.Key)
	var active int64
	db.Model(&models.CallBooking{}).Where("slot_key = ? AND status = 'confirmed' AND cancelled_at IS NULL", slot.Key).Count(&active)
	if active != 1 {
		t.Fatalf("expected slot to be reusable, got %d active", active)
	}
}
