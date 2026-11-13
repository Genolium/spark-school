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

// The accelerator price experiment was retired: every visitor sees and pays the
// price configured in the admin panel, whatever bucket they were once in.
func TestPricing_IgnoresVisitorAndMatchesCheckout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupHandlerDB(t)
	if err := db.AutoMigrate(&models.CoursePricing{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CoursePricing{
		BasicPrice: 3333, AcceleratorPrice: 9990, VIPPrice: 15990,
	}).Error; err != nil {
		t.Fatal(err)
	}

	handler := NewPricingHandler(db)
	router := gin.New()
	router.GET("/pricing", handler.GetPricing)

	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pricing?visitor_id=visitor-%d", i), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("pricing status %d", w.Code)
		}
		var pricing models.CoursePricing
		if err := json.Unmarshal(w.Body.Bytes(), &pricing); err != nil {
			t.Fatal(err)
		}
		if pricing.AcceleratorPrice != 9990 {
			t.Fatalf("visitor-%d accelerator price = %v, want 9990", i, pricing.AcceleratorPrice)
		}
	}

	pricing := models.CoursePricing{BasicPrice: 3333, AcceleratorPrice: 9990, VIPPrice: 15990}
	if got := checkoutBasePrice("accelerator", pricing); got != 9990 {
		t.Fatalf("checkout accelerator price = %v, want 9990", got)
	}
}

func TestExperimentVariants_NoPriceExperiment(t *testing.T) {
	if _, ok := experimentVariants["pricing_price"]; ok {
		t.Fatal("pricing_price experiment must stay retired")
	}
	for _, name := range []string{"hero_cta_copy", "pricing_cta_copy"} {
		if len(experimentVariants[name]) < 2 {
			t.Fatalf("experiment %s must have at least two variants", name)
		}
	}
}
