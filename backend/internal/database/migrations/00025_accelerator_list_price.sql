-- +goose Up
-- Accelerator list price is 10 900 ₽. The "lower" experiment sells at 9 990 ₽.
UPDATE course_pricings
SET accelerator_price = 10900.00
WHERE accelerator_price IN (6900.00, 0) OR accelerator_price IS NULL;

UPDATE course_pricings
SET accelerator_old_price = 10900.00
WHERE accelerator_old_price IN (9900.00, 6900.00, 0)
   OR accelerator_old_price IS NULL
   OR accelerator_old_price < 10900.00;

-- +goose Down
UPDATE course_pricings
SET accelerator_price = 6900.00
WHERE accelerator_price = 10900.00;

UPDATE course_pricings
SET accelerator_old_price = 9900.00
WHERE accelerator_old_price = 10900.00;
