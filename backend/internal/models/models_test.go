package models

import "testing"

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
