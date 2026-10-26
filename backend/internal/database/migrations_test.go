package database

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func migrationTestSQLDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL migration test")
	}
	prepareSchema(t, dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

func TestPostgresMigrations_EmptyDatabaseAndRerun(t *testing.T) {
	sqlDB := migrationTestSQLDB(t)

	if err := RunMigrations(sqlDB); err != nil {
		t.Fatalf("migrate empty database: %v", err)
	}
	// Re-running must be a no-op and must not fail.
	if err := RunMigrations(sqlDB); err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}

	var version int64
	if err := sqlDB.QueryRow("SELECT MAX(version_id) FROM goose_db_version").Scan(&version); err != nil {
		t.Fatalf("goose version: %v", err)
	}
	if version < 14 {
		t.Fatalf("expected at least migration 14, got %d", version)
	}

	for _, table := range []string{
		"users", "payment_receipts", "payment_transactions", "payment_orders",
		"outbox_events", "referral_transactions", "student_submissions",
		"student_progress", "call_bookings", "analytics_events",
		"processed_telegram_updates", "course_pricings", "app_meta",
		"order_intents", "experiment_assignments", "backup_heartbeat",
		"revoked_tokens", "admin_audit_log", "rate_limits", "bot_sessions", "scheduler_leases",
		"admin_revenue_daily", "diagnostics", "campaign_links", "campaign_link_clicks",
	} {
		var exists bool
		if err := sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)", table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("expected table %s", table)
		}
	}

	var kopecksColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND column_name='amount_kopecks' AND table_name IN ('payment_receipts','payment_transactions','payment_orders','referral_transactions','payout_requests')`).Scan(&kopecksColumns); err != nil {
		t.Fatal(err)
	}
	if kopecksColumns != 5 {
		t.Errorf("expected amount_kopecks on five financial tables, got %d", kopecksColumns)
	}
	var paidKopecks int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name IN ('paid_amount_kopecks','anonymized_at')`).Scan(&paidKopecks); err != nil {
		t.Fatal(err)
	}
	if paidKopecks != 2 {
		t.Errorf("expected user kopeck/anonymization columns, got %d", paidKopecks)
	}

	var priceExperimentColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema='public' AND table_name='bot_sessions' AND column_name='price_experiment'`).Scan(&priceExperimentColumns); err != nil {
		t.Fatal(err)
	}
	if priceExperimentColumns != 1 {
		t.Errorf("expected bot_sessions.price_experiment, got %d", priceExperimentColumns)
	}

	var vipPriceExperimentColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema='public' AND table_name='bot_sessions' AND column_name='vip_price_experiment'`).Scan(&vipPriceExperimentColumns); err != nil {
		t.Fatal(err)
	}
	if vipPriceExperimentColumns != 1 {
		t.Errorf("expected bot_sessions.vip_price_experiment, got %d", vipPriceExperimentColumns)
	}

	var onboardingColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name IN ('onboarding_status','intro_sent_at','onboarding_replied_at','onboarding_completed_at','onboarding_notes')`).Scan(&onboardingColumns); err != nil {
		t.Fatal(err)
	}
	if onboardingColumns != 0 {
		t.Errorf("onboarding columns should be removed, got %d", onboardingColumns)
	}

	var recoveryColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name IN ('reminders_opt_out','diagnostic_started_reminder_sent','diagnostic_completed_reminder_sent')`).Scan(&recoveryColumns); err != nil {
		t.Fatal(err)
	}
	if recoveryColumns != 3 {
		t.Errorf("expected recovery columns on users, got %d", recoveryColumns)
	}

	var reminderColumns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='call_bookings' AND column_name IN ('reminder_24_sent','reminder_1_sent')`).Scan(&reminderColumns); err != nil {
		t.Fatal(err)
	}
	if reminderColumns != 2 {
		t.Errorf("expected both Zoom reminder columns, got %d", reminderColumns)
	}

	var partitionName string
	if _, err := sqlDB.Exec(`INSERT INTO analytics_events (step, created_at) VALUES ('landing_view', NOW())`); err != nil {
		t.Fatalf("insert into partitioned analytics: %v", err)
	}
	if err := sqlDB.QueryRow(`SELECT tableoid::regclass::text FROM analytics_events WHERE step='landing_view' ORDER BY created_at DESC LIMIT 1`).Scan(&partitionName); err != nil {
		t.Fatal(err)
	}
	if partitionName == "analytics_events_default" {
		t.Errorf("current event landed in the default partition")
	}
	t.Logf("analytics partition: %s", partitionName)

	for _, index := range []string{
		"idx_active_slot_key", "idx_active_user_booking",
		"idx_student_submission_version", "idx_referral_payment_level",
		"idx_campaign_clicks_created_at",
	} {
		var exists bool
		if err := sqlDB.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname='public' AND indexname=$1)", index).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("expected index %s", index)
		}
	}

	var slugUnique bool
	if err := sqlDB.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM pg_index i
		JOIN pg_class t ON t.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(i.indkey)
		WHERE n.nspname = 'public' AND t.relname = 'campaign_links'
		  AND i.indisunique AND a.attname = 'slug')`).Scan(&slugUnique); err != nil {
		t.Fatal(err)
	}
	if !slugUnique {
		t.Errorf("expected a unique index on campaign_links.slug")
	}
}

func TestPostgresMigrations_UpgradeOldSchemaWithDuplicates(t *testing.T) {
	sqlDB := migrationTestSQLDB(t)
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(sqlDB, "migrations", 3); err != nil {
		t.Fatalf("apply old schema: %v", err)
	}

	now := time.Now()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := sqlDB.Exec(query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}
	// Simulate tables that were previously created by GORM AutoMigrate.
	mustExec(`CREATE TABLE IF NOT EXISTS call_bookings (
		id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, telegram_id BIGINT NOT NULL,
		username VARCHAR(64), slot_key VARCHAR(64) NOT NULL, slot_time VARCHAR(128) NOT NULL,
		call_type VARCHAR(64), status VARCHAR(32), zoom_link VARCHAR(255),
		created_at TIMESTAMPTZ DEFAULT NOW(), cancelled_at TIMESTAMPTZ)`)
	mustExec(`CREATE TABLE IF NOT EXISTS student_submissions (
		id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, telegram_id BIGINT NOT NULL,
		username VARCHAR(64), work_type VARCHAR(32) NOT NULL, content TEXT NOT NULL,
		version INT DEFAULT 1, status VARCHAR(32), curator_notes TEXT,
		submitted_at TIMESTAMPTZ DEFAULT NOW(), reviewed_at TIMESTAMPTZ)`)
	mustExec(`CREATE TABLE IF NOT EXISTS processed_telegram_updates (
		update_id BIGINT PRIMARY KEY, created_at TIMESTAMPTZ DEFAULT NOW())`)
	mustExec(`INSERT INTO users (telegram_id, first_name, has_access) VALUES (501, 'Old', TRUE)`)
	// Two active bookings for the same slot and two for the same user.
	mustExec(`INSERT INTO call_bookings (user_id, telegram_id, slot_key, slot_time, status, created_at) VALUES
		(1, 501, '20261101_1800', 'slot', 'confirmed', $1),
		(1, 502, '20261101_1800', 'slot', 'confirmed', $1),
		(1, 501, '20261102_1800', 'slot2', 'confirmed', $1)`, now)
	mustExec(`INSERT INTO student_submissions (user_id, telegram_id, work_type, content, version, status)
		VALUES (1, 501, 'essay', 'a', 1, 'pending'), (1, 501, 'essay', 'b', 1, 'pending')`)
	mustExec(`INSERT INTO payment_receipts (file_hash, telegram_id, amount, tier, status, created_at)
		VALUES ('legacy-hash', 501, 6900, 'accelerator', 'approved', $1)`, now)
	mustExec(`INSERT INTO processed_telegram_updates (update_id, created_at) VALUES (9001, $1)`, now)

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		t.Fatalf("upgrade old schema: %v", err)
	}

	var active int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM call_bookings WHERE status='confirmed' AND cancelled_at IS NULL").Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("expected exactly one active booking after deduplication, got %d", active)
	}

	var uniqueVersions int
	if err := sqlDB.QueryRow("SELECT COUNT(DISTINCT version) FROM student_submissions WHERE telegram_id=501 AND work_type='essay'").Scan(&uniqueVersions); err != nil {
		t.Fatal(err)
	}
	if uniqueVersions != 2 {
		t.Fatalf("expected duplicate versions to be renumbered, got %d distinct", uniqueVersions)
	}

	var status string
	if err := sqlDB.QueryRow("SELECT status FROM processed_telegram_updates WHERE update_id=9001").Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "completed" {
		t.Fatalf("legacy processed update must be marked completed, got %q", status)
	}

	// The approved legacy receipt must still be present and payment tables
	// must accept the new columns.
	var columns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='payment_transactions' AND column_name IN ('source','visitor_id','attribution_token','confirmed_at')`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 4 {
		t.Fatalf("expected payment_transactions new columns, got %d", columns)
	}
}

func TestPostgresMigrations_RepairsLegacyCoursePricingSchema(t *testing.T) {
	sqlDB := migrationTestSQLDB(t)
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(sqlDB, "migrations", 3); err != nil {
		t.Fatalf("apply old schema: %v", err)
	}

	// This is the production drift that broke the previous deploy: an old
	// GORM-created table without vip_price. CREATE TABLE IF NOT EXISTS in
	// migration 00004 does not add it, so migration 00019 must repair it.
	if _, err := sqlDB.Exec(`CREATE TABLE course_pricings (
		id BIGSERIAL PRIMARY KEY,
		basic_price NUMERIC(10,2) DEFAULT 3990.00,
		accelerator_price NUMERIC(10,2) DEFAULT 6900.00,
		updated_at TIMESTAMPTZ DEFAULT NOW())`); err != nil {
		t.Fatalf("create legacy pricing table: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO course_pricings (basic_price, accelerator_price) VALUES (3990, 6900)`); err != nil {
		t.Fatalf("seed legacy pricing row: %v", err)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		t.Fatalf("upgrade legacy pricing schema: %v", err)
	}

	var columns int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema='public' AND table_name='course_pricings'
		  AND column_name IN ('vip_price','vip_old_price','basic_old_price','accelerator_old_price','bank_name')`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 5 {
		t.Fatalf("expected legacy pricing columns to be repaired, got %d/5", columns)
	}

	var vip float64
	if err := sqlDB.QueryRow(`SELECT vip_price FROM course_pricings ORDER BY id LIMIT 1`).Scan(&vip); err != nil {
		t.Fatal(err)
	}
	if vip <= 0 {
		t.Fatalf("expected repaired vip_price, got %v", vip)
	}

	db, err := gorm.Open(postgres.Open(os.Getenv("TEST_DATABASE_URL")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateSchema(db); err != nil {
		t.Fatalf("schema validation after repair: %v", err)
	}
}

func TestPostgresMigrations_BackfillsPaymentConfirmedEvents(t *testing.T) {
	sqlDB := migrationTestSQLDB(t)
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(sqlDB, "migrations", 20); err != nil {
		t.Fatalf("apply schema up to 20: %v", err)
	}

	now := time.Now()
	if _, err := sqlDB.Exec(`INSERT INTO users (telegram_id, first_name, visitor_id) VALUES (777, 'Legacy', 'visitor-legacy')`); err != nil {
		t.Fatalf("seed legacy user: %v", err)
	}
	// A receipt reconciled by the legacy code path: a financial operation
	// without the matching funnel event.
	if _, err := sqlDB.Exec(`INSERT INTO payment_transactions
		(payment_id, order_id, telegram_id, tier, amount, amount_kopecks, status, currency, pay_method, source, created_at)
		VALUES ('RECEIPT_LEGACY', 'RECEIPT_LEGACY', 777, 'accelerator', 6900, 690000, 'SUCCESS', 'RUB', 'sbp_receipt', 'sbp_receipt', $1)`, now); err != nil {
		t.Fatalf("seed legacy payment: %v", err)
	}
	// Manual grants are not funnel confirmations and must stay untouched.
	if _, err := sqlDB.Exec(`INSERT INTO payment_transactions
		(payment_id, telegram_id, tier, amount, status, currency, pay_method, source, created_at)
		VALUES ('MANUAL_LEGACY', 777, 'accelerator', 6900, 'SUCCESS', 'RUB', 'admin_manual', 'manual', $1)`, now); err != nil {
		t.Fatalf("seed manual payment: %v", err)
	}

	if err := goose.Up(sqlDB, "migrations"); err != nil {
		t.Fatalf("upgrade to latest: %v", err)
	}

	var step, tier, visitorID, metadata string
	if err := sqlDB.QueryRow(`SELECT step, tier, COALESCE(visitor_id,''), COALESCE(metadata,'')
		FROM analytics_events WHERE payment_id='RECEIPT_LEGACY' AND step='payment_confirmed'`).
		Scan(&step, &tier, &visitorID, &metadata); err != nil {
		t.Fatalf("expected a backfilled payment_confirmed event: %v", err)
	}
	if tier != "accelerator" {
		t.Errorf("expected tier accelerator on the backfilled event, got %q", tier)
	}
	if visitorID != "visitor-legacy" {
		t.Errorf("expected the buyer attribution to be restored, got %q", visitorID)
	}
	if !strings.Contains(metadata, "RECEIPT_LEGACY") {
		t.Errorf("expected payment id in metadata, got %q", metadata)
	}

	var manualEvents int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM analytics_events WHERE payment_id='MANUAL_LEGACY'`).Scan(&manualEvents); err != nil {
		t.Fatal(err)
	}
	if manualEvents != 0 {
		t.Fatalf("manual grants must not become funnel events, got %d", manualEvents)
	}
}

func TestPostgresMigrations_ReadinessChecksVersion(t *testing.T) {
	sqlDB := migrationTestSQLDB(t)
	if err := RunMigrations(sqlDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db, err := gorm.Open(postgres.Open(os.Getenv("TEST_DATABASE_URL")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	version, err := Ready(context.Background(), db)
	if err != nil {
		t.Fatalf("expected ready database: %v", err)
	}
	if version < RequiredSchemaVersion {
		t.Fatalf("expected version >= %d, got %d", RequiredSchemaVersion, version)
	}
	if err := ValidateSchema(db); err != nil {
		t.Fatalf("expected schema validation to pass: %v", err)
	}
}

func TestPostgresMigrations_PartitionPreservesData(t *testing.T) {
	sqlDB := migrationTestSQLDB(t)
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(sqlDB, "migrations", 11); err != nil {
		t.Fatalf("apply pre-partition schema: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO analytics_events (visitor_id, step, metadata, created_at)
		VALUES ('visitor-old', 'landing_view', '{}', NOW() - INTERVAL '2 days')`); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		t.Fatalf("apply partition migration: %v", err)
	}
	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM analytics_events WHERE visitor_id='visitor-old'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("old analytics event was lost, got %d", count)
	}
	if _, err := sqlDB.Exec(`INSERT INTO analytics_events (visitor_id, step, created_at) VALUES ('visitor-new','landing_view',NOW())`); err != nil {
		t.Fatal(err)
	}
	var partition string
	if err := sqlDB.QueryRow(`SELECT tableoid::regclass::text FROM analytics_events WHERE visitor_id='visitor-new'`).Scan(&partition); err != nil {
		t.Fatal(err)
	}
	if partition == "analytics_events_default" {
		t.Fatalf("new event landed in default partition")
	}
}
