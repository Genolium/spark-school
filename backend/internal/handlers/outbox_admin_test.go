package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

func TestOutboxDeadLetterListAndRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	event := models.OutboxEvent{
		EventType: "telegram_notification", Payload: `{}`, Status: "failed",
		RetryCount: 5, MaxRetries: 5, LastError: "telegram unavailable", CreatedAt: time.Now(),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewAdminHandler(db, nil, nil, nil)
	r := gin.New()
	r.GET("/dead", handler.ListDeadOutbox)
	r.POST("/:id/retry", handler.RetryOutbox)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/dead", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list failed: %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/1/retry", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("retry failed: %d", w.Code)
	}
	var reloaded models.OutboxEvent
	db.First(&reloaded, event.ID)
	if reloaded.Status != "pending" || reloaded.RetryCount != 0 || reloaded.LastError != "" {
		t.Fatalf("event was not reset: %+v", reloaded)
	}
}
