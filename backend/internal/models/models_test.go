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
