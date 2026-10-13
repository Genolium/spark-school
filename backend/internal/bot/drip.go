package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
)

// RegisterOrUpdateDripLead registers a prospective student into the 72-hour lead nurture sequence.
func (s *BotService) RegisterOrUpdateDripLead(telegramID int64, username, firstName string, score int) {
	if s.db == nil || telegramID == 0 {
		return
	}

	// 1. If student already has access, never enroll in drip campaign
	var user models.User
	if err := s.db.Where("telegram_id = ?", telegramID).First(&user).Error; err == nil {
		if user.HasAccess {
			return
		}
	}

	var lead models.DripLead
	err := s.db.Where("telegram_id = ?", telegramID).First(&lead).Error
	if err != nil {
		// Create new lead starting at stage 0 (trigger step 1 in +2 hours)
		newLead := models.DripLead{
			TelegramID:      telegramID,
			Username:        username,
			FirstName:       firstName,
			Stage:           0,
			CalculatedScore: score,
			NextTriggerAt:   time.Now().Add(2 * time.Hour),
			IsCompleted:     false,
		}
		if err := s.db.Create(&newLead).Error; err != nil {
			log.Printf("[Drip Error] Failed to create drip lead for %d: %v", telegramID, err)
		} else {
			log.Printf("[Drip Nurture] Registered new lead %d (@%s), next trigger at +2h: %v", telegramID, username, newLead.NextTriggerAt)
		}
		return
	}

	// If lead already completed or cancelled, do not restart
	if lead.IsCompleted || lead.Stage < 0 {
		return
	}

	// If lead is at stage 0, update score and contact info if passed
	if lead.Stage == 0 {
		if score > 0 {
			lead.CalculatedScore = score
		}
		if username != "" {
			lead.Username = username
		}
		if firstName != "" {
			lead.FirstName = firstName
		}
		s.db.Save(&lead)
	}
}

// CancelDrip cancels the automated drip sequence immediately upon course purchase.
func (s *BotService) CancelDrip(telegramID int64) {
	if s.db == nil || telegramID == 0 {
		return
	}

	res := s.db.Model(&models.DripLead{}).
		Where("telegram_id = ? AND is_completed = false", telegramID).
		Updates(map[string]interface{}{
			"is_completed": true,
			"stage":        -1, // -1 means cancelled/converted
		})
	if res.RowsAffected > 0 {
		log.Printf("[Drip Nurture] Successfully cancelled drip campaign for converted student %d", telegramID)
	}
}

// startDripWorker runs background ticker processing due drip steps.
func (s *BotService) startDripWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			s.processDueDripLeads()
		}
	}
}

func (s *BotService) processDueDripLeads() {
	if s.db == nil || s.bot == nil {
		return
	}

	var dueLeads []models.DripLead
	now := time.Now()
	err := s.db.Where("is_completed = false AND stage >= 0 AND next_trigger_at <= ?", now).
		Order("next_trigger_at ASC").
		Limit(20).
		Find(&dueLeads).Error
	if err != nil || len(dueLeads) == 0 {
		return
	}

	for _, lead := range dueLeads {
		// Double check if student has purchased
		var user models.User
		if err := s.db.Where("telegram_id = ?", lead.TelegramID).First(&user).Error; err == nil {
			if user.HasAccess {
				lead.IsCompleted = true
				lead.Stage = -1
				s.db.Save(&lead)
				continue
			}
		}

		chatID := tu.ID(lead.TelegramID)

		switch lead.Stage {
		case 0:
			// Step 1: +2 hours — Напоминание со скидкой 5%
			s.sendDripStep1(chatID, lead)
			lead.Stage = 1
			lead.NextTriggerAt = now.Add(22 * time.Hour) // Total 24 hours from start
			s.db.Save(&lead)

		case 1:
			// Step 2: +24 hours — Разбор кейса эссе
			s.sendDripStep2(chatID, lead)
			lead.Stage = 2
			lead.NextTriggerAt = now.Add(24 * time.Hour) // Total 48 hours from start
			s.db.Save(&lead)

		case 2:
			// Step 3: +48 hours — Дедлайн до закрытия мест (финал)
			s.sendDripStep3(chatID, lead)
			lead.Stage = 3
			lead.IsCompleted = true
			s.db.Save(&lead)
		}
	}
}

func (s *BotService) sendDripStep1(chatID telego.ChatID, lead models.DripLead) {
	fire := TgEmoji(EmojiFire, "🔥")
	lightning := TgEmoji(EmojiLightning, "⚡️")

	score := lead.CalculatedScore
	if score <= 0 {
		score = 88
	}

	greeting := "Привет!"
	if lead.FirstName != "" {
		greeting = fmt.Sprintf("Привет, %s!", lead.FirstName)
	}

	text := fmt.Sprintf(`%s <b>Твоя бронь со скидкой 5%% на поток SPARK 2027</b>

%s Твой профиль набрал <b>%d/100</b> в расчёте шансов на грант $20,000. Это высокий потенциал, но отбор строгий: из 500+ претендентов грант получают единицы.

%s Для тебя зафиксирована <b>скидка 5%% по промокоду SPARK5</b>:
• <b>Акселератор:</b> <s>6 900 ₽</s> ➔ <b>6 555 ₽</b>
• <b>VIP:</b> <s>14 900 ₽</s> ➔ <b>14 155 ₽</b>

<i>В потоке действует строгий лимит мест на личные Zoom-интервью 1-на-1. Зафиксируй участие со скидкой прямо сейчас:</i>`, lightning, greeting, score, fire)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💳 Оплатить со скидкой (6 555 ₽)").WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Задать вопрос Илье").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)

	s.sendDripMessageWithFallback(chatID, text, kb)
	log.Printf("[Drip Nurture] Sent Step 1 (+2h) to %d", lead.TelegramID)
}

func (s *BotService) sendDripStep2(chatID telego.ChatID, lead models.DripLead) {
	idea := TgEmoji(EmojiIdea, "💡")
	star := TgEmoji(EmojiStarGreen, "⭐")
	books := TgEmoji(EmojiBooks, "📚")

	text := fmt.Sprintf(`%s <b>Разбор кейса: Почему 90%% срезаются на эссе (и как написать хук)</b>

Привет! Главная ошибка 9 из 10 кандидатов на грант в США — писать скучное академическое сочинение: <i>«Я с детства мечтал изучать науку...»</i>. 

Американская комиссия тратит <b>всего 45 секунд</b> на первое чтение и отсеивает шаблонные работы.

%s <b>Секрет победного эссе:</b>
Первое предложение должно быть сценой из жизни (Action Hook) с конкретным личным вызовом и решением. В Акселераторе мы разбираем 12 реальных выигравших эссе и переписываем твой текст строка за строкой.

%s В программе:
• Аудит эссе и резюме американского стандарта
• 45-мин персональное Zoom-интервью 1-на-1
• Пошаговый визовый гайд J-1

Забирай место в потоке, пока набор открыт:`, books, idea, star)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⚡️ Занять место в Акселераторе").WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("👑 Выбрать VIP-менторинг").WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Консультация с так называемым Илем").WithURL("https://t.me/ilyan_vas"),
		),
	)

	s.sendDripMessageWithFallback(chatID, text, kb)
	log.Printf("[Drip Nurture] Sent Step 2 (+24h) to %d", lead.TelegramID)
}

func (s *BotService) sendDripStep3(chatID telego.ChatID, lead models.DripLead) {
	warning := TgEmoji(EmojiWarning, "⚠️")

	greeting := "Привет!"
	if lead.FirstName != "" {
		greeting = fmt.Sprintf("Привет, %s!", lead.FirstName)
	}

	text := fmt.Sprintf(`⏳ <b>Финальное напоминание: места в потоке SPARK закрываются</b>

%s Мы формируем финальный список студентов на текущий поток подготовки к гранту $20,000 в США.

%s <b>Бронь скидки 5%% и свободные слоты на личные Zoom-интервью 1-на-1 сгорают сегодня.</b>

Если ты действительно хочешь провести это лето на кампусе в США за счёт гранта:
• Аудит всех документов и видеовизитки
• Личная репетиция интервью с финалистом SPARK 2026
• Визовый этап (DS-160, запись, собеседование)

Сделай шаг навстречу гранту прямо сейчас:`, greeting, warning)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⚡️ Занять последнее место (6 555 ₽)").WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Написать Илье лично").WithURL("https://t.me/ilyan_vas"),
		),
	)

	s.sendDripMessageWithFallback(chatID, text, kb)
	log.Printf("[Drip Nurture] Sent Step 3 (+48h) to %d", lead.TelegramID)
}

func (s *BotService) sendDripMessageWithFallback(chatID telego.ChatID, text string, kb *telego.InlineKeyboardMarkup) {
	if s.bot == nil {
		log.Printf("[Bot Mock] Drip message to %v: %s", chatID, StripTgEmoji(text))
		return
	}
	m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	if _, err := s.bot.SendMessage(m); err != nil {
		cleanMsg := tu.Message(chatID, StripTgEmoji(text)).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		_, _ = s.bot.SendMessage(cleanMsg)
	}
}

