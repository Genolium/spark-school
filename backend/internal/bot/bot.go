package bot

import (
	"context"
	"fmt"
	"log"
	"os"
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
	notifyQueue      chan queuedNotification
	quitChan         chan struct{}
}

func NewBotService(cfg *config.Config, db *gorm.DB) *BotService {
	affService := services.NewAffiliateService(db)
	s := &BotService{
		db:               db,
		cfg:              cfg,
		affiliateService: affService,
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

// startQueueWorker drains the outbound message queue with a 35ms rate limit (~28 msgs/sec max)
// to prevent Telegram API HTTP 429 Too Many Requests errors.
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
				m := tu.Message(chatID, item.message)
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
		// already closed
	default:
		close(s.quitChan)
	}
}

// Start launches the bot polling worker.
func (s *BotService) Start() {
	if s.bot == nil {
		return
	}

	if s.cfg.Domain != "localhost" && s.cfg.Domain != "127.0.0.1" && (s.cfg.TelegramAPIServer != "" || s.cfg.TelegramBotToken != "") {
		// When webhook is active (e.g. n8n or cloudflare worker), long polling must not be enabled
		// unless explicitly forced via FORCE_LONG_POLLING=true.
		if os.Getenv("FORCE_LONG_POLLING") != "true" {
			log.Println("Production webhook environment detected. Skipping Telegram Long Polling to avoid 409 Conflict.")
			return
		}
	}

	updates, err := s.bot.UpdatesViaLongPolling(nil)
	if err != nil {
		log.Printf("Failed to get Telegram updates channel: %v", err)
		return
	}

	log.Println("Telegram Bot worker started listening for commands...")

	for update := range updates {
		if update.Message != nil {
			s.handleMessage(update.Message)
		}
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

	text := strings.TrimSpace(msg.Text)
	chatID := tu.ID(msg.Chat.ID)

	if strings.HasPrefix(text, "/withdraw") {
		s.handleWithdraw(chatID, msg.From)
		return true
	}

	if strings.HasPrefix(text, "/start") {
		parts := strings.Fields(text)
		param := ""
		if len(parts) > 1 {
			param = parts[1]
		}

		// 1. Lead Magnet Delivery (?start=guide_essay or ?start=free_audit)
		if strings.HasPrefix(param, "guide_essay") || strings.HasPrefix(param, "free_audit") {
			reply := `🎁 <b>Анатомия победной заявки на $20,000</b>

Поздравляем! Ты сделал первый шаг к учёбе в США этим летом.

Внутри этого PDF-чеклиста:
• Структура победного эссе на 500 слов: хук, конфликт, импакт
• Резюме по формуле Google XYZ: как избежать банальностей
• Разбор 5 критических ошибок, убивающих заявку на корню

<i>Изучай материал и примени его к своей заявке!</i>`

			btnWeb := tu.InlineKeyboardButton("Перейти на сайт 🌐").WithURL(s.cfg.FrontendURL)
			btnTariff := tu.InlineKeyboardButton("Выбрать тариф от 2 900 ₽ ➔").WithURL(s.cfg.FrontendURL + "/#pricing")
			keyboard := tu.InlineKeyboard(
				tu.InlineKeyboardRow(btnWeb),
				tu.InlineKeyboardRow(btnTariff),
			)

			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
			if s.bot != nil {
				_, _ = s.bot.SendMessage(m)
			}
			return true
		}

		// 2. Payment Checkout Helper (?start=pay_<username> or ?start=pay_<username>_ref_<refCode> or ?start=pay)
		if strings.HasPrefix(param, "pay") {
			payload := strings.TrimPrefix(param, "pay_")
			if param == "pay" {
				payload = ""
			}
			username := payload
			refCode := ""

			if idx := strings.Index(payload, "_ref_"); idx != -1 {
				username = payload[:idx]
				refCode = payload[idx+len("_ref_"):]
			} else if strings.HasPrefix(payload, "ref_") {
				username = ""
				refCode = strings.TrimPrefix(payload, "ref_")
			}
			username = strings.TrimPrefix(username, "@")

			if msg.From != nil {
				s.associateBotReferral(msg.From, username, refCode)
			}

			displayUser := username
			if displayUser == "" && msg.From != nil {
				displayUser = msg.From.Username
				if displayUser == "" {
					displayUser = msg.From.FirstName
				}
			}
			if displayUser == "" {
				displayUser = "Student"
			}

			reply := fmt.Sprintf(`💳 <b>Бронирование места в проекте «так называемый SPARK»</b>

Аккаунт: @%s
Тариф: <b>Акселератор (Full Mentorship)</b>
Стоимость со скидкой: <b>6 900 ₽</b>
(Доступны также тарифы: <b>Базовый 2 900 ₽</b> и <b>VIP 14 900 ₽</b>)

<b>Способы оплаты:</b>
1. СБП (Система быстрых платежей): перевод по номеру телефона
2. Банковская карта РФ

После подтверждения оплаты так называемый Иль активирует доступ, и вам придёт персональная ссылка-приглашение в закрытый Telegram-канал участников:`, displayUser)

			btnCheck := tu.InlineKeyboardButton("Написать так называемому Илю ➔").WithURL("https://t.me/ilyan_vas")
			keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnCheck))

			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
			if s.bot != nil {
				_, _ = s.bot.SendMessage(m)
			}
			return true
		}

		// 3. Partner Payout / Withdraw (?start=withdraw)
		if strings.HasPrefix(param, "withdraw") {
			s.handleWithdraw(chatID, msg.From)
			return true
		}

		// 4. Referral link entry (?start=ref_<code>)
		if strings.HasPrefix(param, "ref_") {
			refCode := strings.TrimPrefix(param, "ref_")
			if msg.From != nil {
				s.associateBotReferral(msg.From, msg.From.Username, refCode)
			}
			welcome := fmt.Sprintf(`👋 Привет, <b>%s</b>!

Тебя пригласили по реферальной ссылке (код: <code>%s</code>).
Добро пожаловать в проект <b>«так называемый SPARK»</b>!

Нажми кнопку ниже, чтобы перейти на сайт со скидкой:`, msg.From.FirstName, refCode)

			btnApp := tu.InlineKeyboardButton("Перейти на сайт со скидкой 🌐").WithURL(s.cfg.FrontendURL + "?ref=" + refCode)
			keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnApp))

			m := tu.Message(chatID, welcome).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
			if s.bot != nil {
				_, _ = s.bot.SendMessage(m)
			}
			return true
		}

		// 5. Standard Welcome Message
		welcome := fmt.Sprintf(`👋 Привет, <b>%s</b>!

Я официальный бот проекта <b>«так называемый SPARK»</b> (куратор: так называемый Иль).
Здесь ты получишь материалы для подготовки к грантовой программе США, разборы эссе и доступ к симуляциям интервью.

Нажми кнопку ниже, чтобы перейти на официальный сайт программы:`, msg.From.FirstName)

		btnApp := tu.InlineKeyboardButton("Перейти на сайт 🌐").WithURL(s.cfg.FrontendURL)
		keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnApp))

		m := tu.Message(chatID, welcome).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		if s.bot != nil {
			_, _ = s.bot.SendMessage(m)
		}
		return true
	}

	// 6. Access Status Check (/status)
	if strings.HasPrefix(text, "/status") {
		var user models.User
		inviteLink := s.cfg.TelegramInviteLink
		if inviteLink == "" {
			inviteLink = "https://t.me/+so_called_spark_private"
		}

		if s.db != nil {
			err := s.db.Where("telegram_id = ?", msg.From.ID).First(&user).Error
			if err == nil && user.HasAccess {
				reply := fmt.Sprintf(`✅ <b>Ваш доступ активен!</b>

Вы являетесь действующим участником проекта <b>«так называемый SPARK»</b>.

Материалы, видеоразборы и комьюнити доступны в закрытом Telegram-канале:
%s`, inviteLink)

				btnChannel := tu.InlineKeyboardButton("Перейти в закрытый канал ➔").WithURL(inviteLink)
				keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnChannel))

				m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
				if s.bot != nil {
					_, _ = s.bot.SendMessage(m)
				}
				return true
			}
		}

		reply := `ℹ️ <b>Статус доступа: Не активен</b>

У вас пока нет активного доступа к проекту <b>«так называемый SPARK»</b>.
Тарифы: <b>Базовый</b> (2 900 ₽), <b>Акселератор</b> (6 900 ₽), <b>VIP</b> (14 900 ₽).

После подтверждения оплаты так называемый Иль предоставит персональную ссылку в закрытый канал.`

		btnTariff := tu.InlineKeyboardButton("Оформить тариф ➔").WithURL(s.cfg.FrontendURL + "/#pricing")
		btnCurator := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
		keyboard := tu.InlineKeyboard(
			tu.InlineKeyboardRow(btnTariff),
			tu.InlineKeyboardRow(btnCurator),
		)

		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		if s.bot != nil {
			_, _ = s.bot.SendMessage(m)
		}
		return true
	}

	// 7. Receipt Screenshot Handling (Photo / Document)
	if len(msg.Photo) > 0 || msg.Document != nil {
		var fileID, fileUniqueID string
		if len(msg.Photo) > 0 {
			largest := msg.Photo[len(msg.Photo)-1]
			fileID = largest.FileID
			fileUniqueID = largest.FileUniqueID
		} else if msg.Document != nil {
			fileID = msg.Document.FileID
			fileUniqueID = msg.Document.FileUniqueID
		}

		if fileUniqueID != "" && s.db != nil {
			var existing models.PaymentReceipt
			if err := s.db.Where("file_hash = ? OR (file_unique_id != '' AND file_unique_id = ?)", fileUniqueID, fileUniqueID).First(&existing).Error; err == nil {
				reply := "⚠️ <b>Этот чек уже был отправлен ранее</b>\n\nДанный скриншот чека уже зарегистрирован в системе. Если у вас возникли вопросы, напишите так называемому Илю (@ilyan_vas)."
				btnCurator := tu.InlineKeyboardButton("Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
				m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnCurator)))
				if s.bot != nil {
					_, _ = s.bot.SendMessage(m)
				}
				return true
			}

			uname := ""
			if msg.From != nil {
				uname = msg.From.Username
			}
			receipt := models.PaymentReceipt{
				FileHash:     fileUniqueID,
				FileUniqueID: fileUniqueID,
				TelegramID:   msg.Chat.ID,
				Username:     uname,
				FileID:       fileID,
				Tier:         "accelerator",
				Status:       "pending",
				CreatedAt:    time.Now(),
			}
			_ = s.db.Create(&receipt)

			reply := "🧾 <b>Чек об оплате успешно получен!</b>\n\nТак называемый Иль проверит поступление средств и активирует доступ к закрытому Telegram-каналу проекта «так называемый SPARK» в ближайшее время."
			m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
			if s.bot != nil {
				_, _ = s.bot.SendMessage(m)
			}
			return true
		}
	}

	// Handling non-/start, non-/status text messages (e.g., submitting essay/homework for review)
	if text != "" {
		reply := `📬 <b>Ваше сообщение принято куратором!</b>

Если вы отправили эссе или резюме на аудит — так называемый Иль проверит материалы и свяжется с вами в личных сообщениях.

Материалы и видеоразборы доступны в закрытом Telegram-канале участников.`
		btnMain := tu.InlineKeyboardButton("Открыть сайт программы ➔").WithURL(s.cfg.FrontendURL)
		keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnMain))

		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		if s.bot != nil {
			_, _ = s.bot.SendMessage(m)
		}
		return true
	}

	return true
}

// handleWithdraw delivers partner withdrawal instructions and balance in Telegram.
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

// SendNotification queues an outbound direct message through the rate-limited worker.
func (s *BotService) SendNotification(telegramID int64, message string) error {
	if telegramID == 0 {
		return nil
	}

	// If queue is initialized, route message through rate limiter
	if s.notifyQueue != nil {
		respChan := make(chan error, 1)
		item := queuedNotification{
			telegramID: telegramID,
			message:    message,
			respChan:   respChan,
		}

		select {
		case s.notifyQueue <- item:
			// Wait for worker result with reasonable timeout
			select {
			case err := <-respChan:
				return err
			case <-time.After(3 * time.Second):
				// Return nil after enqueue to avoid blocking caller indefinitely during massive bursts
				return nil
			}
		default:
			// If queue buffer is full, fallback to direct send or log
			log.Printf("[Bot Rate Limiter] Queue buffer full, sending directly to %d", telegramID)
		}
	}

	if s.bot == nil {
		log.Printf("[Bot Notification Mock] to %d: %s", telegramID, message)
		return nil
	}

	chatID := tu.ID(telegramID)
	m := tu.Message(chatID, message)
	_, err := s.bot.SendMessage(m)
	return err
}

// CreateOneTimeInviteLink generates a single-use invite link with a 24-hour expiration for a channel.
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

// associateBotReferral associates the student with a referrer when starting the bot.
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
		// User does not exist yet. Only check for placeholder accounts where telegram_id is unset
		// to strictly prevent hijacking of existing users' accounts.
		if uname != "" {
			var existingByName models.User
			if errName := s.db.Where("LOWER(username) = LOWER(?) AND (telegram_id = 0 OR telegram_id IS NULL)", uname).First(&existingByName).Error; errName == nil {
				existingByName.TelegramID = tgID
				if existingByName.FirstName == "" {
					existingByName.FirstName = from.FirstName
				}
				if existingByName.LastName == "" {
					existingByName.LastName = from.LastName
				}
				s.db.Save(&existingByName)
				user = existingByName
			}
		}

		if user.ID == 0 {
			firstName := from.FirstName
			if firstName == "" {
				firstName = uname
			}
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
				log.Printf("Failed to create user from bot: %v", err)
				return
			}
		}

		// Ensure AffiliateProfile exists for this user with guaranteed unique code
		cleanUname := strings.ToLower(uname)
		if cleanUname == "" {
			cleanUname = fmt.Sprintf("id%d", tgID)
		}
		var affProfile models.AffiliateProfile
		if errAff := s.db.Where("user_id = ?", user.ID).First(&affProfile).Error; errAff != nil {
			finalCode := cleanUname
			var existingCode models.AffiliateProfile
			if errCode := s.db.Where("referral_code = ?", finalCode).First(&existingCode).Error; errCode == nil && existingCode.UserID != user.ID {
				finalCode = fmt.Sprintf("%s_%d", cleanUname, user.ID)
			}
			affProfile = models.AffiliateProfile{
				UserID:       user.ID,
				ReferralCode: finalCode,
				CreatedAt:    time.Now(),
			}
			_ = s.db.Create(&affProfile).Error
		}
	} else {
		// Update username if it was empty or changed
		if user.Username == "" && uname != "" {
			user.Username = uname
			s.db.Save(&user)
		}
		// Also ensure existing user has an AffiliateProfile if it was missing
		var affProfile models.AffiliateProfile
		if errAff := s.db.Where("user_id = ?", user.ID).First(&affProfile).Error; errAff != nil {
			cleanUname := strings.ToLower(user.Username)
			if cleanUname == "" {
				cleanUname = fmt.Sprintf("id%d", tgID)
			}
			finalCode := cleanUname
			var existingCode models.AffiliateProfile
			if errCode := s.db.Where("referral_code = ?", finalCode).First(&existingCode).Error; errCode == nil && existingCode.UserID != user.ID {
				finalCode = fmt.Sprintf("%s_%d", cleanUname, user.ID)
			}
			affProfile = models.AffiliateProfile{
				UserID:       user.ID,
				ReferralCode: finalCode,
				CreatedAt:    time.Now(),
			}
			_ = s.db.Create(&affProfile).Error
		}
	}

	// Link referral if refCode is provided
	if refCode != "" && user.ID != 0 && s.affiliateService != nil {
		if err := s.affiliateService.TrackReferral(user.ID, refCode); err != nil {
			log.Printf("TrackReferral error for user %d and ref %s: %v", user.ID, refCode, err)
		}
	}
}
