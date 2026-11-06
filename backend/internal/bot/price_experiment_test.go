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
