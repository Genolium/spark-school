package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
)

const lowerAcceleratorPrice = 9990.0

// pricingForSession applies the landing price experiment to the bot price too,
// so the amount shown on the site and the amount requested in Telegram match.
func (s *BotService) pricingForSession(session *UserSession) models.CoursePricing {
	pricing := s.getPricing()
	if session != nil && session.PriceExperiment == "lower" {
		pricing.AcceleratorPrice = lowerAcceleratorPrice
	}
	return pricing
}

func (s *BotService) sendPaymentDetails(chatID telego.ChatID, from *telego.User, withPromo bool) {
	if s.bot == nil {
		return
	}

	session := s.getSession(chatID.ID)
	session.State = StateWaitingReceipt

	if from != nil {
		s.RegisterOrUpdateDripLead(chatID.ID, from.Username, from.FirstName, 0)
	}

	var text string
	var keyboard *telego.InlineKeyboardMarkup

	sparkle := TgEmoji(EmojiStarGreen, "✨")
	clipGreen := TgEmoji(EmojiPaperclipGreen, "📎")

	num1 := TgEmoji(EmojiNum1, "1️⃣")
	num2 := TgEmoji(EmojiNum2, "2️⃣")
	num3 := TgEmoji(EmojiNum3, "3️⃣")
	paperclip := TgEmoji(EmojiPaperclip, "📎")

	pricing := s.pricingForSession(session)
	basicBase := int(pricing.BasicPrice)
	accBase := int(pricing.AcceleratorPrice)
	vipBase := int(pricing.VIPPrice)
	bankName := pricing.BankName
	if strings.TrimSpace(bankName) == "" {
		bankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

	if withPromo || session.DiscountPct > 0 {
		pct := session.DiscountPct
		if pct <= 0 {
			pct = 5
		}
		basicDiscount := (basicBase * pct) / 100
		basicPrice := basicBase - basicDiscount
		accDiscount := (accBase * pct) / 100
		accPrice := accBase - accDiscount
		vipDiscount := (vipBase * pct) / 100
		vipPrice := vipBase - vipDiscount

		text = fmt.Sprintf(`%s <b>Промокод успешно активирован! Скидка %d%%</b>

<b>Стоимость со скидкой %d%%:</b>
%s <b>Базовый:</b> <b>%s ₽</b> (скидка %s ₽) <i>[без обратной связи]</i>
%s <b>Акселератор [Хит]:</b> <b>%s ₽</b> (скидка %s ₽)
%s <b>VIP:</b> <b>%s ₽</b> (скидка %s ₽)

🏛 <b>Прямой перевод по СБП (0%% без комиссии):</b>
• Банк: <b>%s</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение платежа: <code>SPARK ПРОМО [Тариф]</code>

%s <b>Как получить доступ:</b>
1. Выберите тариф кнопкой ниже или переведите нужную сумму по СБП.
2. <b>Отправьте скриншот чека прямо сюда в чат!</b>
3. Бот проверит чек и сразу выдаст вам одноразовую ссылку в закрытый канал потока.`, sparkle, pct, pct, num1, formatPriceRub(basicPrice), formatPriceRub(basicDiscount), num2, formatPriceRub(accPrice), formatPriceRub(accDiscount), num3, formatPriceRub(vipPrice), formatPriceRub(vipDiscount), bankName, clipGreen)
		keyboard = PromoAppliedKeyboard(pricing)
	} else {
		basicPromoPrice := basicBase - (basicBase*5)/100
		accPromoPrice := accBase - (accBase*5)/100
		vipPromoPrice := vipBase - (vipBase*5)/100

		text = fmt.Sprintf(`%s <b>Бронирование места в проекте «так называемый SPARK»</b>

<b>Тарифы участия:</b>
%s <b>Базовый:</b> %s ₽ (по промокоду: <b>%s ₽</b>) <i>[без обратной связи]</i>
%s <b>Акселератор [Хит]:</b> %s ₽ (по промокоду: <b>%s ₽</b>)
%s <b>VIP:</b> %s ₽ (по промокоду: <b>%s ₽</b>) <i>[Строго ограниченное количество мест]</i>

🏛 <b>Оплата через СБП (0%% без комиссии):</b>
• Банк: <b>%s</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение: <code>SPARK [Тариф]</code>

%s <b>Как получить доступ:</b>
1. Выберите тариф кнопкой ниже или сделайте перевод по номеру телефона через СБП.
2. <b>Отправьте скриншот чека прямо сюда в чат бота!</b>
3. Бот зафиксирует платёж и сразу выдаст вам персональную ссылку в закрытый канал потока.`, sparkle, num1, formatPriceRub(basicBase), formatPriceRub(basicPromoPrice), num2, formatPriceRub(accBase), formatPriceRub(accPromoPrice), num3, formatPriceRub(vipBase), formatPriceRub(vipPromoPrice), bankName, paperclip)
		keyboard = PaymentKeyboard(pricing)
	}

	targetMedia := MediaBuyCourse
	if withPromo || session.DiscountPct > 0 {
		targetMedia = MediaPromoApplied
	}

	photoFile := s.ResolvePhotoFile(targetMedia)
	photoMsg := tu.Photo(chatID, photoFile).
		WithCaption(text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(keyboard)

	sentMsg, err := s.bot.SendPhoto(photoMsg)
	if err == nil && sentMsg != nil {
		s.CacheSentPhoto(targetMedia, sentMsg)
	} else if err != nil {
		log.Printf("[Bot Error] sendPaymentDetails SendPhoto failed: %v. Sending fallback...", err)
		cleanText := StripTgEmoji(text)
		m := tu.Message(chatID, cleanText).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		if _, textErr := s.bot.SendMessage(m); textErr != nil {
			plainMsg := tu.Message(chatID, cleanText).WithReplyMarkup(keyboard)
			_, _ = s.bot.SendMessage(plainMsg)
		}
	}
}

func (s *BotService) handleSbpPayment(chatID telego.ChatID, from *telego.User, tier string) {
	session := s.getSession(chatID.ID)
	session.State = StateWaitingReceipt
	session.SelectedTier = tier

	if s.bot == nil {
		return
	}

	hasPromo := session.DiscountPct > 0 || session.PromoCode != ""

	pricing := s.pricingForSession(session)
	tierTitle := "Акселератор"
	basePrice := int(pricing.AcceleratorPrice)
	if tier == "basic" {
		tierTitle = "Базовый"
		basePrice = int(pricing.BasicPrice)
	} else if tier == "vip" {
		tierTitle = "VIP"
		basePrice = int(pricing.VIPPrice)
	}

	price := basePrice
	discountText := ""
	if hasPromo {
		pct := session.DiscountPct
		if pct <= 0 {
			pct = 5
		}
		discount := (basePrice * pct) / 100
		price = basePrice - discount
		discountText = fmt.Sprintf(" (скидка %d%% по промокоду <code>%s</code>)", pct, session.PromoCode)
	}

	bankName := pricing.BankName
	if strings.TrimSpace(bankName) == "" {
		bankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

	offerCondition := "услуга считается оказанной в полном объёме в момент предоставления доступа к закрытому каналу и материалам курса"
	if tier == "accelerator" || tier == "vip" {
		offerCondition = "услуга считается оказанной при добавлении в закрытый канал и одном раунде обратной связи"
	}

	sbpText := fmt.Sprintf(`🏛 <b>Оплата тарифа «%s» через СБП (0%% без комиссии)</b>

💰 <b>Сумма к переводу: %s ₽</b>%s

<b>Реквизиты для перевода:</b>
• Банк: <b>%s</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение платежа: <code>SPARK %s</code>

📎 <b>Как получить доступ:</b>
1. Совершите перевод по номеру телефона через СБП в вашем банковском приложении.
2. <b>Отправьте скриншот чека прямо в этот чат бота!</b>
3. Бот проверит чек и сразу выдаст вам персональную ссылку в закрытый канал потока.

ℹ️ <i>Оплачивая участие, вы соглашаетесь с условиями <a href="https://so-called-spark.ru/offer">Публичной оферты</a> (%s) и <a href="https://so-called-spark.ru/privacy">Политики конфиденциальности</a>.</i>

<i>При любых вопросах вы всегда можете написать так называемому Илю (@ilyan_vas).</i>`, tierTitle, formatPriceRub(price), discountText, bankName, strings.ToUpper(tier), offerCondition)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📄 Оферта").WithURL("https://so-called-spark.ru/offer"),
			tu.InlineKeyboardButton("💬 Написать Илю").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ Назад к тарифам").WithCallbackData("action_payment"),
		),
	)

	s.recordOrderIntent(from, tier, float64(price), session.PromoCode, "telegram_sbp")

	m := tu.Message(chatID, sbpText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	if _, sendErr := s.bot.SendMessage(m); sendErr != nil {
		cleanMsg := tu.Message(chatID, StripTgEmoji(sbpText)).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		_, _ = s.bot.SendMessage(cleanMsg)
	}
}

func (s *BotService) handleYooKassaPayment(chatID telego.ChatID, from *telego.User, tier string) {
	s.handleSbpPayment(chatID, from, tier)
}

func (s *BotService) handleAuditUpsellPayment(chatID telego.ChatID, from *telego.User) {
	if !learningToolsEnabled {
		s.sendFeaturePaused(chatID, "Аудит эссе")
		return
	}
	session := s.getSession(chatID.ID)
	session.State = StateWaitingReceipt
	session.SelectedTier = "audit_essay"
	session.IsUpgrade = true
	session.UpgradeAmount = 1900.0
	session.UpgradeToTier = "audit_essay"

	if s.bot == nil {
		return
	}

	pricing := s.getPricing()
	bankName := pricing.BankName
	if strings.TrimSpace(bankName) == "" {
		bankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

	s.recordOrderIntent(from, "audit_essay", 1900, session.PromoCode, "telegram_sbp")

	text := fmt.Sprintf(`✍️ <b>Оплата текстового аудита эссе — 1 900 ₽</b>

Вы получите подробный построчный аудит черновика вашего эссе на 500 слов с комментариями, правками структуры и формулировок от финалиста SPARK 2026.

💰 <b>Сумма к переводу: 1 900 ₽</b>

<b>Реквизиты для перевода через СБП:</b>
• Банк: <b>%s</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение платежа: <code>АУДИТ ЭССЕ</code>

📎 <b>Как получить аудит:</b>
1. Совершите перевод <b>1 900 ₽</b> через СБП в вашем банке.
2. <b>Отправьте скриншот или PDF чека прямо в этот чат!</b>
3. Куратор проверит платёж и сразу подтвердит доступ к проверке черновика в Google Docs.

<i>Разбор с правками готовится в течение 24–48 часов после получения текста. По всем вопросам: @spark_prep_bot</i>`, bankName)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Задать вопрос так называемому Илю").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)

	m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	if _, sendErr := s.bot.SendMessage(m); sendErr != nil {
		cleanMsg := tu.Message(chatID, StripTgEmoji(text)).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		_, _ = s.bot.SendMessage(cleanMsg)
	}
}

func (s *BotService) processPromoInput(chatID telego.ChatID, from *telego.User, rawCode string) {
	if s.bot == nil {
		return
	}

	code := strings.ToUpper(strings.TrimSpace(rawCode))
	session := s.sessions.Get(chatID.ID)

	// Validate promo code in DB
	var promo models.PromoCode
	err := s.db.Where("UPPER(code) = ? AND is_active = true", code).First(&promo).Error

	if err == nil && promo.ID != 0 {
		session.PromoCode = promo.Code
		discount := promo.DiscountPercent
		if discount <= 0 {
			discount = 5
		}
		session.DiscountPct = discount
		session.State = StateWaitingReceipt

		if u := s.syncTelegramUser(from); u != nil {
			u.PromoCode = promo.Code
			s.db.Save(u)
		}

		s.sendPaymentDetails(chatID, from, true)
	} else {
		session.State = StateDefault
		text := `❌ <b>Промокод не найден или отключён.</b>

Проверь написание и отправь код ещё раз — или оплати без скидки.`
		keyboard := tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("🎟 Ввести другой промокод").WithCallbackData("promo_enter"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("💳 Оплатить без промокода").WithCallbackData("action_payment"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
			),
		)
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		_, _ = s.bot.SendMessage(m)
	}
}

func (s *BotService) sendStatusMessage(chatID telego.ChatID, from *telego.User) {
	if s.bot == nil {
		return
	}

	var user models.User
	hasAccess := false
	if from != nil && s.db != nil {
		if err := s.db.Where("telegram_id = ?", from.ID).First(&user).Error; err == nil {
			hasAccess = user.HasAccess
		}
	}

	if hasAccess {
		inviteLink := s.cfg.TelegramInviteLink
		heart := TgEmoji(EmojiHeartSolidGreen, "💚")
		sun := TgEmoji(EmojiSunGreen, "☀️")

		tierTitle := "Акселератор"
		if strings.ToLower(strings.TrimSpace(user.Tier)) == "vip" {
			tierTitle = "VIP"
		}

		statusDetails := s.studentStatusDetails(&user)
		reply := fmt.Sprintf(`%s <b>Ваш доступ активен!</b>

🎓 Тариф: <b>%s</b>

%s Добро пожаловать в проект <b>«так называемый SPARK»</b>.
Ваша ссылка в закрытый канал потока:
%s%s`, heart, tierTitle, sun, inviteLink, statusDetails)

		btnChannel := tu.InlineKeyboardButton("Открыть закрытый канал 🚀").WithURL(inviteLink)
		rows := [][]telego.InlineKeyboardButton{
			{btnChannel},
		}
		if strings.ToLower(strings.TrimSpace(user.Tier)) != "vip" {
			rows = append(rows, []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton("🚀 Апгрейд тарифа").WithCallbackData("action_upgrade"),
			})
		}
		kb := tu.InlineKeyboard(rows...)

		photoFile := s.ResolvePhotoFile(MediaStatusGranted)
		photoMsg := tu.Photo(chatID, photoFile).
			WithCaption(reply).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(kb)

		sentMsg, err := s.bot.SendPhoto(photoMsg)
		if err == nil && sentMsg != nil {
			s.CacheSentPhoto(MediaStatusGranted, sentMsg)
		} else {
			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
			_, _ = s.bot.SendMessage(m)
		}
		return
	}

	pricing := s.getPricing()
	reply := fmt.Sprintf(`ℹ️ <b>Статус доступа: Не активен</b>

У вас пока нет активного доступа к проекту <b>«так называемый SPARK»</b>.
Тарифы: <b>Акселератор</b> (%s ₽), <b>VIP</b> (%s ₽).

После подтверждения оплаты так называемый Иль предоставит персональную ссылку в закрытый канал.`, formatPriceRub(int(pricing.AcceleratorPrice)), formatPriceRub(int(pricing.VIPPrice)))
	btnTariff := tu.InlineKeyboardButton("Оформить тариф ➔").WithCallbackData("action_payment")
	btnCurator := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
	m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnTariff), tu.InlineKeyboardRow(btnCurator)))
	_, _ = s.bot.SendMessage(m)
}

func (s *BotService) handleWithdraw(chatID telego.ChatID, from *telego.User) {
	balanceInfo := ""
	if from != nil && s.db != nil {
		var user models.User
		if err := s.db.Preload("AffiliateProfile").Where("telegram_id = ?", from.ID).First(&user).Error; err == nil && user.AffiliateProfile != nil {
			balanceInfo = fmt.Sprintf("\n\n💰 <b>Ваш партнёрский баланс:</b> %.0f ₽\nДоступно к выводу: %.0f ₽", user.AffiliateProfile.CurrentBalance, user.AffiliateProfile.CurrentBalance)
		}
	}

	reply := fmt.Sprintf(`💼 <b>Вывод средств партнёрской программы проекта «так называемый SPARK»</b>%s

Чтобы запросить выплату накопленного вознаграждения:
1. Откройте партнёрский кабинет по кнопке ниже
2. Укажите сумму и реквизиты (СБП / номер карты)
3. Заявка будет обработана в течение 24 часов`, balanceInfo)

	btnPartner := tu.InlineKeyboardButton("Открыть кабинет партнёра ➔").WithURL(s.cfg.FrontendURL + "/partner")
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnPartner))

	m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
	if s.bot != nil {
		_, _ = s.bot.SendMessage(m)
	}
}

func (s *BotService) sendUpgradeOptions(chatID telego.ChatID, from *telego.User) {
	if s.bot == nil {
		return
	}

	var user models.User
	userFound := false
	if from != nil && s.db != nil {
		if err := s.db.Where("telegram_id = ?", from.ID).First(&user).Error; err == nil {
			userFound = true
		}
	}

	if !userFound || !user.HasAccess {
		reply := `ℹ️ <b>Апгрейд тарифа</b>

У вас пока нет активного тарифа. Выберите подходящий тариф для старта подготовки:`
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(PaymentKeyboard())
		_, _ = s.bot.SendMessage(m)
		return
	}

	currentTier := strings.ToLower(strings.TrimSpace(user.Tier))
	if currentTier == "vip" {
		reply := `👑 <b>У вас уже максимальный тариф VIP!</b>

Вам доступны все материалы, индивидуальные Zoom-интервью 1-на-1 и персональное сопровождение так называемого Иля (@ilyan_vas).`
		btnCurator := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
		btnMenu := tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")
		kb := tu.InlineKeyboard(tu.InlineKeyboardRow(btnCurator), tu.InlineKeyboardRow(btnMenu))
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		_, _ = s.bot.SendMessage(m)
		return
	}

	pricing := s.getPricing()

	if currentTier == "basic" {
		diffAcc := pricing.AcceleratorPrice - pricing.BasicPrice
		if diffAcc <= 0 {
			diffAcc = 2910.0
		}
		diffVIP := pricing.VIPPrice - pricing.BasicPrice
		if diffVIP <= 0 {
			diffVIP = 10910.0
		}

		reply := fmt.Sprintf(`🚀 <b>Апгрейд тарифа «Базовый»</b>

Ваш текущий тариф: <b>Базовый</b> (%s ₽).
Вы можете перейти на тариф с менторством или VIP, доплатив только разницу:

• <b>Акселератор:</b> доплата <b>%s ₽</b> (вместо %s ₽)
  <i>+ Личный аудит всех эссе и резюме</i>
  <i>+ Созвон 45–60 мин: симуляция 20-мин интервью 1-на-1 с разбором</i>
  <i>+ Визовый гайд J-1 и закрытый чат</i>

• <b>VIP:</b> доплата <b>%s ₽</b> (вместо %s ₽)
  <i>+ Всё из Акселератора + 3 мок-интервью</i>
  <i>+ Приоритетный личный чат до вылета в США</i>

Выберите вариант апгрейда:`,
			formatPriceRub(int(pricing.BasicPrice)),
			formatPriceRub(int(diffAcc)),
			formatPriceRub(int(pricing.AcceleratorPrice)),
			formatPriceRub(int(diffVIP)),
			formatPriceRub(int(pricing.VIPPrice)),
		)
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(UpgradeOptionsKeyboard("basic", diffAcc, diffVIP))
		_, _ = s.bot.SendMessage(m)
		return
	}

	// Default: currentTier == accelerator
	diff := pricing.VIPPrice - pricing.AcceleratorPrice
	if diff <= 0 {
		diff = 8000.0
	}

	reply := fmt.Sprintf(`👑 <b>Апгрейд тарифа «Акселератор» до VIP</b>

Ваш текущий тариф: <b>Акселератор</b> (%s ₽).
Вы можете перейти на максимальный уровень <b>VIP</b>, доплатив только разницу:

• <b>VIP:</b> доплата <b>%s ₽</b> (вместо %s ₽)
  <i>+ Персональное менторство так называемого Иля 24/7</i>
  <i>+ Неограниченный аудит всех эссе и документов</i>
  <i>+ Личное сопровождение визового этапа (J-1)</i>

Нажмите кнопку ниже, чтобы перейти на VIP:`,
		formatPriceRub(int(pricing.AcceleratorPrice)),
		formatPriceRub(int(diff)),
		formatPriceRub(int(pricing.VIPPrice)),
	)
	m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(UpgradeOptionsKeyboard("accelerator", diff))
	_, _ = s.bot.SendMessage(m)
}

func (s *BotService) handleUpgradePayment(chatID telego.ChatID, from *telego.User, targetTier string, amountDiff float64) {
	session := s.getSession(chatID.ID)
	session.State = StateWaitingReceipt
	session.IsUpgrade = true
	session.UpgradeToTier = targetTier
	session.SelectedTier = targetTier
	session.UpgradeAmount = amountDiff

	if from != nil && s.db != nil {
		var u models.User
		if err := s.db.Where("telegram_id = ?", from.ID).First(&u).Error; err == nil {
			session.UpgradeFromTier = u.Tier
		}
	}

	if s.bot == nil {
		return
	}

	targetName := "Акселератор"
	if targetTier == "vip" {
		targetName = "VIP"
	}

	pricing := s.getPricing()
	bankName := pricing.BankName
	if strings.TrimSpace(bankName) == "" {
		bankName = "БСПБ (Банк «Санкт-Петербург»)"
	}

	s.recordOrderIntent(from, targetTier, amountDiff, session.PromoCode, "telegram_sbp")

	text := fmt.Sprintf(`🚀 <b>Оплата апгрейда до тарифа «%s»</b>

💰 <b>Сумма доплаты: %s ₽</b>

🏛 <b>Оплата через СБП (0%% без комиссии):</b>
• Банк: <b>%s</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение платежа: <code>SPARK Апгрейд %s</code>

📎 <b>Как завершить апгрейд:</b>
1. Переведите точную сумму доплаты (<b>%s ₽</b>) через СБП.
2. <b>Отправьте скриншот чека прямо сюда в чат бота!</b>
3. Так называемый Иль подтвердит оплату и обновит ваш тариф.`, targetName, formatPriceRub(int(amountDiff)), bankName, strings.ToUpper(targetTier), formatPriceRub(int(amountDiff)))

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Написать так называемому Илю").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)

	m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	_, _ = s.bot.SendMessage(m)
}
