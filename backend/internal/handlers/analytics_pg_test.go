package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

func TestPostgres_FunnelRestrictsStepsToLandingCohort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := analyticsPGDB(t)
	now := time.Now()

	// v1 entered through the landing page; v2 only paid (imported legacy row).
	db.Create(&models.AnalyticsEvent{VisitorID: "v1", Step: "landing_view", CreatedAt: now})
	db.Create(&models.AnalyticsEvent{VisitorID: "v1", Step: "payment_confirmed", PaymentID: "cohort-1", CreatedAt: now})
	db.Create(&models.AnalyticsEvent{VisitorID: "v2", Step: "payment_confirmed", PaymentID: "cohort-2", CreatedAt: now})
	db.Create(&models.PaymentTransaction{PaymentID: "cohort-1", TelegramID: 1, VisitorID: "v1", Tier: "accelerator", Amount: 1000, Status: "SUCCESS", Currency: "RUB", CreatedAt: now})
	db.Create(&models.PaymentTransaction{PaymentID: "cohort-2", TelegramID: 2, VisitorID: "v2", Tier: "accelerator", Amount: 2000, Status: "SUCCESS", Currency: "RUB", CreatedAt: now})

	r := gin.New()
	r.GET("/funnel", GetFunnel(db))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/funnel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Steps    []funnelStepResponse `json:"steps"`
		Payments struct {
			Count            int64   `json:"count"`
			Revenue          float64 `json:"revenue"`
			TrackedCount     int64   `json:"tracked_count"`
			TrackedRevenue   float64 `json:"tracked_revenue"`
			UntrackedCount   int64   `json:"untracked_count"`
			UntrackedRevenue float64 `json:"untracked_revenue"`
		} `json:"payments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, step := range response.Steps {
		counts[step.Step] = step.Count
	}
	if counts["landing_view"] != 1 || counts["payment_confirmed"] != 1 {
		t.Fatalf("funnel steps must stay inside the landing cohort, got %+v", counts)
	}
	if response.Payments.Count != 2 || response.Payments.Revenue != 3000 {
		t.Fatalf("payments block must keep reporting all settled money: %+v", response.Payments)
	}
	if response.Payments.TrackedCount != 1 || response.Payments.TrackedRevenue != 1000 {
		t.Fatalf("unexpected tracked payments: %+v", response.Payments)
	}
	if response.Payments.UntrackedCount != 1 || response.Payments.UntrackedRevenue != 2000 {
		t.Fatalf("unexpected untracked payments: %+v", response.Payments)
	}
}

func TestPostgres_ExperimentsRevenuePerVisitorDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := analyticsPGDB(t)
	now := time.Now()

	// Control: 120 visitors, 2 payments of 5000 = 10 000 ₽ (83.33 ₽/visitor).
	// Lower:  120 visitors, 20 payments of 5000 = 100 000 ₽ (833.33 ₽/visitor).
	for i := 0; i < 120; i++ {
		db.Create(&models.ExperimentAssignment{Experiment: "pricing_price", Variant: "control", VisitorID: fmt.Sprintf("c-%d", i), CreatedAt: now})
		db.Create(&models.ExperimentAssignment{Experiment: "pricing_price", Variant: "lower", VisitorID: fmt.Sprintf("l-%d", i), CreatedAt: now})
	}
	for i := 0; i < 2; i++ {
		db.Create(&models.PaymentTransaction{
			PaymentID: fmt.Sprintf("control-pay-%d", i), TelegramID: int64(i + 1), VisitorID: fmt.Sprintf("c-%d", i),
			Tier: "accelerator", Amount: 5000, Status: "SUCCESS", Currency: "RUB", CreatedAt: now,
		})
	}
	for i := 0; i < 20; i++ {
		db.Create(&models.PaymentTransaction{
			PaymentID: fmt.Sprintf("lower-pay-%d", i), TelegramID: int64(i + 100), VisitorID: fmt.Sprintf("l-%d", i),
			Tier: "accelerator", Amount: 5000, Status: "SUCCESS", Currency: "RUB", CreatedAt: now,
		})
	}
	// Outside the 14-day window: neither the assignment nor its payment count.
	db.Create(&models.ExperimentAssignment{Experiment: "pricing_price", Variant: "control", VisitorID: "old-control", CreatedAt: now.AddDate(0, 0, -20)})
	db.Create(&models.PaymentTransaction{
		PaymentID: "old-payment", TelegramID: 999, VisitorID: "old-control",
		Tier: "accelerator", Amount: 9000, Status: "SUCCESS", Currency: "RUB", CreatedAt: now.AddDate(0, 0, -20),
	})

	r := gin.New()
	r.GET("/experiments", GetExperiments(db))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/experiments", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var response struct {
		Experiments []experimentRow `json:"experiments"`
		WindowDays  int             `json:"window_days"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.WindowDays != 14 {
		t.Fatalf("expected a 14-day decision window, got %d", response.WindowDays)
	}
	byVariant := map[string]experimentRow{}
	for _, row := range response.Experiments {
		byVariant[row.Variant] = row
	}
	control := byVariant["control"]
	lower := byVariant["lower"]
	if control.Visitors != 120 || control.Revenue != 10000 {
		t.Fatalf("control must exclude out-of-window rows: %+v", control)
	}
	if lower.Visitors != 120 || lower.Revenue != 100000 {
		t.Fatalf("unexpected lower metrics: %+v", lower)
	}
	if control.RevenuePerVisitor != 83.33 || lower.RevenuePerVisitor != 833.33 {
		t.Fatalf("unexpected revenue per visitor: control=%v lower=%v", control.RevenuePerVisitor, lower.RevenuePerVisitor)
	}
	if lower.UpliftPercent <= 0 {
		t.Fatalf("expected positive RPV uplift, got %v", lower.UpliftPercent)
	}
	if !lower.MinSampleReached || !lower.Significant {
		t.Fatalf("expected a significant result at 120 visitors per variant: %+v", lower)
	}
	if lower.Decision != "switch" {
		t.Fatalf("expected the lower variant to win, got %q", lower.Decision)
	}
	if control.Decision != "reference" {
		t.Fatalf("control decision must be reference, got %q", control.Decision)
	}
}

func TestPostgres_ExperimentsRequireHundredVisitorsPerVariant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := analyticsPGDB(t)
	now := time.Now()
	for i := 0; i < 99; i++ {
		db.Create(&models.ExperimentAssignment{Experiment: "pricing_price", Variant: "control", VisitorID: fmt.Sprintf("small-c-%d", i), CreatedAt: now})
		db.Create(&models.ExperimentAssignment{Experiment: "pricing_price", Variant: "lower", VisitorID: fmt.Sprintf("small-l-%d", i), CreatedAt: now})
	}

	r := gin.New()
	r.GET("/experiments", GetExperiments(db))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/experiments", nil))
	var response struct {
		Experiments []experimentRow `json:"experiments"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, row := range response.Experiments {
		if row.MinSampleReached {
			t.Fatalf("99 visitors per variant must not reach the threshold: %+v", row)
		}
		expected := "collect"
		if row.Variant == "control" {
			expected = "reference"
		}
		if row.Decision != expected {
			t.Fatalf("expected decision %q for %s, got %q", expected, row.Variant, row.Decision)
		}
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
