package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

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
	yooKassaService  *services.YooKassaService
	sessions         *SessionManager
	mediaCache       *MediaCache
	notifyQueue      chan queuedNotification
	quitChan         chan struct{}
}

func NewBotService(cfg *config.Config, db *gorm.DB) *BotService {
	affService := services.NewAffiliateService(db)
	ykService := services.NewYooKassaService(cfg)
	s := &BotService{
		db:               db,
		cfg:              cfg,
		affiliateService: affService,
		yooKassaService:  ykService,
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

func extractTelegramLink(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "https://t.me/") || strings.HasPrefix(trimmed, "http://t.me/") {
			return trimmed
		}
	}
	return ""
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

	log.Printf("[Bot Inbound] chat_id=%d, type=%s, from=%s, text=%q", msg.Chat.ID, msg.Chat.Type, msg.From.FirstName, msg.Text)

	// Private Chat Filter (R5): strictly ignore group, supergroup, channel messages
	if msg.Chat.Type != "private" || msg.Chat.ID <= 0 {
		log.Printf("[Bot Inbound Ignored] chat_type=%s, chat_id=%d", msg.Chat.Type, msg.Chat.ID)
		return false
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
			payload := strings.TrimPrefix(param, "pay_")
			if param == "pay" {
				payload = ""
			}

			// 1. Extract tier
			tier := "accelerator"
			if strings.HasPrefix(payload, "basic") || strings.HasPrefix(payload, "base") {
				tier = "basic"
				payload = strings.TrimPrefix(strings.TrimPrefix(payload, "basic"), "base")
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

			// 2. Extract referral or promo code
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

			if u := s.syncTelegramUser(msg.From); u != nil {
				if tier != "" {
					u.Tier = tier
				}
				if withPromo && session.PromoCode != "" {
					u.PromoCode = session.PromoCode
				}
				s.db.Save(u)
			}

			s.sendPaymentDetails(chatID, msg.From, withPromo)
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
			log.Printf("[Bot /emoji] No custom emoji entities found in msg. Text=%q, EntitiesCount=%d", msg.Text, len(msg.Entities))
			reply := `ℹ️ <b>Как узнать Custom Emoji ID:</b>

Отправьте команду <code>/emoji</code> вместе с эмодзи из вашего пака прямо в одном сообщении, например:
<code>/emoji </code> [вставьте эмодзи из пака]

Бот моментально покажет символ, сам эмодзи и его ID!`
			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
			if _, err := s.bot.SendMessage(m); err != nil {
				log.Printf("[Bot Error] /emoji info SendMessage failed: %v", err)
			}
		}
		return true
	}

	if strings.HasPrefix(text, "/status") {
		s.sendStatusMessage(chatID, msg.From)
		return true
	}

	// State-dependent handling:
	// Only parse promo code if user actually requested to enter a promo code
	if session.State == StateWaitingPromo && text != "" && !strings.HasPrefix(text, "/") {
		s.processPromoInput(chatID, msg.From, text)
		return true
	}

	// Default fallback for any unexpected text: guide user back to main menu
	if text != "" && !strings.HasPrefix(text, "/") {
		helpMsg := tu.Message(chatID, `👋 Я вас понял! Чтобы выбрать действие, воспользуйтесь кнопками меню ниже или отправьте команду /start:`)
		helpMsg.WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL))
		_, _ = s.bot.SendMessage(helpMsg)
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

	star := TgEmoji(EmojiSparkleGreen, "✨")
	spark := TgEmoji(EmojiStarGreen, "⭐")
	spiral := TgEmoji(EmojiSpiralGreen, "🌀")
	heart := TgEmoji(EmojiHeartGreen, "💚")
	sun := TgEmoji(EmojiSunGreen, "☀️")
	arrowDown := TgEmoji(EmojiArrowDown, "⬇️")

	caption := fmt.Sprintf(`%s <b>Проект «так называемый SPARK»</b> %s

%s Здравствуйте, <b>%s</b>!

%s Наше обучение - комплексная программа подготовки к прохождению всероссийского конкурсного отбора на грантовую стажировку в США ($20,000).

%s В рамках проекта участники получают проверенные шаблоны резюме, методологию написания эссе, банк вопросов прошлых лет к Zoom-интервью и сопровождение визового этапа (J-1).

%s <b>Выберите интересующий раздел:</b>`, star, spark, spiral, firstName, heart, sun, arrowDown)

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
		log.Printf("[Bot Error] SendPhoto with local media failed: %v. Retrying with verified photo file ID...", err)
		// Fallback 1: Try sending with verified Telegram photo ID and KEEP custom emojis!
		fallbackPhoto := tu.Photo(chatID, tu.FileFromID(PhotoGoldenGate)).
			WithCaption(caption).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL))
		if _, fbErr := s.bot.SendPhoto(fallbackPhoto); fbErr != nil {
			log.Printf("[Bot Error] Fallback SendPhoto also failed: %v. Sending message...", fbErr)
			textMsg := tu.Message(chatID, caption).
				WithParseMode(telego.ModeHTML).
				WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL))
			if _, textErr := s.bot.SendMessage(textMsg); textErr != nil {
				log.Printf("[Bot Error] SendMessage with custom emoji failed: %v. Cleaning tags...", textErr)
				cleanCaption := StripTgEmoji(caption)
				cleanMsg := tu.Message(chatID, cleanCaption).
					WithParseMode(telego.ModeHTML).
					WithReplyMarkup(MainMenuKeyboard(s.cfg.FrontendURL))
				_, _ = s.bot.SendMessage(cleanMsg)
			}
		}
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

	sparkle := TgEmoji(EmojiStarGreen, "✨")
	checkV := TgEmoji(EmojiCheckVGreen, "✔️")
	clipGreen := TgEmoji(EmojiPaperclipGreen, "📎")
	bullet := TgEmoji(EmojiStarBigGreen, "⭐")

	lightning := TgEmoji(EmojiLightning, "⚡️")
	num1 := TgEmoji(EmojiNum1, "1️⃣")
	num2 := TgEmoji(EmojiNum2, "2️⃣")
	calendar := TgEmoji(EmojiCalendarTag, "🗓")
	paperclip := TgEmoji(EmojiPaperclip, "📎")

	if withPromo || session.DiscountPct > 0 {
		pct := session.DiscountPct
		if pct <= 0 {
			pct = 5
		}
		accDiscount := (6900 * pct) / 100
		accPrice := 6900 - accDiscount
		vipDiscount := (14900 * pct) / 100
		vipPrice := 14900 - vipDiscount

		text = fmt.Sprintf(`%s <b>Промокод успешно активирован! Скидка %d%%</b>

<b>Стоимость со скидкой %d%%:</b>
%s Акселератор: <b>%d ₽</b> (скидка %d ₽)
%s VIP: <b>%d ₽</b> (скидка %d ₽)

💳 <b>Способы оплаты:</b>
1️⃣ <b>Быстрая онлайн-оплата (Tribute / Telegram):</b>
• Оплата картами и через Telegram в 1 клик по кнопкам тарифа ниже

2️⃣ <b>ЮKassa (Официальный эквайринг РФ — СБП / Карты):</b>
• Безопасная оплата с чеком ФНС (54-ФЗ)
• Нажмите кнопку «Оплатить через ЮKassa со скидкой» ниже

%s <b>Или перевод по СБП (0%%):</b>
• Банк: <b>Т-Банк (Тинькофф)</b>
• Номер телефона: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение платежа: <code>SPARK ПРОМО [Тариф]</code>

%s <b>После онлайн-оплаты доступ в закрытый канал выдаётся автоматически!</b> При переводе по СБП прикрепи чек в этот чат.`, sparkle, pct, pct, bullet, accPrice, accDiscount, bullet, vipPrice, vipDiscount, checkV, clipGreen)
		keyboard = PromoAppliedKeyboard()
	} else {
		text = fmt.Sprintf(`%s <b>Бронирование места в проекте «так называемый SPARK»</b>

<b>Тарифы участия:</b>
%s <b>Акселератор [Хит]:</b> 6 900 ₽ (по промокоду: <b>6 555 ₽</b>)
%s <b>VIP:</b> 14 900 ₽ (по промокоду: <b>14 155 ₽</b>) <i>[Строго 3 места]</i>

💳 <b>Способы оплаты:</b>
1️⃣ <b>Быстрая онлайн-оплата (Tribute / Telegram):</b>
• Оплата картами и через Telegram в 1 клик по кнопкам тарифа ниже

2️⃣ <b>ЮKassa (Официальный эквайринг РФ — СБП / Карты):</b>
• Безопасная оплата с чеком ФНС (54-ФЗ)
• Нажмите кнопку «Оплатить через ЮKassa» ниже

%s <b>Или прямой перевод по реквизитам СБП (0%%):</b>
• Банк: <b>Т-Банк (Тинькофф)</b>
• Телефон: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Назначение: <code>SPARK [Тариф]</code>

%s <b>После онлайн-оплаты доступ выдаётся сразу</b>, а при переводе по СБП просто отправь скриншот чека в этот чат!`, lightning, num1, num2, calendar, paperclip)
		keyboard = PaymentKeyboard()
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

func (s *BotService) handleYooKassaPayment(chatID telego.ChatID, from *telego.User, tier string) {
	if s.bot == nil {
		return
	}

	session := s.getSession(chatID.ID)
	hasPromo := session.DiscountPct > 0 || session.PromoCode != ""

	tierTitle := "Акселератор"
	basePrice := 6900.0
	if tier == "vip" {
		tierTitle = "VIP"
		basePrice = 14900.0
	}

	price := basePrice
	if hasPromo {
		pct := session.DiscountPct
		if pct <= 0 {
			pct = 5
		}
		price = basePrice - (basePrice*float64(pct))/100.0
	}

	// If YooKassaSecretKey is configured in .env, create live checkout session via API
	if s.cfg != nil && s.cfg.YooKassaSecretKey != "" && s.yooKassaService != nil {
		username := ""
		if from != nil {
			username = from.Username
		}
		botUsername := s.cfg.TelegramBotUsername
		if botUsername == "" {
			botUsername = "spark_prep_bot"
		}

		ykReq := services.YooKassaPaymentRequest{
			Amount: services.YooKassaAmount{
				Value:    fmt.Sprintf("%.2f", price),
				Currency: "RUB",
			},
			Capture: true,
			Confirmation: services.YooKassaConfirmation{
				Type:      "redirect",
				ReturnURL: fmt.Sprintf("https://t.me/%s", botUsername),
			},
			Description: fmt.Sprintf("Обучение в проекте «так называемый SPARK» — Тариф %s (@%s)", tierTitle, username),
			Metadata: map[string]string{
				"tier":        tier,
				"telegram_id": strconv.FormatInt(chatID.ID, 10),
				"username":    username,
				"promo_code":  session.PromoCode,
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := s.yooKassaService.CreatePayment(ctx, ykReq)
		if err == nil && resp != nil && resp.Confirmation.ConfirmationURL != "" {
			successText := fmt.Sprintf(`✅ <b>Счёт в ЮKassa успешно сформирован!</b>

📦 Тариф: <b>%s</b>
💰 Сумма к оплате: <b>%d ₽</b>
🧾 Фискальный чек ФНС РФ: Самозанятый Васюнин Илья Олегович (ИНН 780739313219)
🔒 Оплата через защищённый шлюз ЮKassa (СБП, Карты РФ, SberPay)

<i>Нажмите кнопку ниже для безопасной оплаты. Доступ в закрытый канал потока будет выдан автоматически сразу после подтверждения!</i>`, tierTitle, int(price))

			checkoutKb := tu.InlineKeyboard(
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton(fmt.Sprintf("💳 Оплатить в ЮKassa (%d ₽)", int(price))).WithURL(resp.Confirmation.ConfirmationURL),
				),
				tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("⬅️ Назад ко всем способам").WithCallbackData("action_payment"),
				),
			)

			m := tu.Message(chatID, successText).WithParseMode(telego.ModeHTML).WithReplyMarkup(checkoutKb)
			if _, sendErr := s.bot.SendMessage(m); sendErr != nil {
				cleanMsg := tu.Message(chatID, StripTgEmoji(successText)).WithParseMode(telego.ModeHTML).WithReplyMarkup(checkoutKb)
				_, _ = s.bot.SendMessage(cleanMsg)
			}
			return
		}
		log.Printf("[Bot YooKassa] API CreatePayment failed: %v", err)
	}

	// Fallback when secret key is not set yet in .env or API error
	fallbackText := fmt.Sprintf(`🇷🇺 <b>Официальная оплата через ЮKassa: Тариф %s (%d ₽)</b>

Официальный магазин ЮKassa (ShopID: <code>1491540</code>) зарегистрирован.
Вы можете оплатить прямо сейчас без комиссии:

⚡️ <b>1. Онлайн в 1 клик через Telegram (Tribute):</b>
Карты РФ и зарубежные, СБП — моментальная выдача доступа!

🏛 <b>2. Прямой перевод по СБП (0%%):</b>
• Банк: <b>Т-Банк (Тинькофф)</b>
• Телефон: <code>+7 981 163-36-91</code>
• Получатель: <b>Васюнин Илья Олегович</b>
• Сумма: <b>%d ₽</b>

Либо напишите @ilyan_vas — куратор сформирует прямой счёт ЮKassa из личного кабинета!`, tierTitle, int(price), int(price))

	fallbackKb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⚡️ Оплатить через Tribute").WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Написать так называемому Илю").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ Назад к тарифам").WithCallbackData("action_payment"),
		),
	)

	m := tu.Message(chatID, fallbackText).WithParseMode(telego.ModeHTML).WithReplyMarkup(fallbackKb)
	if _, sendErr := s.bot.SendMessage(m); sendErr != nil {
		cleanMsg := tu.Message(chatID, StripTgEmoji(fallbackText)).WithParseMode(telego.ModeHTML).WithReplyMarkup(fallbackKb)
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
		reply := fmt.Sprintf(`%s <b>Ваш доступ активен!</b>

%s Добро пожаловать в проект <b>«так называемый SPARK»</b>.
Ваша ссылка в закрытый канал потока:
%s`, heart, sun, inviteLink)
		btnChannel := tu.InlineKeyboardButton("Открыть закрытый канал 🚀").WithURL(inviteLink)
		kb := tu.InlineKeyboard(tu.InlineKeyboardRow(btnChannel))

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

	// Always sync every interacting user as a lead in Postgres CRM
	s.syncTelegramUser(&query.From)

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

	case "pay_yk_menu":
		hasPromo := session.DiscountPct > 0 || session.PromoCode != ""
		ykText := `🇷🇺 <b>Официальная оплата через ЮKassa (ShopID: 1491540)</b>

Платёжный шлюз с официальным чеком ФНС (54-ФЗ):
• СБП (Система быстрых платежей 0%)
• Любые банковские карты РФ (МИР, Visa, Mastercard)
• SberPay / T-Pay / Mir Pay

<b>Выберите тариф для выставления счёта:</b>`
		kb := YooKassaSelectTierKeyboard(hasPromo)
		m := tu.Message(chatID, ykText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		if _, err := s.bot.SendMessage(m); err != nil {
			cleanMsg := tu.Message(chatID, StripTgEmoji(ykText)).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
			_, _ = s.bot.SendMessage(cleanMsg)
		}
		return true

	case "pay_yk_accelerator":
		s.handleYooKassaPayment(chatID, &query.From, "accelerator")
		return true

	case "pay_yk_vip":
		s.handleYooKassaPayment(chatID, &query.From, "vip")
		return true

	case "promo_enter":
		session.State = StateWaitingPromo
		checkCircle := TgEmoji(EmojiCheckCircleGreen, "✅")
		sparkle := TgEmoji(EmojiSparkleGreen, "✨")

		text := fmt.Sprintf(`%s <b>Активация промокода на скидку 5%%</b> %s

Напиши промокод прямо в ответном сообщении!

<i>Укажи промокод твоего друга-партнёра или из официального блога.</i>

Скидка 5%% действует на любой тариф:
• Акселератор: 6 900 ₽ ➔ <b>6 555 ₽</b>
• VIP: 14 900 ₽ ➔ <b>14 155 ₽</b>`, checkCircle, sparkle)
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(PromoRequestKeyboard())
		if _, err := s.bot.SendMessage(m); err != nil {
			log.Printf("[Bot Error] promo_enter SendMessage failed: %v. Retrying stripped...", err)
			cleanText := StripTgEmoji(text)
			cleanMsg := tu.Message(chatID, cleanText).WithParseMode(telego.ModeHTML).WithReplyMarkup(PromoRequestKeyboard())
			_, _ = s.bot.SendMessage(cleanMsg)
		}
		return true

	case "info_program":
		star := TgEmoji(EmojiSparkleGreen, "✨")
		spark := TgEmoji(EmojiStarGreen, "⭐")
		num1 := TgEmoji(EmojiNum1, "1️⃣")
		num2 := TgEmoji(EmojiNum2, "2️⃣")

		text := fmt.Sprintf(`%s <b>Программа сопровождения «так называемый SPARK»</b> %s

• <b>Закрытый канал потока</b> со всеми материалами и апдейтами
• <b>Подробный курс по каждому элементу заявки</b>
• <b>X-Factor Video framework:</b> режиссура и хуки первых 5 секунд
• <b>Банк вопросов</b> к Zoom-интервью с комиссией
• <b>Языковая подготовка</b> к собеседованию
• <b>Визовый этап:</b> DS-160, запись в дипмиссию, привязка к родине
• <b>Персональный аудит</b> и 45-мин Zoom-интервью с так называемым Илем

<b>Тарифные планы:</b>
%s <b>Акселератор [Хит]:</b> 6 900 ₽ (6 555 ₽ по промокоду)
%s <b>VIP:</b> 14 900 ₽ (14 155 ₽ по промокоду)`, star, spark, num1, num2)

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

	case "admin_receipts":
		if s.IsAdmin(query.From.ID) {
			s.SendAdminReceipts(chatID)
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

// sendAccessNotificationToStudent reliably delivers channel invite link to student using 6-stage fallback logic
func (s *BotService) sendAccessNotificationToStudent(studentTelegramID int64, studentMsg string, inviteLink string) error {
	if studentTelegramID == 0 {
		return fmt.Errorf("student Telegram ID is 0")
	}
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
	plainText := fmt.Sprintf("Поздравляем! Оплата подтверждена.\n\nДобро пожаловать в проект «так называемый SPARK»!\nТвоя ссылка для входа в закрытый канал потока:\n%s\n\nСсылка является персональной и одноразовой.", inviteLink)
	plainMsg := tu.Message(studentChatID, plainText)

	if _, err6 := s.bot.SendMessage(plainMsg); err6 == nil {
		log.Printf("[Bot Approval] Step 6 plain SendMessage succeeded for student %d", studentTelegramID)
		return nil
	} else {
		log.Printf("[Bot Approval] FATAL: All 6 send attempts failed for student %d: %v", studentTelegramID, err6)
		return err6
	}
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

	inviteLink := strings.TrimSpace(s.cfg.TelegramInviteLink)
	if s.cfg.TelegramChannelID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if link, err := s.CreateOneTimeInviteLink(ctx, s.cfg.TelegramChannelID); err == nil && link != "" {
			inviteLink = link
		} else if err != nil {
			log.Printf("[Bot Approval] Failed to create one-time invite link for channel %d: %v", s.cfg.TelegramChannelID, err)
		}
	}
	if inviteLink == "" {
		inviteLink = "https://t.me/+so_called_spark_private"
	}

	studentMsg := FormatPaymentSuccessMessage(inviteLink)

	// If receipt was ALREADY approved, re-send invite link to student without duplicating DB actions
	if receipt.Status == "approved" {
		log.Printf("[Bot Approval] Receipt #%d was already approved. Re-sending invite link to student %d...", receipt.ID, receipt.TelegramID)
		sendErr := s.sendAccessNotificationToStudent(receipt.TelegramID, studentMsg, inviteLink)
		if sendErr != nil {
			_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            fmt.Sprintf("❌ Ошибка отправки ссылки: %v", sendErr),
				ShowAlert:       true,
			})
		} else {
			_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
				CallbackQueryID: query.ID,
				Text:            "✅ Ссылка успешно отправлена студенту повторно!",
				ShowAlert:       true,
			})
		}
		return
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

	sendErr := s.sendAccessNotificationToStudent(receipt.TelegramID, studentMsg, inviteLink)
	if sendErr != nil {
		log.Printf("[Bot Approval] Failed delivering invite link to student %d: %v", receipt.TelegramID, sendErr)
		_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            fmt.Sprintf("⚠️ Чек одобрен, но ошибка отправки сообщения: %v", sendErr),
			ShowAlert:       true,
		})
	} else {
		_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
			CallbackQueryID: query.ID,
			Text:            "✅ Доступ успешно выдан, ссылка отправлена студенту!",
		})
	}

	// Update Curator's Inline Buttons
	if query.Message != nil {
		curatorChatID := tu.ID(query.Message.GetChat().ID)
		msgID := query.Message.GetMessageID()
		approvedKb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("🔄 Отправить ссылку повторно").WithCallbackData(fmt.Sprintf("receipt_approve_%d", receipt.ID)),
			),
		)
		if receipt.Username != "" {
			approvedKb.InlineKeyboard = append(approvedKb.InlineKeyboard, tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf("💬 Написать @%s", receipt.Username)).WithURL(fmt.Sprintf("https://t.me/%s", receipt.Username)),
			))
		}
		editMarkup := &telego.EditMessageReplyMarkupParams{
			ChatID:      curatorChatID,
			MessageID:   msgID,
			ReplyMarkup: approvedKb,
		}
		_, _ = s.bot.EditMessageReplyMarkup(editMarkup)
	}
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

Пожалуйста, свяжитесь с так называемым Илем: @ilyan_vas`

	btnHelp := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
	m := tu.Message(tu.ID(receipt.TelegramID), rejectMsg).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnHelp)))
	if _, err := s.bot.SendMessage(m); err != nil {
		log.Printf("[Bot Rejection] SendMessage failed for student %d: %v. Retrying clean...", receipt.TelegramID, err)
		cleanReject := StripTgEmoji(rejectMsg)
		cleanM := tu.Message(tu.ID(receipt.TelegramID), cleanReject).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnHelp)))
		if _, cleanErr := s.bot.SendMessage(cleanM); cleanErr != nil {
			plainM := tu.Message(tu.ID(receipt.TelegramID), cleanReject)
			_, _ = s.bot.SendMessage(plainM)
		}
	}

	_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            "❌ Чек отклонён",
	})

	if query.Message != nil {
		curatorChatID := tu.ID(query.Message.GetChat().ID)
		msgID := query.Message.GetMessageID()
		rejectedKb := tu.InlineKeyboard()
		if receipt.Username != "" {
			rejectedKb = tu.InlineKeyboard(tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf("💬 Написать @%s", receipt.Username)).WithURL(fmt.Sprintf("https://t.me/%s", receipt.Username)),
			))
		}
		editMarkup := &telego.EditMessageReplyMarkupParams{
			ChatID:      curatorChatID,
			MessageID:   msgID,
			ReplyMarkup: rejectedKb,
		}
		_, _ = s.bot.EditMessageReplyMarkup(editMarkup)
	}
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

// syncTelegramUser ensures every user interacting with the bot is persisted in Postgres,
// enabling full CRM visibility for leads and unpaid students in the web admin panel.
func (s *BotService) syncTelegramUser(from *telego.User) *models.User {
	if s.db == nil || from == nil || from.ID <= 0 {
		return nil
	}

	tgID := from.ID
	cleanUsername := strings.TrimPrefix(strings.TrimSpace(from.Username), "@")
	firstName := strings.TrimSpace(from.FirstName)
	if firstName == "" {
		firstName = "Студент"
	}
	lastName := strings.TrimSpace(from.LastName)

	var user models.User
	err := s.db.Where("telegram_id = ?", tgID).First(&user).Error
	if err != nil {
		role := "student"
		hasAccess := false
		if s.cfg != nil && s.cfg.AdminTelegramID != 0 && tgID == s.cfg.AdminTelegramID {
			role = "admin"
			hasAccess = true
		}

		user = models.User{
			TelegramID: tgID,
			Username:   cleanUsername,
			FirstName:  firstName,
			LastName:   lastName,
			Role:       role,
			HasAccess:  hasAccess,
			Tier:       "accelerator",
			PaidAmount: 0,
			CreatedAt:  time.Now(),
		}
		if err := s.db.Create(&user).Error; err != nil {
			log.Printf("[Bot User Sync] Failed creating user %d: %v", tgID, err)
			return nil
		}

		refCode := strings.ToLower(cleanUsername)
		if refCode == "" {
			refCode = fmt.Sprintf("id%d", tgID)
		}
		affProfile := models.AffiliateProfile{
			UserID:       user.ID,
			ReferralCode: refCode,
			CreatedAt:    time.Now(),
		}
		_ = s.db.Create(&affProfile).Error

		log.Printf("[Bot User Sync] New lead registered in DB: @%s (ID: %d)", cleanUsername, tgID)
		return &user
	}

	// Keep names up to date
	changed := false
	if cleanUsername != "" && user.Username != cleanUsername {
		user.Username = cleanUsername
		changed = true
	}
	if firstName != "" && user.FirstName != firstName {
		user.FirstName = firstName
		changed = true
	}
	if lastName != "" && user.LastName != lastName {
		user.LastName = lastName
		changed = true
	}
	if changed {
		s.db.Save(&user)
	}

	return &user
}

func (s *BotService) associateBotReferral(from *telego.User, inputUsername, refCode string) {
	if s.db == nil || from == nil {
		return
	}

	user := s.syncTelegramUser(from)
	if user == nil {
		return
	}

	if refCode != "" {
		_ = s.affiliateService.TrackReferral(user.ID, refCode)
	}
}
