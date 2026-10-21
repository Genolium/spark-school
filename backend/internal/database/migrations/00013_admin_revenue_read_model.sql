-- +goose Up
CREATE TABLE IF NOT EXISTS admin_revenue_daily (
    day DATE NOT NULL,
    tier VARCHAR(32) NOT NULL,
    source VARCHAR(64) NOT NULL DEFAULT '',
    promo_code VARCHAR(32) NOT NULL DEFAULT '',
    purchases BIGINT NOT NULL DEFAULT 0,
    revenue_kopecks BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (day, tier, source, promo_code)
);
CREATE INDEX IF NOT EXISTS idx_admin_revenue_daily_day ON admin_revenue_daily(day);

-- +goose Down
DROP TABLE IF EXISTS admin_revenue_daily;
