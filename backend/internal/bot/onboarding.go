package bot

import (
	"fmt"
	"html"
	"log"
	"time"

	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
	"gorm.io/gorm"
)

// startOnboardingWorker reminds the curator about new students who paid but
// have not received or answered the personal introduction.
func (s *BotService) startOnboardingWorker() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			if !s.acquireSchedulerLease("bot:onboarding", 20*time.Minute) {
				continue
			}
			if err := s.processOnboardingReminders(time.Now()); err != nil {
				log.Printf("[Onboarding] reminders: %v", err)
			}
		}
	}
}

func (s *BotService) processOnboardingReminders(now time.Time) error {
	if s.db == nil || s.cfg == nil || s.cfg.AdminTelegramID == 0 {
		return nil
	}
	var pending []models.User
	if err := s.db.Where("onboarding_status = ? AND intro_reminder_sent_at IS NULL AND access_granted_at IS NOT NULL AND access_granted_at < ?",
		"pending_intro", now.Add(-2*time.Hour)).Limit(50).Find(&pending).Error; err != nil {
		return err
	}
	for _, user := range pending {
		message := fmt.Sprintf(`👋 <b>Онбординг:</b> студент @%s (ID %d) получил доступ, но приветственное сообщение ещё не отправлено.

Напиши ему в личку и отметь «Сообщение отправлено» в админке.`,
			html.EscapeString(user.Username), user.TelegramID)
		if err := s.markOnboardingReminder(user.ID, "intro_reminder_sent_at", message, now); err != nil {
			log.Printf("[Onboarding] intro reminder for user %d: %v", user.ID, err)
		}
	}

	var awaiting []models.User
	if err := s.db.Where("onboarding_status = ? AND reply_reminder_sent_at IS NULL AND intro_sent_at IS NOT NULL AND intro_sent_at < ?",
		"awaiting_reply", now.Add(-48*time.Hour)).Limit(50).Find(&awaiting).Error; err != nil {
		return err
	}
	for _, user := range awaiting {
		message := fmt.Sprintf(`⏳ <b>Онбординг:</b> студент @%s (ID %d) не ответил на приветствие больше 48 часов.

Можно мягко напомнить о себе и отметить ответ в админке.`,
			html.EscapeString(user.Username), user.TelegramID)
		if err := s.markOnboardingReminder(user.ID, "reply_reminder_sent_at", message, now); err != nil {
			log.Printf("[Onboarding] reply reminder for user %d: %v", user.ID, err)
		}
	}
	return nil
}

func (s *BotService) markOnboardingReminder(userID uint, column, message string, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.User{}).
			Where("id = ? AND "+column+" IS NULL", userID).
			UpdateColumn(column, &now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil
		}
		return outbox.EnqueueNotificationTx(tx, s.cfg.AdminTelegramID, message)
	})
}
