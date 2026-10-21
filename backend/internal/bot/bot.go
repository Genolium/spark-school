package bot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/payments"
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
	paymentsService  *payments.Service
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
		paymentsService:  payments.NewService(db, affService),
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

	// Start 72-hour drip lead nurture worker
	go s.startDripWorker()

	// Start daily curator digest cron worker
	go s.startDigestWorker()

	// Start Zoom reminder worker (24h and 1h before the call)
	go s.startReminderWorker()

	return s
}

// getSession returns user session, initializing session manager if nil.
func (s *BotService) getSession(chatID int64) *UserSession {
	if s.sessions == nil {
		s.sessions = NewSessionManager(30 * time.Minute)
	}
	return s.sessions.Get(chatID)
}

func sanitizeAttributionToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "_"); idx != -1 {
		raw = raw[:idx]
	}
	if len(raw) > 32 {
		raw = raw[:32]
	}
	for _, r := range raw {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return ""
		}
	}
	return raw
}

// attributeLead links an anonymous landing visitor to a Telegram user using the
// short token that travelled in the /start parameter. It also records the
// server-side bot_lead step, which a browser cannot fabricate.
func (s *BotService) attributeLead(from *telego.User, token string) error {
	if s.db == nil || from == nil || token == "" {
		return nil
	}
	var user models.User
	if err := s.db.Where("telegram_id = ?", from.ID).First(&user).Error; err != nil {
		return err
	}
	if user.AttributionToken != "" {
		return nil
	}
	var landing models.AnalyticsEvent
	err := s.db.Where("attribution_token = ? AND visitor_id <> ''", token).Order("created_at DESC").First(&landing).Error
	visitorID := ""
	if err == nil {
		visitorID = landing.VisitorID
	}
	if err := s.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"attribution_token": token,
		"visitor_id":        visitorID,
	}).Error; err != nil {
		return err
	}
	meta := fmt.Sprintf(`{"telegram_id":%d,"attribution_token":%q}`, from.ID, token)
	return s.db.Create(&models.AnalyticsEvent{
		VisitorID:        visitorID,
		AttributionToken: token,
		Step:             "bot_lead",
		Source:           landing.Source,
		Campaign:         landing.Campaign,
		Tier:             user.Tier,
		Metadata:         meta,
		CreatedAt:        time.Now(),
	}).Error
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

// ProcessUpdate is kept as a convenience wrapper for long polling and tests.
// Webhook processing must call HandleUpdate with the request context.
func (s *BotService) ProcessUpdate(update *telego.Update) error {
	return s.HandleUpdate(context.Background(), update)
}

// HandleUpdate executes one Telegram update and reports every failure to the
// caller. A recovered panic is returned as an error (named return value), so a
// crash can never be mistaken for successful processing.
func (s *BotService) HandleUpdate(ctx context.Context, update *telego.Update) (err error) {
	if update == nil {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("bot update panic: %v", r)
			log.Printf("[Bot Panic] update_id=%d: %v", update.UpdateID, r)
		}
	}()

	if ctx != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}

	if update.CallbackQuery != nil {
		return s.HandleCallbackQuery(update.CallbackQuery)
	}
	if update.Message != nil {
		return s.handleMessage(update.Message)
	}
	return nil
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
		if err := s.HandleUpdate(context.Background(), &update); err != nil {
			log.Printf("[Bot Polling] update %d failed: %v", update.UpdateID, err)
		}
	}
}

func (s *BotService) handleMessage(msg *telego.Message) error {
	if msg == nil {
		return nil
	}

	log.Printf("[Bot Inbound] chat_id=%d, type=%s, from=%s, text=%q", msg.Chat.ID, msg.Chat.Type, msg.From.FirstName, msg.Text)

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
		return nil
	}

	if strings.HasPrefix(text, "/status") || strings.HasPrefix(text, "/orders") {
		s.SendOrdersHistory(chatID, msg.From)
		return nil
	}

	if strings.HasPrefix(text, "/plan") {
		s.SendPreparationPlan(chatID, msg.From)
		return nil
	}

	if strings.HasPrefix(text, "/done_step") {
		parts := strings.Fields(text)
		stepNum := 1
		if len(parts) > 1 {
			if num, err := strconv.Atoi(parts[1]); err == nil {
				stepNum = num
			}
		}
		s.TogglePlanStep(chatID, msg.From, stepNum)
		return nil
	}

	// [Временно приостановлено: Сценарий 2 (Сдача работ на аудит) и Сценарий 3 (Запись на Zoom-созвон 1-на-1)]
	/*
		if strings.HasPrefix(text, "/submit_work") {
			return s.SubmitWork(chatID, msg.From, text)
			return nil
		}

		if strings.HasPrefix(text, "/book_call") {
			s.BookCallDialog(chatID, msg.From, "")
			return nil
		}
	*/

	if strings.HasPrefix(text, "/interview") {
		s.SendInterviewTrainer(chatID)
		return nil
	}

	if strings.HasPrefix(text, "/queue") {
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

		if strings.HasPrefix(text, "/cancel_call") {
			s.CancelBooking(chatID, msg.From, 0)
			return nil
		}
	*/

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

	canUpgrade := false
	if from != nil && s.db != nil {
		var u models.User
		if err := s.db.Where("telegram_id = ?", from.ID).First(&u).Error; err == nil {
			if u.HasAccess && strings.ToLower(strings.TrimSpace(u.Tier)) != "vip" {
				canUpgrade = true
			}
		}
	}

	mainKb := MainMenuKeyboard(s.cfg.FrontendURL, canUpgrade, s.getPricing())

	// Send rich photo card using cached or local file
	photoFile := s.ResolvePhotoFile(MediaMainBanner)
	photoMsg := tu.Photo(chatID, photoFile).
		WithCaption(caption).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(mainKb)

	sentMsg, err := s.bot.SendPhoto(photoMsg)
	if err == nil && sentMsg != nil {
		s.CacheSentPhoto(MediaMainBanner, sentMsg)
	} else if err != nil {
		log.Printf("[Bot Error] SendPhoto with local media failed: %v. Retrying with verified photo file ID...", err)
		// Fallback 1: Try sending with verified Telegram photo ID and KEEP custom emojis!
		fallbackPhoto := tu.Photo(chatID, tu.FileFromID(PhotoGoldenGate)).
			WithCaption(caption).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(mainKb)
		if _, fbErr := s.bot.SendPhoto(fallbackPhoto); fbErr != nil {
			log.Printf("[Bot Error] Fallback SendPhoto also failed: %v. Sending message...", fbErr)
			textMsg := tu.Message(chatID, caption).
				WithParseMode(telego.ModeHTML).
				WithReplyMarkup(mainKb)
			if _, textErr := s.bot.SendMessage(textMsg); textErr != nil {
				log.Printf("[Bot Error] SendMessage with custom emoji failed: %v. Cleaning tags...", textErr)
				cleanCaption := StripTgEmoji(caption)
				cleanMsg := tu.Message(chatID, cleanCaption).
					WithParseMode(telego.ModeHTML).
					WithReplyMarkup(mainKb)
				_, _ = s.bot.SendMessage(cleanMsg)
			}
		}
	}
}

func (s *BotService) getPricing() models.CoursePricing {
	if s.db != nil {
		return models.GetCoursePricing(s.db)
	}
	return models.DefaultCoursePricing
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

	pricing := s.getPricing()
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

	pricing := s.getPricing()
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

// studentStatusDetails renders the student's current program state: progress,
// audit balance, pending work deadlines and the nearest Zoom call.
func (s *BotService) studentStatusDetails(user *models.User) string {
	if s.db == nil || user == nil {
		return ""
	}
	var sb strings.Builder

	var progress models.StudentProgress
	if err := s.db.Where("telegram_id = ?", user.TelegramID).First(&progress).Error; err == nil {
		done := 0
		for _, step := range []bool{progress.Step1Done, progress.Step2Done, progress.Step3Done, progress.Step4Done, progress.Step5Done, progress.Step6Done} {
			if step {
				done++
			}
		}
		sb.WriteString(fmt.Sprintf("\n📈 <b>Программа:</b> %d/6 этапов пройдено", done))
	}

	if strings.EqualFold(strings.TrimSpace(user.Tier), "basic") {
		sb.WriteString(fmt.Sprintf("\n🔍 <b>Оплаченных аудитов:</b> %d", user.AuditCredits))
	}

	var pendingWorks int64
	s.db.Model(&models.StudentSubmission{}).
		Where("telegram_id = ? AND status IN ?", user.TelegramID, []string{"pending", "reviewing"}).
		Count(&pendingWorks)
	if pendingWorks > 0 {
		var oldest models.StudentSubmission
		if err := s.db.Where("telegram_id = ? AND status IN ?", user.TelegramID, []string{"pending", "reviewing"}).
			Order("submitted_at ASC").First(&oldest).Error; err == nil {
			deadline := oldest.SubmittedAt.Add(48 * time.Hour)
			overdue := time.Now().After(deadline)
			label := deadline.Format("02.01 15:04")
			if overdue {
				label = "просрочен — куратор уже уведомлён"
			}
			sb.WriteString(fmt.Sprintf("\n📝 <b>Работ в очереди:</b> %d (ближайший ответ до %s)", pendingWorks, label))
		} else {
			sb.WriteString(fmt.Sprintf("\n📝 <b>Работ в очереди:</b> %d", pendingWorks))
		}
	}

	var booking models.CallBooking
	if err := s.db.Where("telegram_id = ? AND status = ? AND cancelled_at IS NULL", user.TelegramID, "confirmed").
		Order("slot_key ASC").First(&booking).Error; err == nil {
		sb.WriteString(fmt.Sprintf("\n📅 <b>Ближайший Zoom:</b> %s", booking.SlotTime))
	}
	return sb.String()
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

// HandleCallbackQuery handles all button taps.
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

	// [Временно приостановлено: Сценарий 3 (Запись и отмена слотов созвона)]
	/*
		if strings.HasPrefix(data, "cancel_booking_") {
			idStr := strings.TrimPrefix(data, "cancel_booking_")
			idNum, _ := strconv.Atoi(idStr)
			s.CancelBooking(chatID, &query.From, uint(idNum))
			return nil
		}

		if strings.HasPrefix(data, "book_slot_") {
			slotKey := strings.TrimPrefix(data, "book_slot_")
			s.BookCallDialog(chatID, &query.From, slotKey)
			return nil
		}
	*/

	if data == "slot_already_taken" {
		m := tu.Message(chatID, "🔒 Этот слот уже занят другим студентом. Пожалуйста, выберите другое свободное время.")
		_, _ = s.bot.SendMessage(m)
		s.BookCallDialog(chatID, &query.From, "")
		return nil
	}

	// Curator submission reviews from queue
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

	case "action_status":
		s.SendOrdersHistory(chatID, &query.From)
		return nil

	case "action_orders":
		s.SendOrdersHistory(chatID, &query.From)
		return nil

	case "action_plan":
		s.SendPreparationPlan(chatID, &query.From)
		return nil

	case "plan_toggle_1":
		s.TogglePlanStep(chatID, &query.From, 1)
		return nil
	case "plan_toggle_2":
		s.TogglePlanStep(chatID, &query.From, 2)
		return nil
	case "plan_toggle_3":
		s.TogglePlanStep(chatID, &query.From, 3)
		return nil
	case "plan_toggle_4":
		s.TogglePlanStep(chatID, &query.From, 4)
		return nil
	case "plan_toggle_5":
		s.TogglePlanStep(chatID, &query.From, 5)
		return nil
	case "plan_toggle_6":
		s.TogglePlanStep(chatID, &query.From, 6)
		return nil

	// [Временно приостановлено: Сценарий 2 (Сдача работ) и Сценарий 3 (Запись на Zoom-созвон 1-на-1)]
	/*
		case "action_submit_work_info":
			return s.SubmitWork(chatID, &query.From, "/submit_work")
			return nil

		case "action_book_call":
			s.BookCallDialog(chatID, &query.From, "")
			return nil

		case "book_slot_tue_18":
			s.BookCallDialog(chatID, &query.From, "Вторник, 18:00 МСК")
			return nil
		case "book_slot_thu_1930":
			s.BookCallDialog(chatID, &query.From, "Четверг, 19:30 МСК")
			return nil
		case "book_slot_sat_14":
			s.BookCallDialog(chatID, &query.From, "Суббота, 14:00 МСК")
			return nil
	*/

	case "action_interview":
		s.SendInterviewTrainer(chatID)
		return nil

	case "admin_queue":
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

// sendAccessNotificationToStudent reliably delivers channel invite link to student using 6-stage fallback logic
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

func (s *BotService) processReceiptApproval(query *telego.CallbackQuery, idStr string) error {
	adminID := query.From.ID
	if !s.IsAdmin(adminID) {
		s.answerCallback(query, "⛔️ У вас нет прав администратора.", true)
		return nil
	}
	if s.db == nil || s.paymentsService == nil {
		s.answerCallback(query, "❌ Сервис платежей недоступен.", true)
		return fmt.Errorf("payment service is unavailable")
	}

	receiptID64, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32)
	if err != nil || receiptID64 == 0 {
		s.answerCallback(query, "Чек не найден в базе данных.", true)
		return fmt.Errorf("invalid receipt id %q", idStr)
	}
	receiptID := uint(receiptID64)

	var receipt models.PaymentReceipt
	if err := s.db.First(&receipt, receiptID).Error; err != nil {
		s.answerCallback(query, "Чек не найден в базе данных.", true)
		return err
	}

	paymentID := fmt.Sprintf("RECEIPT_%d", receipt.ID)
	result, err := s.paymentsService.Confirm(payments.ConfirmInput{
		Source:      "sbp_receipt",
		PaymentID:   paymentID,
		ReceiptID:   receipt.ID,
		TelegramID:  receipt.TelegramID,
		Username:    receipt.Username,
		Tier:        receipt.Tier,
		Amount:      receipt.Amount,
		Currency:    "RUB",
		PayMethod:   "sbp_receipt",
		ProductName: "Оплата участия в проекте «так называемый SPARK»",
		CreatedAt:   receipt.CreatedAt,
	})
	if err != nil {
		log.Printf("[Bot Approval] Receipt #%d confirmation failed: %v", receipt.ID, err)
		s.answerCallback(query, "❌ Не удалось подтвердить чек: "+err.Error(), true)
		return err
	}

	// A repeated approval is a deliberate resend: it must not change money,
	// credits or commissions, only enqueue the access message again.
	if result.AlreadyProcessed {
		if err := s.paymentsService.RequeueAccessNotification(paymentID); err != nil {
			s.answerCallback(query, "⚠️ Платёж уже подтверждён, но повторная отправка не поставлена в очередь: "+err.Error(), true)
			return err
		}
		s.answerCallback(query, "✅ Платёж уже подтверждён. Ссылка поставлена в очередь на повторную отправку.", false)
	} else if result.AccessGranted {
		msg := "✅ Чек подтверждён. Уведомление студенту поставлено в очередь."
		if result.AuditCredits > 0 {
			msg = "✅ Аудит эссе подтверждён. Уведомление студенту поставлено в очередь."
		} else if result.Tier == "vip" {
			msg = "✅ Апгрейд до VIP подтверждён. Уведомление студенту поставлено в очередь."
		}
		s.answerCallback(query, msg, false)
	} else {
		s.answerCallback(query, "⚠️ Платёж записан без получателя. Проверьте Telegram ID вручную.", true)
	}

	// Update curator's inline buttons.
	if s.bot != nil && query.Message != nil {
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
		_, _ = s.bot.EditMessageReplyMarkup(&telego.EditMessageReplyMarkupParams{
			ChatID:      curatorChatID,
			MessageID:   msgID,
			ReplyMarkup: approvedKb,
		})
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

func (s *BotService) processReceiptRejection(query *telego.CallbackQuery, idStr string) {
	adminID := query.From.ID
	if !s.IsAdmin(adminID) {
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

Пожалуйста, свяжитесь с куратором в поддержке: @spark_prep_bot`

	btnHelp := tu.InlineKeyboardButton("💬 Написать куратору").WithURL("https://t.me/spark_prep_bot")
	if s.bot != nil {
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
	}

	if s.bot != nil && query.Message != nil {
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

func (s *BotService) processReceiptTierToggle(query *telego.CallbackQuery, idStr string, newTier string) {
	adminID := query.From.ID
	if !s.IsAdmin(adminID) {
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

	var student models.User
	userFound := false
	isUpgrade := false
	prevTier := ""
	if err := s.db.Where("telegram_id = ?", receipt.TelegramID).First(&student).Error; err == nil {
		userFound = true
		if student.HasAccess {
			isUpgrade = true
			prevTier = strings.ToLower(strings.TrimSpace(student.Tier))
		}
	}

	receipt.Tier = newTier
	pricing := s.getPricing()
	if isUpgrade {
		if prevTier == "basic" {
			if newTier == "vip" {
				diff := pricing.VIPPrice - pricing.BasicPrice
				if diff <= 0 {
					diff = 10910.0
				}
				receipt.Amount = diff
				receipt.Tier = "vip"
			} else {
				diff := pricing.AcceleratorPrice - pricing.BasicPrice
				if diff <= 0 {
					diff = 2910.0
				}
				receipt.Amount = diff
				receipt.Tier = "accelerator"
			}
		} else if prevTier == "accelerator" {
			diff := pricing.VIPPrice - pricing.AcceleratorPrice
			if diff <= 0 {
				diff = 8000.0
			}
			receipt.Amount = diff
			receipt.Tier = "vip"
		}
	} else {
		switch newTier {
		case "basic":
			receipt.Amount = pricing.BasicPrice
			receipt.Tier = "basic"
		case "vip":
			receipt.Amount = pricing.VIPPrice
			receipt.Tier = "vip"
		default:
			receipt.Amount = pricing.AcceleratorPrice
			receipt.Tier = "accelerator"
		}
	}
	s.db.Save(&receipt)

	if userFound && !isUpgrade {
		student.Tier = receipt.Tier
		s.db.Save(&student)
	}

	_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		Text:            fmt.Sprintf("✅ Тариф чека #%d переключён на %s!", receipt.ID, strings.ToUpper(receipt.Tier)),
	})

	if query.Message != nil {
		curatorChatID := tu.ID(query.Message.GetChat().ID)
		msgID := query.Message.GetMessageID()

		studentDisplay := receipt.Username
		if studentDisplay == "" {
			studentDisplay = fmt.Sprintf("ID %d", receipt.TelegramID)
		} else if !strings.HasPrefix(studentDisplay, "@") {
			studentDisplay = "@" + studentDisplay
		}

		dateStr := ""
		if !receipt.CreatedAt.IsZero() {
			dateStr = receipt.CreatedAt.Format("15:04 02.01.2006")
		} else {
			dateStr = time.Now().Format("15:04 02.01.2006")
		}

		var newCaption string
		var btnApprove telego.InlineKeyboardButton

		if isUpgrade {
			newCaption = fmt.Sprintf(`🚀 <b>Чек на АПГРЕЙД тарифа #%d</b>

👤 Студент: <b>%s</b>
🆔 ID: <code>%d</code>
🎓 Апгрейд: <b>%s ➔ %s</b>
💰 Сумма доплаты: <b>%.0f ₽</b>
📅 Дата: <i>%s</i>

Подтвердите поступление средств и повышение тарифа:`,
				receipt.ID,
				studentDisplay,
				receipt.TelegramID,
				strings.ToUpper(prevTier), strings.ToUpper(receipt.Tier),
				receipt.Amount,
				dateStr,
			)
			btnApprove = tu.InlineKeyboardButton("✅ Одобрить апгрейд").WithCallbackData(fmt.Sprintf("receipt_approve_%d", receipt.ID))
		} else {
			newCaption = fmt.Sprintf(`🧾 <b>Чек об оплате #%d</b>

👤 Студент: <b>%s</b>
🆔 ID: <code>%d</code>
🎓 Тариф: <b>%s</b>
💰 Сумма: <b>%.0f ₽</b>
📅 Дата: <i>%s</i>

Подтвердите поступление средств и выдачу доступа:`,
				receipt.ID,
				studentDisplay,
				receipt.TelegramID,
				strings.ToUpper(receipt.Tier),
				receipt.Amount,
				dateStr,
			)
			btnApprove = tu.InlineKeyboardButton("✅ Одобрить и выдать доступ").WithCallbackData(fmt.Sprintf("receipt_approve_%d", receipt.ID))
		}

		btnReject := tu.InlineKeyboardButton("❌ Отклонить").WithCallbackData(fmt.Sprintf("receipt_reject_%d", receipt.ID))

		var switchTierButtons []telego.InlineKeyboardButton
		if !isUpgrade {
			tierLow := strings.ToLower(strings.TrimSpace(receipt.Tier))
			if tierLow != "basic" {
				switchTierButtons = append(switchTierButtons, tu.InlineKeyboardButton("⚡️ На Базовый").WithCallbackData(fmt.Sprintf("receipt_tier_basic_%d", receipt.ID)))
			}
			if tierLow != "accelerator" {
				switchTierButtons = append(switchTierButtons, tu.InlineKeyboardButton("🔄 На Акселератор").WithCallbackData(fmt.Sprintf("receipt_tier_accelerator_%d", receipt.ID)))
			}
			if tierLow != "vip" {
				switchTierButtons = append(switchTierButtons, tu.InlineKeyboardButton("👑 На VIP").WithCallbackData(fmt.Sprintf("receipt_tier_vip_%d", receipt.ID)))
			}
		}

		rows := [][]telego.InlineKeyboardButton{
			{btnApprove, btnReject},
		}
		if len(switchTierButtons) > 0 {
			rows = append(rows, switchTierButtons)
		}
		if receipt.Username != "" {
			cleanUser := strings.TrimPrefix(receipt.Username, "@")
			rows = append(rows, []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton(fmt.Sprintf("💬 Написать @%s", cleanUser)).WithURL("https://t.me/" + cleanUser),
			})
		}
		newKb := tu.InlineKeyboard(rows...)

		editCap := &telego.EditMessageCaptionParams{
			ChatID:      curatorChatID,
			MessageID:   msgID,
			Caption:     newCaption,
			ParseMode:   telego.ModeHTML,
			ReplyMarkup: newKb,
		}
		if _, err := s.bot.EditMessageCaption(editCap); err != nil {
			editTxt := &telego.EditMessageTextParams{
				ChatID:      curatorChatID,
				MessageID:   msgID,
				Text:        newCaption,
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: newKb,
			}
			_, _ = s.bot.EditMessageText(editTxt)
		}
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

// SendFiscalReceipt formats and delivers an official fiscal receipt notification with an optional inline button to view the receipt.
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
		log.Printf("[Bot Fiscal Receipt Mock] to %d (amount: %.2f, url: %s): %s", telegramID, amount, receiptURL, msg)
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

// sendUpgradeOptions presents upgrade options based on student's current tier.
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

// handleUpgradePayment sets upgrade state and sends payment instructions for exact price difference.
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

// startDigestWorker runs background ticker and fires daily digest at configured hour (default 21:00).
func (s *BotService) startDigestWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	targetHour := 21
	if s.cfg != nil && s.cfg.DigestHour >= 0 && s.cfg.DigestHour <= 23 {
		targetHour = s.cfg.DigestHour
	}

	lastSentDate := ""

	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			now := time.Now()
			today := now.Format("2006-01-02")
			if now.Hour() == targetHour && now.Minute() == 0 && lastSentDate != today {
				lastSentDate = today
				log.Printf("[Daily Digest Cron] Automatically triggering curator daily digest for %s at %02d:%02d...", today, now.Hour(), now.Minute())
				_, _ = s.SendCuratorDailyDigest()
			}
		}
	}
}
