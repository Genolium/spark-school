-- +goose Up
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS intro_sent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_replied_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_notes TEXT;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS intro_reminder_sent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS reply_reminder_sent_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_users_onboarding_status ON users(onboarding_status);

-- Existing students are not re-onboarded; only new payments after this
-- migration start the flow.
UPDATE users SET onboarding_status = 'skipped'
WHERE onboarding_status IS NULL AND has_access = TRUE;

-- +goose Down
DROP INDEX IF EXISTS idx_users_onboarding_status;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS reply_reminder_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS intro_reminder_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_notes;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_completed_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_replied_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS intro_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_status;
