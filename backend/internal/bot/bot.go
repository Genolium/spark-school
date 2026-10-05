package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
)

type queuedNotification struct {
	telegramID int64
	message    string
	respChan   chan error
}

type BotService struct {
	bot              *telego.Bot
	db               *gorm.DB
	cfg              *config.Config
	affiliateService *services.AffiliateService
	sessions         *SessionManager
	mediaCache       *MediaCache
	notifyQueue      chan queuedNotification
	quitChan         chan struct{}
}

func NewBotService(cfg *config.Config, db *gorm.DB) *BotService {
	affService := services.NewAffiliateService(db)
	s := &BotService{
		db:               db,
		cfg:              cfg,
		affiliateService: affService,
		sessions:         NewSessionManager(30 * time.Minute),
		mediaCache:       NewMediaCache("media"),
		notifyQueue:      make(chan queuedNotification, 500),
		quitChan:         make(chan struct{}),
	}

	if cfg.TelegramBotToken != "" {
		var botOpts []telego.BotOption
		botOpts = append(botOpts, telego.WithDefaultLogger(false, true))
		if cfg.TelegramAPIServer != "" {
			botOpts = append(botOpts, telego.WithAPIServer(cfg.TelegramAPIServer))
			log.Printf("Telegram Bot initialized with custom API server: %s", cfg.TelegramAPIServer)
		}

		b, err := telego.NewBot(cfg.TelegramBotToken, botOpts...)
		if err != nil {
			log.Printf("Failed to initialize Telego bot: %v. Running in disabled mode.", err)
		} else {
			s.bot = b
		}
	} else {
		log.Println("TELEGRAM_BOT_TOKEN not provided. Bot service is running in mock/dry-run mode.")
	}

	// Start rate-limited outbound message worker
	go s.startQueueWorker()

	return s
}

// getSession returns user session, initializing session manager if nil.
func (s *BotService) getSession(chatID int64) *UserSession {
	if s.sessions == nil {
		s.sessions = NewSessionManager(30 * time.Minute)
	}
	return s.sessions.Get(chatID)
}

// startQueueWorker drains outbound queue with 35ms rate limiting
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
				chatID := tu.ID(item.telegramID)
				m := tu.Message(chatID, item.message).WithParseMode(telego.ModeHTML)
				_, err = s.bot.SendMessage(m)
			} else if item.telegramID != 0 {
				log.Printf("[Bot Notification Mock] to %d: %s", item.telegramID, item.message)
			}

			if item.respChan != nil {
				item.respChan <- err
			}
		}
	}
}

// Stop gracefully stops the outbound queue worker.
func (s *BotService) Stop() {
	select {
	case <-s.quitChan:
	default:
		close(s.quitChan)
	}
}

// ProcessUpdate handles incoming updates directly (from Webhook or Long Polling).
func (s *BotService) ProcessUpdate(update *telego.Update) {
	if update == nil {
		return
	}

	if update.CallbackQuery != nil {
		s.HandleCallbackQuery(update.CallbackQuery)
	} else if update.Message != nil {
		s.handleMessage(update.Message)
	}
}

// Start launches the bot polling worker if Long Polling is configured.
func (s *BotService) Start() {
	if s.bot == nil {
		return
	}

	log.Println("Telegram Bot worker started listening for updates...")
	updates, err := s.bot.UpdatesViaLongPolling(nil)
	if err != nil {
		log.Printf("Failed to get Telegram updates channel: %v", err)
		return
	}

	for update := range updates {
		s.ProcessUpdate(&update)
	}
}

func (s *BotService) handleMessage(msg *telego.Message) bool {
	if msg == nil {
		return false
	}

	// Private Chat Filter (R5): strictly ignore group, supergroup, channel messages
	if msg.Chat.Type != "private" || msg.Chat.ID <= 0 {
		return false
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
		return true
	}

	if strings.HasPrefix(text, "/withdraw") {
		s.handleWithdraw(chatID, msg.From)
		return true
	}

	if strings.HasPrefix(text, "/start") {
		session.State = StateDefault
		parts := strings.Fields(text)
		param := ""
		if len(parts) > 1 {
			param = parts[1]
		}

		if strings.HasPrefix(param, "pay") {
			s.sendPaymentDetails(chatID, msg.From, false)
			return true
		}

		if strings.HasPrefix(param, "withdraw") {
			s.handleWithdraw(chatID, msg.From)
			return true
		}

		if strings.HasPrefix(param, "ref_") {
			refCode := strings.TrimPrefix(param, "ref_")
			if msg.From != nil {
				s.associateBotReferral(msg.From, msg.From.Username, refCode)
			}
		}

		s.sendWelcomeMenu(chatID, msg.From)
		return true
	}

	// Curator Broadcast input state
	if session.State == StateAdminBroadcast && !strings.HasPrefix(text, "/") {
		session.State = StateDefault
		s.ExecuteBroadcast(chatID, text)
		return true
	}

	// Curator In-line Command: /admin
	if strings.HasPrefix(text, "/admin") {
		if msg.From != nil && s.IsAdmin(msg.From.ID) {
			s.SendAdminDashboard(chatID)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return true
	}

	// Curator In-line Command: /promo create [CODE] [DISCOUNT%]
	if strings.HasPrefix(text, "/promo") {
		if msg.From != nil && s.IsAdmin(msg.From.ID) {
			s.HandleAdminPromoCommand(chatID, msg.From, text)
		} else {
			m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
			_, _ = s.bot.SendMessage(m)
		}
		return true
	}

	// Diagnostic tool for Telegram Premium Custom Emojis
	if strings.HasPrefix(text, "/emoji") {
		var detectedIDs []string
		for _, entity := range msg.Entities {
			if entity.Type == telego.EntityTypeCustomEmoji && entity.CustomEmojiID != "" {
				detectedIDs = append(detectedIDs, entity.CustomEmojiID)
			}
		}
		if len(detectedIDs) > 0 {
			reply := "🎉 <b>Обнаружены Custom Emoji ID:</b>\n\n"
			for i, id := range detectedIDs {
				reply += fmt.Sprintf("%d. <code>%s</code>\n", i+1, id)
			}
			reply += "\n<i>Скопируйте их сюда в чат, и мы внедрим их в оформление бота!</i>"
			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
			_, _ = s.bot.SendMessage(m)
		} else {
			reply := `ℹ️ <b>Как узнать Custom Emoji ID:</b>

Отправьте команду <code>/emoji</code> вместе с эмодзи из вашего пака прямо в одном сообщении, например:
<code>/emoji </code> [вставьте эмодзи из пака @emojiabc]

Бот моментально считает их Telegram ID и пришлёт вам!`
			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
			_, _ = s.bot.SendMessage(m)
		}
		return true
	}

	if strings.HasPrefix(text, "/status") {
		s.sendStatusMessage(chatID, msg.From)
		return true
	}

	// If ordinary text sent, check if it matches a promo code in DB
	if text != "" && !strings.HasPrefix(text, "/") {
		s.processPromoInput(chatID, msg.From, text)
		return true
	}

	return true
}

func (s *BotService) sendWelcomeMenu(chatID telego.ChatID, from *telego.User) {
	if s.bot == nil {
		return
	}

	firstName := "Студент"
	if from != nil && from.FirstName != "" {
		firstName = from.FirstName
	}

	caption := fmt.Sprintf(`🎓 <b>Проект «так называемый SPARK»</b>

Здравствуйте, <b>%s</b>!

Комплексная программа подготовки к прохождению всероссийского конкурсного отбора на грантовую стажировку в США ($20,000).

В рамках проекта участники получают проверенные шаблоны резюме (Google XYZ), методологию написания эссе, банк вопросов прошлых лет к Zoom-интервью и сопровождение визового этапа (J-1).

👇 <b>Выберите интересующий раздел:</b>`, firstName)

	// Send rich photo card using cached or local file
	photoFile := s.ResolvePhotoFile(MediaMainBanner)
	photoMsg := tu.Photo(chatID, photoFile).
		WithCaption(caption).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL))

	sentMsg, err := s.bot.SendPhoto(photoMsg)
	if err == nil && sentMsg != nil {
		s.CacheSentPhoto(MediaMainBanner, sentMsg)
	} else if err != nil {
		// Fallback to text message if photo failed
		textMsg := tu.Message(chatID, caption).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL))
		_, _ = s.bot.SendMessage(textMsg)
	}
}

func (s *BotService) sendPaymentDetails(chatID telego.ChatID, from *telego.User, withPromo bool) {
	if s.bot == nil {
		return
	}

	session := s.getSession(chatID.ID)
	session.State = StateWaitingReceipt

	var text string
	var keyboard *telego.InlineKeyboardMarkup

	if withPromo || session.DiscountPct > 0 {
		text = `🎉 <b>Промокод успешно активирован! Скидка 5%</b>

<b>Стоимость с учётом скидки 5%:</b>
• Акселератор: <b>6 555 ₽</b> (скидка 345 ₽)
• VIP: <b>14 155 ₽</b> (скидка 745 ₽)

📍 <b>Реквизиты для оплаты со скидкой (СБП 0%):</b>
• Банк: <b>Т-Банк (Тинькофф)</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение платежа: <code>SPARK ПРОМО [Тариф]</code>

📌 <b>После перевода прикрепи скриншот чека прямо в этот чат!</b>
Бот передаст его так называемому Илю и сразу выдаст персональную ссылку в закрытый канал потока.`
		keyboard = PromoAppliedKeyboard()
	} else {
		text = `💳 <b>Бронирование места в проекте «так называемый SPARK»</b>

<b>Тарифы участия:</b>
1️⃣ <b>Акселератор [Хит]:</b> 6 900 ₽ (по промокоду: <b>6 555 ₽</b>)
2️⃣ <b>VIP:</b> 14 900 ₽ (по промокоду: <b>14 155 ₽</b>) <i>[Строго 3 места]</i>

📍 <b>Реквизиты для оплаты (СБП 0%):</b>
• Банк: <b>Т-Банк (Тинькофф)</b>
• Телефон: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение: <code>SPARK [Тариф]</code>

📌 <b>После перевода просто отправь скриншот чека в этот чат!</b>
Бот передаст его так называемому Илю и сразу пришлёт персональную ссылку в закрытый канал потока.`
		keyboard = PaymentKeyboard()
	}

	photoFile := s.ResolvePhotoFile(MediaBuyCourse)
	photoMsg := tu.Photo(chatID, photoFile).
		WithCaption(text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(keyboard)

	sentMsg, err := s.bot.SendPhoto(photoMsg)
	if err == nil && sentMsg != nil {
		s.CacheSentPhoto(MediaBuyCourse, sentMsg)
	} else if err != nil {
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		_, _ = s.bot.SendMessage(m)
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

	// Also support default promo START5, IL5, SPARK5
	isValid := err == nil || code == "START5" || code == "IL5" || code == "SPARK5" || code == "ILYA5"

	if isValid {
		session.PromoCode = code
		session.DiscountPct = 5
		session.State = StateWaitingReceipt
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
		reply := fmt.Sprintf(`✅ <b>Ваш доступ активен!</b>

Добро пожаловать в проект <b>«так называемый SPARK»</b>.
Ваша ссылка в закрытый канал потока:
%s`, inviteLink)
		btnChannel := tu.InlineKeyboardButton("Открыть закрытый канал 🚀").WithURL(inviteLink)
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnChannel)))
		_, _ = s.bot.SendMessage(m)
		return
	}

	reply := `ℹ️ <b>Статус доступа: Не активен</b>

У вас пока нет активного доступа к проекту <b>«так называемый SPARK»</b>.
Тарифы: <b>Акселератор</b> (6 900 ₽), <b>VIP</b> (14 900 ₽).

После подтверждения оплаты так называемый Иль предоставит персональную ссылку в закрытый канал.`
	btnTariff := tu.InlineKeyboardButton("Оформить тариф ➔").WithCallbackData("action_payment")
	btnCurator := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
	m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnTariff), tu.InlineKeyboardRow(btnCurator)))
	_, _ = s.bot.SendMessage(m)
}

// HandleCallbackQuery handles all button taps.
func (s *BotService) HandleCallbackQuery(query *telego.CallbackQuery) bool {
	if query == nil {
		return false
	}

	data := query.Data

	// Admin Receipt Approvals - Security check for admin rights
	if strings.HasPrefix(data, "receipt_approve_") {
		idStr := strings.TrimPrefix(data, "receipt_approve_")
		s.processReceiptApproval(query, idStr)
		return true
	}
	if strings.HasPrefix(data, "receipt_reject_") {
		idStr := strings.TrimPrefix(data, "receipt_reject_")
		s.processReceiptRejection(query, idStr)
		return true
	}

	if query.Message == nil {
		return false
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

	switch data {
	case "action_menu":
		session.State = StateDefault
		s.sendWelcomeMenu(chatID, &query.From)
		return true

	case "action_payment":
		session.State = StateWaitingReceipt
		s.sendPaymentDetails(chatID, &query.From, false)
		return true

	case "promo_enter":
		session.State = StateWaitingPromo
		text := `🎟 <b>Активация промокода на скидку 5%</b>

Напиши промокод прямо в ответном сообщении!

<i>Например: <code>START5</code> или промокод твоего друга-партнёра.</i>

Скидка 5% действует на любой тариф:
• Акселератор: 6 900 ₽ ➔ <b>6 555 ₽</b>
• VIP: 14 900 ₽ ➔ <b>14 155 ₽</b>`
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(PromoRequestKeyboard())
		_, _ = s.bot.SendMessage(m)
		return true

	case "info_program":
		text := `🚀 <b>Программа сопровождения «так называемый SPARK»:</b>

• <b>Закрытый канал потока</b> со всеми материалами и апдейтами
• <b>Шаблоны:</b> Google XYZ резюме, эссе на 500 слов
• <b>X-Factor Video framework:</b> режиссура и хуки первых 5 секунд
• <b>Банк вопросов</b> к Zoom-интервью с комиссией
• <b>Языковая подготовка</b> к собеседованию
• <b>Визовый этап:</b> DS-160, запись в дипмиссию, привязка к родине
• <b>Персональный аудит</b> и 45-мин Zoom-интервью с так называемым Илем

<b>Тарифные планы:</b>
1️⃣ <b>Акселератор [Хит]:</b> 6 900 ₽ (6 555 ₽ по промокоду)
2️⃣ <b>VIP:</b> 14 900 ₽ (14 155 ₽ по промокоду)`

		btnPay := tu.InlineKeyboardButton("💳 Купить курс").WithCallbackData("action_payment")
		btnPromo := tu.InlineKeyboardButton("🎟 Применить промокод -5%").WithCallbackData("promo_enter")
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
		return true

	case "admin_menu":
		if s.IsAdmin(query.From.ID) {
			s.SendAdminDashboard(chatID)
			return true
		}

	case "admin_stats":
		if s.IsAdmin(query.From.ID) {
			s.SendAdminLiveStats(chatID)
			return true
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
			return true
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
			return true
		}
	}

	return false
}

func (s *BotService) processReceiptApproval(query *telego.CallbackQuery, idStr string) {
	adminID := query.From.ID
	if s.cfg != nil && s.cfg.AdminTelegramID != 0 && adminID != s.cfg.AdminTelegramID {
		if s.bot != nil {
			_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "⛔️ У вас нет прав администратора.",
				ShowAlert:       true,
			})
		}
		return
	}

	if s.db == nil {
		return
	}

	var receipt models.PaymentReceipt
	if err := s.db.First(&receipt, "id = ?", idStr).Error; err != nil {
		_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Чек не найден в базе данных.",
			ShowAlert:       true,
		})
		return
	}

	if receipt.Status == "approved" {
		_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "Этот чек уже был одобрен ранее.",
			ShowAlert:       true,
		})
		return
	}

	inviteLink := s.cfg.TelegramInviteLink
	if s.cfg.TelegramChannelID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if link, err := s.CreateOneTimeInviteLink(ctx, s.cfg.TelegramChannelID); err == nil && link != "" {
			inviteLink = link
		}
	}

	receipt.Status = "approved"
	s.db.Save(&receipt)

	var student models.User
	if err := s.db.Where("telegram_id = ?", receipt.TelegramID).First(&student).Error; err == nil {
		student.HasAccess = true
		now := time.Now()
		student.AccessGrantedAt = &now
		s.db.Save(&student)
		_ = s.affiliateService.ProcessCoursePurchase(student.ID, receipt.Amount)
	}

	studentMsg := fmt.Sprintf(`🎉 <b>Поздравляем! Оплата подтверждена.</b>

Добро пожаловать в проект <b>«так называемый SPARK»</b>!
Твоя ссылка для входа в закрытый канал потока:
%s

<i>Ссылка является персональной и одноразовой.</i>`, inviteLink)

	btnChannel := tu.InlineKeyboardButton("Войти в закрытый канал ➔").WithURL(inviteLink)
	m := tu.Message(tu.ID(receipt.TelegramID), studentMsg).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnChannel)))
	_, _ = s.bot.SendMessage(m)

	_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "✅ Доступ успешно выдан!",
	})
}

func (s *BotService) processReceiptRejection(query *telego.CallbackQuery, idStr string) {
	adminID := query.From.ID
	if s.cfg != nil && s.cfg.AdminTelegramID != 0 && adminID != s.cfg.AdminTelegramID {
		if s.bot != nil {
			_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "⛔️ У вас нет прав администратора.",
				ShowAlert:       true,
			})
		}
		return
	}

	if s.db == nil {
		return
	}

	var receipt models.PaymentReceipt
	if err := s.db.First(&receipt, "id = ?", idStr).Error; err != nil {
		return
	}

	receipt.Status = "rejected"
	s.db.Save(&receipt)

	rejectMsg := `⚠️ <b>К сожалению, ваш чек не был подтверждён.</b>

Возможные причины:
• Неверная сумма перевода
• Нечитаемый скриншот или отсутствие фискального подтверждения банка

Пожалуйста, свяжитесь с куратором: @ilyan_vas`

	btnHelp := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
	m := tu.Message(tu.ID(receipt.TelegramID), rejectMsg).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnHelp)))
	_, _ = s.bot.SendMessage(m)

	_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "❌ Чек отклонён",
	})
}

// handleWithdraw handles partner balance withdrawals
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

// SendNotification queues an outbound direct message through rate limiter
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
			case <-time.After(3 * time.Second):
				return nil
			}
		default:
			log.Printf("[Bot Rate Limiter] Queue full, sending directly to %d", telegramID)
		}
	}

	if s.bot == nil {
		log.Printf("[Bot Notification Mock] to %d: %s", telegramID, message)
		return nil
	}

	chatID := tu.ID(telegramID)
	m := tu.Message(chatID, message).WithParseMode(telego.ModeHTML)
	_, err := s.bot.SendMessage(m)
	return err
}

// CreateOneTimeInviteLink generates a single-use invite link with a 24-hour expiration.
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

func (s *BotService) associateBotReferral(from *telego.User, inputUsername, refCode string) {
	if s.db == nil || from == nil {
		return
	}

	tgID := from.ID
	uname := from.Username
	if uname == "" {
		uname = inputUsername
	}
	uname = strings.TrimPrefix(uname, "@")

	var user models.User
	err := s.db.Where("telegram_id = ?", tgID).First(&user).Error
	if err != nil {
		firstName := from.FirstName
		if firstName == "" {
			firstName = "Student"
		}
		role := "student"
		if s.cfg.AdminTelegramID != 0 && tgID == s.cfg.AdminTelegramID {
			role = "admin"
		}

		user = models.User{
			TelegramID: tgID,
			Username:   uname,
			FirstName:  firstName,
			LastName:   from.LastName,
			Role:       role,
			CreatedAt:  time.Now(),
		}
		if err := s.db.Create(&user).Error; err != nil {
			return
		}

		cleanUname := strings.ToLower(uname)
		if cleanUname == "" {
			cleanUname = fmt.Sprintf("id%d", tgID)
		}
		affProfile := models.AffiliateProfile{
			UserID:       user.ID,
			ReferralCode: cleanUname,
			CreatedAt:    time.Now(),
		}
		_ = s.db.Create(&affProfile).Error
	}

	if refCode != "" {
		_ = s.affiliateService.TrackReferral(user.ID, refCode)
	}
}
