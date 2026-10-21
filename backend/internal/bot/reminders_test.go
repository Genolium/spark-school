package bot

import (
	"testing"
	"time"

	"github.com/spark-school/backend/internal/models"
)

func TestZoomReminders_SentOnce(t *testing.T) {
	service, db := setupTestBot(t)
	now := time.Now()
	slot := now.Add(2 * time.Hour).In(time.FixedZone("MSK", 3*3600))
	booking := models.CallBooking{
		UserID: 1, TelegramID: 100, Username: "student",
		SlotKey: slot.Format("20060102_1504"), SlotTime: "скоро",
		Status: "confirmed", CreatedAt: now,
	}
	if err := db.Create(&booking).Error; err != nil {
		t.Fatal(err)
	}

	if err := service.sendDueZoomReminders(now); err != nil {
		t.Fatal(err)
	}
	if err := service.sendDueZoomReminders(now); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'telegram_notification'").Count(&count)
	if count != 1 {
		t.Fatalf("expected one 24h reminder, got %d", count)
	}
	var reloaded models.CallBooking
	db.First(&reloaded, booking.ID)
	if !reloaded.Reminder24Sent {
		t.Fatal("24h reminder flag was not persisted")
	}
}
