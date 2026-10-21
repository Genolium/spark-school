-- +goose Up
-- Integer-kopeck shadow columns. Existing ruble columns stay for API/JSON
-- compatibility; all new writes populate both.
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS paid_amount_kopecks BIGINT DEFAULT 0;
ALTER TABLE IF EXISTS payment_receipts ADD COLUMN IF NOT EXISTS amount_kopecks BIGINT DEFAULT 0;
ALTER TABLE IF EXISTS payment_transactions ADD COLUMN IF NOT EXISTS amount_kopecks BIGINT DEFAULT 0;
ALTER TABLE IF EXISTS payment_orders ADD COLUMN IF NOT EXISTS amount_kopecks BIGINT DEFAULT 0;
ALTER TABLE IF EXISTS referral_transactions ADD COLUMN IF NOT EXISTS amount_kopecks BIGINT DEFAULT 0;
ALTER TABLE IF EXISTS payout_requests ADD COLUMN IF NOT EXISTS amount_kopecks BIGINT DEFAULT 0;

UPDATE users SET paid_amount_kopecks = ROUND(paid_amount * 100) WHERE paid_amount_kopecks = 0 AND paid_amount <> 0;
UPDATE payment_receipts SET amount_kopecks = ROUND(amount * 100) WHERE amount_kopecks = 0 AND amount <> 0;
UPDATE payment_transactions SET amount_kopecks = ROUND(amount * 100) WHERE amount_kopecks = 0 AND amount <> 0;
UPDATE payment_orders SET amount_kopecks = ROUND(amount * 100) WHERE amount_kopecks = 0 AND amount <> 0;
UPDATE referral_transactions SET amount_kopecks = ROUND(amount * 100) WHERE amount_kopecks = 0 AND amount <> 0;
UPDATE payout_requests SET amount_kopecks = ROUND(amount * 100) WHERE amount_kopecks = 0 AND amount <> 0;

-- Link a confirmed analytics event to the financial operation.
ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS payment_id VARCHAR(64);
UPDATE analytics_events
SET payment_id = substring(metadata FROM '"payment_id"\s*:\s*"([^"]+)"')
WHERE step = 'payment_confirmed' AND payment_id IS NULL AND metadata LIKE '%"payment_id"%';
CREATE INDEX IF NOT EXISTS idx_analytics_events_payment_id ON analytics_events(payment_id);

-- Privacy: allow anonymization without destroying financial history.
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS anonymized_at TIMESTAMP WITH TIME ZONE;

-- Backup worker heartbeat. The backup container creates this table itself for
-- the very first run before migrations are applied.
CREATE TABLE IF NOT EXISTS backup_heartbeat (
    id INT PRIMARY KEY,
    last_success_at TIMESTAMP WITH TIME ZONE NOT NULL,
    file_name VARCHAR(255),
    size_bytes BIGINT
);

-- +goose Down
DROP TABLE IF EXISTS backup_heartbeat;
DROP INDEX IF EXISTS idx_analytics_events_payment_id;
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS payment_id;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS anonymized_at;
ALTER TABLE IF EXISTS payout_requests DROP COLUMN IF EXISTS amount_kopecks;
ALTER TABLE IF EXISTS referral_transactions DROP COLUMN IF EXISTS amount_kopecks;
ALTER TABLE IF EXISTS payment_orders DROP COLUMN IF EXISTS amount_kopecks;
ALTER TABLE IF EXISTS payment_transactions DROP COLUMN IF EXISTS amount_kopecks;
ALTER TABLE IF EXISTS payment_receipts DROP COLUMN IF EXISTS amount_kopecks;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS paid_amount_kopecks;
