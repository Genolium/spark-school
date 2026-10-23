-- +goose Up
-- Synchronize payment transaction created_at with actual receipt created_at
-- so revenue is accounted on the payment date rather than the curator moderation click date.
UPDATE payment_transactions pt
SET created_at = pr.created_at
FROM payment_receipts pr
WHERE pt.payment_id = 'RECEIPT_' || pr.id::text
  AND pr.created_at IS NOT NULL
  AND pt.created_at <> pr.created_at;

-- Refresh admin_revenue_daily read model immediately if table exists
DELETE FROM admin_revenue_daily;
INSERT INTO admin_revenue_daily (day, tier, source, promo_code, purchases, revenue_kopecks, updated_at)
SELECT (created_at AT TIME ZONE 'Europe/Moscow')::date,
       COALESCE(NULLIF(LOWER(tier), ''), 'accelerator'),
       COALESCE(source, ''),
       COALESCE(promo_code, ''),
       COUNT(*),
       COALESCE(SUM(COALESCE(NULLIF(amount_kopecks, 0), ROUND(amount * 100)::bigint)), 0),
       NOW()
FROM payment_transactions
WHERE status = 'SUCCESS'
GROUP BY 1, 2, 3, 4;

-- +goose Down
SELECT 1;
