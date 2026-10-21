package bot

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/spark-school/backend/internal/outbox"
)

// DeliverPaymentNotice is the outbox delivery callback for confirmed payments.
// It runs after the financial transaction commits, so a Telegram failure can
// be retried without re-applying money, credits or commissions.
func (s *BotService) DeliverPaymentNotice(p outbox.PaymentNoticePayload) error {
	switch p.Kind {
	case "admin":
		adminID := p.AdminID
		if adminID == 0 && s.cfg != nil {
			adminID = s.cfg.AdminTelegramID
		}
		if adminID == 0 {
			return nil
		}
		student := fmt.Sprintf("ID: %d", p.TelegramID)
		if p.Username != "" {
			student = fmt.Sprintf("@%s (%s)", html.EscapeString(p.Username), student)
		}
		msg := fmt.Sprintf(
			"⚡️ <b>Оплата подтверждена</b>\n\n"+
				"👤 <b>Студент:</b> %s\n"+
				"📦 <b>Продукт:</b> %s\n"+
				"💰 <b>Сумма:</b> %.2f ₽\n"+
				"🆔 <b>Платёж:</b> <code>%s</code>\n"+
				"🏦 <b>Источник:</b> %s\n"+
				"🎟 <b>Промокод:</b> %s",
			student, html.EscapeString(tierDisplay(p.Tier)), p.Amount, html.EscapeString(p.PaymentID),
			html.EscapeString(p.Source), html.EscapeString(p.PromoCode),
		)
		return s.SendNotification(adminID, msg)

	case "audit":
		msg := `🎉 <b>Оплата текстового аудита эссе успешно подтверждена!</b>

Теперь ты можешь отправить черновик эссе через команду:
<code>/submit_work essay [ссылка на Google Docs]</code>

Куратор проведёт подробный построчный аудит твоего эссе и подготовит разбор с правками в течение 24–48 часов.`
		return s.SendNotification(p.TelegramID, msg)

	case "upgrade":
		msg := fmt.Sprintf(`🎉 <b>Поздравляем! Ваш тариф успешно повышен до %s!</b>

Вам открыты все расширенные материалы и привилегии потока проекта «так называемый SPARK».
По всем вопросам вы всегда можете обратиться к куратору через поддержку бота.`, html.EscapeString(tierDisplay(p.Tier)))
		return s.SendNotification(p.TelegramID, msg)

	default: // access
		inviteLink := ""
		if s.cfg != nil {
			inviteLink = strings.TrimSpace(s.cfg.TelegramInviteLink)
			if s.cfg.TelegramChannelID != 0 && s.bot != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if link, err := s.CreateOneTimeInviteLink(ctx, s.cfg.TelegramChannelID); err == nil && link != "" {
					inviteLink = link
				}
			}
		}
		if inviteLink == "" {
			return fmt.Errorf("cannot deliver access for payment %s: invite link is not configured", p.PaymentID)
		}
		return s.sendAccessNotificationToStudent(p.TelegramID, FormatPaymentSuccessMessage(inviteLink), inviteLink)
	}
}

func tierDisplay(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "basic":
		return "Базовый"
	case "vip":
		return "VIP-сопровождение"
	case "audit_essay", "audit":
		return "Аудит эссе"
	default:
		return "Акселератор"
	}
}
