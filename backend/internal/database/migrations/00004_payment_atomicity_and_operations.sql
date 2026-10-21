-- +goose Up
-- Canonical production schema. This migration is safe on both an empty
-- database created by migrations 00001-00003 and on an existing installation
-- that was previously maintained by GORM AutoMigrate and ad-hoc DDL.

-- ---------------------------------------------------------------------------
-- 1. users: entitlement, payment and attribution columns
-- ---------------------------------------------------------------------------
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS tier VARCHAR(32) DEFAULT 'accelerator';
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS paid_amount NUMERIC(10,2) DEFAULT 0.00;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS promo_code VARCHAR(32);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_issued BOOLEAN DEFAULT FALSE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_issued_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_url VARCHAR(512);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_number VARCHAR(64);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS upsell_audit_sent BOOLEAN DEFAULT FALSE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS upsell_audit_sent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS has_audit_upsell BOOLEAN DEFAULT FALSE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS audit_credits INT DEFAULT 0;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS attribution_token VARCHAR(64);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS visitor_id VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_users_attribution_token ON users(attribution_token);
CREATE INDEX IF NOT EXISTS idx_users_visitor_id ON users(visitor_id);
CREATE INDEX IF NOT EXISTS idx_users_receipt_issued ON users(receipt_issued);
CREATE INDEX IF NOT EXISTS idx_users_has_audit_upsell ON users(has_audit_upsell);
CREATE INDEX IF NOT EXISTS idx_users_upsell_audit_sent ON users(upsell_audit_sent);

-- Legacy rows with an upsell flag but no credit counter (one-time only).
UPDATE users SET audit_credits = 1 WHERE has_audit_upsell = TRUE AND audit_credits = 0;
UPDATE users SET tier = 'accelerator' WHERE tier IS NULL OR tier = '';

-- Historical receipts that predate the users table.
INSERT INTO users (telegram_id, username, first_name, role, has_access, tier, paid_amount, created_at)
SELECT DISTINCT pr.telegram_id, pr.username, COALESCE(NULLIF(pr.username, ''), 'Студент'), 'student',
       (pr.status = 'approved'), COALESCE(NULLIF(pr.tier, ''), 'accelerator'),
       CASE WHEN pr.status = 'approved' THEN pr.amount ELSE 0 END,
       pr.created_at
FROM payment_receipts pr
WHERE pr.telegram_id > 0
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.telegram_id = pr.telegram_id);

-- ---------------------------------------------------------------------------
-- 2. payments
-- ---------------------------------------------------------------------------
ALTER TABLE IF EXISTS payment_receipts ADD COLUMN IF NOT EXISTS promo_code VARCHAR(32);
ALTER TABLE IF EXISTS payment_receipts ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE IF EXISTS payment_transactions ADD COLUMN IF NOT EXISTS source VARCHAR(32);
ALTER TABLE IF EXISTS payment_transactions ADD COLUMN IF NOT EXISTS visitor_id VARCHAR(64);
ALTER TABLE IF EXISTS payment_transactions ADD COLUMN IF NOT EXISTS attribution_token VARCHAR(64);
ALTER TABLE IF EXISTS payment_transactions ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_payment_transactions_visitor_id ON payment_transactions(visitor_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_attribution_token ON payment_transactions(attribution_token);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_status ON payment_transactions(status);

CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    order_id VARCHAR(100) UNIQUE NOT NULL,
    telegram_id BIGINT NOT NULL,
    username VARCHAR(64),
    tier VARCHAR(32) NOT NULL,
    amount NUMERIC(10,2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'RUB',
    promo_code VARCHAR(32),
    provider VARCHAR(32) NOT NULL,
    provider_payment_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'pending',
    visitor_id VARCHAR(64),
    attribution_token VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_orders_telegram_id ON payment_orders(telegram_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_status ON payment_orders(status);
CREATE INDEX IF NOT EXISTS idx_payment_orders_provider_payment_id ON payment_orders(provider_payment_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_visitor_id ON payment_orders(visitor_id);
CREATE INDEX IF NOT EXISTS idx_payment_orders_attribution_token ON payment_orders(attribution_token);

-- ---------------------------------------------------------------------------
-- 3. outbox leases and backoff
-- ---------------------------------------------------------------------------
ALTER TABLE IF EXISTS outbox_events ADD COLUMN IF NOT EXISTS available_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS outbox_events ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_outbox_events_locked_until ON outbox_events(locked_until);
CREATE INDEX IF NOT EXISTS idx_outbox_events_available_at ON outbox_events(available_at);

-- ---------------------------------------------------------------------------
-- 4. affiliate commissions per payment
-- ---------------------------------------------------------------------------
ALTER TABLE IF EXISTS referral_transactions ADD COLUMN IF NOT EXISTS payment_id VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_referral_transactions_payment_id ON referral_transactions(payment_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_payment_level
    ON referral_transactions(payment_id, level)
    WHERE payment_id IS NOT NULL AND payment_id <> '';

-- ---------------------------------------------------------------------------
-- 5. pricing, learning operations and analytics
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS course_pricings (
    id BIGSERIAL PRIMARY KEY,
    basic_price NUMERIC(10,2) DEFAULT 3990.00,
    basic_old_price NUMERIC(10,2) DEFAULT 5900.00,
    accelerator_price NUMERIC(10,2) DEFAULT 6900.00,
    accelerator_old_price NUMERIC(10,2) DEFAULT 9900.00,
    vip_price NUMERIC(10,2) DEFAULT 14900.00,
    vip_old_price NUMERIC(10,2) DEFAULT 19900.00,
    bank_name VARCHAR(128) DEFAULT 'БСПБ (Банк «Санкт-Петербург»)',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS student_submissions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    telegram_id BIGINT NOT NULL,
    username VARCHAR(64),
    work_type VARCHAR(32) NOT NULL,
    content TEXT NOT NULL,
    version INT DEFAULT 1,
    status VARCHAR(32) DEFAULT 'pending',
    curator_notes TEXT,
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    reviewed_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_student_submissions_user_id ON student_submissions(user_id);
CREATE INDEX IF NOT EXISTS idx_student_submissions_telegram_id ON student_submissions(telegram_id);
CREATE INDEX IF NOT EXISTS idx_student_submissions_status ON student_submissions(status);

-- Resolve pre-existing duplicate versions before enforcing uniqueness.
WITH duplicates AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY telegram_id, work_type, version ORDER BY id) AS rn
    FROM student_submissions
)
UPDATE student_submissions s
SET version = s.version + 100000
FROM duplicates d
WHERE s.id = d.id AND d.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_student_submission_version
    ON student_submissions(telegram_id, work_type, version);

CREATE TABLE IF NOT EXISTS student_progress (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    step1_done BOOLEAN DEFAULT FALSE,
    step2_done BOOLEAN DEFAULT FALSE,
    step3_done BOOLEAN DEFAULT FALSE,
    step4_done BOOLEAN DEFAULT FALSE,
    step5_done BOOLEAN DEFAULT FALSE,
    step6_done BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS call_bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    telegram_id BIGINT NOT NULL,
    username VARCHAR(64),
    slot_key VARCHAR(64) NOT NULL,
    slot_time VARCHAR(128) NOT NULL,
    call_type VARCHAR(64) DEFAULT 'zoom_simulation',
    status VARCHAR(32) DEFAULT 'confirmed',
    zoom_link VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    cancelled_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_call_bookings_user_id ON call_bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_call_bookings_telegram_id ON call_bookings(telegram_id);
CREATE INDEX IF NOT EXISTS idx_call_bookings_slot_key ON call_bookings(slot_key);
CREATE INDEX IF NOT EXISTS idx_call_bookings_status ON call_bookings(status);

-- Resolve pre-existing double bookings before enforcing the partial indexes.
-- The oldest confirmed booking wins; later duplicates are cancelled.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY slot_key ORDER BY id) AS rn
    FROM call_bookings
    WHERE status = 'confirmed' AND cancelled_at IS NULL
)
UPDATE call_bookings b
SET status = 'cancelled', cancelled_at = NOW()
FROM ranked r
WHERE b.id = r.id AND r.rn > 1;

WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY telegram_id ORDER BY id) AS rn
    FROM call_bookings
    WHERE status = 'confirmed' AND cancelled_at IS NULL
)
UPDATE call_bookings b
SET status = 'cancelled', cancelled_at = NOW()
FROM ranked r
WHERE b.id = r.id AND r.rn > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_active_slot_key
    ON call_bookings(slot_key)
    WHERE status = 'confirmed' AND cancelled_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_active_user_booking
    ON call_bookings(telegram_id)
    WHERE status = 'confirmed' AND cancelled_at IS NULL;

CREATE TABLE IF NOT EXISTS analytics_events (
    id BIGSERIAL PRIMARY KEY,
    visitor_id VARCHAR(64),
    step VARCHAR(64) NOT NULL,
    metadata TEXT,
    ip VARCHAR(45),
    user_agent VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_analytics_events_visitor_id ON analytics_events(visitor_id);
CREATE INDEX IF NOT EXISTS idx_analytics_events_step ON analytics_events(step);
CREATE INDEX IF NOT EXISTS idx_analytics_events_created_at ON analytics_events(created_at);

-- ---------------------------------------------------------------------------
-- 6. Telegram update claims
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS processed_telegram_updates (
    update_id BIGINT PRIMARY KEY,
    status VARCHAR(20) DEFAULT 'processing',
    attempts INT DEFAULT 1,
    locked_until TIMESTAMP WITH TIME ZONE,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
ALTER TABLE IF EXISTS processed_telegram_updates ADD COLUMN IF NOT EXISTS status VARCHAR(20);
ALTER TABLE IF EXISTS processed_telegram_updates ADD COLUMN IF NOT EXISTS attempts INT DEFAULT 1;
ALTER TABLE IF EXISTS processed_telegram_updates ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS processed_telegram_updates ADD COLUMN IF NOT EXISTS last_error TEXT;
ALTER TABLE IF EXISTS processed_telegram_updates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
-- Rows written by the old implementation were successfully processed.
UPDATE processed_telegram_updates SET status = 'completed' WHERE status IS NULL OR status = '';
ALTER TABLE IF EXISTS processed_telegram_updates ALTER COLUMN status SET DEFAULT 'processing';
CREATE INDEX IF NOT EXISTS idx_processed_updates_status ON processed_telegram_updates(status);
CREATE INDEX IF NOT EXISTS idx_processed_updates_locked_until ON processed_telegram_updates(locked_until);

-- ---------------------------------------------------------------------------
-- 7. One-time bookkeeping marker
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS app_meta (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS app_meta;
DROP INDEX IF EXISTS idx_processed_updates_locked_until;
DROP INDEX IF EXISTS idx_processed_updates_status;
DROP TABLE IF EXISTS processed_telegram_updates;
DROP INDEX IF EXISTS idx_analytics_events_created_at;
DROP INDEX IF EXISTS idx_analytics_events_step;
DROP INDEX IF EXISTS idx_analytics_events_visitor_id;
DROP TABLE IF EXISTS analytics_events;
DROP INDEX IF EXISTS idx_active_user_booking;
DROP INDEX IF EXISTS idx_active_slot_key;
DROP INDEX IF EXISTS idx_call_bookings_status;
DROP INDEX IF EXISTS idx_call_bookings_slot_key;
DROP INDEX IF EXISTS idx_call_bookings_telegram_id;
DROP INDEX IF EXISTS idx_call_bookings_user_id;
DROP TABLE IF EXISTS call_bookings;
DROP TABLE IF EXISTS student_progress;
DROP INDEX IF EXISTS idx_student_submission_version;
DROP INDEX IF EXISTS idx_student_submissions_status;
DROP INDEX IF EXISTS idx_student_submissions_telegram_id;
DROP INDEX IF EXISTS idx_student_submissions_user_id;
DROP TABLE IF EXISTS student_submissions;
DROP TABLE IF EXISTS course_pricings;
DROP INDEX IF EXISTS idx_referral_payment_level;
DROP INDEX IF EXISTS idx_referral_transactions_payment_id;
ALTER TABLE IF EXISTS referral_transactions DROP COLUMN IF EXISTS payment_id;
DROP INDEX IF EXISTS idx_outbox_events_available_at;
DROP INDEX IF EXISTS idx_outbox_events_locked_until;
ALTER TABLE IF EXISTS outbox_events DROP COLUMN IF EXISTS available_at;
ALTER TABLE IF EXISTS outbox_events DROP COLUMN IF EXISTS locked_until;
DROP INDEX IF EXISTS idx_payment_orders_attribution_token;
DROP INDEX IF EXISTS idx_payment_orders_visitor_id;
DROP INDEX IF EXISTS idx_payment_orders_provider_payment_id;
DROP INDEX IF EXISTS idx_payment_orders_status;
DROP INDEX IF EXISTS idx_payment_orders_telegram_id;
DROP TABLE IF EXISTS payment_orders;
DROP INDEX IF EXISTS idx_payment_transactions_status;
DROP INDEX IF EXISTS idx_payment_transactions_attribution_token;
DROP INDEX IF EXISTS idx_payment_transactions_visitor_id;
ALTER TABLE IF EXISTS payment_transactions DROP COLUMN IF EXISTS confirmed_at;
ALTER TABLE IF EXISTS payment_transactions DROP COLUMN IF EXISTS attribution_token;
ALTER TABLE IF EXISTS payment_transactions DROP COLUMN IF EXISTS visitor_id;
ALTER TABLE IF EXISTS payment_transactions DROP COLUMN IF EXISTS source;
ALTER TABLE IF EXISTS payment_receipts DROP COLUMN IF EXISTS confirmed_at;
ALTER TABLE IF EXISTS payment_receipts DROP COLUMN IF EXISTS promo_code;
DROP INDEX IF EXISTS idx_users_upsell_audit_sent;
DROP INDEX IF EXISTS idx_users_has_audit_upsell;
DROP INDEX IF EXISTS idx_users_receipt_issued;
DROP INDEX IF EXISTS idx_users_visitor_id;
DROP INDEX IF EXISTS idx_users_attribution_token;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS visitor_id;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS attribution_token;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS audit_credits;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS has_audit_upsell;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS upsell_audit_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS upsell_audit_sent;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS receipt_number;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS receipt_url;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS receipt_issued_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS receipt_issued;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS promo_code;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS paid_amount;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS tier;
