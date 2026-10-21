package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAdminLogin_SuccessAndFailure(t *testing.T) {
	cfg := &config.Config{
		AdminUsername:   "admin",
		AdminPassword:   "SparkAdmin2026!",
		AdminTelegramID: 123456789,
		JWTSecret:       "test-secret-key-32-chars-spark-2027",
	}

	handler := NewAdminHandler(nil, cfg, nil, nil)

	r := gin.New()
	r.POST("/admin/login", handler.Login)

	// 1. Success case
	bodySuccess, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "SparkAdmin2026!",
	})
	req := httptest.NewRequest("POST", "/admin/login", bytes.NewReader(bodySuccess))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for valid login, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["token"] == nil || res["token"] == "" {
		t.Error("expected token in response")
	}

	// 2. Invalid password case
	bodyFail, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "WrongPassword!",
	})
	reqFail := httptest.NewRequest("POST", "/admin/login", bytes.NewReader(bodyFail))
	reqFail.Header.Set("Content-Type", "application/json")
	wFail := httptest.NewRecorder()
	r.ServeHTTP(wFail, reqFail)

	if wFail.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for wrong password, got %d", wFail.Code)
	}

	// 3. Missing payload case
	reqEmpty := httptest.NewRequest("POST", "/admin/login", bytes.NewReader([]byte("{}")))
	reqEmpty.Header.Set("Content-Type", "application/json")
	wEmpty := httptest.NewRecorder()
	r.ServeHTTP(wEmpty, reqEmpty)

	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for empty body, got %d", wEmpty.Code)
	}
}

func TestAffiliateSubmitPayout_Validation(t *testing.T) {
	cfg := &config.Config{Domain: "sparkprep.ru"}
	handler := NewAffiliateHandler(nil, cfg, nil)

	r := gin.New()
	r.POST("/affiliate/payout", func(c *gin.Context) {
		// Mock authenticated user
		c.Set("user_id", uint(42))
		handler.SubmitPayout(c)
	})

	// Test 1: Empty body should fail binding with 400
	req1 := httptest.NewRequest("POST", "/affiliate/payout", bytes.NewReader([]byte("{}")))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", w1.Code)
	}

	// Test 2: Details too short (< 4 chars) should fail with 400
	bodyShort, _ := json.Marshal(map[string]interface{}{
		"amount":  2000,
		"details": "12",
	})
	req2 := httptest.NewRequest("POST", "/affiliate/payout", bytes.NewReader(bodyShort))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short details, got %d", w2.Code)
	}

	// Test 3: Invalid amount (<= 0) should fail with 400
	bodyZeroAmount, _ := json.Marshal(map[string]interface{}{
		"amount":  0,
		"details": "+79991234567",
	})
	req3 := httptest.NewRequest("POST", "/affiliate/payout", bytes.NewReader(bodyZeroAmount))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero amount, got %d", w3.Code)
	}
}

func TestTelegramWidgetLogin_Validation(t *testing.T) {
	cfg := &config.Config{
		TelegramBotToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
		JWTSecret:        "test-secret-key-32-chars-spark-2027",
	}
	handler := NewAuthHandler(nil, cfg, nil)

	r := gin.New()
	r.POST("/auth/telegram-widget", handler.TelegramWidgetLogin)

	// 1. Invalid JSON body
	reqEmpty := httptest.NewRequest("POST", "/auth/telegram-widget", bytes.NewReader([]byte("invalid json")))
	reqEmpty.Header.Set("Content-Type", "application/json")
	wEmpty := httptest.NewRecorder()
	r.ServeHTTP(wEmpty, reqEmpty)
	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", wEmpty.Code)
	}

	// 2. Invalid signature / hash
	payloadBadHash, _ := json.Marshal(map[string]string{
		"id":         "123456789",
		"first_name": "Test",
		"hash":       "fakehash",
	})
	reqBadHash := httptest.NewRequest("POST", "/auth/telegram-widget", bytes.NewReader(payloadBadHash))
	reqBadHash.Header.Set("Content-Type", "application/json")
	wBadHash := httptest.NewRecorder()
	r.ServeHTTP(wBadHash, reqBadHash)
	if wBadHash.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", wBadHash.Code)
	}

	// 3. Missing/invalid Telegram ID with test bypass
	payloadBadID, _ := json.Marshal(map[string]string{
		"id":   "not-a-number",
		"hash": "test",
	})
	reqBadID := httptest.NewRequest("POST", "/auth/telegram-widget", bytes.NewReader(payloadBadID))
	reqBadID.Header.Set("Content-Type", "application/json")
	wBadID := httptest.NewRecorder()
	r.ServeHTTP(wBadID, reqBadID)
	if wBadID.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid Telegram ID, got %d", wBadID.Code)
	}
}

func TestCalculateCapacityAndSpots(t *testing.T) {
	testCases := []struct {
		activeStudents int64
		wantCapacity   int64
		wantSpotsLeft  int64
	}{
		{activeStudents: 0, wantCapacity: 25, wantSpotsLeft: 25},
		{activeStudents: 1, wantCapacity: 25, wantSpotsLeft: 24},
		{activeStudents: 24, wantCapacity: 25, wantSpotsLeft: 1},
		// At 25, reaches capacity, doubles to 50
		{activeStudents: 25, wantCapacity: 50, wantSpotsLeft: 25},
		{activeStudents: 26, wantCapacity: 50, wantSpotsLeft: 24},
		{activeStudents: 49, wantCapacity: 50, wantSpotsLeft: 1},
		// At 50, reaches capacity, doubles to 100
		{activeStudents: 50, wantCapacity: 100, wantSpotsLeft: 50},
		{activeStudents: 55, wantCapacity: 100, wantSpotsLeft: 45},
		{activeStudents: 99, wantCapacity: 100, wantSpotsLeft: 1},
		// At 100, reaches capacity, doubles to 200
		{activeStudents: 100, wantCapacity: 200, wantSpotsLeft: 100},
	}

	for _, tc := range testCases {
		gotCap, gotSpots := CalculateCapacityAndSpots(tc.activeStudents)
		if gotCap != tc.wantCapacity || gotSpots != tc.wantSpotsLeft {
			t.Errorf("CalculateCapacityAndSpots(%d): got capacity=%d, spots=%d; want capacity=%d, spots=%d",
				tc.activeStudents, gotCap, gotSpots, tc.wantCapacity, tc.wantSpotsLeft)
		}
	}
}

func TestGetPlacesStats_Endpoint(t *testing.T) {
	handler := NewStatsHandler(nil)
	r := gin.New()
	r.GET("/api/v1/stats/places", handler.GetPlacesStats)

	req := httptest.NewRequest("GET", "/api/v1/stats/places", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res PlacesStatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if res.TotalCapacity != 25 || res.SpotsLeft != 25 {
		t.Errorf("expected 25/25, got cap=%d, spots=%d", res.TotalCapacity, res.SpotsLeft)
	}
}

func TestReceiptVerify_Validation(t *testing.T) {
	handler := NewReceiptHandler(nil)
	r := gin.New()
	r.POST("/api/v1/receipts/verify", handler.VerifyOrSubmit)

	// Missing both file_hash and file_id
	payload, _ := json.Marshal(map[string]interface{}{
		"telegram_id": 12345,
	})
	req := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing hash/file_id, got %d", w.Code)
	}

	// Valid hash without db returns 200
	payloadValid, _ := json.Marshal(map[string]interface{}{
		"file_hash":   "abc123def456",
		"telegram_id": 12345,
	})
	reqValid := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(payloadValid))
	reqValid.Header.Set("Content-Type", "application/json")
	wValid := httptest.NewRecorder()
	r.ServeHTTP(wValid, reqValid)

	if wValid.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid hash without db, got %d", wValid.Code)
	}
}

func TestPromoValidate_TierPricing(t *testing.T) {
	// Test helper GetTierPrice (only 2 active tiers: accelerator and vip)
	if price := GetTierPrice("accelerator", 0); price != 6900.00 {
		t.Errorf("expected accelerator price 6900, got %f", price)
	}
	if price := GetTierPrice("акселератор", 0); price != 6900.00 {
		t.Errorf("expected акселератор price 6900, got %f", price)
	}
	if price := GetTierPrice("vip", 0); price != 14900.00 {
		t.Errorf("expected vip price 14900, got %f", price)
	}
	if price := GetTierPrice("unknown", 0); price != 6900.00 {
		t.Errorf("expected default price 6900, got %f", price)
	}
	if price := GetTierPrice("", 5000); price != 5000.00 {
		t.Errorf("expected custom price 5000, got %f", price)
	}
}

func TestSubmitReceipt_RejectsDuplicateFileUniqueID(t *testing.T) {
	receipts := make(map[string]*models.PaymentReceipt)
	handler := NewReceiptHandler(nil).WithCustomStore(
		func(fileHash, fileUniqueID string) (*models.PaymentReceipt, bool) {
			for _, r := range receipts {
				if (fileHash != "" && r.FileHash == fileHash) || (fileUniqueID != "" && r.FileUniqueID == fileUniqueID) {
					return r, true
				}
			}
			return nil, false
		},
		func(receipt *models.PaymentReceipt) error {
			receipt.ID = uint(len(receipts) + 1)
			receipts[receipt.FileUniqueID] = receipt
			return nil
		},
	)

	r := gin.New()
	r.POST("/api/v1/receipts/verify", handler.SubmitReceipt)

	// 1. Initial valid submission with file_unique_id
	payload1, _ := json.Marshal(map[string]interface{}{
		"file_unique_id": "AQAD_test_unique_id_1001",
		"telegram_id":    123456,
		"username":       "student_one",
		"amount":         6900.0,
		"tier":           "accelerator",
	})
	req1 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(payload1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("expected status 201 for first receipt submission, got %d: %s", w1.Code, w1.Body.String())
	}

	var res1 map[string]interface{}
	if err := json.Unmarshal(w1.Body.Bytes(), &res1); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res1["duplicate"] != false || res1["status"] != "recorded" {
		t.Errorf("unexpected response on first submission: %v", res1)
	}

	// 2. Duplicate submission with SAME file_unique_id
	payload2, _ := json.Marshal(map[string]interface{}{
		"file_unique_id": "AQAD_test_unique_id_1001",
		"telegram_id":    999999,
		"username":       "fraudulent_user",
		"amount":         6900.0,
		"tier":           "accelerator",
	})
	req2 := httptest.NewRequest("POST", "/api/v1/receipts/verify", bytes.NewReader(payload2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected status 409 Conflict for duplicate file_unique_id, got %d: %s", w2.Code, w2.Body.String())
	}

	var res2 map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &res2); err != nil {
		t.Fatalf("failed to decode duplicate response: %v", err)
	}
	if res2["duplicate"] != true {
		t.Errorf("expected duplicate=true, got %v", res2["duplicate"])
	}
}

func TestAdminToggleAccess_DynamicInviteLinkAndRebranding(t *testing.T) {
	cfg := &config.Config{
		TelegramInviteLink: "https://t.me/+static_fallback",
		TelegramChannelID:  -1001234567890,
	}

	var notifiedID int64
	var notifiedMsg string
	notifyMock := func(telegramID int64, message string) error {
		notifiedID = telegramID
		notifiedMsg = message
		return nil
	}

	dynamicGeneratorCalled := false
	dynamicLinkMock := func(ctx context.Context, channelID int64) (string, error) {
		dynamicGeneratorCalled = true
		if channelID != -1001234567890 {
			t.Errorf("expected channelID -1001234567890, got %d", channelID)
		}
		return "https://t.me/+one_time_link_dynamic", nil
	}

	handler := NewAdminHandler(nil, cfg, nil, notifyMock, dynamicLinkMock)

	r := gin.New()
	r.POST("/admin/users/:id/access", handler.ToggleAccess)

	payload, _ := json.Marshal(map[string]bool{"has_access": true})
	req := httptest.NewRequest("POST", "/admin/users/42/access", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if !dynamicGeneratorCalled {
		t.Error("expected dynamic invite link generator to be called")
	}

	if notifiedID == 0 {
		t.Error("expected notification to be sent to user")
	}

	if !strings.Contains(notifiedMsg, "https://t.me/+one_time_link_dynamic") {
		t.Errorf("expected dynamic link in notification message, got %s", notifiedMsg)
	}

	if !strings.Contains(notifiedMsg, "так называемый SPARK") {
		t.Errorf("expected rebranded name in notification message, got %s", notifiedMsg)
	}

	if strings.Contains(notifiedMsg, "SPARK Prep 2027") {
		t.Errorf("notification contains obsolete branding 'SPARK Prep 2027': %s", notifiedMsg)
	}
}

func TestAdminToggleAccess_WithPromoCode_Mock(t *testing.T) {
	cfg := &config.Config{
		TelegramInviteLink: "https://t.me/+static_fallback",
	}
	handler := NewAdminHandler(nil, cfg, nil, nil)

	r := gin.New()
	r.POST("/admin/users/:id/access", handler.ToggleAccess)

	payload, _ := json.Marshal(map[string]interface{}{
		"has_access": true,
		"promo_code": "spark5",
	})
	req := httptest.NewRequest("POST", "/admin/users/99/access", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["has_access"] != true {
		t.Errorf("expected has_access=true, got %v", res["has_access"])
	}
	if res["promo_code"] != "SPARK5" {
		t.Errorf("expected promo_code=SPARK5, got %v", res["promo_code"])
	}
}

func TestTributeSignatureVerification(t *testing.T) {
	apiKey := "test_tribute_secret_key"
	body := []byte(`{"name":"new_subscription","payload":{"amount":6900,"telegram_user_id":12345678}}`)

	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	if !VerifyTributeSignature(apiKey, body, validSig) {
		t.Error("expected valid signature to pass verification")
	}

	if VerifyTributeSignature(apiKey, body, "invalid_signature_hex") {
		t.Error("expected invalid signature to fail verification")
	}

	// Strictly fail-closed: empty API key or empty signature must fail
	if VerifyTributeSignature("", body, "any_signature") {
		t.Error("expected empty API key to fail verification (fail-closed)")
	}
	if VerifyTributeSignature(apiKey, body, "") {
		t.Error("expected empty signature to fail verification (fail-closed)")
	}
}

func TestHandleTributeWebhook(t *testing.T) {
	apiKey := "test_tribute_secret_key"
	cfg := &config.Config{
		TributeAPIKey:   apiKey,
		AdminTelegramID: 999888,
	}

	db := setupHandlerDB(t)
	var notifiedStudentMsg string
	notifyFunc := func(telegramID int64, message string) error {
		notifiedStudentMsg = message
		return nil
	}

	handler := NewPaymentWebhookHandler(db, cfg, nil, notifyFunc)

	r := gin.New()
	r.POST("/api/payments/tribute", handler.HandleTributeWebhook)

	payload := map[string]interface{}{
		"name": "new_subscription",
		"payload": map[string]interface{}{
			"subscription_id":   "sub_test_123",
			"subscription_name": "Акселератор",
			"amount":            6900,
			"currency":          "rub",
			"telegram_user_id":  777666,
			"user": map[string]interface{}{
				"username": "tribute_student",
			},
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(bodyBytes)
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/api/payments/tribute", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trbt-signature", sig)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var tx models.PaymentTransaction
	if err := db.Where("payment_id = ?", "TRBT_sub_test_123").First(&tx).Error; err != nil {
		t.Fatalf("expected payment transaction: %v", err)
	}
	if tx.Amount != 6900 || tx.Status != "SUCCESS" || tx.Tier != "accelerator" {
		t.Fatalf("unexpected transaction: %+v", tx)
	}
	var user models.User
	if err := db.Where("telegram_id = ?", 777666).First(&user).Error; err != nil {
		t.Fatalf("expected user to be created: %v", err)
	}
	if !user.HasAccess || user.Tier != "accelerator" || user.PaidAmount != 6900 {
		t.Fatalf("unexpected user rights: %+v", user)
	}
	var outboxCount int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'payment_notice'").Count(&outboxCount)
	if outboxCount != 2 {
		t.Fatalf("expected user and admin notices in outbox, got %d", outboxCount)
	}

	// A repeated provider delivery must not duplicate money or rights.
	req2 := httptest.NewRequest("POST", "/api/payments/tribute", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("trbt-signature", sig)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for replay, got %d", w2.Code)
	}
	var txCount int64
	db.Model(&models.PaymentTransaction{}).Where("telegram_id = ?", 777666).Count(&txCount)
	if txCount != 1 {
		t.Fatalf("expected exactly one transaction after replay, got %d", txCount)
	}
	var reloaded models.User
	db.Where("telegram_id = ?", 777666).First(&reloaded)
	if reloaded.PaidAmount != 6900 {
		t.Fatalf("replay changed paid amount: %f", reloaded.PaidAmount)
	}
	_ = notifiedStudentMsg
}

func TestHandleTributeWebhook_Security_MissingSignature(t *testing.T) {
	cfg := &config.Config{
		TributeAPIKey: "secret_key_123",
	}
	handler := NewPaymentWebhookHandler(nil, cfg, nil, nil)
	r := gin.New()
	r.POST("/api/payments/tribute", handler.HandleTributeWebhook)

	body := []byte(`{"payload":{"amount":6900,"telegram_user_id":1234}}`)
	req := httptest.NewRequest("POST", "/api/payments/tribute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No signature header

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing signature header, got %d", w.Code)
	}
}

func TestHandleTributeWebhook_Security_InvalidSignature(t *testing.T) {
	cfg := &config.Config{
		TributeAPIKey: "secret_key_123",
	}
	handler := NewPaymentWebhookHandler(nil, cfg, nil, nil)
	r := gin.New()
	r.POST("/api/payments/tribute", handler.HandleTributeWebhook)

	body := []byte(`{"payload":{"amount":6900,"telegram_user_id":1234}}`)
	req := httptest.NewRequest("POST", "/api/payments/tribute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trbt-signature", "deadbeefbadsignature1234567890")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid signature, got %d", w.Code)
	}
}

func TestHandleTributeWebhook_Security_UnderpaymentFlagged(t *testing.T) {
	apiKey := "test_tribute_secret_key"
	cfg := &config.Config{
		TributeAPIKey:   apiKey,
		AdminTelegramID: 999888,
	}

	var studentNotified bool
	var adminAlertReceived bool
	notifyFunc := func(telegramID int64, message string) error {
		if telegramID == 999888 {
			adminAlertReceived = true
			if !strings.Contains(message, "ПОДОЗРИТЕЛЬНАЯ ОПЛАТА") {
				t.Errorf("expected admin alert to contain suspicion warning, got: %s", message)
			}
		} else {
			studentNotified = true
		}
		return nil
	}

	handler := NewPaymentWebhookHandler(nil, cfg, nil, notifyFunc)
	r := gin.New()
	r.POST("/api/payments/tribute", handler.HandleTributeWebhook)

	// Attacker tries to pay 100 RUB instead of 6900 RUB
	payload := map[string]interface{}{
		"name": "new_subscription",
		"payload": map[string]interface{}{
			"subscription_id":  "sub_underpaid_1",
			"amount":           100.0, // 100 RUB
			"currency":         "rub",
			"telegram_user_id": 999999,
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(bodyBytes)
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("POST", "/api/payments/tribute", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trbt-signature", sig)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK acknowledgment, got %d", w.Code)
	}

	if studentNotified {
		t.Error("CRITICAL SECURITY VULNERABILITY: student received access for an underpaid order (100 RUB)!")
	}

	if !adminAlertReceived {
		t.Error("expected admin to be notified of suspicious underpayment attempt")
	}
}

func TestYooKassaHandler_Webhook_Success(t *testing.T) {
	cfg := &config.Config{
		YooKassaShopID:  "1491540",
		AdminTelegramID: 888777,
	}

	db := setupHandlerDB(t)
	handler := NewYooKassaHandler(db, cfg, nil, nil, nil)

	r := gin.New()
	r.POST("/api/payments/yookassa", handler.HandleWebhook)

	payload := map[string]interface{}{
		"type":  "notification",
		"event": "payment.succeeded",
		"object": map[string]interface{}{
			"id":     "22d6d5eb-000f-5000-8000-18db351245c7",
			"status": "succeeded",
			"paid":   true,
			"amount": map[string]interface{}{
				"value":    "6900.00",
				"currency": "RUB",
			},
			"description": "Оплата участия в проекте «так называемый SPARK» (Тариф Акселератор)",
			"metadata": map[string]string{
				"telegram_id": "555444",
				"username":    "yookassa_buyer",
				"tier":        "accelerator",
			},
			"payment_method": map[string]string{
				"type": "bank_card",
			},
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/payments/yookassa", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var tx models.PaymentTransaction
	if err := db.Where("payment_id = ?", "22d6d5eb-000f-5000-8000-18db351245c7").First(&tx).Error; err != nil {
		t.Fatalf("expected transaction: %v", err)
	}
	if tx.Amount != 6900 || tx.Status != "SUCCESS" {
		t.Fatalf("unexpected transaction: %+v", tx)
	}
	var user models.User
	if err := db.Where("telegram_id = ?", 555444).First(&user).Error; err != nil {
		t.Fatalf("expected user: %v", err)
	}
	if !user.HasAccess || user.PaidAmount != 6900 {
		t.Fatalf("unexpected rights: %+v", user)
	}
	var outboxCount int64
	db.Model(&models.OutboxEvent{}).Where("event_type = 'payment_notice'").Count(&outboxCount)
	if outboxCount != 2 {
		t.Fatalf("expected two outbox notices, got %d", outboxCount)
	}
}

func TestYooKassaHandler_Webhook_Underpayment(t *testing.T) {
	cfg := &config.Config{
		YooKassaShopID:  "1491540",
		AdminTelegramID: 888777,
	}

	var studentNotified bool
	var adminAlertReceived bool
	notifyFunc := func(telegramID int64, message string) error {
		if telegramID == 888777 {
			adminAlertReceived = true
		} else {
			studentNotified = true
		}
		return nil
	}

	handler := NewYooKassaHandler(nil, cfg, nil, nil, notifyFunc)

	r := gin.New()
	r.POST("/api/payments/yookassa", handler.HandleWebhook)

	// Underpayment of 10 RUB
	payload := map[string]interface{}{
		"type":  "notification",
		"event": "payment.succeeded",
		"object": map[string]interface{}{
			"id":     "underpaid_yk_1",
			"status": "succeeded",
			"paid":   true,
			"amount": map[string]interface{}{
				"value":    "10.00",
				"currency": "RUB",
			},
			"metadata": map[string]string{
				"telegram_id": "555444",
				"tier":        "accelerator",
			},
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/payments/yookassa", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK acknowledgment, got %d", w.Code)
	}

	if studentNotified {
		t.Error("CRITICAL: Student received access for underpaid YooKassa transaction!")
	}

	if !adminAlertReceived {
		t.Error("Expected admin alert for underpaid YooKassa transaction")
	}
}

func TestYooKassaHandler_Webhook_Production_FailClosedWithoutKey(t *testing.T) {
	// In production, an unconfigured secret key MUST reject incoming webhooks with 503
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		YooKassaShopID:    "1491540",
		YooKassaSecretKey: "", // Missing key
	}

	handler := NewYooKassaHandler(nil, cfg, nil, nil, nil)
	r := gin.New()
	r.POST("/api/payments/yookassa", handler.HandleWebhook)

	payload := map[string]interface{}{
		"type":  "notification",
		"event": "payment.succeeded",
		"object": map[string]interface{}{
			"id":     "fraud_attempt_1",
			"status": "succeeded",
			"paid":   true,
			"amount": map[string]interface{}{
				"value":    "6900.00",
				"currency": "RUB",
			},
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/payments/yookassa", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable when secret key is unset in production, got %d", w.Code)
	}
}

func TestYooKassaHandler_CreateCheckout_InputSanitization(t *testing.T) {
	cfg := &config.Config{
		YooKassaShopID:    "1491540",
		YooKassaSecretKey: "test_secret_key",
		Domain:            "so-called-spark.ru",
	}

	handler := NewYooKassaHandler(nil, cfg, nil, nil, nil)
	r := gin.New()
	r.POST("/api/payments/yookassa/create", handler.CreateCheckout)

	// Test negative telegram ID and malicious username with script injection
	reqBody, _ := json.Marshal(map[string]interface{}{
		"tier":              "accelerator",
		"telegram_username": "@valid_user<script>alert(1)</script>",
		"telegram_id":       -999, // negative ID
		"promo_code":        "SPARK5",
	})

	req := httptest.NewRequest("POST", "/api/payments/yookassa/create", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// A payment must always have an identifiable recipient.
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for negative telegram id, got %d", w.Code)
	}
}

func TestAdminHandler_ListUsers_FiltersAndPagination(t *testing.T) {
	handler := NewAdminHandler(nil, &config.Config{}, nil, nil)
	r := gin.New()
	r.GET("/admin/users", handler.ListUsers)

	// 1. Request with filters returns 200 OK and proper JSON structure
	req := httptest.NewRequest("GET", "/admin/users?tier=accelerator&access=active&receipt_status=pending&q=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}
	if res["users"] == nil {
		t.Fatal("expected users field in response")
	}
	if res["total"] == nil {
		t.Fatal("expected total field in response")
	}
}

func TestAdminHandler_TriggerDigest(t *testing.T) {
	called := false
	handler := NewAdminHandler(nil, &config.Config{}, nil, nil)
	handler.SetDigestSender(func() error {
		called = true
		return nil
	})

	r := gin.New()
	r.POST("/admin/digest", handler.TriggerDigest)

	req := httptest.NewRequest("POST", "/admin/digest", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	if !called {
		t.Fatal("expected digest function to be triggered")
	}
}

func TestPricingHandler_GetAndUpdatePricing(t *testing.T) {
	handler := NewPricingHandler(nil)
	r := gin.New()
	r.GET("/api/v1/pricing", handler.GetPricing)
	r.PUT("/api/v1/admin/pricing", handler.UpdatePricing)

	// 1. GET default pricing
	req := httptest.NewRequest("GET", "/api/v1/pricing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var p models.CoursePricing
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("failed to decode pricing response: %v", err)
	}
	if p.AcceleratorPrice != 6900.0 || p.VIPPrice != 14900.0 {
		t.Fatalf("expected default prices 6900 and 14900, got %f and %f", p.AcceleratorPrice, p.VIPPrice)
	}

	// 2. PUT invalid pricing (VIP < Accelerator)
	invalidBody := []byte(`{"accelerator_price": 10000, "vip_price": 8000}`)
	putReq := httptest.NewRequest("PUT", "/api/v1/admin/pricing", bytes.NewReader(invalidBody))
	putReq.Header.Set("Content-Type", "application/json")
	wPut := httptest.NewRecorder()
	r.ServeHTTP(wPut, putReq)

	if wPut.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when VIP < Accelerator, got %d", wPut.Code)
	}

	// 3. PUT valid pricing
	validBody := []byte(`{"accelerator_price": 7900, "accelerator_old_price": 10900, "vip_price": 16900, "vip_old_price": 21900}`)
	putReq2 := httptest.NewRequest("PUT", "/api/v1/admin/pricing", bytes.NewReader(validBody))
	putReq2.Header.Set("Content-Type", "application/json")
	wPut2 := httptest.NewRecorder()
	r.ServeHTTP(wPut2, putReq2)

	if wPut2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on valid update, got %d: %s", wPut2.Code, wPut2.Body.String())
	}
}

func TestAdminHandler_GetStats_DetailedRevenueAndBreakdown(t *testing.T) {
	// Test nil DB graceful response
	handler := NewAdminHandler(nil, nil, nil, nil)
	r := gin.New()
	r.GET("/api/v1/admin/stats", handler.GetStats)

	req := httptest.NewRequest("GET", "/api/v1/admin/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for nil DB stats, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}

	if _, ok := res["daily_stats"]; !ok {
		t.Error("expected daily_stats field in GetStats response")
	}
	if _, ok := res["tier_breakdown"]; !ok {
		t.Error("expected tier_breakdown field in GetStats response")
	}
	if _, ok := res["today_revenue"]; !ok {
		t.Error("expected today_revenue field in GetStats response")
	}
	if _, ok := res["average_order_value"]; !ok {
		t.Error("expected average_order_value field in GetStats response")
	}
}

func TestAdminHandler_ListUsers_SortingAndReceiptIssued(t *testing.T) {
	handler := NewAdminHandler(nil, &config.Config{}, nil, nil)
	r := gin.New()
	r.GET("/admin/users", handler.ListUsers)

	sortParams := []string{"purchase_date", "paid_amount", "name", "username", "tier", "receipt_issued", "created_at"}
	for _, sp := range sortParams {
		req := httptest.NewRequest("GET", "/admin/users?sort_by="+sp+"&sort_dir=desc&receipt_issued=issued", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for sort_by=%s, got %d", sp, w.Code)
		}
	}
}

func TestAdminHandler_ToggleReceiptIssued_Mock(t *testing.T) {
	handler := NewAdminHandler(nil, &config.Config{}, nil, nil)
	r := gin.New()
	r.POST("/admin/users/:id/receipt-issued", handler.ToggleReceiptIssued)

	req := httptest.NewRequest("POST", "/admin/users/999/receipt-issued", bytes.NewReader([]byte(`{"receipt_issued":true}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when DB is nil mock, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}
	if res["receipt_issued"] != true {
		t.Fatalf("expected receipt_issued=true, got %v", res["receipt_issued"])
	}
}

func TestAdminHandler_IssueReceipt_Mock(t *testing.T) {
	handler := NewAdminHandler(nil, &config.Config{}, nil, nil)
	r := gin.New()
	r.POST("/admin/users/:id/issue-receipt", handler.IssueReceipt)

	payload := `{"receipt_url":"https://lknpd.nalog.ru/api/v1/receipt/123/print","receipt_number":"CHK-2026-001","amount":6900,"service_name":"Обучение SPARK 2027","send_telegram":true}`
	req := httptest.NewRequest("POST", "/admin/users/1/issue-receipt", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when issuing receipt in mock mode, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}
	if res["success"] != true {
		t.Fatalf("expected success=true, got %v", res["success"])
	}
	if res["receipt_issued"] != true {
		t.Fatalf("expected receipt_issued=true, got %v", res["receipt_issued"])
	}
}
