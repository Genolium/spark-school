package bot

import (
	"testing"

	"github.com/spark-school/backend/internal/models"
)

// The accelerator price experiment was retired. Sessions saved earlier with
// PriceExperiment "lower" must get the regular admin price.
func TestPricingForSessionIgnoresLegacyPriceExperiment(t *testing.T) {
	service := &BotService{}

	legacy := service.pricingForSession(&UserSession{PriceExperiment: "lower"})
	if legacy.AcceleratorPrice != models.DefaultCoursePricing.AcceleratorPrice {
		t.Fatalf("legacy lower session accelerator price = %v, want %v", legacy.AcceleratorPrice, models.DefaultCoursePricing.AcceleratorPrice)
	}

	fresh := service.pricingForSession(&UserSession{})
	if fresh.AcceleratorPrice != legacy.AcceleratorPrice {
		t.Fatalf("fresh session price %v differs from legacy %v", fresh.AcceleratorPrice, legacy.AcceleratorPrice)
	}
}
