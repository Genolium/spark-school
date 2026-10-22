-- +goose Up
-- Production installations created before migration 00004 may have a
-- GORM-generated course_pricings table without vip_price. CREATE TABLE IF
-- NOT EXISTS never adds missing columns, so repair the full contract here.
CREATE TABLE IF NOT EXISTS course_pricings (
    id BIGSERIAL PRIMARY KEY,
    basic_price NUMERIC(10,2) DEFAULT 3990.00,
    basic_old_price NUMERIC(10,2) DEFAULT 5900.00,
    accelerator_price NUMERIC(10,2) DEFAULT 6900.00,
    accelerator_old_price NUMERIC(10,2) DEFAULT 9900.00,
    vip_price NUMERIC(10,2) DEFAULT 14900.00,
    vip_old_price NUMERIC(10,2) DEFAULT 19900.00,
    bank_name VARCHAR(128) DEFAULT 'БСПБ (Банк «Санкт-Петербург»)',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS basic_price NUMERIC(10,2) DEFAULT 3990.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS basic_old_price NUMERIC(10,2) DEFAULT 5900.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS accelerator_price NUMERIC(10,2) DEFAULT 6900.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS accelerator_old_price NUMERIC(10,2) DEFAULT 9900.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS vip_price NUMERIC(10,2) DEFAULT 14900.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS vip_old_price NUMERIC(10,2) DEFAULT 19900.00;
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS bank_name VARCHAR(128) DEFAULT 'БСПБ (Банк «Санкт-Петербург»)';
ALTER TABLE course_pricings ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

UPDATE course_pricings SET basic_price = COALESCE(basic_price, 3990.00);
UPDATE course_pricings SET basic_old_price = COALESCE(basic_old_price, 5900.00);
UPDATE course_pricings SET accelerator_price = COALESCE(accelerator_price, 6900.00);
UPDATE course_pricings SET accelerator_old_price = COALESCE(accelerator_old_price, 9900.00);
UPDATE course_pricings SET vip_price = COALESCE(vip_price, 14900.00);
UPDATE course_pricings SET vip_old_price = COALESCE(vip_old_price, 19900.00);
UPDATE course_pricings SET bank_name = COALESCE(NULLIF(bank_name, ''), 'БСПБ (Банк «Санкт-Петербург»)');
UPDATE course_pricings SET updated_at = COALESCE(updated_at, NOW());

INSERT INTO course_pricings (basic_price, basic_old_price, accelerator_price, accelerator_old_price, vip_price, vip_old_price, bank_name)
SELECT 3990.00, 5900.00, 6900.00, 9900.00, 14900.00, 19900.00, 'БСПБ (Банк «Санкт-Петербург»)'
WHERE NOT EXISTS (SELECT 1 FROM course_pricings);

-- +goose Down
SELECT 1;
