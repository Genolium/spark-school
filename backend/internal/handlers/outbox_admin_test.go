package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func TestAdminHandler_IgnoreOutboxEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	h := &AdminHandler{db: db}

	dead := models.OutboxEvent{EventType: "telegram_notification", Payload: "{}", Status: "failed", RetryCount: 5, MaxRetries: 5}
	if err := db.Create(&dead).Error; err != nil {
		t.Fatal(err)
	}
	pending := models.OutboxEvent{EventType: "telegram_notification", Payload: "{}", Status: "pending", MaxRetries: 5}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.POST("/outbox/:id/ignore", h.IgnoreOutbox)

	post := func(id uint) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, fmt.Sprintf("/outbox/%d/ignore", id), nil))
		return w.Code
	}

	if code := post(dead.ID); code != http.StatusOK {
		t.Fatalf("ignoring a dead event must succeed, got %d", code)
	}
	var updated models.OutboxEvent
	if err := db.First(&updated, dead.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Status != "ignored" {
		t.Fatalf("expected status ignored, got %q", updated.Status)
	}
	if updated.ProcessedAt == nil {
		t.Fatal("expected processed_at to be set")
	}

	if code := post(pending.ID); code != http.StatusNotFound {
		t.Fatalf("a pending event must not be ignored, got %d", code)
	}

	// The ignored event must leave the dead-letter list.
	listRouter := gin.New()
	listRouter.GET("/outbox/dead", h.ListDeadOutbox)
	w := httptest.NewRecorder()
	listRouter.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/outbox/dead", nil))
	var listResponse struct {
		Events []models.OutboxEvent `json:"events"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResponse); err != nil {
		t.Fatal(err)
	}
	for _, event := range listResponse.Events {
		if event.ID == dead.ID {
			t.Fatal("ignored event must not appear in the dead-letter list")
		}
	}
}
