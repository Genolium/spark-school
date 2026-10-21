-- +goose Up
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS reminders_opt_out BOOLEAN DEFAULT FALSE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS diagnostic_started_reminder_sent BOOLEAN DEFAULT FALSE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS diagnostic_completed_reminder_sent BOOLEAN DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_reminders_opt_out ON users(reminders_opt_out);
CREATE INDEX IF NOT EXISTS idx_users_diag_started_sent ON users(diagnostic_started_reminder_sent);
CREATE INDEX IF NOT EXISTS idx_users_diag_completed_sent ON users(diagnostic_completed_reminder_sent);

CREATE TABLE IF NOT EXISTS order_intents (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    telegram_id BIGINT NOT NULL,
    tier VARCHAR(32) NOT NULL,
    amount NUMERIC(10,2) NOT NULL,
    promo_code VARCHAR(32),
    source VARCHAR(32),
    status VARCHAR(20) DEFAULT 'pending',
    payment_id VARCHAR(64),
    reminder_count INT DEFAULT 0,
    last_reminder_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_order_intents_user_id ON order_intents(user_id);
CREATE INDEX IF NOT EXISTS idx_order_intents_telegram_id ON order_intents(telegram_id);
CREATE INDEX IF NOT EXISTS idx_order_intents_status ON order_intents(status);
CREATE INDEX IF NOT EXISTS idx_order_intents_payment_id ON order_intents(payment_id);

CREATE TABLE IF NOT EXISTS experiment_assignments (
    id BIGSERIAL PRIMARY KEY,
    experiment VARCHAR(64) NOT NULL,
    variant VARCHAR(64) NOT NULL,
    visitor_id VARCHAR(64),
    attribution_token VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_experiment_assignments_experiment ON experiment_assignments(experiment);
CREATE INDEX IF NOT EXISTS idx_experiment_assignments_visitor_id ON experiment_assignments(visitor_id);
CREATE INDEX IF NOT EXISTS idx_experiment_assignments_attribution_token ON experiment_assignments(attribution_token);
CREATE UNIQUE INDEX IF NOT EXISTS idx_experiment_visitor_unique
    ON experiment_assignments(experiment, visitor_id)
    WHERE visitor_id IS NOT NULL AND visitor_id <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_experiment_visitor_unique;
DROP INDEX IF EXISTS idx_experiment_assignments_attribution_token;
DROP INDEX IF EXISTS idx_experiment_assignments_visitor_id;
DROP INDEX IF EXISTS idx_experiment_assignments_experiment;
DROP TABLE IF EXISTS experiment_assignments;
DROP INDEX IF EXISTS idx_order_intents_payment_id;
DROP INDEX IF EXISTS idx_order_intents_status;
DROP INDEX IF EXISTS idx_order_intents_telegram_id;
DROP INDEX IF EXISTS idx_order_intents_user_id;
DROP TABLE IF EXISTS order_intents;
DROP INDEX IF EXISTS idx_users_diag_completed_sent;
DROP INDEX IF EXISTS idx_users_diag_started_sent;
DROP INDEX IF EXISTS idx_users_reminders_opt_out;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS diagnostic_completed_reminder_sent;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS diagnostic_started_reminder_sent;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS reminders_opt_out;
