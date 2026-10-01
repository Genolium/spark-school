package handlers

import (
	"bytes"
	"context"
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
	// Test helper GetTierPrice
	if price := GetTierPrice("basic", 0); price != 2900.00 {
		t.Errorf("expected basic price 2900, got %f", price)
	}
	if price := GetTierPrice("accelerator", 0); price != 6900.00 {
		t.Errorf("expected accelerator price 6900, got %f", price)
	}
	if price := GetTierPrice("vip", 0); price != 14900.00 {
		t.Errorf("expected vip price 14900, got %f", price)
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

