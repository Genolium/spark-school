-- +goose Up
-- Canonical shop prices confirmed on 2026-11-13. This also repairs the
-- legacy vip_price column, which GORM used to address as v_ip_price.
UPDATE course_pricings
SET basic_price = 3333.00,
    basic_old_price = 5900.00,
    accelerator_price = 9990.00,
    accelerator_old_price = 15400.00,
    vip_price = 15990.00,
    vip_old_price = 20000.00;

-- +goose Down
SELECT 1;
