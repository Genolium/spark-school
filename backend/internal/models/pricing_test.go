package models

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestDefaultCoursePricingUsesConstants(t *testing.T) {
	d := DefaultCoursePricing
	if d.BasicPrice != DefaultBasicPrice || d.AcceleratorPrice != DefaultAcceleratorPrice || d.VIPPrice != DefaultVIPPrice {
		t.Fatalf("DefaultCoursePricing = %+v does not match Default* constants", d)
	}
	cases := map[string]float64{
		"basic":       DefaultBasicPrice,
		" Базовый ":   DefaultBasicPrice,
		"vip":         DefaultVIPPrice,
		"accelerator": DefaultAcceleratorPrice,
		"unknown":     DefaultAcceleratorPrice,
	}
	for tier, want := range cases {
		if got := DefaultTierPrice(tier); got != want {
			t.Errorf("DefaultTierPrice(%q) = %v, want %v", tier, got, want)
		}
	}
	if got := DefaultUpgradeDiff("basic", "vip"); got != DefaultVIPPrice-DefaultBasicPrice {
		t.Errorf("DefaultUpgradeDiff(basic, vip) = %v", got)
	}
	if got := DefaultUpgradeDiff("vip", "basic"); got != 0 {
		t.Errorf("DefaultUpgradeDiff(vip, basic) = %v, want 0", got)
	}
}

func TestGetCoursePricingNilDBReturnsDefaults(t *testing.T) {
	if got := GetCoursePricing(nil); got != DefaultCoursePricing {
		t.Fatalf("GetCoursePricing(nil) = %+v, want %+v", got, DefaultCoursePricing)
	}
}

// TestDefaultCoursePricingMatchesFrontendDefaults fails when the backend
// default prices and frontend/src/lib/pricing-defaults.ts diverge.
func TestDefaultCoursePricingMatchesFrontendDefaults(t *testing.T) {
	path := filepath.Join("..", "..", "..", "frontend", "src", "lib", "pricing-defaults.ts")
	src, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skipf("frontend sources not available at %s", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{
		"basic_price":           DefaultBasicPrice,
		"basic_old_price":       DefaultBasicOldPrice,
		"accelerator_price":     DefaultAcceleratorPrice,
		"accelerator_old_price": DefaultAcceleratorOldPrice,
		"vip_price":             DefaultVIPPrice,
		"vip_old_price":         DefaultVIPOldPrice,
	}
	for field, backend := range want {
		m := regexp.MustCompile(`(?m)^\s*` + field + `:\s*([0-9.]+)`).FindSubmatch(src)
		if m == nil {
			t.Errorf("%s not found in %s", field, path)
			continue
		}
		frontend, err := strconv.ParseFloat(string(m[1]), 64)
		if err != nil {
			t.Errorf("%s: %v", field, err)
			continue
		}
		if frontend != backend {
			t.Errorf("%s: frontend default %v != backend default %v", field, frontend, backend)
		}
	}
}
