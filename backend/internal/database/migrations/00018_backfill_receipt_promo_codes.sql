-- +goose Up
-- Receipt backfill used to copy the amount but not the promo code, so paid
-- promo activations were missing from the financial ledger and promo revenue.
UPDATE payment_transactions pt
SET promo_code = pr.promo_code
FROM payment_receipts pr
WHERE pt.payment_id = 'RECEIPT_' || pr.id::text
  AND COALESCE(pt.promo_code, '') = ''
  AND COALESCE(pr.promo_code, '') <> '';

-- +goose Down
SELECT 1;
