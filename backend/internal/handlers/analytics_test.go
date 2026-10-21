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

func TestTrackAnalytics_WhitelistPIIAndSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	r := gin.New()
	r.POST("/track", TrackAnalytics(db))

	post := func(payload map[string]interface{}) *httptest.ResponseRecorder {
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/track", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w := post(map[string]interface{}{
		"step":              "landing_view",
		"visitor_id":        "visitor-1",
		"attribution_token": "a12345678",
		"experiments":       map[string]string{"hero_cta_copy": "benefit", "unknown": "x"},
		"metadata": map[string]interface{}{
			"utm_source": "telegram",
			"username":   "should-not-be-stored",
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var event models.AnalyticsEvent
	if err := db.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Source != "telegram" || event.AttributionToken != "a12345678" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if bytes.Contains([]byte(event.Metadata), []byte("should-not-be-stored")) {
		t.Fatalf("PII leaked into metadata: %s", event.Metadata)
	}
	var assignments []models.ExperimentAssignment
	db.Find(&assignments)
	if len(assignments) != 1 || assignments[0].Experiment != "hero_cta_copy" || assignments[0].Variant != "benefit" {
		t.Fatalf("unexpected experiment assignments: %+v", assignments)
	}

	// The browser may not fabricate a paid conversion.
	if w := post(map[string]interface{}{"step": "payment_confirmed"}); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for server-only step, got %d", w.Code)
	}
	if w := post(map[string]interface{}{"step": "not_a_real_step"}); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown step, got %d", w.Code)
	}
	large := map[string]interface{}{"step": "landing_view", "metadata": map[string]interface{}{"blob": string(make([]byte, 5000))}}
	if w := post(large); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized metadata, got %d", w.Code)
	}
}

func TestTrackAnalytics_ReturnsErrorWhenPersistenceFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	r := gin.New()
	r.POST("/track", TrackAnalytics(db))
	body, _ := json.Marshal(map[string]interface{}{"step": "landing_view", "visitor_id": "v"})
	req := httptest.NewRequest("POST", "/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when the event cannot be stored, got %d", w.Code)
	}
}
