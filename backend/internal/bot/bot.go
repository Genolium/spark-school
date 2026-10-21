package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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
		sessions:         NewSessionManager(30*time.Minute, db),
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

	return s
}

// getSession returns user session, initializing session manager if nil.

// attributeLead links an anonymous landing visitor to a Telegram user using the
// short token that travelled in the /start parameter. It also records the
// server-side bot_lead step, which a browser cannot fabricate.

// startQueueWorker drains outbound queue with 35ms rate limiting

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
		if chatID := updateChatID(update); chatID != 0 && s.sessions != nil {
			if session := s.sessions.Peek(chatID); session != nil {
				s.sessions.Save(chatID, session)
			}
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

// studentStatusDetails renders the student's current program state: progress,
// audit balance and pending work deadlines.
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

	return sb.String()
}

// HandleCallbackQuery handles all button taps.

// sendAccessNotificationToStudent reliably delivers channel invite link to student using 6-stage fallback logic

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

	// Only a pending receipt may be rejected. A stale button pressed after
	// approval must not overwrite a confirmed payment.
	res := s.db.Model(&models.PaymentReceipt{}).
		Where("id = ? AND status = ?", receipt.ID, "pending").
		Updates(map[string]interface{}{"status": "rejected", "updated_at": time.Now()})
	if res.Error != nil {
		log.Printf("[Bot Rejection] failed to update receipt #%d: %v", receipt.ID, res.Error)
		return
	}
	if res.RowsAffected != 1 {
		s.answerCallback(query, "ℹ️ Чек уже обработан. Обновите список чеков.", true)
		return
	}
	receipt.Status = "rejected"

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
	if receipt.Status != "pending" {
		s.answerCallback(query, "ℹ️ Чек уже обработан. Изменение тарифа недоступно.", true)
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
	if err := s.db.Save(&receipt).Error; err != nil {
		log.Printf("[Bot Receipt] failed to save tier toggle for #%d: %v", receipt.ID, err)
		return
	}

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

// SendNotification queues an outbound direct message through rate limiter

// CreateOneTimeInviteLink generates a single-use invite link with a 24-hour expiration.

// SendFiscalReceipt formats and delivers an official fiscal receipt notification with an optional inline button to view the receipt.

// syncTelegramUser ensures every user interacting with the bot is persisted in Postgres,
// enabling full CRM visibility for leads and unpaid students in the web admin panel.

// sendUpgradeOptions presents upgrade options based on student's current tier.

// handleUpgradePayment sets upgrade state and sends payment instructions for exact price difference.

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
				if !s.acquireSchedulerLease("bot:digest", 2*time.Minute) {
					continue
				}
				lastSentDate = today
				log.Printf("[Daily Digest Cron] Automatically triggering curator daily digest for %s at %02d:%02d...", today, now.Hour(), now.Minute())
				_, _ = s.SendCuratorDailyDigest()
			}
		}
	}
}
