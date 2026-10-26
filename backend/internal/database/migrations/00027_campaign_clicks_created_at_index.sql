-- +goose Up
-- Retention deletes campaign_link_clicks by created_at alone; the existing
-- indexes start with link_id, so add a dedicated one for the daily job.
CREATE INDEX IF NOT EXISTS idx_campaign_clicks_created_at ON campaign_link_clicks(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_campaign_clicks_created_at;
