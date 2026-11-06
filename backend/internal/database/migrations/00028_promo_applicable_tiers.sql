-- +goose Up
ALTER TABLE IF EXISTS promo_codes ADD COLUMN IF NOT EXISTS applicable_tiers VARCHAR(64);

-- +goose Down
ALTER TABLE IF EXISTS promo_codes DROP COLUMN IF EXISTS applicable_tiers;
