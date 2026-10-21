-- +goose Up
ALTER TABLE IF EXISTS call_bookings ADD COLUMN IF NOT EXISTS reminder_24_sent BOOLEAN DEFAULT FALSE;
ALTER TABLE IF EXISTS call_bookings ADD COLUMN IF NOT EXISTS reminder_1_sent BOOLEAN DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_call_bookings_reminder_24_sent ON call_bookings(reminder_24_sent);
CREATE INDEX IF NOT EXISTS idx_call_bookings_reminder_1_sent ON call_bookings(reminder_1_sent);

-- +goose Down
DROP INDEX IF EXISTS idx_call_bookings_reminder_1_sent;
DROP INDEX IF EXISTS idx_call_bookings_reminder_24_sent;
ALTER TABLE IF EXISTS call_bookings DROP COLUMN IF EXISTS reminder_1_sent;
ALTER TABLE IF EXISTS call_bookings DROP COLUMN IF EXISTS reminder_24_sent;
