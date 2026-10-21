package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/config"
)

func TestDiagnosticCreateAndGet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	cfg := &config.Config{TelegramBotUsername: "spark_prep_bot"}
	h := NewDiagnosticHandler(db, cfg)
	r := gin.New()
	r.POST("/diagnostic", h.Create)
	r.GET("/diagnostic/:token", h.Get)

	body, _ := json.Marshal(map[string]interface{}{
		"visitor_id":        "visitor-plan",
		"attribution_token": "a12345678",
		"age_citizenship":   true,
		"university_year":   "3",
		"english_level":     "b2",
		"source":            "landing",
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/diagnostic", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Token       string `json:"token"`
		TelegramURL string `json:"telegram_url"`
		Plan        struct {
			Score int                      `json:"score"`
			Steps []struct{ Title string } `json:"steps"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Token == "" || created.Plan.Score != 88 || len(created.Plan.Steps) == 0 {
		t.Fatalf("unexpected response: %+v", created)
	}
	if created.TelegramURL != "https://t.me/spark_prep_bot?start=plan_"+created.Token {
		t.Fatalf("unexpected telegram url: %s", created.TelegramURL)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/diagnostic/"+created.Token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get failed: %d", w.Code)
	}
}
