-- +goose Up
ALTER TABLE IF EXISTS bot_sessions ADD COLUMN IF NOT EXISTS vip_price_experiment VARCHAR(16);

-- +goose Down
ALTER TABLE IF EXISTS bot_sessions DROP COLUMN IF EXISTS vip_price_experiment;
