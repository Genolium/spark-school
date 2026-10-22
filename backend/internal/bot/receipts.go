package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/money"
)

// HandleReceiptUpload processes receipt screenshot from user with SHA-256 deduplication and fraud prevention.
func (s *BotService) HandleReceiptUpload(msg *telego.Message) error {
	if s.bot == nil || msg == nil {
		return nil
	}

	if msg.From != nil {
		s.syncTelegramUser(msg.From)
	}

	chatID := tu.ID(msg.Chat.ID)
	session := s.getSession(msg.Chat.ID)

	var fileID string
	var fileUniqueID string

	if len(msg.Photo) > 0 {
		highestRes := msg.Photo[len(msg.Photo)-1]
		fileID = highestRes.FileID
		fileUniqueID = highestRes.FileUniqueID
	} else if msg.Document != nil {
		fileID = msg.Document.FileID
		fileUniqueID = msg.Document.FileUniqueID
	} else {
		return nil
	}

	// Calculate deterministic unique hash based on fileUniqueID or fileID
	hashInput := fileUniqueID
	if hashInput == "" {
		hashInput = fileID
	}
	hashBytes := sha256.Sum256([]byte(hashInput))
	fileHash := hex.EncodeToString(hashBytes[:])

	// Check if this receipt was already submitted (Anti-Fraud deduplication)
	var existingReceipt models.PaymentReceipt
	if err := s.db.Where("file_hash = ? OR (file_unique_id = ? AND file_unique_id != '')", fileHash, fileUniqueID).First(&existingReceipt).Error; err == nil {
		reply := "⚠️ <b>Этот чек уже был загружен ранее!</b>\n\nЕсли произошла ошибка или вы отправили чек повторно, пожалуйста, напишите куратору: @ilyan_vas"
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return nil
	}

	var dbUser models.User
	userFound := false
	if s.db != nil {
		if err := s.db.Where("telegram_id = ?", msg.Chat.ID).First(&dbUser).Error; err == nil {
			userFound = true
		}
	}

	chosenTier := strings.ToLower(strings.TrimSpace(session.SelectedTier))
	isUpgrade := false
	var upgradeCurrentTier string
	var amount float64

	if chosenTier == "audit_essay" || strings.ToLower(strings.TrimSpace(session.UpgradeToTier)) == "audit_essay" {
		isUpgrade = true
		chosenTier = "audit_essay"
		amount = 1900.0
	} else if session.IsUpgrade && session.UpgradeAmount > 0 {
		isUpgrade = true
		amount = session.UpgradeAmount
		chosenTier = strings.ToLower(strings.TrimSpace(session.UpgradeToTier))
		upgradeCurrentTier = strings.ToLower(strings.TrimSpace(session.UpgradeFromTier))
	} else if userFound && dbUser.HasAccess {
		dbTier := strings.ToLower(strings.TrimSpace(dbUser.Tier))
		if dbTier == "accelerator" {
			isUpgrade = true
			upgradeCurrentTier = "accelerator"
			chosenTier = "vip"
			amount = 8000.0
		}
	}

	if !isUpgrade {
		if chosenTier == "" || chosenTier == "accelerator" {
			if userFound && strings.ToLower(strings.TrimSpace(dbUser.Tier)) == "vip" {
				chosenTier = "vip"
			}
		}
		if chosenTier == "" {
			chosenTier = "accelerator"
		}

		pricing := s.pricingForSession(session)
		switch chosenTier {
		case "basic":
			amount = pricing.BasicPrice
			chosenTier = "basic"
		case "vip":
			amount = pricing.VIPPrice
			chosenTier = "vip"
		default:
			amount = pricing.AcceleratorPrice
			chosenTier = "accelerator"
		}

		if session.DiscountPct > 0 || session.PromoCode == "SPARK5" {
			pct := session.DiscountPct
			if pct <= 0 {
				pct = 5
			}
			amount = amount * (1.0 - float64(pct)/100.0)
		}
	}

	username := msg.From.Username
	if username == "" {
		username = msg.From.FirstName
	}
	username = strings.TrimPrefix(username, "@")

	// Legacy sessions may carry the discount percentage but lose the code
	// itself. The user's profile is the source of truth in that case.
	promoCode := receiptPromoCode(session.PromoCode, session.DiscountPct, dbUser.PromoCode)
	if session.PromoCode == "" && promoCode != "" {
		session.PromoCode = promoCode
	}

	receipt := models.PaymentReceipt{
		FileHash:      fileHash,
		FileUniqueID:  fileUniqueID,
		TelegramID:    msg.Chat.ID,
		Username:      username,
		FileID:        fileID,
		Tier:          chosenTier,
		Amount:        amount,
		AmountKopecks: int64(money.FromRubles(amount)),
		Status:        "pending",
		PromoCode:     promoCode,
		CreatedAt:     time.Now(),
	}

	if err := s.db.Create(&receipt).Error; err != nil {
		reply := "❌ Ошибка при сохранении чека в системе. Пожалуйста, напишите так называемому Илю (@ilyan_vas)."
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return nil
	}
	// Server-side order creation event, linked to the anonymous visitor.
	if u := s.syncTelegramUser(msg.From); u != nil {
		metadata := fmt.Sprintf(`{"order_id":"RECEIPT_%d","telegram_id":%d,"amount":%f,"tier":%q}`, receipt.ID, receipt.TelegramID, receipt.Amount, chosenTier)
		_ = s.db.Create(&models.AnalyticsEvent{
			VisitorID:        u.VisitorID,
			AttributionToken: u.AttributionToken,
			Step:             "order_created",
			Source:           "telegram",
			Tier:             chosenTier,
			PromoCode:        promoCode,
			Metadata:         metadata,
			CreatedAt:        time.Now(),
		}).Error
	}

	if u := s.syncTelegramUser(msg.From); u != nil {
		// The selected tier is an intent, not an entitlement. It is applied
		// only after the payment is confirmed, so an unpaid receipt cannot
		// silently change the student's paid tier.
		if promoCode != "" && u.PromoCode == "" {
			u.PromoCode = promoCode
		} else if u.PromoCode == "" && (amount == 6555.0 || amount == 14155.0 || amount == 3790.0) {
			u.PromoCode = "SPARK5"
		}
		s.db.Save(u)
	}

	session.State = StateDefault
	session.IsUpgrade = false
	session.UpgradeAmount = 0

	// Send confirmation to student
	var reply string
	if chosenTier == "audit_essay" {
		reply = `🧾 <b>Чек на текстовый аудит эссе (1 900 ₽) успешно принят!</b>

Так называемый Иль проверит поступление средств и сразу напишет вам с подробной инструкцией, как отправить черновик в Google Docs.`
	} else if isUpgrade {
		reply = fmt.Sprintf(`🧾 <b>Чек на апгрейд тарифа успешно принят!</b>

Новый тариф: <b>%s</b>
Сумма доплаты: <b>%.0f ₽</b>

Так называемый Иль проверит поступление средств и сразу повысит ваш тариф в закрытом канале потока.`, strings.ToUpper(chosenTier), amount)
	} else {
		reply = fmt.Sprintf(`🧾 <b>Чек об оплате успешно принят!</b>

Тариф: <b>%s</b>
Сумма: <b>%.0f ₽</b>

Так называемый Иль проверит поступление средств и сразу пришлёт персональную ссылку-приглашение в закрытый канал потока.`, strings.ToUpper(chosenTier), amount)
	}

	btnBlog := tu.InlineKeyboardButton("🇺🇸 Пока можно почитать блог").WithURL("https://t.me/so_called_spark")
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnBlog))
	m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
	_, _ = s.bot.SendMessage(m)

	// Send actionable review card to Curator (Admin)
	if s.cfg.AdminTelegramID != 0 {
		_ = s.SendAdminReceiptCard(s.cfg.AdminTelegramID, &receipt)
	}

	_ = upgradeCurrentTier
	return nil
}

// SendAdminReceiptCard sends the review notification with photo/document and 1-click action buttons,
// with robust text fallback if Telegram media API fails (e.g. invalid fileID, mock or expired media).
func (s *BotService) SendAdminReceiptCard(adminID int64, receipt *models.PaymentReceipt) error {
	if s.bot == nil || adminID == 0 || receipt == nil {
		return nil
	}

	adminChatID := tu.ID(adminID)

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

	isUpgradeCard := false
	var prevTier string
	if s.db != nil {
		var student models.User
		if err := s.db.Where("telegram_id = ?", receipt.TelegramID).First(&student).Error; err == nil {
			if student.HasAccess && (receipt.Amount == 4000 || receipt.Amount == 8000 || receipt.Amount == 12000) {
				isUpgradeCard = true
				prevTier = student.Tier
			}
		}
	}

	var caption string
	var btnApprove telego.InlineKeyboardButton
	if strings.ToLower(receipt.Tier) == "audit_essay" {
		caption = fmt.Sprintf(`✍️ <b>Чек на ТЕКСТОВЫЙ АУДИТ ЭССЕ #%d (Order Bump)</b>

👤 Студент: <b>%s</b>
🆔 ID: <code>%d</code>
💰 Сумма: <b>%.0f ₽</b>
📅 Дата: <i>%s</i>

Подтвердите оплату текстового аудита черновика в Google Docs:`,
			receipt.ID,
			studentDisplay,
			receipt.TelegramID,
			receipt.Amount,
			dateStr,
		)
		btnApprove = tu.InlineKeyboardButton("✅ Одобрить аудит (1 900 ₽)").WithCallbackData(fmt.Sprintf("receipt_approve_%d", receipt.ID))
	} else if isUpgradeCard {
		caption = fmt.Sprintf(`🚀 <b>Чек на АПГРЕЙД тарифа #%d</b>

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
		caption = fmt.Sprintf(`🧾 <b>Чек об оплате #%d</b>

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

	// Buttons to toggle tier if wrong tier was detected
	var switchTierButtons []telego.InlineKeyboardButton
	if !isUpgradeCard {
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
	keyboard := tu.InlineKeyboard(rows...)

	// 1. Try sending as Photo if FileID exists
	if receipt.FileID != "" {
		photoMsg := tu.Photo(adminChatID, tu.FileFromID(receipt.FileID)).
			WithCaption(caption).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(keyboard)
		if _, err := s.bot.SendPhoto(photoMsg); err == nil {
			return nil
		} else {
			log.Printf("[Bot Receipt Card] SendPhoto failed for receipt #%d (FileID=%s): %v. Trying SendDocument...", receipt.ID, receipt.FileID, err)
		}

		// 2. Try sending as Document (if uploaded as uncompressed image or PDF)
		docMsg := tu.Document(adminChatID, tu.FileFromID(receipt.FileID)).
			WithCaption(caption).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(keyboard)
		if _, err := s.bot.SendDocument(docMsg); err == nil {
			return nil
		} else {
			log.Printf("[Bot Receipt Card] SendDocument failed for receipt #%d (FileID=%s): %v. Falling back to text message...", receipt.ID, receipt.FileID, err)
		}
	}

	// 3. Fallback: Always guarantee delivering the actionable text card to the curator
	textMsg := tu.Message(adminChatID, caption).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(keyboard)
	_, err := s.bot.SendMessage(textMsg)
	if err != nil {
		log.Printf("[Bot Receipt Card] SendMessage failed for receipt #%d: %v", receipt.ID, err)
	}
	return err
}

// receiptPromoCode keeps the code attached to a discounted receipt even when
// an old session persisted DiscountPct but lost PromoCode.
func receiptPromoCode(sessionPromo string, discountPct int, userPromo string) string {
	if code := strings.ToUpper(strings.TrimSpace(sessionPromo)); code != "" {
		return code
	}
	if discountPct > 0 {
		return strings.ToUpper(strings.TrimSpace(userPromo))
	}
	return ""
}
