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
)

// HandleReceiptUpload processes receipt screenshot from user with SHA-256 deduplication and fraud prevention.
func (s *BotService) HandleReceiptUpload(msg *telego.Message) bool {
	if s.bot == nil || msg == nil {
		return false
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
		return false
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
		return true
	}

	chosenTier := strings.ToLower(strings.TrimSpace(session.SelectedTier))
	if chosenTier == "" {
		chosenTier = "accelerator"
	}

	amount := 6900.0
	switch chosenTier {
	case "vip":
		amount = 14900.0
		chosenTier = "vip"
	default:
		amount = 6900.0
		chosenTier = "accelerator"
	}

	if session.DiscountPct > 0 {
		amount = amount * (1.0 - float64(session.DiscountPct)/100.0)
	}

	username := msg.From.Username
	if username == "" {
		username = msg.From.FirstName
	}
	username = strings.TrimPrefix(username, "@")

	receipt := models.PaymentReceipt{
		FileHash:     fileHash,
		FileUniqueID: fileUniqueID,
		TelegramID:   msg.Chat.ID,
		Username:     username,
		FileID:       fileID,
		Tier:         chosenTier,
		Amount:       amount,
		Status:       "pending",
		CreatedAt:    time.Now(),
	}

	if err := s.db.Create(&receipt).Error; err != nil {
		reply := "❌ Ошибка при сохранении чека в системе. Пожалуйста, напишите так называемому Илю (@ilyan_vas)."
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return true
	}

	session.State = StateDefault

	// Send confirmation to student
	reply := fmt.Sprintf(`🧾 <b>Чек об оплате успешно принят!</b>

Тариф: <b>%s</b>
Сумма: <b>%.0f ₽</b>

Так называемый Иль проверит поступление средств и сразу пришлёт персональную ссылку-приглашение в закрытый канал потока.`, strings.ToUpper(chosenTier), amount)

	btnBlog := tu.InlineKeyboardButton("🇺🇸 Пока можно почитать блог").WithURL("https://t.me/so_called_spark")
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnBlog))
	m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
	_, _ = s.bot.SendMessage(m)

	// Send actionable review card to Curator (Admin)
	if s.cfg.AdminTelegramID != 0 {
		_ = s.SendAdminReceiptCard(s.cfg.AdminTelegramID, &receipt)
	}

	return true
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

	caption := fmt.Sprintf(`🧾 <b>Чек об оплате #%d</b>

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

	btnApprove := tu.InlineKeyboardButton("✅ Одобрить и выдать доступ").WithCallbackData(fmt.Sprintf("receipt_approve_%d", receipt.ID))
	btnReject := tu.InlineKeyboardButton("❌ Отклонить").WithCallbackData(fmt.Sprintf("receipt_reject_%d", receipt.ID))

	rows := [][]telego.InlineKeyboardButton{
		{btnApprove, btnReject},
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
