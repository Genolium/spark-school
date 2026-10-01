package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateAndVerifyJWT(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum-length-spark-2027"
	user := &models.User{
		ID:         42,
		TelegramID: 123456789,
		Username:   "alex_tester",
		Role:       "student",
		HasAccess:  true,
	}

	token, err := GenerateJWT(user, secret)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Test AuthMiddleware with valid token
	r := gin.New()
	r.GET("/protected", AuthMiddleware(secret), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")
		hasAccess, _ := c.Get("has_access")
		c.JSON(http.StatusOK, gin.H{
			"user_id":    userID,
			"role":       role,
			"has_access": hasAccess,
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if uint(res["user_id"].(float64)) != 42 {
		t.Errorf("expected user_id 42, got %v", res["user_id"])
	}
	if res["role"] != "student" {
		t.Errorf("expected role student, got %v", res["role"])
	}
	if res["has_access"] != true {
		t.Errorf("expected has_access true, got %v", res["has_access"])
	}
}

func TestAdminMiddleware_RejectionAndAcceptance(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum-length-spark-2027"

	studentUser := &models.User{ID: 1, Role: "student"}
	studentToken, _ := GenerateJWT(studentUser, secret)

	adminUser := &models.User{ID: 2, Role: "admin"}
	adminToken, _ := GenerateJWT(adminUser, secret)

	r := gin.New()
	r.GET("/admin-only", AdminMiddleware(secret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Student should be rejected with 403 Forbidden
	reqStudent := httptest.NewRequest("GET", "/admin-only", nil)
	reqStudent.Header.Set("Authorization", "Bearer "+studentToken)
	wStudent := httptest.NewRecorder()
	r.ServeHTTP(wStudent, reqStudent)

	if wStudent.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for student, got %d", wStudent.Code)
	}

	// Admin should be accepted with 200 OK
	reqAdmin := httptest.NewRequest("GET", "/admin-only", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin, got %d", wAdmin.Code)
	}
}

func TestVerifyTelegramWidget(t *testing.T) {
	botToken := "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
	now := time.Now().Unix()

	data := map[string]string{
		"id":         "123456789",
		"first_name": "Ilya",
		"username":   "ilya_vas",
		"auth_date":  fmt.Sprintf("%d", now),
	}

	// Calculate correct hash
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, data[k]))
	}
	checkStr := strings.Join(parts, "\n")
	secretKey := sha256.Sum256([]byte(botToken))
	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(checkStr))
	data["hash"] = hex.EncodeToString(h.Sum(nil))

	if !VerifyTelegramWidget(data, botToken) {
		t.Fatal("expected VerifyTelegramWidget to return true for valid hash")
	}

	// Tampered data should fail
	data["first_name"] = "Attacker"
	if VerifyTelegramWidget(data, botToken) {
		t.Fatal("expected VerifyTelegramWidget to return false for tampered data")
	}
}
