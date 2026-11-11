package handlers

import (
	"testing"

	"github.com/spark-school/backend/internal/models"
)

// A full-price payment must never fall below the fraud floor, whatever the
// admin sets as the price. Basic at 3 333 ₽ used to sit under a fixed 3 500 ₽
// floor and was flagged as underpaid.
func TestMinimumForTier_FullPricePaymentIsAccepted(t *testing.T) {
	db := setupHandlerDB(t)
	if err := db.AutoMigrate(&models.CoursePricing{}); err != nil {
		t.Fatalf("migrate pricing: %v", err)
	}
	pricing := models.GetCoursePricing(db)

	for tier, price := range map[string]float64{
		"basic":       pricing.BasicPrice,
		"accelerator": pricing.AcceleratorPrice,
		"vip":         pricing.VIPPrice,
	} {
		if floor := minimumForTier(db, tier); price < floor {
			t.Errorf("%s: full price %.2f is below the floor %.2f", tier, price, floor)
		}
	}
	if got := minimumForTier(nil, "basic"); models.DefaultCoursePricing.BasicPrice < got {
		t.Errorf("default basic price %.2f is below the floor %.2f", models.DefaultCoursePricing.BasicPrice, got)
	}
}

func TestMinimumForTier_FollowsLargestActivePromo(t *testing.T) {
	db := setupHandlerDB(t)
	if err := db.AutoMigrate(&models.CoursePricing{}); err != nil {
		t.Fatalf("migrate pricing: %v", err)
	}
	pricing := models.GetCoursePricing(db)

	promos := []models.PromoCode{
		{Code: "FIVE", DiscountPercent: 5, IsActive: true},
		{Code: "DEADLINE", DiscountPercent: 14, IsActive: true},
	}
	if err := db.Create(&promos).Error; err != nil {
		t.Fatalf("create promos: %v", err)
	}
	// An inactive code must not lower the floor. GORM skips a false bool on
	// insert because of the column default, so switch it off explicitly.
	off := models.PromoCode{Code: "OFF", DiscountPercent: 80, IsActive: true}
	if err := db.Create(&off).Error; err != nil {
		t.Fatalf("create promo: %v", err)
	}
	if err := db.Model(&off).Update("is_active", false).Error; err != nil {
		t.Fatalf("deactivate promo: %v", err)
	}

	discounted := roundAmount(pricing.AcceleratorPrice * 0.86)
	floor := minimumForTier(db, "accelerator")
	if discounted < floor {
		t.Errorf("payment with the 14%% promo (%.2f) is below the floor %.2f", discounted, floor)
	}
	if tooLow := roundAmount(pricing.AcceleratorPrice * 0.5); tooLow >= floor {
		t.Errorf("half price %.2f must stay below the floor %.2f", tooLow, floor)
	}
	if floor := minimumCoursePayment(db); floor <= 100 {
		t.Errorf("course floor %.2f must still reject token payments", floor)
	}
	if got := minimumForTier(db, "audit_essay"); got != minAuditPayment {
		t.Errorf("audit floor = %.2f, want %.2f", got, minAuditPayment)
	}
}
