-- +goose Up
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS basic_feedback_price NUMERIC(10,2) NOT NULL DEFAULT 5000.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS basic_feedback_old_price NUMERIC(10,2) NOT NULL DEFAULT 6900.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS basic_feedback_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS sales_close_at TIMESTAMPTZ DEFAULT '2026-11-21T00:00:00+03:00';

UPDATE course_pricings
SET sales_close_at = '2026-11-21T00:00:00+03:00'
WHERE sales_close_at IS NULL;

-- +goose Down
ALTER TABLE course_pricings DROP COLUMN IF EXISTS sales_close_at;
ALTER TABLE course_pricings DROP COLUMN IF EXISTS basic_feedback_enabled;
ALTER TABLE course_pricings DROP COLUMN IF EXISTS basic_feedback_old_price;
ALTER TABLE course_pricings DROP COLUMN IF EXISTS basic_feedback_price;
