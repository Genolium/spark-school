package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
)

func signedInitData(botToken string, userJSON string) string {
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	values.Set("query_id", "AAExample")
	values.Set("user", userJSON)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func TestStudentMiniAppAuthAndMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	cfg := &config.Config{
		TelegramBotToken: "123456:test-bot-token",
		JWTSecret:        "test-secret-key-32-chars-spark-2027",
	}
	h := NewStudentHandler(db, cfg)
	r := gin.New()
	r.POST("/student/auth", h.TelegramAuth)
	me := r.Group("/student")
	me.Use(auth.AuthMiddleware(cfg.JWTSecret))
	me.GET("/me", h.Me)

	initData := signedInitData(cfg.TelegramBotToken, `{"id":777,"first_name":"Test","username":"test"}`)
	body, _ := json.Marshal(map[string]string{"init_data": initData})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/student/auth", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("auth failed: %d %s", w.Code, w.Body.String())
	}
	var response struct {
		Token string `json:"token"`
		User  struct {
			ID int64 `json:"telegram_id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Token == "" || response.User.ID != 777 {
		t.Fatalf("unexpected response: %+v", response)
	}
	var user models.User
	if err := db.Where("telegram_id = ?", 777).First(&user).Error; err != nil {
		t.Fatal("student was not created")
	}

	req := httptest.NewRequest("GET", "/student/me", nil)
	req.Header.Set("Authorization", "Bearer "+response.Token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me failed: %d %s", w.Code, w.Body.String())
	}
}
