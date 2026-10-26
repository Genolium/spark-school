-- +goose Up
CREATE TABLE IF NOT EXISTS campaign_links (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(40) NOT NULL UNIQUE,
    title VARCHAR(120) NOT NULL,
    target_type VARCHAR(16) NOT NULL CHECK (target_type IN ('channel','bot','landing')),
    target_value VARCHAR(255) NOT NULL DEFAULT '',
    utm_source VARCHAR(64),
    utm_medium VARCHAR(64),
    utm_campaign VARCHAR(80),
    utm_content VARCHAR(80),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS campaign_link_clicks (
    id BIGSERIAL PRIMARY KEY,
    link_id BIGINT NOT NULL REFERENCES campaign_links(id) ON DELETE CASCADE,
    visitor_id VARCHAR(64),
    attribution_token VARCHAR(64),
    ip_hash VARCHAR(64),
    user_agent VARCHAR(255),
    referer VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_link_created ON campaign_link_clicks(link_id, created_at);
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_visitor ON campaign_link_clicks(visitor_id);

-- +goose Down
DROP TABLE IF EXISTS campaign_link_clicks;
DROP TABLE IF EXISTS campaign_links;
