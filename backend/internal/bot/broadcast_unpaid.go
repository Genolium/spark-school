package bot

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
)

const (
	// broadcastConfirmTTL is how long a previewed broadcast waits for the
	// curator's confirmation before it is dropped.
	broadcastConfirmTTL = 15 * time.Minute
	// broadcastSendDelay keeps the send rate below Telegram's 30 msgs/sec.
	broadcastSendDelay = 40 * time.Millisecond
)

type pendingBroadcast struct {
	text      string
	createdAt time.Time
}

// stageBroadcast remembers the previewed text for the admin chat. A new preview
// replaces the previous one.
func (s *BotService) stageBroadcast(chatID int64, text string, now time.Time) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	if s.pendingBroadcasts == nil {
		s.pendingBroadcasts = make(map[int64]pendingBroadcast)
	}
	s.pendingBroadcasts[chatID] = pendingBroadcast{text: text, createdAt: now}
}

// takeBroadcast returns the staged text and removes it, so a confirmation can
// send it at most once. An expired preview is dropped and not returned.
func (s *BotService) takeBroadcast(chatID int64, now time.Time) (string, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	pending, ok := s.pendingBroadcasts[chatID]
	if !ok {
		return "", false
	}
	delete(s.pendingBroadcasts, chatID)
	if now.Sub(pending.createdAt) > broadcastConfirmTTL {
		return "", false
	}
	return pending.text, true
}

// unpaidBroadcastRecipients returns the people a "not paid yet" message may
// reach: they started the bot, have no access and no successful payment, did
// not opt out of reminders and are not anonymized. Money comes from
// payment_transactions, not from users.paid_amount.
func (s *BotService) unpaidBroadcastRecipients() ([]models.User, error) {
	if s.db == nil {
		return nil, errors.New("database is unavailable")
	}
	var adminID int64
	if s.cfg != nil {
		adminID = s.cfg.AdminTelegramID
	}
	var users []models.User
	err := s.db.
		Where("telegram_id <> 0 AND telegram_id <> ?", adminID).
		Where("COALESCE(role, '') <> 'admin'").
		Where("has_access = ?", false).
		Where("reminders_opt_out = ?", false).
		Where("anonymized_at IS NULL").
		Where("NOT EXISTS (SELECT 1 FROM payment_transactions pt WHERE pt.telegram_id = users.telegram_id AND pt.status = 'SUCCESS')").
		Order("id").
		Find(&users).Error
	return users, err
}

// deliverBroadcast sends text to every user through send and counts the
// results. One failed delivery (for example a blocked bot) never stops the rest.
func deliverBroadcast(users []models.User, text string, delay time.Duration, send func(chatID int64, text string) error) (sent, failed int) {
	for _, u := range users {
		if err := send(u.TelegramID, text); err != nil {
			failed++
			log.Printf("[Broadcast Fail] telegram_id=%d: %v", u.TelegramID, err)
		} else {
			sent++
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	return sent, failed
}

func broadcastMessage(chatID telego.ChatID, text string) *telego.SendMessageParams {
	return tu.Message(chatID, text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Открыть меню").WithCallbackData("action_menu"),
		)))
}

// runBroadcast sends text to users in the background and reports to the admin.
func (s *BotService) runBroadcast(adminChatID telego.ChatID, users []models.User, text string) {
	recipients := make([]models.User, 0, len(users))
	for _, u := range users {
		if s.cfg != nil && u.TelegramID == s.cfg.AdminTelegramID {
			continue // the admin does not receive their own broadcast
		}
		recipients = append(recipients, u)
	}

	start := tu.Message(adminChatID, fmt.Sprintf("🚀 Запуск рассылки на <b>%d</b> получателей...", len(recipients))).
		WithParseMode(telego.ModeHTML)
	_, _ = s.bot.SendMessage(start)

	go func() {
		sent, failed := deliverBroadcast(recipients, text, broadcastSendDelay, func(chatID int64, body string) error {
			_, err := s.bot.SendMessage(broadcastMessage(tu.ID(chatID), body))
			return err
		})
		report := tu.Message(adminChatID, fmt.Sprintf(`✅ <b>Рассылка завершена!</b>

📬 Успешно доставлено: <b>%d</b>
⚠️ Ошибок доставки (блокировка бота): <b>%d</b>`, sent, failed)).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(report)
	}()
}

// promptUnpaidBroadcast asks the curator for the message text.
func (s *BotService) promptUnpaidBroadcast(chatID telego.ChatID) {
	recipients, err := s.unpaidBroadcastRecipients()
	if err != nil {
		_, _ = s.bot.SendMessage(tu.Message(chatID, "❌ Не удалось загрузить список: "+err.Error()))
		return
	}
	text := fmt.Sprintf(`📢 <b>Рассылка неоплатившим</b>

Получат <b>%d</b> человек: запускали бота, нет доступа и успешной оплаты, не отключали напоминания. Оплатившие и вы сами сообщение не получат.

Отправь следующим сообщением текст. Перед отправкой я покажу предпросмотр, и ты подтвердишь.

<i>Поддерживается HTML-разметка: &lt;b&gt;жирный&lt;/b&gt;, &lt;i&gt;курсив&lt;/i&gt;, ссылки.</i>

Для отмены напиши /admin.`, len(recipients))
	_, _ = s.bot.SendMessage(tu.Message(chatID, text).WithParseMode(telego.ModeHTML))
}

// previewUnpaidBroadcast shows the exact message and asks for confirmation.
// Nothing is sent to the audience here.
func (s *BotService) previewUnpaidBroadcast(chatID telego.ChatID, text string) {
	recipients, err := s.unpaidBroadcastRecipients()
	if err != nil {
		_, _ = s.bot.SendMessage(tu.Message(chatID, "❌ Не удалось загрузить список: "+err.Error()))
		return
	}
	if len(recipients) == 0 {
		_, _ = s.bot.SendMessage(tu.Message(chatID, "Некому отправлять: все, кто запускал бота, уже оплатили или отключили напоминания."))
		return
	}
	// The preview is the real message sent to the curator. If Telegram rejects
	// the HTML, the curator learns it here and not after sending to everyone.
	if _, err := s.bot.SendMessage(broadcastMessage(chatID, text)); err != nil {
		_, _ = s.bot.SendMessage(tu.Message(chatID, "❌ Telegram не принял текст, проверь HTML-разметку. Рассылка не запущена."))
		return
	}
	s.stageBroadcast(chatID.ID, text, time.Now())
	confirm := tu.Message(chatID, fmt.Sprintf("Так увидят %d человек. Отправить?", len(recipients))).
		WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("✅ Отправить").WithCallbackData("admin_broadcast_unpaid_send"),
			tu.InlineKeyboardButton("✖️ Отмена").WithCallbackData("admin_broadcast_unpaid_cancel"),
		)))
	_, _ = s.bot.SendMessage(confirm)
}

// sendStagedUnpaidBroadcast runs after the curator confirms. The audience is
// recalculated now, because people may have paid since the preview.
func (s *BotService) sendStagedUnpaidBroadcast(chatID telego.ChatID) {
	text, ok := s.takeBroadcast(chatID.ID, time.Now())
	if !ok {
		_, _ = s.bot.SendMessage(tu.Message(chatID, "Предпросмотр устарел или уже отправлен. Начни заново: /admin → «Рассылка неоплатившим»."))
		return
	}
	recipients, err := s.unpaidBroadcastRecipients()
	if err != nil {
		_, _ = s.bot.SendMessage(tu.Message(chatID, "❌ Не удалось загрузить список: "+err.Error()))
		return
	}
	s.runBroadcast(chatID, recipients, text)
}

func (s *BotService) cancelStagedBroadcast(chatID telego.ChatID) {
	_, _ = s.takeBroadcast(chatID.ID, time.Now())
	_, _ = s.bot.SendMessage(tu.Message(chatID, "Рассылка отменена, ничего не отправлено."))
}
