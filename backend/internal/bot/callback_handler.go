package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
)

func (s *BotService) HandleCallbackQuery(query *telego.CallbackQuery) error {
	if query == nil {
		return nil
	}

	// Always sync every interacting user as a lead in Postgres CRM
	s.syncTelegramUser(&query.From)

	data := query.Data

	// Admin Receipt Approvals - Security check for admin rights
	if strings.HasPrefix(data, "receipt_approve_") {
		idStr := strings.TrimPrefix(data, "receipt_approve_")
		return s.processReceiptApproval(query, idStr)
	}
	if strings.HasPrefix(data, "receipt_reject_") {
		idStr := strings.TrimPrefix(data, "receipt_reject_")
		s.processReceiptRejection(query, idStr)
		return nil
	}
	if strings.HasPrefix(data, "receipt_tier_basic_") {
		idStr := strings.TrimPrefix(data, "receipt_tier_basic_")
		s.processReceiptTierToggle(query, idStr, "basic")
		return nil
	}
	if strings.HasPrefix(data, "receipt_tier_vip_") {
		idStr := strings.TrimPrefix(data, "receipt_tier_vip_")
		s.processReceiptTierToggle(query, idStr, "vip")
		return nil
	}
	if strings.HasPrefix(data, "receipt_tier_accelerator_") {
		idStr := strings.TrimPrefix(data, "receipt_tier_accelerator_")
		s.processReceiptTierToggle(query, idStr, "accelerator")
		return nil
	}
	if query.Message == nil {
		return nil
	}

	chatID := tu.ID(query.Message.GetChat().ID)
	session := s.getSession(query.Message.GetChat().ID)

	// Calculator Steps
	if strings.HasPrefix(data, "calc_") {
		return s.HandleCalculatorCallback(query)
	}

	if s.bot != nil {
		_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
		})
	}

	// Curator submission reviews from queue
	if strings.HasPrefix(data, "sub_review_") || strings.HasPrefix(data, "sub_changes_") {
		if !learningToolsEnabled {
			return s.sendFeaturePaused(chatID, "Проверка работ")
		}
	}
	if strings.HasPrefix(data, "sub_review_") {
		if s.IsAdmin(query.From.ID) {
			idStr := strings.TrimPrefix(data, "sub_review_")
			idNum, _ := strconv.Atoi(idStr)
			return s.ProcessSubmissionReview(chatID, &query.From, uint(idNum), "Работа проверена и одобрена куратором.", true)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Действие доступно только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return nil
	}

	if strings.HasPrefix(data, "sub_changes_") {
		if s.IsAdmin(query.From.ID) {
			idStr := strings.TrimPrefix(data, "sub_changes_")
			idNum, _ := strconv.Atoi(idStr)
			return s.ProcessSubmissionReview(chatID, &query.From, uint(idNum), "Куратор рекомендует доработать материал по комментариям в документе.", false)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Действие доступно только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return nil
	}

	// Guaranteed drip promo discount
	if strings.HasPrefix(data, "pay_drip_") {
		tier := strings.TrimPrefix(data, "pay_drip_")
		session.PromoCode = "SPARK5"
		session.DiscountPct = 5
		if s.db != nil {
			var u models.User
			if err := s.db.Where("telegram_id = ?", query.From.ID).First(&u).Error; err == nil {
				u.PromoCode = "SPARK5"
				s.db.Save(&u)
			}
		}
		s.handleSbpPayment(chatID, &query.From, tier)
		return nil
	}

	switch data {
	case "action_menu":
		session.State = StateDefault
		s.sendWelcomeMenu(chatID, &query.From)
		return nil

	case "action_status", "action_orders", "action_plan",
		"plan_toggle_1", "plan_toggle_2", "plan_toggle_3",
		"plan_toggle_4", "plan_toggle_5", "plan_toggle_6":
		if !learningToolsEnabled {
			return s.sendFeaturePaused(chatID, "Учебный кабинет")
		}
		switch data {
		case "action_status", "action_orders":
			s.SendOrdersHistory(chatID, &query.From)
		case "action_plan":
			s.SendPreparationPlan(chatID, &query.From)
		default:
			step, _ := strconv.Atoi(strings.TrimPrefix(data, "plan_toggle_"))
			s.TogglePlanStep(chatID, &query.From, step)
		}
		return nil

	// [Временно приостановлено: Сценарий 2 (Сдача работ)]
	/*
		case "action_submit_work_info":
			return s.SubmitWork(chatID, &query.From, "/submit_work")
			return nil
	*/

	case "admin_queue":
		if !learningToolsEnabled {
			return s.sendFeaturePaused(chatID, "Очередь куратора")
		}
		if s.IsAdmin(query.From.ID) {
			s.SendCuratorQueue(chatID)
			return nil
		}

	case "action_payment":
		session.State = StateWaitingReceipt
		s.sendPaymentDetails(chatID, &query.From, false)
		return nil

	case "pay_yk_menu":
		session.State = StateWaitingReceipt
		s.sendPaymentDetails(chatID, &query.From, false)
		return nil

	case "pay_yk_basic", "pay_sbp_basic":
		s.handleSbpPayment(chatID, &query.From, "basic")
		return nil

	case "pay_yk_accelerator", "pay_sbp_accelerator":
		s.handleSbpPayment(chatID, &query.From, "accelerator")
		return nil

	case "pay_yk_vip", "pay_sbp_vip":
		s.handleSbpPayment(chatID, &query.From, "vip")
		return nil

	case "pay_upsell_audit":
		s.handleAuditUpsellPayment(chatID, &query.From)
		return nil

	case "action_upgrade":
		s.sendUpgradeOptions(chatID, &query.From)
		return nil

	case "upgrade_to_accelerator":
		pricing := s.getPricing()
		diff := pricing.AcceleratorPrice - pricing.BasicPrice
		if diff <= 0 {
			diff = 2910.0
		}
		s.handleUpgradePayment(chatID, &query.From, "accelerator", diff)
		return nil

	case "upgrade_to_vip":
		pricing := s.getPricing()
		diff := pricing.VIPPrice - pricing.AcceleratorPrice
		if query.From.ID != 0 && s.db != nil {
			var u models.User
			if err := s.db.Where("telegram_id = ?", query.From.ID).First(&u).Error; err == nil {
				if strings.ToLower(strings.TrimSpace(u.Tier)) == "basic" {
					diff = pricing.VIPPrice - pricing.BasicPrice
				}
			}
		}
		if diff <= 0 {
			diff = 8000.0
		}
		s.handleUpgradePayment(chatID, &query.From, "vip", diff)
		return nil

	case "pay_sbp_info":
		session.State = StateWaitingReceipt
		s.handleSbpPayment(chatID, &query.From, "accelerator")
		return nil

	case "promo_enter":
		session.State = StateWaitingPromo
		checkCircle := TgEmoji(EmojiCheckCircleGreen, "✅")
		sparkle := TgEmoji(EmojiSparkleGreen, "✨")
		pricing := s.getPricing()

		text := fmt.Sprintf(`%s <b>Активация промокода на скидку 5%%</b> %s

Напиши промокод прямо в ответном сообщении!

<i>Укажи промокод твоего друга-партнёра или из официального блога.</i>

Скидка 5%% действует на любой тариф:
• Акселератор: %s ₽ ➔ <b>%s ₽</b>
• VIP: %s ₽ ➔ <b>%s ₽</b>`, checkCircle, sparkle,
			formatPriceRub(int(pricing.AcceleratorPrice)), formatPriceRub(int(pricing.AcceleratorPrice*0.95)),
			formatPriceRub(int(pricing.VIPPrice)), formatPriceRub(int(pricing.VIPPrice*0.95)))
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(PromoRequestKeyboard())
		if _, err := s.bot.SendMessage(m); err != nil {
			log.Printf("[Bot Error] promo_enter SendMessage failed: %v. Retrying stripped...", err)
			cleanText := StripTgEmoji(text)
			cleanMsg := tu.Message(chatID, cleanText).WithParseMode(telego.ModeHTML).WithReplyMarkup(PromoRequestKeyboard())
			_, _ = s.bot.SendMessage(cleanMsg)
		}
		return nil

	case "info_program":
		star := TgEmoji(EmojiSparkleGreen, "✨")
		spark := TgEmoji(EmojiStarGreen, "⭐")
		num1 := TgEmoji(EmojiNum1, "1️⃣")
		num2 := TgEmoji(EmojiNum2, "2️⃣")
		num3 := TgEmoji(EmojiNum3, "3️⃣")
		pricing := s.getPricing()

		text := fmt.Sprintf(`%s <b>Программа сопровождения «так называемый SPARK»</b> %s

• <b>Закрытый канал потока</b> со всеми материалами и апдейтами
• <b>Подробный курс по каждому элементу заявки</b>
• <b>X-Factor Video framework:</b> режиссура и хуки первых 5 секунд
• <b>Банк вопросов</b> к Zoom-интервью с комиссией
• <b>Языковая подготовка</b> к собеседованию
• <b>Визовый этап:</b> DS-160, запись в дипмиссию, привязка к родине
• <b>Персональный аудит</b> и созвон 45–60 мин с симуляцией 20-мин интервью с так называемым Илем

<b>Тарифные планы:</b>
%s <b>Базовый:</b> %s ₽ (%s ₽ по промокоду) <i>[без обратной связи]</i>
%s <b>Акселератор [Хит]:</b> %s ₽ (%s ₽ по промокоду)
%s <b>VIP:</b> %s ₽ (%s ₽ по промокоду)`, star, spark,
			num1, formatPriceRub(int(pricing.BasicPrice)), formatPriceRub(int(pricing.BasicPrice*0.95)),
			num2, formatPriceRub(int(pricing.AcceleratorPrice)), formatPriceRub(int(pricing.AcceleratorPrice*0.95)),
			num3, formatPriceRub(int(pricing.VIPPrice)), formatPriceRub(int(pricing.VIPPrice*0.95)))

		btnPay := tu.InlineKeyboardButton(fmt.Sprintf("%s Купить курс", "💳")).WithCallbackData("action_payment")
		btnPromo := tu.InlineKeyboardButton(fmt.Sprintf("%s Применить промокод -5%%", "🎟")).WithCallbackData("promo_enter")
		btnBlog := tu.InlineKeyboardButton("🇺🇸 Блог про мою поездку в США").WithURL("https://t.me/so_called_spark")
		btnMenu := tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")

		kb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(btnPay),
			tu.InlineKeyboardRow(btnPromo),
			tu.InlineKeyboardRow(btnBlog),
			tu.InlineKeyboardRow(btnMenu),
		)

		photoFile := s.ResolvePhotoFile(MediaBuyCourse)
		photoMsg := tu.Photo(chatID, photoFile).
			WithCaption(text).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(kb)

		sentMsg, err := s.bot.SendPhoto(photoMsg)
		if err == nil && sentMsg != nil {
			s.CacheSentPhoto(MediaBuyCourse, sentMsg)
		} else if err != nil {
			m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
			_, _ = s.bot.SendMessage(m)
		}
		return nil

	case "admin_menu":
		if s.IsAdmin(query.From.ID) {
			s.SendAdminDashboard(chatID)
			return nil
		}

	case "admin_stats":
		if s.IsAdmin(query.From.ID) {
			s.SendAdminLiveStats(chatID)
			return nil
		}

	case "admin_digest":
		if s.IsAdmin(query.From.ID) {
			_, _ = s.SendCuratorDailyDigest(chatID.ID)
			return nil
		}

	case "admin_receipts":
		if s.IsAdmin(query.From.ID) {
			s.SendAdminReceipts(chatID)
			return nil
		}

	case "admin_promos":
		if s.IsAdmin(query.From.ID) && s.db != nil {
			var promos []models.PromoCode
			s.db.Order("id desc").Limit(10).Find(&promos)
			sb := strings.Builder{}
			sb.WriteString("🎟 <b>Список активных промокодов:</b>\n\n")
			for _, p := range promos {
				statusIcon := "🟢"
				if !p.IsActive {
					statusIcon = "🔴"
				}
				sb.WriteString(fmt.Sprintf("%s <code>%s</code> — <b>%d%%</b> (исп: %d)\n", statusIcon, p.Code, p.DiscountPercent, p.UsesCount))
			}
			sb.WriteString("\nСоздать новый: <code>/promo create КОД 10%</code>")
			btnBack := tu.InlineKeyboardButton("⬅️ В админку").WithCallbackData("admin_menu")
			m := tu.Message(chatID, sb.String()).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnBack)))
			_, _ = s.bot.SendMessage(m)
			return nil
		}

	case "admin_broadcast_prompt":
		if s.IsAdmin(query.From.ID) {
			session.State = StateAdminBroadcast
			text := `📢 <b>Режим рассылки сообщений</b>

Отправь следующее сообщение с текстом анонса, и бот доставит его <b>каждому пользователю</b> из базы данных!

<i>Поддерживается HTML-разметка: &lt;b&gt;жирный&lt;/b&gt;, &lt;i&gt;курсив&lt;/i&gt;, ссылки.</i>

Для отмены просто напиши /admin.`
			m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML)
			_, _ = s.bot.SendMessage(m)
			return nil
		}
	}

	return nil
}

func (s *BotService) answerCallback(query *telego.CallbackQuery, text string, alert bool) {
	if s.bot == nil || query == nil {
		return
	}
	_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            text,
		ShowAlert:       alert,
	})
}

func (s *BotService) sendFeaturePaused(chatID telego.ChatID, feature string) error {
	msg := fmt.Sprintf("ℹ️ <b>%s</b> временно недоступен.\n\nДоступны выбор тарифа, оплата, промокод и диагностика.", feature)
	return s.send(tu.Message(chatID, msg).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("💳 Выбрать тариф").WithCallbackData("action_payment")),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
	)))
}
