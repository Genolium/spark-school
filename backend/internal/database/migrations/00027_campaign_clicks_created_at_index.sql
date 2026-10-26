-- +goose Up
-- Retention deletes campaign_link_clicks by created_at alone; the existing
-- indexes start with link_id, so add a dedicated one for the daily job.
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_created_at ON campaign_link_clicks(created_at);
-- EraseUser removes clicks by attribution_token and the model declares this
-- index, but migration 00026 never created it.
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_attribution_token ON campaign_link_clicks(attribution_token);

-- +goose Down
DROP INDEX IF EXISTS idx_campaign_clicks_attribution_token;
DROP INDEX IF EXISTS idx_campaign_clicks_created_at;
