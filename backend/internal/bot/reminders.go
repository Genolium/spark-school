package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
	"gorm.io/gorm"
)

// startReminderWorker sends 24h and 1h Zoom reminders at most once per booking.
// Flags are flipped in a conditional UPDATE inside the same transaction as the
// outbox insert, so multiple instances cannot send the same reminder twice.
func (s *BotService) startReminderWorker() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			if !s.acquireSchedulerLease("bot:zoom_reminders", 15*time.Minute) {
				continue
			}
			if err := s.sendDueZoomReminders(time.Now()); err != nil {
				log.Printf("[Reminders] failed: %v", err)
			}
		}
	}
}

func (s *BotService) sendDueZoomReminders(now time.Time) error {
	if s.db == nil {
		return nil
	}
	var bookings []models.CallBooking
	if err := s.db.Where("status = ? AND cancelled_at IS NULL", "confirmed").Find(&bookings).Error; err != nil {
		return err
	}
	msk := time.FixedZone("MSK", 3*3600)
	for _, booking := range bookings {
		slot, err := time.ParseInLocation("20060102_1504", booking.SlotKey, msk)
		if err != nil {
			continue
		}
		until := slot.Sub(now)
		switch {
		case !booking.Reminder24Sent && until <= 24*time.Hour && until > time.Hour:
			if err := s.markAndEnqueueReminder(booking, "reminder_24_sent", "24 часа", slot); err != nil {
				log.Printf("[Reminders] booking #%d 24h: %v", booking.ID, err)
			}
		case !booking.Reminder1Sent && until <= time.Hour && until > -15*time.Minute:
			if err := s.markAndEnqueueReminder(booking, "reminder_1_sent", "1 час", slot); err != nil {
				log.Printf("[Reminders] booking #%d 1h: %v", booking.ID, err)
			}
		}
	}
	return nil
}

func (s *BotService) markAndEnqueueReminder(booking models.CallBooking, flag, label string, slot time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.CallBooking{}).
			Where("id = ? AND status = 'confirmed' AND cancelled_at IS NULL AND "+flag+" = FALSE", booking.ID).
			UpdateColumn(flag, true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil
		}
		zoom := ""
		if strings.HasPrefix(booking.ZoomLink, "http") {
			zoom = "\n🔗 " + booking.ZoomLink
		}
		msg := fmt.Sprintf(`⏰ <b>Напоминание о Zoom-симуляции</b>

До созвона осталось <b>%s</b>.
📅 %s%s

Если планы изменились, отмените запись командой <code>/cancel_call</code>, чтобы освободить слот.`, label, booking.SlotTime, zoom)
		return outbox.EnqueueNotificationTx(tx, booking.TelegramID, msg)
	})
}

// enqueueAdminAlert is used by background jobs that need a durable alert.
func enqueueAdminAlert(tx *gorm.DB, adminID int64, payload outbox.AdminAlertPayload) error {
	if adminID == 0 {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return tx.Create(&models.OutboxEvent{
		EventType:  "admin_receipt_card",
		Payload:    string(raw),
		Status:     "pending",
		MaxRetries: 5,
		CreatedAt:  time.Now(),
	}).Error
}
