package bot

import (
	"testing"
	"time"

	"github.com/spark-school/backend/internal/models"
)

func TestOnboardingReminders_Once(t *testing.T) {
	service, db := setupTestBot(t)
	old := time.Now().Add(-3 * time.Hour)
	user := models.User{
		TelegramID: 88001, Username: "newbie", FirstName: "New", Role: "student",
		HasAccess: true, Tier: "accelerator", AccessGrantedAt: &old,
		OnboardingStatus: "pending_intro",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.processOnboardingReminders(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := service.processOnboardingReminders(time.Now()); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 1 {
		t.Fatalf("expected one intro reminder, got %d", count)
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if reloaded.IntroReminderSentAt == nil {
		t.Fatal("intro reminder flag not persisted")
	}

	// Move to awaiting reply, older than 48h: one reply reminder.
	intro := time.Now().Add(-49 * time.Hour)
	db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"onboarding_status": "awaiting_reply",
		"intro_sent_at":     &intro,
	})
	if err := service.processOnboardingReminders(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := service.processOnboardingReminders(time.Now()); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 2 {
		t.Fatalf("expected one reply reminder, got %d total", count)
	}
}
