package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func (s *BotService) startQueueWorker() {
	ticker := time.NewTicker(35 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.quitChan:
			return
		case item, ok := <-s.notifyQueue:
			if !ok {
				return
			}
			<-ticker.C

			var err error
			if s.bot != nil && item.telegramID != 0 {
				if strings.Contains(item.message, "Добро пожаловать на моё обучение") ||
					strings.Contains(item.message, "Твоя ссылка для входа в закрытый канал потока:") {
					link := extractTelegramLink(item.message)
					err = s.sendAccessNotificationToStudent(item.telegramID, item.message, link)
				} else {
					chatID := tu.ID(item.telegramID)
					m := tu.Message(chatID, item.message).WithParseMode(telego.ModeHTML)
					_, err = s.bot.SendMessage(m)
				}
			} else if item.telegramID != 0 {
				log.Printf("[Bot Notification Mock] to %d (len=%d)", item.telegramID, len(item.message))
			}

			if item.respChan != nil {
				item.respChan <- err
			}
		}
	}
}

func (s *BotService) sendAccessNotificationToStudent(studentTelegramID int64, studentMsg string, inviteLink string) error {
	if studentTelegramID == 0 {
		return fmt.Errorf("student Telegram ID is 0")
	}

	// Student received access — immediately cancel any automated drip follow-ups
	s.CancelDrip(studentTelegramID)

	if s.bot == nil {
		log.Printf("[Bot Approval Mock] sendAccessNotificationToStudent to %d: %s", studentTelegramID, studentMsg)
		return nil
	}

	studentChatID := tu.ID(studentTelegramID)
	cleanMsg := StripTgEmoji(studentMsg)

	var kb *telego.InlineKeyboardMarkup
	if strings.HasPrefix(inviteLink, "http://") || strings.HasPrefix(inviteLink, "https://") || strings.HasPrefix(inviteLink, "tg://") {
		kb = tu.InlineKeyboard(tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Войти в закрытый канал ➔").WithURL(inviteLink),
		))
	} else {
		kb = tu.InlineKeyboard(tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Написать так называемому Илю").WithURL("https://t.me/ilyan_vas"),
		))
	}

	photoFile := s.ResolvePhotoFile(MediaStatusGranted)

	// Step 1: Rich HTML Photo with custom emojis + keyboard
	photoMsg := tu.Photo(studentChatID, photoFile).
		WithCaption(studentMsg).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(kb)

	sentMsg, err := s.bot.SendPhoto(photoMsg)
	if err == nil && sentMsg != nil {
		s.CacheSentPhoto(MediaStatusGranted, sentMsg)
		log.Printf("[Bot Approval] Step 1 SendPhoto (rich) succeeded for student %d", studentTelegramID)
		return nil
	}
	log.Printf("[Bot Approval] Step 1 SendPhoto failed for student %d: %v. Retrying with stripped emojis...", studentTelegramID, err)

	// Step 2: Clean Photo with StripTgEmoji + keyboard
	photoMsgClean := tu.Photo(studentChatID, photoFile).
		WithCaption(cleanMsg).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(kb)

	sentMsg2, err2 := s.bot.SendPhoto(photoMsgClean)
	if err2 == nil && sentMsg2 != nil {
		s.CacheSentPhoto(MediaStatusGranted, sentMsg2)
		log.Printf("[Bot Approval] Step 2 SendPhoto (clean) succeeded for student %d", studentTelegramID)
		return nil
	}
	log.Printf("[Bot Approval] Step 2 SendPhoto failed for student %d: %v. Retrying SendMessage...", studentTelegramID, err2)

	// Step 3: Text SendMessage with rich HTML + custom emojis + keyboard
	textMsg := tu.Message(studentChatID, studentMsg).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(kb)

	if _, err3 := s.bot.SendMessage(textMsg); err3 == nil {
		log.Printf("[Bot Approval] Step 3 SendMessage (rich) succeeded for student %d", studentTelegramID)
		return nil
	} else {
		log.Printf("[Bot Approval] Step 3 SendMessage failed for student %d: %v. Retrying clean text...", studentTelegramID, err3)
	}

	// Step 4: Text SendMessage with cleanMsg + keyboard
	cleanTextMsg := tu.Message(studentChatID, cleanMsg).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(kb)

	if _, err4 := s.bot.SendMessage(cleanTextMsg); err4 == nil {
		log.Printf("[Bot Approval] Step 4 SendMessage (clean) succeeded for student %d", studentTelegramID)
		return nil
	} else {
		log.Printf("[Bot Approval] Step 4 SendMessage failed for student %d: %v. Retrying without keyboard...", studentTelegramID, err4)
	}

	// Step 5: Text SendMessage cleanMsg WITHOUT keyboard (in case keyboard URL caused rejection)
	noMarkupMsg := tu.Message(studentChatID, cleanMsg).
		WithParseMode(telego.ModeHTML)

	if _, err5 := s.bot.SendMessage(noMarkupMsg); err5 == nil {
		log.Printf("[Bot Approval] Step 5 SendMessage (no markup) succeeded for student %d", studentTelegramID)
		return nil
	} else {
		log.Printf("[Bot Approval] Step 5 SendMessage failed for student %d: %v. Retrying plain text...", studentTelegramID, err5)
	}

	// Step 6: Pure plain unformatted text
	plainText := fmt.Sprintf("Поздравляем! Оплата подтверждена.\n\nДобро пожаловать в проект «так называемый SPARK»!\nТвоя ссылка для входа в закрытый канал потока:\n%s\n\nЧто дальше:\n1. Зайди в канал по ссылке.\n2. Иль напишет тебе в личку с несколькими вопросами — ответь, пожалуйста.\n3. Все задания и материалы — в канале.\n\nСсылка является персональной и одноразовой.", inviteLink)
	plainMsg := tu.Message(studentChatID, plainText)

	if _, err6 := s.bot.SendMessage(plainMsg); err6 == nil {
		log.Printf("[Bot Approval] Step 6 plain SendMessage succeeded for student %d", studentTelegramID)
		return nil
	} else {
		log.Printf("[Bot Approval] FATAL: All 6 send attempts failed for student %d: %v", studentTelegramID, err6)
		return err6
	}
}

func (s *BotService) SendNotification(telegramID int64, message string) error {
	if telegramID == 0 {
		return nil
	}

	if s.notifyQueue != nil {
		respChan := make(chan error, 1)
		item := queuedNotification{
			telegramID: telegramID,
			message:    message,
			respChan:   respChan,
		}

		select {
		case s.notifyQueue <- item:
			select {
			case err := <-respChan:
				return err
			case <-time.After(10 * time.Second):
				// Delivery was not confirmed. Outbox must retry instead of
				// treating an unconfirmed send as success.
				return fmt.Errorf("telegram notification timed out before delivery confirmation")
			}
		default:
			log.Printf("[Bot Rate Limiter] Queue full, sending directly to %d", telegramID)
		}
	}

	if s.bot == nil {
		log.Printf("[Bot Notification Mock] to %d (len=%d)", telegramID, len(message))
		return nil
	}

	if strings.Contains(message, "Добро пожаловать на моё обучение") ||
		strings.Contains(message, "Твоя ссылка для входа в закрытый канал потока:") {
		link := extractTelegramLink(message)
		return s.sendAccessNotificationToStudent(telegramID, message, link)
	}

	chatID := tu.ID(telegramID)
	m := tu.Message(chatID, message).WithParseMode(telego.ModeHTML)
	_, err := s.bot.SendMessage(m)
	if err != nil {
		cleanMsg := StripTgEmoji(message)
		cleanM := tu.Message(chatID, cleanMsg).WithParseMode(telego.ModeHTML)
		if _, cleanErr := s.bot.SendMessage(cleanM); cleanErr == nil {
			return nil
		}
		rawM := tu.Message(chatID, cleanMsg)
		_, rawErr := s.bot.SendMessage(rawM)
		return rawErr
	}
	return nil
}

func (s *BotService) CreateOneTimeInviteLink(ctx context.Context, channelID int64) (string, error) {
	if s.bot == nil {
		return "", fmt.Errorf("bot service is not initialized")
	}
	if channelID == 0 {
		return "", fmt.Errorf("channel ID is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	link, err := s.bot.CreateChatInviteLink(&telego.CreateChatInviteLinkParams{
		ChatID:      tu.ID(channelID),
		MemberLimit: 1,
		ExpireDate:  time.Now().Add(24 * time.Hour).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create chat invite link: %w", err)
	}
	return link.InviteLink, nil
}

func (s *BotService) SendFiscalReceipt(telegramID int64, tier string, amount float64, receiptURL string, receiptNumber string, serviceName string) error {
	if telegramID == 0 {
		return fmt.Errorf("student Telegram ID is 0")
	}

	tierDisplay := "«Акселератор»"
	switch strings.ToLower(tier) {
	case "basic":
		tierDisplay = "«Базовый»"
	case "vip":
		tierDisplay = "«VIP»"
	}

	if strings.TrimSpace(serviceName) == "" {
		serviceName = fmt.Sprintf("Информационно-консультационные услуги программы «так называемый SPARK 2027» (Тариф %s)", tierDisplay)
	}

	msk := time.FixedZone("MSK", 3*3600)
	dateStr := time.Now().In(msk).Format("02.01.2006 15:04")

	msg := fmt.Sprintf("🧾 <b>Электронный чек об оплате</b>\n\n"+
		"Здравствуйте! Ваш платёж за обучение в проекте «так называемый SPARK» успешно зарегистрирован и фискализирован в соответствии с законодательством РФ (ФЗ № 422-ФЗ / 54-ФЗ).\n\n"+
		"<b>Реквизиты операции:</b>\n"+
		"• <b>Услуга:</b> %s\n"+
		"• <b>Тариф:</b> %s\n"+
		"• <b>Сумма:</b> %.2f ₽\n",
		html.EscapeString(serviceName), tierDisplay, amount)

	cleanNumber := strings.TrimSpace(receiptNumber)
	if cleanNumber != "" {
		msg += fmt.Sprintf("• <b>Номер чека:</b> <code>%s</code>\n", html.EscapeString(cleanNumber))
	}
	msg += fmt.Sprintf("• <b>Дата формирования:</b> %s МСК\n\n", dateStr)
	msg += "Электронный чек имеет полную юридическую силу и подтверждает факт оплаты образовательных материалов."

	var kb *telego.InlineKeyboardMarkup
	receiptURL = strings.TrimSpace(receiptURL)
	var rows [][]telego.InlineKeyboardButton

	if strings.HasPrefix(receiptURL, "http://") || strings.HasPrefix(receiptURL, "https://") {
		rows = append(rows, tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🧾 Открыть официальный чек").WithURL(receiptURL),
		))
	}

	rows = append(rows, tu.InlineKeyboardRow(
		tu.InlineKeyboardButton("💬 Связаться с так называемым Илем").WithURL("https://t.me/ilyan_vas"),
	))
	kb = tu.InlineKeyboard(rows...)

	if s.bot == nil {
		log.Printf("[Bot Fiscal Receipt Mock] to %d (amount: %.2f, len=%d)", telegramID, amount, len(msg))
		return nil
	}

	studentChatID := tu.ID(telegramID)
	m := tu.Message(studentChatID, msg).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	_, err := s.bot.SendMessage(m)
	if err != nil {
		log.Printf("[Bot Fiscal Receipt] HTML send failed for %d: %v. Retrying with stripped formatting...", telegramID, err)
		cleanMsg := StripTgEmoji(msg)
		cleanM := tu.Message(studentChatID, cleanMsg).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		if _, cleanErr := s.bot.SendMessage(cleanM); cleanErr == nil {
			return nil
		}
		rawM := tu.Message(studentChatID, cleanMsg).WithReplyMarkup(kb)
		_, rawErr := s.bot.SendMessage(rawM)
		return rawErr
	}
	return nil
}
