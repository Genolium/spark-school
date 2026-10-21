-- +goose Up
-- The manual onboarding queue was replaced by a short "what happens next"
-- paragraph in the access message.
DROP INDEX IF EXISTS idx_users_onboarding_status;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS reply_reminder_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS intro_reminder_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_notes;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_completed_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_replied_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS intro_sent_at;
ALTER TABLE IF EXISTS users DROP COLUMN IF EXISTS onboarding_status;

-- +goose Down
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_status VARCHAR(32);
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS intro_sent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_replied_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS onboarding_notes TEXT;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS intro_reminder_sent_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS reply_reminder_sent_at TIMESTAMP WITH TIME ZONE;
CREATE INDEX IF NOT EXISTS idx_users_onboarding_status ON users(onboarding_status);
