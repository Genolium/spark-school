-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_url VARCHAR(512);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS receipt_number VARCHAR(64);

CREATE TABLE IF NOT EXISTS fiscal_receipts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    telegram_id BIGINT NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    tier VARCHAR(32) DEFAULT 'accelerator',
    receipt_url VARCHAR(512),
    receipt_number VARCHAR(64),
    service_name VARCHAR(255),
    sent_to_bot BOOLEAN DEFAULT FALSE,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fiscal_receipts_user_id ON fiscal_receipts(user_id);
CREATE INDEX IF NOT EXISTS idx_fiscal_receipts_telegram_id ON fiscal_receipts(telegram_id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS fiscal_receipts;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS receipt_number;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS receipt_url;
