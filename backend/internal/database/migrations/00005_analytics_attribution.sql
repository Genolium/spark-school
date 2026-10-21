-- +goose Up
ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS attribution_token VARCHAR(64);
ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS source VARCHAR(64);
ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS campaign VARCHAR(80);
ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS tier VARCHAR(32);
ALTER TABLE IF EXISTS analytics_events ADD COLUMN IF NOT EXISTS promo_code VARCHAR(32);

CREATE INDEX IF NOT EXISTS idx_analytics_events_attribution_token ON analytics_events(attribution_token);
CREATE INDEX IF NOT EXISTS idx_analytics_events_source ON analytics_events(source);
CREATE INDEX IF NOT EXISTS idx_analytics_events_campaign ON analytics_events(campaign);
CREATE INDEX IF NOT EXISTS idx_analytics_events_tier ON analytics_events(tier);
CREATE INDEX IF NOT EXISTS idx_analytics_events_promo_code ON analytics_events(promo_code);

-- +goose Down
DROP INDEX IF EXISTS idx_analytics_events_promo_code;
DROP INDEX IF EXISTS idx_analytics_events_tier;
DROP INDEX IF EXISTS idx_analytics_events_campaign;
DROP INDEX IF EXISTS idx_analytics_events_source;
DROP INDEX IF EXISTS idx_analytics_events_attribution_token;
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS promo_code;
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS tier;
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS campaign;
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS source;
ALTER TABLE IF EXISTS analytics_events DROP COLUMN IF EXISTS attribution_token;
