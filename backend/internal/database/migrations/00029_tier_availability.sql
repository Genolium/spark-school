-- +goose Up
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS basic_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS accelerator_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS vip_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE course_pricings DROP COLUMN IF EXISTS basic_enabled;
ALTER TABLE course_pricings DROP COLUMN IF EXISTS accelerator_enabled;
ALTER TABLE course_pricings DROP COLUMN IF EXISTS vip_enabled;
