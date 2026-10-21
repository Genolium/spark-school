package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
	"gorm.io/gorm"
)

// recordOrderIntent stores a checkout intention. Re-opening the same tier
// updates the amount/promo but keeps the original creation time and reminder
// counter, so recovery is not reset by simply viewing the payment screen.
func (s *BotService) recordOrderIntent(from *telego.User, tier string, amount float64, promoCode, source string) {
	if !recoveryEnabled || s.db == nil || from == nil {
		return
	}
	tier = strings.ToLower(strings.TrimSpace(tier))
	if tier == "" || amount <= 0 {
		return
	}
	var user models.User
	if err := s.db.Where("telegram_id = ?", from.ID).First(&user).Error; err != nil {
		return
	}
	now := time.Now()
	var existing models.OrderIntent
	if err := s.db.Where("telegram_id = ? AND tier = ? AND status = 'pending'", from.ID, tier).
		Order("created_at DESC").First(&existing).Error; err == nil {
		_ = s.db.Model(&models.OrderIntent{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
			"amount":     amount,
			"promo_code": promoCode,
			"source":     source,
			"updated_at": now,
		}).Error
		return
	}
	_ = s.db.Model(&models.OrderIntent{}).
		Where("telegram_id = ? AND status = 'pending'", from.ID).
		Update("status", "superseded").Error
	_ = s.db.Create(&models.OrderIntent{
		UserID:     user.ID,
		TelegramID: from.ID,
		Tier:       tier,
		Amount:     amount,
		PromoCode:  promoCode,
		Source:     source,
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error
}

func (s *BotService) startRecoveryWorker() {
	if !recoveryEnabled {
		return
	}
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			now := time.Now()
			if err := s.processAbandonedOrders(now); err != nil {
				log.Printf("[Recovery] abandoned orders: %v", err)
			}
			if err := s.processDiagnosticRecovery(now); err != nil {
				log.Printf("[Recovery] diagnostics: %v", err)
			}
		}
	}
}

func (s *BotService) processAbandonedOrders(now time.Time) error {
	if s.db == nil {
		return nil
	}
	var intents []models.OrderIntent
	if err := s.db.Where("status = 'pending' AND reminder_count < 2 AND created_at <= ?", now.Add(-2*time.Hour)).
		Where("last_reminder_at IS NULL OR last_reminder_at < ?", now.Add(-20*time.Hour)).
		Order("created_at ASC").Limit(50).Find(&intents).Error; err != nil {
		return err
	}
	for _, intent := range intents {
		var user models.User
		if err := s.db.Where("telegram_id = ?", intent.TelegramID).First(&user).Error; err != nil {
			continue
		}
		if user.RemindersOptOut {
			_ = s.db.Model(&models.OrderIntent{}).Where("id = ?", intent.ID).Update("status", "cancelled").Error
			continue
		}
		var paidCount int64
		s.db.Model(&models.PaymentTransaction{}).
			Where("telegram_id = ? AND status = 'SUCCESS' AND created_at >= ?", intent.TelegramID, intent.CreatedAt).
			Count(&paidCount)
		if paidCount > 0 {
			_ = s.db.Model(&models.OrderIntent{}).Where("id = ?", intent.ID).Update("status", "paid").Error
			continue
		}

		stage := intent.ReminderCount + 1
		message := recoveryOrderMessage(intent, stage)
		if err := s.enqueueRecovery(intent.ID, intent.ReminderCount, intent.TelegramID, message, now); err != nil {
			log.Printf("[Recovery] order intent #%d: %v", intent.ID, err)
		}
	}
	return nil
}

func recoveryOrderMessage(intent models.OrderIntent, stage int) string {
	tier := "Акселератор"
	switch intent.Tier {
	case "basic":
		tier = "Базовый"
	case "vip":
		tier = "VIP"
	case "audit_essay":
		tier = "Аудит эссе"
	}
	if stage <= 1 {
		return fmt.Sprintf(`⏳ <b>Вы выбрали тариф «%s», но оплата ещё не поступила.</b>

Сумма: <b>%.0f ₽</b>%s

Если вы уже перевели деньги — отправьте скриншот или PDF чека прямо в этот чат.
Если планы изменились — вы можете вернуться к тарифам в любой момент.`, tier, intent.Amount, promoLine(intent.PromoCode))
	}
	return fmt.Sprintf(`📌 <b>Последнее напоминание по тарифу «%s»</b>

Сумма к оплате: <b>%.0f ₽</b>. Чек можно отправить прямо в этот чат.
Если передумали — просто проигнорируйте сообщение, больше напоминаний не будет.`, tier, intent.Amount)
}

func promoLine(code string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}
	return fmt.Sprintf("\nПромокод: <b>%s</b>", code)
}

func (s *BotService) enqueueRecovery(intentID uint, expectedCount int, telegramID int64, message string, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.OrderIntent{}).
			Where("id = ? AND status = 'pending' AND reminder_count = ?", intentID, expectedCount).
			Updates(map[string]interface{}{
				"reminder_count":   expectedCount + 1,
				"last_reminder_at": now,
				"updated_at":       now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil
		}
		return outbox.EnqueueNotificationTx(tx, telegramID, message)
	})
}

func (s *BotService) processDiagnosticRecovery(now time.Time) error {
	if s.db == nil {
		return nil
	}

	type diagRow struct {
		UserID     uint
		TelegramID int64
		VisitorID  string
	}

	// Started but never completed.
	var started []diagRow
	if err := s.db.Raw(`
		SELECT DISTINCT u.id AS user_id, u.telegram_id, a.visitor_id
		FROM users u
		JOIN analytics_events a ON a.attribution_token = u.attribution_token
		WHERE u.attribution_token <> '' AND u.reminders_opt_out = FALSE
		  AND u.diagnostic_started_reminder_sent = FALSE
		  AND a.step = 'diagnostic_start' AND a.created_at < ?
		  AND NOT EXISTS (
			SELECT 1 FROM analytics_events c
			WHERE c.visitor_id = a.visitor_id AND c.step = 'diagnostic_complete' AND c.created_at >= a.created_at
		  )
		LIMIT 50`, now.Add(-time.Hour)).Scan(&started).Error; err != nil {
		return err
	}
	for _, row := range started {
		message := fmt.Sprintf(`🧭 <b>Вы начали диагностику, но не завершили её.</b>

Это займёт меньше минуты: результат покажет готовность к конкурсу и ближайшие шаги.
Продолжить: %s`, strings.TrimRight(s.cfg.FrontendURL, "/"))
		if err := s.enqueueDiagnosticReminder(row.UserID, "diagnostic_started_reminder_sent", row.TelegramID, message); err != nil {
			log.Printf("[Recovery] diagnostic start for user %d: %v", row.UserID, err)
		}
	}

	// Completed diagnostic but no order and no payment.
	var completed []diagRow
	if err := s.db.Raw(`
		SELECT DISTINCT u.id AS user_id, u.telegram_id, a.visitor_id
		FROM users u
		JOIN analytics_events a ON a.attribution_token = u.attribution_token
		WHERE u.attribution_token <> '' AND u.reminders_opt_out = FALSE
		  AND u.diagnostic_completed_reminder_sent = FALSE
		  AND a.step = 'diagnostic_complete' AND a.created_at < ?
		  AND NOT EXISTS (
			SELECT 1 FROM order_intents oi
			WHERE oi.telegram_id = u.telegram_id AND oi.created_at >= a.created_at - INTERVAL '2 days'
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM payment_transactions pt
			WHERE pt.telegram_id = u.telegram_id AND pt.status = 'SUCCESS' AND pt.created_at >= a.created_at - INTERVAL '2 days'
		  )
		LIMIT 50`, now.Add(-6*time.Hour)).Scan(&completed).Error; err != nil {
		return err
	}
	for _, row := range completed {
		message := `🎯 <b>Диагностика пройдена, но тариф ещё не выбран.</b>

Посмотрите программу и формат сопровождения — можно начать с Базового тарифа или выбрать Акселератор с обратной связью куратора.
Выбрать тариф: /start`
		if err := s.enqueueDiagnosticReminder(row.UserID, "diagnostic_completed_reminder_sent", row.TelegramID, message); err != nil {
			log.Printf("[Recovery] diagnostic completion for user %d: %v", row.UserID, err)
		}
	}
	return nil
}

func (s *BotService) enqueueDiagnosticReminder(userID uint, column string, telegramID int64, message string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.User{}).
			Where("id = ? AND reminders_opt_out = FALSE AND "+column+" = FALSE", userID).
			UpdateColumn(column, true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil
		}
		return outbox.EnqueueNotificationTx(tx, telegramID, message)
	})
}

// setRemindersOptOut is used by /notifications_off and /notifications_on.
func (s *BotService) setRemindersOptOut(chatID telego.ChatID, from *telego.User, optOut bool) error {
	if s.db == nil || from == nil {
		return fmt.Errorf("database is unavailable")
	}
	if err := s.db.Model(&models.User{}).Where("telegram_id = ?", from.ID).
		Update("reminders_opt_out", optOut).Error; err != nil {
		return err
	}
	if optOut {
		_ = s.db.Model(&models.DripLead{}).Where("telegram_id = ?", from.ID).
			Updates(map[string]interface{}{"stage": -1, "is_completed": true}).Error
	}
	text := "🔕 Напоминания отключены. Вы по-прежнему можете выбрать тариф и написать куратору."
	if !optOut {
		text = "🔔 Напоминания включены."
	}
	return s.send(tu.Message(chatID, text).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("💳 Выбрать тариф").WithCallbackData("action_payment")),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
	)))
}
