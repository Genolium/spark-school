-- +goose Up
CREATE TABLE IF NOT EXISTS diagnostics (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(32) UNIQUE NOT NULL,
    visitor_id VARCHAR(64),
    attribution_token VARCHAR(64),
    telegram_id BIGINT,
    age_citizenship BOOLEAN DEFAULT FALSE,
    university_year VARCHAR(32),
    english_level VARCHAR(16),
    score INT DEFAULT 0,
    source VARCHAR(64),
    linked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_diagnostics_visitor_id ON diagnostics(visitor_id);
CREATE INDEX IF NOT EXISTS idx_diagnostics_attribution_token ON diagnostics(attribution_token);
CREATE INDEX IF NOT EXISTS idx_diagnostics_telegram_id ON diagnostics(telegram_id);

-- +goose Down
DROP TABLE IF EXISTS diagnostics;
