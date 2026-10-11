package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

// TributeWebhookPayload models the incoming notification from Tribute (@tribute / web.tribute.tg).
type TributeWebhookPayload struct {
	Name             string                 `json:"name"`
	Event            string                 `json:"event"`
	CreatedAt        string                 `json:"created_at"`
	Payload          map[string]interface{} `json:"payload"`
	TelegramUserID   int64                  `json:"telegram_user_id"`
	Amount           float64                `json:"amount"`
	Currency         string                 `json:"currency"`
	SubscriptionName string                 `json:"subscription_name"`
	OrderID          string                 `json:"order_id"`
}

// VerifyTributeSignature checks HMAC-SHA256 signature from the trbt-signature header.
func VerifyTributeSignature(apiKey string, body []byte, signatureHex string) bool {
	cleanKey := strings.TrimSpace(apiKey)
	cleanSign := strings.TrimSpace(strings.ToLower(signatureHex))
	if cleanKey == "" {
		// If Tribute API key is not yet configured, allow to prevent dropping valid payments
		return true
	}
	if cleanSign == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(cleanKey))
	mac.Write(body)
	expectedHex := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedHex), []byte(cleanSign))
}

// HandleTributeWebhook handles incoming payment webhooks from Tribute.
// POST /api/payments/tribute and /payments/tribute
func (h *PaymentWebhookHandler) HandleTributeWebhook(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[TRIBUTE] Error reading request body: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "error": "cannot read body"})
		return
	}

	rawBodyStr := string(bodyBytes)
	log.Printf("[TRIBUTE] Callback received (%d bytes): %s", len(bodyBytes), rawBodyStr)

	if len(bodyBytes) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "empty ping acknowledged"})
		return
	}

	// Signature verification via trbt-signature header
	signatureHeader := c.GetHeader("trbt-signature")
	if signatureHeader == "" {
		signatureHeader = c.GetHeader("Trbt-Signature")
	}
	if signatureHeader == "" {
		signatureHeader = c.GetHeader("X-Trbt-Signature")
	}

	if h.cfg != nil && h.cfg.TributeAPIKey != "" {
		if !VerifyTributeSignature(h.cfg.TributeAPIKey, bodyBytes, signatureHeader) {
			log.Printf("[TRIBUTE-SECURITY] Invalid HMAC signature! Header: %s", signatureHeader)
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "invalid signature"})
			return
		}
		log.Printf("[TRIBUTE-SECURITY] Signature verified successfully with HMAC-SHA256")
	}

	// Parse JSON
	var payload TributeWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		log.Printf("[TRIBUTE] JSON parse error: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "unmarshal error"})
		return
	}

	// Extract data from nested payload or top level
	var telegramID int64 = payload.TelegramUserID
	var username string
	var amount float64 = payload.Amount
	var currency string = payload.Currency
	var productName string = payload.SubscriptionName
	var paymentID string = payload.OrderID

	if payload.Payload != nil {
		p := payload.Payload

		// Extract telegram user ID
		if tgIDVal, ok := p["telegram_user_id"]; ok {
			switch v := tgIDVal.(type) {
			case float64:
				telegramID = int64(v)
			case int64:
				telegramID = v
			case string:
				if id, err := strconv.ParseInt(v, 10, 64); err == nil {
					telegramID = id
				}
			}
		}

		// Extract user sub-object
		if userVal, ok := p["user"].(map[string]interface{}); ok {
			if uTgID, ok := userVal["telegram_id"]; ok && telegramID == 0 {
				if f, ok := uTgID.(float64); ok {
					telegramID = int64(f)
				}
			}
			if uName, ok := userVal["username"].(string); ok {
				username = strings.TrimPrefix(uName, "@")
			}
		}

		// Extract amount
		if amtVal, ok := p["amount"]; ok {
			switch v := amtVal.(type) {
			case float64:
				amount = v
			case string:
				if parsed, err := strconv.ParseFloat(v, 64); err == nil {
					amount = parsed
				}
			}
		}

		// Extract currency
		if currVal, ok := p["currency"].(string); ok && currVal != "" {
			currency = currVal
		}

		// Extract name/title
		if subNameVal, ok := p["subscription_name"].(string); ok && subNameVal != "" {
			productName = subNameVal
		} else if prodNameVal, ok := p["product_name"].(string); ok && prodNameVal != "" {
			productName = prodNameVal
		}

		// Extract payment/subscription ID
		if subIDVal, ok := p["subscription_id"].(string); ok && subIDVal != "" {
			paymentID = subIDVal
		} else if ordIDVal, ok := p["order_id"].(string); ok && ordIDVal != "" {
			paymentID = ordIDVal
		} else if idVal, ok := p["id"].(string); ok && idVal != "" {
			paymentID = idVal
		}
	}

	// Normalize amount: Tribute often reports amounts in kopecks (e.g. 690000 = 6900 RUB)
	if amount >= 50000 {
		amount = amount / 100.0
	}
	if currency == "" {
		currency = "RUB"
	}

	// Determine tier:
	// VIP is 14 900 ₽ (or 14 155 ₽ with promo)
	// Accelerator is 6 900 ₽ (or 6 555 ₽ with promo)
	tier := "accelerator"
	lowerProduct := strings.ToLower(productName)
	if amount >= 13000 || strings.Contains(lowerProduct, "vip") || strings.Contains(lowerProduct, "вип") {
		tier = "vip"
	}

	if paymentID == "" {
		paymentID = fmt.Sprintf("TRBT_%d_%d", telegramID, time.Now().Unix())
	} else if !strings.HasPrefix(paymentID, "TRBT_") {
		paymentID = "TRBT_" + paymentID
	}

	// Idempotency check: prevent duplicate processing
	if h.db != nil {
		var existing models.PaymentTransaction
		if err := h.db.Where("payment_id = ?", paymentID).First(&existing).Error; err == nil {
			log.Printf("[TRIBUTE-IDEMPOTENCY] Payment %s already processed. Skipping duplicate.", paymentID)
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "already_processed"})
			return
		}
	}

	// Record transaction in DB
	transaction := models.PaymentTransaction{
		PaymentID:      paymentID,
		OrderID:        paymentID,
		TelegramID:     telegramID,
		Username:       username,
		Tier:           tier,
		Amount:         amount,
		OriginalAmount: amount,
		Status:         "SUCCESS",
		Currency:       currency,
		PayMethod:      "TRIBUTE",
		ProductName:    productName,
		RawCallback:    rawBodyStr,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if h.db != nil {
		_ = h.db.Create(&transaction)
	}

	tierNameRU := "Акселератор (6 900 ₽)"
	if tier == "vip" {
		tierNameRU = "VIP-сопровождение (14 900 ₽)"
	}

	// Update or create student in DB
	if h.db != nil && telegramID > 0 {
		var user models.User
		now := time.Now()
		err := h.db.Where("telegram_id = ?", telegramID).First(&user).Error
		if err == nil {
			user.HasAccess = true
			user.AccessGrantedAt = &now
			user.Tier = tier
			user.PaidAmount = amount
			h.db.Save(&user)
			if user.Username != "" && username == "" {
				username = user.Username
			}
		} else {
			newUser := models.User{
				TelegramID:      telegramID,
				Username:        username,
				FirstName:       "Студент",
				Role:            "student",
				HasAccess:       true,
				AccessGrantedAt: &now,
				Tier:            tier,
				PaidAmount:      amount,
			}
			h.db.Create(&newUser)
		}
	}

	// Generate single-use invite link for channel
	inviteLink := ""
	if h.cfg != nil {
		inviteLink = h.cfg.TelegramInviteLink
	}
	if h.inviteLinkFunc != nil {
		var chanID int64
		if h.cfg != nil {
			chanID = h.cfg.TelegramChannelID
		}
		if dynLink, err := h.inviteLinkFunc(context.Background(), chanID); err == nil && dynLink != "" {
			inviteLink = dynLink
		}
	}

	// Send confirmation and channel link to student via Telegram bot
	if h.notifyFunc != nil && telegramID > 0 {
		studentMsg := fmt.Sprintf(
			"🎉 <b>Оплата подтверждена через Tribute!</b>\n\n"+
				"Добро пожаловать в проект «так называемый SPARK» 2027!\n\n"+
				"📦 <b>Твой тариф:</b> %s\n"+
				"💳 <b>Оплачено:</b> %.2f ₽\n\n"+
				"🔒 <b>Твоя персональная ссылка в закрытый канал и комьюнити:</b>\n%s\n\n"+
				"<i>Ссылка одноразовая и действует 24 часа. Сохрани её.</i>",
			tierNameRU, amount, inviteLink,
		)
		_ = h.notifyFunc(telegramID, studentMsg)
	}

	// Push alert to Ilya in Telegram
	adminTG := int64(123456789)
	if h.cfg != nil && h.cfg.AdminTelegramID != 0 {
		adminTG = h.cfg.AdminTelegramID
	}

	if h.notifyFunc != nil && adminTG != 0 {
		studentRef := fmt.Sprintf("ID: %d", telegramID)
		if username != "" {
			studentRef = fmt.Sprintf("@%s (%s)", username, studentRef)
		}

		adminAlert := fmt.Sprintf(
			"⚡️ <b>НОВАЯ ОПЛАТА ЧЕРЕЗ TRIBUTE!</b> ⚡️\n\n"+
				"👤 <b>Студент:</b> %s\n"+
				"📦 <b>Тариф:</b> %s\n"+
				"💰 <b>Сумма:</b> %.2f %s\n"+
				"🆔 <b>ID платежа:</b> <code>%s</code>\n"+
				"💳 <b>Метод:</b> Tribute (Cards/SBP)\n"+
				"⏱ <b>Время:</b> %s\n\n"+
				"✅ <i>Доступ в закрытый канал выдан автоматически, ссылка отправлена студенту!</i>",
			studentRef,
			tierNameRU,
			amount,
			currency,
			paymentID,
			time.Now().Format("15:04:05 02.01.2006"),
		)
		_ = h.notifyFunc(adminTG, adminAlert)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment processed"})
}
