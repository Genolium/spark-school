package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// RunMigrations executes all pending migrations using Goose against the SQL database.
func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	log.Println("[Database] Checking and running database migrations with Goose...")
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run goose up: %w", err)
	}

	log.Println("[Database] Migrations applied successfully.")
	return nil
}
