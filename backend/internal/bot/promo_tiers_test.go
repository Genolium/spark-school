package bot

import (
	"testing"

	"github.com/spark-school/backend/internal/models"
)

func TestSessionDiscountForTier(t *testing.T) {
	service, db := setupTestBot(t)
	if err := db.Create(&models.PromoCode{
		Code:            "DEADLINE",
		DiscountPercent: 14,
		IsActive:        true,
		ApplicableTiers: "accelerator,vip",
	}).Error; err != nil {
		t.Fatal(err)
	}

	session := &UserSession{PromoCode: "DEADLINE", DiscountPct: 14}
	if got := service.sessionDiscountForTier(session, "basic"); got != 0 {
		t.Fatalf("basic discount = %d, want 0", got)
	}
	if got := service.sessionDiscountForTier(session, "accelerator"); got != 14 {
		t.Fatalf("accelerator discount = %d, want 14", got)
	}
	if got := service.sessionDiscountForTier(session, "vip"); got != 14 {
		t.Fatalf("vip discount = %d, want 14", got)
	}

	if err := db.Create(&models.PromoCode{Code: "ALL5", DiscountPercent: 5, IsActive: true}).Error; err != nil {
		t.Fatal(err)
	}
	all := &UserSession{PromoCode: "ALL5", DiscountPct: 5}
	for _, tier := range []string{"basic", "accelerator", "vip"} {
		if got := service.sessionDiscountForTier(all, tier); got != 5 {
			t.Fatalf("%s discount = %d, want 5", tier, got)
		}
	}
}
