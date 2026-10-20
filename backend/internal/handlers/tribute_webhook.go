package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

// Minimum allowable payment amount to prevent underpayment/tampering attacks.
// 3 500 ₽ safely covers Basic with 5% promo (3 790.50 ₽).
const MinCoursePaymentThreshold = 3500.00
const MinVipPaymentThreshold = 13000.00

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
// It is strictly fail-closed: requires non-empty secret key and non-empty signature.
func VerifyTributeSignature(apiKey string, body []byte, signatureHex string) bool {
	cleanKey := strings.TrimSpace(apiKey)
	cleanSign := strings.TrimSpace(strings.ToLower(signatureHex))

	// Strict fail-closed: never permit verification without configured key or signature
	if cleanKey == "" || cleanSign == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(cleanKey))
	mac.Write(body)
	expectedHex := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedHex), []byte(cleanSign))
}

// HandleTributeWebhook handles incoming payment webhooks from Tribute with multi-layer defense.
// POST /api/payments/tribute and /payments/tribute
func (h *PaymentWebhookHandler) HandleTributeWebhook(c *gin.Context) {
	// 1. Read raw body (enforces 2MB maximum limit from middleware)
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("[TRIBUTE-SECURITY] Error reading request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "cannot read request body"})
		return
	}

	rawBodyStr := string(bodyBytes)

	// If empty ping / test check from reverse proxy
	if len(bodyBytes) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "empty ping acknowledged"})
		return
	}

	// 2. Strict Authentication & HMAC-SHA256 Signature Verification
	signatureHeader := c.GetHeader("trbt-signature")
	if signatureHeader == "" {
		signatureHeader = c.GetHeader("Trbt-Signature")
	}
	if signatureHeader == "" {
		signatureHeader = c.GetHeader("X-Trbt-Signature")
	}

	apiKey := ""
	if h.cfg != nil {
		apiKey = strings.TrimSpace(h.cfg.TributeAPIKey)
	}

	isTestMode := gin.Mode() == gin.TestMode || os.Getenv("ENV") == "test"

	if apiKey == "" {
		if !isTestMode {
			log.Printf("[CRITICAL-SECURITY] Tribute webhook invoked but TRIBUTE_API_KEY is not configured! Rejecting request to prevent unauthorized access grant.")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "payment gateway signature verification is not configured",
			})
			return
		}
	} else {
		if signatureHeader == "" {
			log.Printf("[TRIBUTE-SECURITY] Missing required signature header (trbt-signature)")
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "missing trbt-signature header"})
			return
		}

		if !VerifyTributeSignature(apiKey, bodyBytes, signatureHeader) {
			log.Printf("[TRIBUTE-SECURITY] Signature verification failed! Header: %s, BodyLen: %d", signatureHeader, len(bodyBytes))
			c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "invalid signature"})
			return
		}
		log.Printf("[TRIBUTE-SECURITY] Signature verified successfully with HMAC-SHA256")
	}

	// 3. Parse JSON Payload
	var payload TributeWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		log.Printf("[TRIBUTE-SECURITY] JSON unmarshal error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid json payload"})
		return
	}

	// 4. Extract and normalize fields
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

	// Sanitize text inputs for logging and Telegram HTML formatting
	username = strings.TrimSpace(html.EscapeString(username))
	productName = strings.TrimSpace(html.EscapeString(productName))
	currency = strings.ToUpper(strings.TrimSpace(html.EscapeString(currency)))
	if currency == "" {
		currency = "RUB"
	}

	// Normalize amount: Tribute reports amounts in minor units (kopecks/cents) when >= 50000
	if amount >= 50000 {
		amount = amount / 100.0
	}

	// 5. Deterministic Replay-Resistant Idempotency Key
	// If paymentID is missing, derive a deterministic hash from payload body to prevent replay attacks
	if paymentID == "" {
		bodyHash := ComputeHash(bodyBytes)
		paymentID = fmt.Sprintf("TRBT_HASH_%s", bodyHash[:20])
	} else if !strings.HasPrefix(paymentID, "TRBT_") {
		paymentID = "TRBT_" + paymentID
	}

	// 6. Anti-Underpayment & Amount Tampering Protection
	// Determine Tier strictly by settled amount and title
	tier := "accelerator"
	lowerProduct := strings.ToLower(productName)
	if amount >= MinVipPaymentThreshold || strings.Contains(lowerProduct, "vip") || strings.Contains(lowerProduct, "вип") {
		tier = "vip"
	} else if amount < 6000 && (strings.Contains(lowerProduct, "basic") || strings.Contains(lowerProduct, "базов")) {
		tier = "basic"
	}

	// Fraud check: if amount is below minimum course price (e.g. attacker sends 1 ₽ or 50 ₽),
	// DO NOT grant access. Flag transaction for manual investigation.
	if amount < MinCoursePaymentThreshold {
		log.Printf("[SECURITY-ALERT] Tribute payment %s underpayment detected: %.2f %s. Minimum required: %.2f. Flagging without access grant.",
			paymentID, amount, currency, MinCoursePaymentThreshold)

		// Record suspicious transaction in DB for audit
		if h.db != nil {
			flaggedTx := models.PaymentTransaction{
				PaymentID:      paymentID,
				OrderID:        paymentID,
				TelegramID:     telegramID,
				Username:       username,
				Tier:           tier,
				Amount:         amount,
				OriginalAmount: amount,
				Status:         "FLAGGED_UNDERPAID",
				Currency:       currency,
				PayMethod:      "TRIBUTE",
				ProductName:    productName,
				RawCallback:    rawBodyStr,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			_ = h.db.Create(&flaggedTx)
		}

		// Alert Ilya about suspicious payment
		adminTG := int64(123456789)
		if h.cfg != nil && h.cfg.AdminTelegramID != 0 {
			adminTG = h.cfg.AdminTelegramID
		}
		if h.notifyFunc != nil && adminTG != 0 {
			alertMsg := fmt.Sprintf(
				"⚠️ <b>ВНИМАНИЕ: ПОДОЗРИТЕЛЬНАЯ ОПЛАТА TRIBUTE!</b> ⚠️\n\n"+
					"👤 <b>Студент:</b> ID %d (@%s)\n"+
					"💰 <b>Сумма:</b> %.2f %s <i>(МЕНЬШЕ МИНИМУМА 6 000 ₽)</i>\n"+
					"📦 <b>Товар:</b> %s\n"+
					"🆔 <b>ID платежа:</b> <code>%s</code>\n\n"+
					"⛔️ <b>Доступ НЕ выдан.</b> Проверьте транзакцию в кабинете Tribute вручную!",
				telegramID, username, amount, currency, productName, paymentID,
			)
			_ = h.notifyFunc(adminTG, alertMsg)
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "payment_flagged_underpaid"})
		return
	}

	// 7. Atomic Concurrency Control & Idempotency Execution
	var isAlreadyProcessed bool
	var user models.User

	if h.db != nil {
		err = h.db.Transaction(func(tx *gorm.DB) error {
			// Idempotency check inside transaction
			var existing models.PaymentTransaction
			if err := tx.Where("payment_id = ?", paymentID).First(&existing).Error; err == nil {
				isAlreadyProcessed = true
				return nil
			}

			// Record PaymentTransaction
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

			if err := tx.Create(&transaction).Error; err != nil {
				// Detect unique index collision (parallel race condition)
				if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
					isAlreadyProcessed = true
					return nil
				}
				return err
			}

			// Update or Create Student Record
			if telegramID > 0 {
				now := time.Now()
				if err := tx.Where("telegram_id = ?", telegramID).First(&user).Error; err == nil {
					user.HasAccess = true
					user.AccessGrantedAt = &now
					user.Tier = tier
					user.PaidAmount = amount
					if username != "" && user.Username == "" {
						user.Username = username
					}
					if err := tx.Save(&user).Error; err != nil {
						return err
					}
				} else {
					user = models.User{
						TelegramID:      telegramID,
						Username:        username,
						FirstName:       "Студент",
						Role:            "student",
						HasAccess:       true,
						AccessGrantedAt: &now,
						Tier:            tier,
						PaidAmount:      amount,
					}
					if err := tx.Create(&user).Error; err != nil {
						return err
					}
				}

				// Process Affiliate Commission atomically
				if h.affiliateService != nil && user.ID != 0 {
					_ = h.affiliateService.ProcessCoursePurchaseTx(tx, user.ID, amount)
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("[TRIBUTE-ERROR] DB transaction error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "database error"})
			return
		}

		if isAlreadyProcessed {
			log.Printf("[TRIBUTE-IDEMPOTENCY] Payment %s already processed. Skipping duplicate callback safely.", paymentID)
			c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "already_processed"})
			return
		}
	}

	// 8. Generate Single-Use Telegram Channel Invite Link (Only on successful settlement)
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

	tierNameRU := "Акселератор"
	if tier == "vip" {
		tierNameRU = "VIP-сопровождение"
	}

	// 9. Deliver Channel Invite Link to Student
	if h.notifyFunc != nil && telegramID > 0 {
		sparkle := "<tg-emoji emoji-id=\"5884407720131436294\">✨</tg-emoji>"
		sun := "<tg-emoji emoji-id=\"5883980366590517942\">🔆</tg-emoji>"
		arrow := "<tg-emoji emoji-id=\"5884424573583106260\">➡️</tg-emoji>"

		studentMsg := fmt.Sprintf(
			"%s Спасибо за покупку! <b>Добро пожаловать на моё обучение!</b>\n\n"+
				"%s Твоя ссылка для входа в закрытый канал потока:\n%s\n\n"+
				"%s <i>Ссылка является персональной и одноразовой.</i>",
			sparkle, sun, inviteLink, arrow,
		)
		_ = h.notifyFunc(telegramID, studentMsg)
	}

	// 10. Instant Push Notification to Ilya (Admin)
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
