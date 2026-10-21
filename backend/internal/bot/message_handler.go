package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
)

func (s *BotService) handleMessage(msg *telego.Message) error {
	if msg == nil {
		return nil
	}
	if msg.From == nil {
		return fmt.Errorf("telegram message has no sender")
	}

	fromID := int64(0)
	if msg.From != nil {
		fromID = msg.From.ID
	}
	// Never log message text or names: they contain personal data.
	log.Printf("[Bot Inbound] chat_id=%d, type=%s, from_id=%d, text_len=%d", msg.Chat.ID, msg.Chat.Type, fromID, len(msg.Text))

	// Private Chat Filter (R5): strictly ignore group, supergroup, channel messages
	if msg.Chat.Type != "private" || msg.Chat.ID <= 0 {
		log.Printf("[Bot Inbound Ignored] chat_type=%s, chat_id=%d", msg.Chat.Type, msg.Chat.ID)
		return nil
	}

	// Always persist and sync every interacting user as a lead in Postgres CRM
	if msg.From != nil {
		s.syncTelegramUser(msg.From)
	}

	chatID := tu.ID(msg.Chat.ID)
	session := s.getSession(msg.Chat.ID)

	// Check if message is a photo/document receipt
	if len(msg.Photo) > 0 || msg.Document != nil {
		return s.HandleReceiptUpload(msg)
	}

	text := strings.TrimSpace(msg.Text)

	// Check if in StateWaitingPromo
	if session.State == StateWaitingPromo && !strings.HasPrefix(text, "/") {
		s.processPromoInput(chatID, msg.From, text)
		return nil
	}

	if strings.HasPrefix(text, "/withdraw") {
		s.handleWithdraw(chatID, msg.From)
		return nil
	}

	if strings.HasPrefix(text, "/start") {
		session.State = StateDefault
		parts := strings.Fields(text)
		param := ""
		if len(parts) > 1 {
			param = parts[1]
		}

		if token := diagnosticTokenFromStart(param); token != "" {
			return s.handleDiagnosticPlan(chatID, msg.From, token)
		}

		if param == "pay_audit" || param == "upsell_audit" || param == "audit" || strings.HasPrefix(param, "pay_audit") {
			s.handleAuditUpsellPayment(chatID, msg.From)
			return nil
		}

		if strings.HasPrefix(param, "pay") {
			payload := strings.TrimPrefix(param, "pay_")
			if param == "pay" {
				payload = ""
			}

			if strings.HasPrefix(payload, "audit") {
				s.handleAuditUpsellPayment(chatID, msg.From)
				return nil
			}

			// 1. Extract tier
			tier := "accelerator"
			if strings.HasPrefix(payload, "basic") {
				tier = "basic"
				payload = strings.TrimPrefix(payload, "basic")
				payload = strings.TrimPrefix(payload, "_")
			} else if strings.HasPrefix(payload, "vip") {
				tier = "vip"
				payload = strings.TrimPrefix(payload, "vip")
				payload = strings.TrimPrefix(payload, "_")
			} else if strings.HasPrefix(payload, "accelerator") {
				tier = "accelerator"
				payload = strings.TrimPrefix(payload, "accelerator")
				payload = strings.TrimPrefix(payload, "_")
			}
			session.SelectedTier = tier

			// 2. Extract the short landing attribution token.
			attrToken := ""
			if idx := strings.Index(payload, "_a_"); idx != -1 {
				attrToken = sanitizeAttributionToken(payload[idx+len("_a_"):])
				payload = payload[:idx]
			}

			// 3. Extract referral or promo code
			refCode := ""
			username := payload
			if idx := strings.Index(payload, "_ref_"); idx != -1 {
				username = payload[:idx]
				refCode = payload[idx+len("_ref_"):]
			} else if strings.HasPrefix(payload, "ref_") {
				username = ""
				refCode = strings.TrimPrefix(payload, "ref_")
			}
			username = strings.TrimPrefix(username, "@")

			if refCode != "" && msg.From != nil {
				s.associateBotReferral(msg.From, username, refCode)
			}

			// 3. If payload contains a valid active promo code in DB, auto-apply it
			withPromo := false
			if username != "" && s.db != nil {
				var promo models.PromoCode
				if err := s.db.Where("UPPER(code) = ? AND is_active = true", strings.ToUpper(username)).First(&promo).Error; err == nil && promo.ID != 0 {
					session.PromoCode = promo.Code
					discount := promo.DiscountPercent
					if discount <= 0 {
						discount = 5
					}
					session.DiscountPct = discount
					withPromo = true
				}
			}

			session.SelectedTier = tier
			if u := s.syncTelegramUser(msg.From); u != nil {
				// Do NOT overwrite user's paid tier if they already have access!
				if !u.HasAccess && tier != "" {
					u.Tier = tier
				}
				if withPromo && session.PromoCode != "" && u.PromoCode == "" {
					u.PromoCode = session.PromoCode
				}
				s.db.Save(u)
			}
			if attrToken != "" {
				if err := s.attributeLead(msg.From, attrToken); err != nil {
					log.Printf("[Analytics] failed to attribute bot lead: %v", err)
				}
			}

			s.sendPaymentDetails(chatID, msg.From, withPromo)
			return nil
		}

		if strings.HasPrefix(param, "withdraw") {
			s.handleWithdraw(chatID, msg.From)
			return nil
		}

		if strings.HasPrefix(param, "ref_") {
			refCode := strings.TrimPrefix(param, "ref_")
			if msg.From != nil {
				s.associateBotReferral(msg.From, msg.From.Username, refCode)
			}
		}

		s.sendWelcomeMenu(chatID, msg.From)
		return nil
	}

	// Curator Broadcast input state
	if session.State == StateAdminBroadcast && !strings.HasPrefix(text, "/") {
		session.State = StateDefault
		s.ExecuteBroadcast(chatID, text)
		return nil
	}

	// Curator In-line Command: /admin
	if strings.HasPrefix(text, "/admin") {
		if msg.From != nil && s.IsAdmin(msg.From.ID) {
			s.SendAdminDashboard(chatID)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return nil
	}

	// Curator In-line Command: /promo create [CODE] [DISCOUNT%]
	if strings.HasPrefix(text, "/promo") {
		if msg.From != nil && s.IsAdmin(msg.From.ID) {
			s.HandleAdminPromoCommand(chatID, msg.From, text)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return nil
	}

	// Curator In-line Command: /digest
	if strings.HasPrefix(text, "/digest") {
		if msg.From != nil && s.IsAdmin(msg.From.ID) {
			_, _ = s.SendCuratorDailyDigest(chatID.ID)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return nil
	}

	// Diagnostic tool for Telegram Premium Custom Emojis
	if strings.HasPrefix(text, "/emoji") {
		type detectedEmoji struct {
			fallback string
			id       string
		}
		var detected []detectedEmoji

		// Telegram entity offsets are in UTF-16 code units
		utf16Text := utf16.Encode([]rune(msg.Text))
		for _, entity := range msg.Entities {
			if entity.Type == telego.EntityTypeCustomEmoji && entity.CustomEmojiID != "" {
				char := ""
				if entity.Offset >= 0 && entity.Offset+entity.Length <= len(utf16Text) {
					sub := utf16.Decode(utf16Text[entity.Offset : entity.Offset+entity.Length])
					char = string(sub)
				}
				// Default to sparkle if extracted char is empty or whitespace
				if strings.TrimSpace(char) == "" {
					char = "✨"
				}
				detected = append(detected, detectedEmoji{
					fallback: char,
					id:       entity.CustomEmojiID,
				})
			}
		}

		if len(detected) > 0 {
			reply := "🎉 <b>Обнаружены Custom Emoji ID:</b>\n"
			plainReply := "🎉 Обнаружены Custom Emoji ID:\n"
			for i, e := range detected {
				reply += fmt.Sprintf("%d.  %s  <tg-emoji emoji-id=\"%s\">%s</tg-emoji>  <code>%s</code>\n",
					i+1, e.fallback, e.id, e.fallback, e.id)
				plainReply += fmt.Sprintf("%d.  %s  ID: %s\n", i+1, e.fallback, e.id)
			}
			reply += "\n<i>Скопируйте нужные строчки сюда, и мы внедрим их в бота!</i>"
			plainReply += "\nСкопируйте нужные строчки сюда, и мы внедрим их в бота!"

			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
			if _, err := s.bot.SendMessage(m); err != nil {
				log.Printf("[Bot Error] /emoji SendMessage with <tg-emoji> failed: %v. Sending plain fallback...", err)
				plainMsg := tu.Message(chatID, plainReply)
				if _, plainErr := s.bot.SendMessage(plainMsg); plainErr != nil {
					log.Printf("[Bot Error] /emoji fallback SendMessage also failed: %v", plainErr)
				}
			}
		} else {
			log.Printf("[Bot /emoji] No custom emoji entities found. EntitiesCount=%d", len(msg.Entities))
			reply := `ℹ️ <b>Как узнать Custom Emoji ID:</b>

Отправьте команду <code>/emoji</code> вместе с эмодзи из вашего пака прямо в одном сообщении, например:
<code>/emoji </code> [вставьте эмодзи из пака]

Бот моментально покажет символ, сам эмодзи и его ID!`
			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
			if _, err := s.bot.SendMessage(m); err != nil {
				log.Printf("[Bot Error] /emoji info SendMessage failed: %v", err)
			}
		}
		return nil
	}

	if strings.HasPrefix(text, "/status") || strings.HasPrefix(text, "/orders") ||
		strings.HasPrefix(text, "/plan") || strings.HasPrefix(text, "/done_step") {
		if !learningToolsEnabled {
			return s.sendFeaturePaused(chatID, "Учебный кабинет")
		}
		if strings.HasPrefix(text, "/plan") {
			s.SendPreparationPlan(chatID, msg.From)
		} else if strings.HasPrefix(text, "/done_step") {
			parts := strings.Fields(text)
			stepNum := 1
			if len(parts) > 1 {
				if num, err := strconv.Atoi(parts[1]); err == nil {
					stepNum = num
				}
			}
			s.TogglePlanStep(chatID, msg.From, stepNum)
		} else {
			s.SendOrdersHistory(chatID, msg.From)
		}
		return nil
	}

	// [Временно приостановлено: Сценарий 2 (Сдача работ на аудит)]
	/*
		if strings.HasPrefix(text, "/submit_work") {
			return s.SubmitWork(chatID, msg.From, text)
			return nil
		}
	*/

	if strings.HasPrefix(text, "/queue") {
		if !learningToolsEnabled {
			return s.sendFeaturePaused(chatID, "Очередь куратора")
		}
		if msg.From != nil && s.IsAdmin(msg.From.ID) {
			s.SendCuratorQueue(chatID)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return nil
	}

	/*
		if strings.HasPrefix(text, "/reviewed") {
			return s.HandleReviewedCommand(chatID, msg.From, text)
			return nil
		}

		if strings.HasPrefix(text, "/changes") {
			return s.HandleChangesCommand(chatID, msg.From, text)
			return nil
		}
	*/

	if strings.HasPrefix(text, "/notifications_off") || strings.HasPrefix(text, "/stop_reminders") {
		return s.setRemindersOptOut(chatID, msg.From, true)
	}

	if strings.HasPrefix(text, "/notifications_on") || strings.HasPrefix(text, "/start_reminders") {
		return s.setRemindersOptOut(chatID, msg.From, false)
	}

	if strings.HasPrefix(text, "/upgrade") {
		s.sendUpgradeOptions(chatID, msg.From)
		return nil
	}

	// State-dependent handling:
	// Only parse promo code if user actually requested to enter a promo code
	if session.State == StateWaitingPromo && text != "" && !strings.HasPrefix(text, "/") {
		s.processPromoInput(chatID, msg.From, text)
		return nil
	}

	// Default fallback for any unexpected text: guide user back to main menu
	if text != "" && !strings.HasPrefix(text, "/") {
		helpMsg := tu.Message(chatID, `👋 Я вас понял! Чтобы выбрать действие, воспользуйтесь кнопками меню ниже или отправьте команду /start:`)
		helpMsg.WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL, s.getPricing()))
		_, _ = s.bot.SendMessage(helpMsg)
		return nil
	}

	return nil
}
