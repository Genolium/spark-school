package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// prepareSchema serializes PostgreSQL integration tests across test binaries
// with a file lock, then resets the public schema. `go test ./...` runs
// packages in parallel; without this lock one package would drop tables while
// another is still using them.
func prepareSchema(t *testing.T, dsn string) {
	t.Helper()
	lockPath := filepath.Join(os.TempDir(), "spark-pg-integration.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("open test lock: %v", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		t.Fatalf("lock test database: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
}
