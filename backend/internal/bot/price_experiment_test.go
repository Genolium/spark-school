package bot

import (
	"testing"

	"github.com/spark-school/backend/internal/models"
)

func TestPricingForSessionPriceExperiment(t *testing.T) {
	service := &BotService{}

	lower := service.pricingForSession(&UserSession{PriceExperiment: "lower"})
	if lower.AcceleratorPrice != 9990 {
		t.Fatalf("lower variant accelerator price = %v, want 9990", lower.AcceleratorPrice)
	}
	if lower.BasicPrice != models.DefaultCoursePricing.BasicPrice || lower.VIPPrice != models.DefaultCoursePricing.VIPPrice {
		t.Fatal("price experiment must only change the accelerator tier")
	}

	control := service.pricingForSession(&UserSession{})
	if control.AcceleratorPrice != models.DefaultCoursePricing.AcceleratorPrice {
		t.Fatalf("control accelerator price = %v, want %v", control.AcceleratorPrice, models.DefaultCoursePricing.AcceleratorPrice)
	}
}

func TestPricingForSessionVIPPriceExperiment(t *testing.T) {
	service := &BotService{}

	low := service.pricingForSession(&UserSession{VIPPriceExperiment: "low"})
	if low.VIPPrice != 14990 {
		t.Fatalf("low VIP price = %v, want 14990", low.VIPPrice)
	}
	mid := service.pricingForSession(&UserSession{VIPPriceExperiment: "mid"})
	if mid.VIPPrice != 17990 {
		t.Fatalf("mid VIP price = %v, want 17990", mid.VIPPrice)
	}
	control := service.pricingForSession(&UserSession{})
	if control.VIPPrice != models.DefaultCoursePricing.VIPPrice {
		t.Fatalf("control VIP price = %v, want %v", control.VIPPrice, models.DefaultCoursePricing.VIPPrice)
	}
}
