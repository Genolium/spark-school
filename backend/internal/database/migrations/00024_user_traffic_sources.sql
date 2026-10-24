-- +goose Up
-- Add traffic acquisition attribution columns to users
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS utm_source VARCHAR(64);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS utm_medium VARCHAR(64);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS utm_campaign VARCHAR(80);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS utm_content VARCHAR(80);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS initial_referrer VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_users_utm_source ON users(utm_source);
CREATE INDEX IF NOT EXISTS idx_users_utm_campaign ON users(utm_campaign);

-- Backfill existing users attribution from analytics_events if available
UPDATE users u
SET utm_source = COALESCE(NULLIF(ae.source, ''), (ae.metadata::jsonb->>'utm_source')),
    utm_campaign = COALESCE(NULLIF(ae.campaign, ''), (ae.metadata::jsonb->>'utm_campaign')),
    utm_medium = (ae.metadata::jsonb->>'utm_medium'),
    utm_content = (ae.metadata::jsonb->>'utm_content')
FROM analytics_events ae
WHERE u.attribution_token IS NOT NULL 
  AND u.attribution_token <> ''
  AND ae.attribution_token = u.attribution_token
  AND (u.utm_source IS NULL OR u.utm_source = '')
  AND (ae.source <> '' OR ae.campaign <> '' OR ae.metadata LIKE '%utm_%');

-- +goose Down
DROP INDEX IF EXISTS idx_users_utm_campaign;
DROP INDEX IF EXISTS idx_users_utm_source;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS initial_referrer;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS utm_content;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS utm_campaign;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS utm_medium;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS utm_source;
