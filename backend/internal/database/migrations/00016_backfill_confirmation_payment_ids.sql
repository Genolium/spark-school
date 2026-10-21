-- +goose Up
-- payment_confirmed events created after migration 00008 carried the payment
-- id only inside metadata, so the monitoring join on analytics_events.payment_id
-- reported settled payments as missing. Backfill the column from metadata.
UPDATE analytics_events
SET payment_id = substring(metadata FROM '"payment_id"\s*:\s*"([^"]+)"')
WHERE step = 'payment_confirmed'
  AND payment_id IS NULL
  AND metadata LIKE '%"payment_id"%';

-- +goose Down
SELECT 1;
