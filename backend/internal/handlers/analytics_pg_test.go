package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func analyticsPGDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}
	lockPath := filepath.Join(os.TempDir(), "spark-pg-integration.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.AnalyticsEvent{}, &models.PaymentTransaction{}, &models.ExperimentAssignment{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPostgres_FunnelUsesEventsAndPaymentTransactions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := analyticsPGDB(t)
	now := time.Now()
	events := []models.AnalyticsEvent{
		{VisitorID: "v1", Step: "landing_view", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v1", Step: "diagnostic_start", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v1", Step: "diagnostic_complete", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v1", Step: "pricing_select_tier", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v1", Step: "bot_handoff", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v1", Step: "bot_lead", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v1", Step: "payment_confirmed", Source: "telegram", Tier: "accelerator", CreatedAt: now},
		{VisitorID: "v2", Step: "landing_view", Source: "youtube", Tier: "vip", CreatedAt: now},
	}
	for _, event := range events {
		if err := db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.PaymentTransaction{
		PaymentID: "funnel-1", TelegramID: 1, Tier: "accelerator", Amount: 6900,
		Status: "SUCCESS", Currency: "RUB", Source: "telegram", CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	r.GET("/funnel", GetFunnel(db))
	req := httptest.NewRequest("GET", "/funnel?source=telegram", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Steps    []funnelStepResponse `json:"steps"`
		Payments struct {
			Count   int64   `json:"count"`
			Revenue float64 `json:"revenue"`
			Source  string  `json:"source"`
		} `json:"payments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Payments.Count != 1 || response.Payments.Revenue != 6900 || response.Payments.Source != "payment_transactions" {
		t.Fatalf("unexpected payments: %+v", response.Payments)
	}
	counts := map[string]int64{}
	for _, step := range response.Steps {
		counts[step.Step] = step.Count
	}
	if counts["landing_view"] != 1 {
		t.Fatalf("source filter should leave one unique landing visitor, got %d", counts["landing_view"])
	}
	if counts["payment_confirmed"] != 1 {
		t.Fatalf("expected one confirmed payment event, got %d", counts["payment_confirmed"])
	}
}

func TestPostgres_ExperimentsUseConfirmedPayments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := analyticsPGDB(t)
	now := time.Now()
	db.Create(&models.ExperimentAssignment{Experiment: "hero_cta_copy", Variant: "control", VisitorID: "exp-v1", CreatedAt: now})
	db.Create(&models.ExperimentAssignment{Experiment: "hero_cta_copy", Variant: "benefit", VisitorID: "exp-v2", CreatedAt: now})
	db.Create(&models.AnalyticsEvent{VisitorID: "exp-v1", Step: "bot_lead", CreatedAt: now})
	db.Create(&models.AnalyticsEvent{VisitorID: "exp-v1", Step: "order_created", CreatedAt: now})
	db.Create(&models.AnalyticsEvent{VisitorID: "exp-v2", Step: "bot_lead", CreatedAt: now})
	db.Create(&models.PaymentTransaction{
		PaymentID: "exp-pay-1", TelegramID: 1, VisitorID: "exp-v1", Tier: "accelerator",
		Amount: 6900, Status: "SUCCESS", Currency: "RUB", CreatedAt: now,
	})

	r := gin.New()
	r.GET("/experiments", GetExperiments(db))
	req := httptest.NewRequest("GET", "/experiments", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Experiments []experimentRow `json:"experiments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Experiments) != 2 {
		t.Fatalf("expected two variants, got %d", len(response.Experiments))
	}
	byVariant := map[string]experimentRow{}
	for _, row := range response.Experiments {
		byVariant[row.Variant] = row
	}
	if byVariant["control"].Payers != 1 || byVariant["control"].Revenue != 6900 || byVariant["control"].Conversion != 100 {
		t.Fatalf("unexpected control metrics: %+v", byVariant["control"])
	}
	if byVariant["benefit"].Payers != 0 || byVariant["benefit"].BotLeads != 1 {
		t.Fatalf("unexpected benefit metrics: %+v", byVariant["benefit"])
	}
}
