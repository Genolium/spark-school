package readmodels

import (
	"database/sql"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func prepareSchema(t *testing.T, dsn string) {
	t.Helper()
	lockPath := filepath.Join(os.TempDir(), "spark-pg-integration.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatal(err)
	}
}
