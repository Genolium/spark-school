package database

import (
	"math"
	"testing"

	"github.com/pressly/goose/v3"
)

// TestEmbeddedMigrationsValid guarantees that all embedded SQL migrations have
// unique sequential versions and can be collected by Goose without panicking.
func TestEmbeddedMigrationsValid(t *testing.T) {
	goose.SetBaseFS(embedMigrations)
	migrations, err := goose.CollectMigrations("migrations", 0, math.MaxInt64)
	if err != nil {
		t.Fatalf("goose.CollectMigrations failed: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected embedded migrations, got 0")
	}
	last := migrations[len(migrations)-1].Version
	if last < RequiredSchemaVersion {
		t.Fatalf("latest migration version %d is older than RequiredSchemaVersion %d", last, RequiredSchemaVersion)
	}
}
