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

func visitorForExperiment(t *testing.T, experiment, variant string) string {
	t.Helper()
	for i := 0; i < 10000; i++ {
		id := fmt.Sprintf("visitor-%d", i)
		if assignVariant(id, experiment) == variant {
			return id
		}
	}
	t.Fatalf("no visitor found for %s/%s", experiment, variant)
	return ""
}

func TestPricingExperiment_PriceIsStickyAndSharedWithCheckout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	if err := db.AutoMigrate(&models.CoursePricing{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoursePricing{
		BasicPrice: 3990, AcceleratorPrice: 10990, VIPPrice: 19990,
	}).Error; err != nil {
		t.Fatal(err)
	}

	lowerVisitor := visitorForExperiment(t, "pricing_price", "lower")
	controlVisitor := visitorForExperiment(t, "pricing_price", "control")

	handler := NewPricingHandler(db)
	router := gin.New()
	router.GET("/pricing", handler.GetPricing)

	getPrice := func(visitorID string) float64 {
		req := httptest.NewRequest(http.MethodGet, "/pricing?visitor_id="+visitorID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("pricing status %d", w.Code)
		}
		var pricing models.CoursePricing
		if err := json.Unmarshal(w.Body.Bytes(), &pricing); err != nil {
			t.Fatal(err)
		}
		return pricing.AcceleratorPrice
	}

	if got := getPrice(lowerVisitor); got != 9990 {
		t.Fatalf("lower variant price = %v, want 9990", got)
	}
	if got := getPrice(lowerVisitor); got != 9990 {
		t.Fatalf("price must stay sticky, got %v", got)
	}
	if got := getPrice(controlVisitor); got != 10990 {
		t.Fatalf("control variant price = %v, want 10990", got)
	}

	pricing := models.CoursePricing{BasicPrice: 3990, AcceleratorPrice: 10990, VIPPrice: 19990}
	if got := checkoutBasePrice("accelerator", pricing, lowerVisitor); got != 9990 {
		t.Fatalf("checkout lower price = %v, want 9990", got)
	}
	if got := checkoutBasePrice("accelerator", pricing, controlVisitor); got != 10990 {
		t.Fatalf("checkout control price = %v, want 10990", got)
	}
}

func TestVIPPriceExperiment_AppliesToPricingAndCheckout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	if err := db.AutoMigrate(&models.CoursePricing{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoursePricing{
		BasicPrice: 3333, AcceleratorPrice: 10990, VIPPrice: 19990,
	}).Error; err != nil {
		t.Fatal(err)
	}

	lowVisitor := visitorForExperiment(t, "vip_price", "low")
	midVisitor := visitorForExperiment(t, "vip_price", "mid")
	controlVisitor := visitorForExperiment(t, "vip_price", "control")

	handler := NewPricingHandler(db)
	router := gin.New()
	router.GET("/pricing", handler.GetPricing)
	getVIPPrice := func(visitorID string) float64 {
		req := httptest.NewRequest(http.MethodGet, "/pricing?visitor_id="+visitorID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("pricing status %d", w.Code)
		}
		var pricing models.CoursePricing
		if err := json.Unmarshal(w.Body.Bytes(), &pricing); err != nil {
			t.Fatal(err)
		}
		return pricing.VIPPrice
	}

	if got := getVIPPrice(lowVisitor); got != 14990 {
		t.Fatalf("low VIP price = %v, want 14990", got)
	}
	if got := getVIPPrice(midVisitor); got != 17990 {
		t.Fatalf("mid VIP price = %v, want 17990", got)
	}
	if got := getVIPPrice(controlVisitor); got != 19990 {
		t.Fatalf("control VIP price = %v, want 19990", got)
	}
	if got := getVIPPrice(lowVisitor); got != 14990 {
		t.Fatalf("VIP price must stay sticky, got %v", got)
	}

	pricing := models.CoursePricing{BasicPrice: 3333, AcceleratorPrice: 10990, VIPPrice: 19990}
	if got := checkoutBasePrice("vip", pricing, lowVisitor); got != 14990 {
		t.Fatalf("checkout low VIP price = %v, want 14990", got)
	}
	if got := checkoutBasePrice("vip", pricing, midVisitor); got != 17990 {
		t.Fatalf("checkout mid VIP price = %v, want 17990", got)
	}
	if got := checkoutBasePrice("vip", pricing, controlVisitor); got != 19990 {
		t.Fatalf("checkout control VIP price = %v, want 19990", got)
	}
}

func TestExperimentVariants_IncludePriceAndHeadline(t *testing.T) {
	for _, name := range []string{"pricing_price", "vip_price", "hero_headline", "hero_cta_copy", "pricing_cta_copy"} {
		if len(experimentVariants[name]) < 2 {
			t.Fatalf("experiment %s must have at least two variants", name)
		}
	}
	if got := acceleratorPriceForVariant(10990, "lower"); got != 9990 {
		t.Fatalf("lower price = %v", got)
	}
	if got := acceleratorPriceForVariant(10990, "control"); got != 10990 {
		t.Fatalf("control price = %v", got)
	}
	if got := vipPriceForVariant(19990, "low"); got != 14990 {
		t.Fatalf("low VIP price = %v", got)
	}
	if got := vipPriceForVariant(19990, "mid"); got != 17990 {
		t.Fatalf("mid VIP price = %v", got)
	}
	if got := vipPriceForVariant(19990, "control"); got != 19990 {
		t.Fatalf("control VIP price = %v", got)
	}
}
