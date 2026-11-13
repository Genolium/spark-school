package bot

import (
	"strings"
	"testing"

	"github.com/spark-school/backend/internal/models"
)

func TestPaymentDetailsContent_UsesPromoPercentAndTierScope(t *testing.T) {
	service, db := setupTestBot(t)
	if err := db.Create(&models.PromoCode{
		Code:            "DEADLINE",
		DiscountPercent: 14,
		IsActive:        true,
		ApplicableTiers: "accelerator,vip",
	}).Error; err != nil {
		t.Fatal(err)
	}

	pricing := models.CoursePricing{BasicPrice: 3333, AcceleratorPrice: 9990, VIPPrice: 15990}
	session := &UserSession{PromoCode: "DEADLINE", DiscountPct: 14, SelectedTier: "accelerator"}

	text, keyboard := service.paymentDetailsContent(session, pricing, true)
	for _, want := range []string{
		"Скидка 14%",
		"8 591",  // 9 990 − 14%
		"13 751", // 15 990 − 14%
		"промокод на этот тариф не действует", // Basic is excluded
		"Акселератор, VIP",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("payment details must contain %q, got:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Скидка 5%") {
		t.Errorf("payment details must not fall back to 5%%")
	}

	if keyboard == nil || len(keyboard.InlineKeyboard) == 0 {
		t.Fatal("expected a keyboard")
	}
	firstButton := keyboard.InlineKeyboard[0][0].Text
	if !strings.Contains(firstButton, "3 333") || strings.Contains(firstButton, "скидкой") {
		t.Errorf("basic button must show the full price without a discount, got %q", firstButton)
	}
	accButton := keyboard.InlineKeyboard[1][0].Text
	if !strings.Contains(accButton, "8 591") {
		t.Errorf("accelerator button must show the discounted price, got %q", accButton)
	}
}

func TestPaymentDetailsContent_UnrestrictedPromoAppliesEverywhere(t *testing.T) {
	service, db := setupTestBot(t)
	if err := db.Create(&models.PromoCode{
		Code:            "SPARK5",
		DiscountPercent: 5,
		IsActive:        true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	pricing := models.CoursePricing{BasicPrice: 3333, AcceleratorPrice: 9990, VIPPrice: 15990}
	session := &UserSession{PromoCode: "SPARK5", DiscountPct: 5}

	text, _ := service.paymentDetailsContent(session, pricing, true)
	if !strings.Contains(text, "Скидка 5%") {
		t.Errorf("expected the 5%% promo copy, got:\n%s", text)
	}
	if strings.Contains(text, "не действует") {
		t.Errorf("unrestricted promo must not exclude tiers:\n%s", text)
	}
}

func TestPaymentDetailsContent_OmitsDisabledTier(t *testing.T) {
	service, _ := setupTestBot(t)
	pricing := models.CoursePricing{
		BasicPrice:          3333,
		AcceleratorPrice:    9990,
		VIPPrice:            15990,
		BasicEnabled:        true,
		AcceleratorEnabled:  false,
		VIPEnabled:          true,
		TierFlagsSet:        true,
	}

	text, keyboard := service.paymentDetailsContent(&UserSession{}, pricing, false)
	if strings.Contains(text, "Акселератор") {
		t.Errorf("disabled accelerator must not be advertised in the text:\n%s", text)
	}
	if keyboard == nil {
		t.Fatal("expected a keyboard")
	}
	callbacks := map[string]bool{}
	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			callbacks[button.CallbackData] = true
		}
	}
	if callbacks["pay_sbp_accelerator"] {
		t.Error("disabled accelerator must not have a payment button")
	}
	if !callbacks["pay_sbp_basic"] || !callbacks["pay_sbp_vip"] {
		t.Errorf("enabled tiers must keep their payment buttons: %v", callbacks)
	}
}
