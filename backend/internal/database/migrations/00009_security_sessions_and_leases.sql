-- +goose Up
CREATE TABLE IF NOT EXISTS revoked_tokens (
    jti VARCHAR(64) PRIMARY KEY,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);

CREATE TABLE IF NOT EXISTS admin_audit_log (
    id BIGSERIAL PRIMARY KEY,
    admin_username VARCHAR(64),
    admin_telegram_id BIGINT,
    method VARCHAR(10) NOT NULL,
    path VARCHAR(255) NOT NULL,
    status INT NOT NULL,
    ip VARCHAR(45),
    request_id VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_created_at ON admin_audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_admin ON admin_audit_log(admin_username);

CREATE TABLE IF NOT EXISTS rate_limits (
    key VARCHAR(255) PRIMARY KEY,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    count INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS bot_sessions (
    telegram_id BIGINT PRIMARY KEY,
    state VARCHAR(32),
    selected_tier VARCHAR(32),
    promo_code VARCHAR(32),
    discount_pct INT DEFAULT 0,
    is_upgrade BOOLEAN DEFAULT FALSE,
    upgrade_from_tier VARCHAR(32),
    upgrade_to_tier VARCHAR(32),
    upgrade_amount NUMERIC(10,2) DEFAULT 0,
    calc_age_citizenship BOOLEAN DEFAULT FALSE,
    calc_study_year VARCHAR(32),
    calc_english_level VARCHAR(32),
    last_activity TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bot_sessions_expires_at ON bot_sessions(expires_at);

CREATE TABLE IF NOT EXISTS scheduler_leases (
    name VARCHAR(64) PRIMARY KEY,
    owner VARCHAR(128) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

ALTER TABLE IF EXISTS drip_leads ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_drip_leads_locked_until ON drip_leads(locked_until);

ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS schema_version INT DEFAULT 1;

-- +goose Down
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS schema_version;
DROP INDEX IF EXISTS idx_drip_leads_locked_until;
ALTER TABLE IF EXISTS drip_leads DROP COLUMN IF EXISTS locked_until;
DROP TABLE IF EXISTS scheduler_leases;
DROP TABLE IF EXISTS bot_sessions;
DROP TABLE IF EXISTS rate_limits;
DROP TABLE IF EXISTS admin_audit_log;
DROP TABLE IF EXISTS revoked_tokens;
