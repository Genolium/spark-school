-- +goose Up
-- Receipt backfill used to copy the amount but not the promo code, so paid
-- promo activations were missing from the financial ledger and promo revenue.
UPDATE payment_transactions pt
SET promo_code = pr.promo_code
FROM payment_receipts pr
WHERE pt.payment_id = 'RECEIPT_' || pr.id::text
  AND COALESCE(pt.promo_code, '') = ''
  AND COALESCE(pr.promo_code, '') <> '';

-- Fallback for legacy SBP receipts where the receipt itself has no promo code
-- but the user's promo and the discounted amount prove the code was applied.
WITH latest_pricing AS (
    SELECT basic_price, accelerator_price, vip_price
    FROM course_pricings
    ORDER BY id DESC
    LIMIT 1
)
UPDATE payment_transactions pt
SET promo_code = UPPER(u.promo_code)
FROM users u
JOIN promo_codes pc ON UPPER(pc.code) = UPPER(u.promo_code)
CROSS JOIN latest_pricing lp
WHERE pt.telegram_id = u.telegram_id
  AND pt.status = 'SUCCESS'
  AND pt.source = 'sbp_receipt'
  AND COALESCE(pt.promo_code, '') = ''
  AND COALESCE(u.promo_code, '') <> ''
  AND ABS(pt.amount - (
      CASE LOWER(pt.tier)
          WHEN 'basic' THEN lp.basic_price
          WHEN 'vip' THEN lp.vip_price
          ELSE lp.accelerator_price
      END * (1 - COALESCE(NULLIF(pc.discount_percent, 0), 5) / 100.0)
  )) < 0.01;

-- +goose Down
SELECT 1;
