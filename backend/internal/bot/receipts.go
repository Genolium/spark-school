package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// SendAdminReceiptCard sends the review notification with photo and 1-click action buttons.
func (s *BotService) SendAdminReceiptCard(adminID int64, receipt *models.PaymentReceipt) error {
	if s.bot == nil || adminID == 0 {
		return nil
	}

	adminChatID := tu.ID(adminID)
	caption := fmt.Sprintf(`🔔 <b>Новый чек на оплату проекта «так называемый SPARK»!</b>

Студент: <b>@%s</b>
ID: <code>%d</code>
Тариф: <b>%s</b>
Сумма: <b>%.0f ₽</b>
Чек ID: #%d

Подтвердите поступление средств и выдачу доступа:`, receipt.Username, receipt.TelegramID, strings.ToUpper(receipt.Tier), receipt.Amount, receipt.ID)

	btnApprove := tu.InlineKeyboardButton("✅ Одобрить и выдать доступ").WithCallbackData(fmt.Sprintf("receipt_approve_%d", receipt.ID))
	btnReject := tu.InlineKeyboardButton("❌ Отклонить").WithCallbackData(fmt.Sprintf("receipt_reject_%d", receipt.ID))
	keyboard := tu.InlineKeyboard(tu.InlineKeyboardRow(btnApprove, btnReject))

	if receipt.FileID != "" {
		photoMsg := tu.Photo(adminChatID, tu.FileFromID(receipt.FileID)).WithCaption(caption).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
		_, err := s.bot.SendPhoto(photoMsg)
		return err
	}

	msg := tu.Message(adminChatID, caption).WithParseMode(telego.ModeHTML).WithReplyMarkup(keyboard)
	_, err := s.bot.SendMessage(msg)
	return err
}
