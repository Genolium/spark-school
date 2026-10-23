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
	if len(assignments) != len(experimentVariants) {
		t.Fatalf("expected assignments for every active experiment, got %+v", assignments)
	}
	for _, assignment := range assignments {
		if assignment.Variant != assignVariant("visitor-1", assignment.Experiment) {
			t.Fatalf("server must choose the variant, got %+v", assignment)
		}
		if assignment.Experiment == "unknown" {
			t.Fatal("unknown experiment must not be stored")
		}
	}

	// The browser may not fabricate a paid conversion.
	if w := post(map[string]interface{}{"step": "payment_confirmed"}); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for server-only step, got %d", w.Code)
	}
	if w := post(map[string]interface{}{"step": "not_a_real_step"}); w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown step, got %d", w.Code)
	}
	large := map[string]interface{}{"step": "landing_view", "metadata": map[string]interface{}{"source": string(make([]byte, 5000))}}
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

func TestExperimentAssignmentAndStatistics(t *testing.T) {
	if assignVariant("visitor-1", "hero_cta_copy") != assignVariant("visitor-1", "hero_cta_copy") {
		t.Fatal("assignment must be deterministic")
	}
	if assignVariant("", "hero_cta_copy") == "" {
		t.Fatal("expected a variant")
	}

	// Revenue per visitor: 1000 visitors with 100 payments of 1000 ₽ vs
	// 1000 visitors with 150 payments of 1000 ₽ = 100 ₽ vs 150 ₽ per visitor.
	rows := []experimentRow{
		{Experiment: "hero_cta_copy", Variant: "control", Visitors: 1000, Payers: 100, Revenue: 100000, RevenueSquared: 100000000},
		{Experiment: "hero_cta_copy", Variant: "benefit", Visitors: 1000, Payers: 150, Revenue: 150000, RevenueSquared: 150000000},
	}
	applyExperimentStatistics(rows)
	for i := range rows {
		rows[i].RevenuePerVisitor = roundAmount(rows[i].Revenue / float64(rows[i].Visitors))
	}
	if rows[0].RevenuePerVisitor != 100 || rows[1].RevenuePerVisitor != 150 {
		t.Fatalf("unexpected revenue per visitor: %v vs %v", rows[0].RevenuePerVisitor, rows[1].RevenuePerVisitor)
	}
	if !rows[1].Significant || rows[1].PValue >= 0.05 {
		t.Fatalf("expected significant RPV uplift, got %+v", rows[1])
	}
	if rows[1].UpliftPercent < 49 || rows[1].UpliftPercent > 51 {
		t.Fatalf("expected +50%% RPV uplift, got %f", rows[1].UpliftPercent)
	}
	if rows[1].Decision != "switch" {
		t.Fatalf("expected the report to switch, got %q", rows[1].Decision)
	}
	if rows[0].Decision != "reference" {
		t.Fatalf("control must stay the reference, got %q", rows[0].Decision)
	}

	small := []experimentRow{
		{Experiment: "hero_cta_copy", Variant: "control", Visitors: 10, Payers: 1, Revenue: 6900},
		{Experiment: "hero_cta_copy", Variant: "benefit", Visitors: 10, Payers: 3, Revenue: 20700},
	}
	applyExperimentStatistics(small)
	if small[1].Significant || small[1].MinSampleReached {
		t.Fatalf("small samples must not be called significant: %+v", small[1])
	}
	if small[1].Decision != "collect" {
		t.Fatalf("small samples must ask for more data, got %q", small[1].Decision)
	}
	if small[1].RequiredSample != experimentMinSample {
		t.Fatalf("the threshold must be %d visitors per variant, got %d", experimentMinSample, small[1].RequiredSample)
	}
}

func TestAssignExperiments_ServerDeterministic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	r := gin.New()
	r.POST("/assign", AssignExperiments(db))

	body, _ := json.Marshal(map[string]string{"visitor_id": "visitor-server", "attribution_token": "a12345678"})
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/assign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var response struct {
			Experiments map[string]string `json:"experiments"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		for name := range experimentVariants {
			if response.Experiments[name] != assignVariant("visitor-server", name) {
				t.Fatalf("unexpected assignment for %s: %v", name, response.Experiments)
			}
		}
	}
	var count int64
	db.Model(&models.ExperimentAssignment{}).Count(&count)
	if count != int64(len(experimentVariants)) {
		t.Fatalf("expected one assignment per experiment, got %d", count)
	}
}
