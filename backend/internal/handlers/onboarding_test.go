package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func TestUpdateOnboarding_TransitionsAndEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	user := models.User{
		TelegramID: 99001, Username: "student", FirstName: "Student", Role: "student",
		HasAccess: true, Tier: "accelerator", OnboardingStatus: "pending_intro",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	h := NewAdminHandler(db, nil, nil, nil)
	r := gin.New()
	r.POST("/users/:id/onboarding", h.UpdateOnboarding)

	post := func(status string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"status": status, "notes": "ok"})
		req := httptest.NewRequest("POST", "/users/"+itoa(user.ID)+"/onboarding", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := post("awaiting_reply"); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var reloaded models.User
	db.First(&reloaded, user.ID)
	if reloaded.OnboardingStatus != "awaiting_reply" || reloaded.IntroSentAt == nil {
		t.Fatalf("unexpected state: %+v", reloaded)
	}
	if w := post("completed"); w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	db.First(&reloaded, user.ID)
	if reloaded.OnboardingStatus != "completed" || reloaded.OnboardingCompletedAt == nil || reloaded.OnboardingRepliedAt == nil {
		t.Fatalf("completion not persisted: %+v", reloaded)
	}
	var events int64
	db.Model(&models.AnalyticsEvent{}).Where("step IN ?", []string{"onboarding_intro_sent", "onboarding_replied"}).Count(&events)
	if events != 2 {
		t.Fatalf("expected two onboarding events, got %d", events)
	}
}

func itoa(id uint) string {
	if id == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for id > 0 {
		i--
		digits[i] = byte('0' + id%10)
		id /= 10
	}
	return string(digits[i:])
}
