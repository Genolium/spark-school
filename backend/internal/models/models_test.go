package models

import (
	"testing"
	"time"
)

// The backup worker and migration 00008 create "backup_heartbeat" (singular).
// GORM would otherwise pluralize it to "backup_heartbeats" and monitoring
// would fail with "relation does not exist".
func TestBackupHeartbeatTableName(t *testing.T) {
	if got := (BackupHeartbeat{}).TableName(); got != "backup_heartbeat" {
		t.Fatalf("BackupHeartbeat table = %q, want %q", got, "backup_heartbeat")
	}
}

func TestCoursePricingTierAvailability(t *testing.T) {
	legacy := CoursePricing{}
	if !legacy.TierEnabled("accelerator") {
		t.Fatal("legacy CoursePricing without TierFlagsSet must keep tiers enabled")
	}

	pricing := CoursePricing{
		BasicEnabled:       true,
		AcceleratorEnabled: false,
		VIPEnabled:         true,
		TierFlagsSet:       true,
	}
	if pricing.TierEnabled("accelerator") {
		t.Fatal("accelerator must be disabled")
	}
	if !pricing.TierEnabled("basic") || !pricing.TierEnabled("vip") {
		t.Fatal("basic and vip must stay enabled")
	}
	if got := pricing.FirstEnabledTier(); got != "basic" {
		t.Fatalf("FirstEnabledTier = %q, want basic", got)
	}

	none := CoursePricing{TierFlagsSet: true}
	if got := none.FirstEnabledTier(); got != "" {
		t.Fatalf("FirstEnabledTier with all tiers off = %q, want empty", got)
	}
}

func TestCoursePricingSalesCloseAtAndBasicFeedbackPromo(t *testing.T) {
	closeAt := DefaultSalesCloseAt()
	if closeAt == nil {
		t.Fatal("DefaultSalesCloseAt must not be nil")
	}
	pricing := CoursePricing{
		BasicEnabled:         true,
		BasicFeedbackEnabled: true,
		AcceleratorEnabled:   true,
		VIPEnabled:           true,
		SalesCloseAt:         closeAt,
		TierFlagsSet:         true,
	}

	beforeClose := closeAt.Add(-time.Second)
	if pricing.IsSalesClosedAt(beforeClose) {
		t.Fatal("sales must remain open before 2026-11-21 00:00 MSK")
	}
	if !pricing.TierEnabledAt("basic_feedback", beforeClose) {
		t.Fatal("basic_feedback must be enabled before sales_close_at")
	}

	atClose := *closeAt
	if !pricing.IsSalesClosedAt(atClose) {
		t.Fatal("sales must be closed at 2026-11-21 00:00:00 MSK")
	}
	for _, tier := range []string{"basic", "basic_feedback", "accelerator", "vip"} {
		if pricing.TierEnabledAt(tier, atClose) {
			t.Fatalf("tier %q must be disabled at sales_close_at", tier)
		}
	}

	promo := PromoCode{Code: "SPARK5", DiscountPercent: 5, IsActive: true}
	if promo.AppliesToTier("basic_feedback") {
		t.Fatal("promo codes must not apply to standalone basic_feedback offer")
	}
	if !promo.AppliesToTier("basic") || !promo.AppliesToTier("accelerator") {
		t.Fatal("unrestricted promo must still apply to basic and accelerator")
	}
}
