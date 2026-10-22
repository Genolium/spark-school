-- +goose Up
-- Legacy receipts reconciled from `payment_receipts` created a financial
-- operation in `payment_transactions` without the matching funnel event, so
-- the monitoring check reported them as "payments without payment_confirmed".
-- Restore the link for every operation that is still missing it.

UPDATE payment_transactions pt
SET visitor_id = u.visitor_id,
    attribution_token = u.attribution_token
FROM users u
WHERE u.telegram_id = pt.telegram_id
  AND COALESCE(pt.visitor_id, '') = ''
  AND COALESCE(pt.source, '') = 'sbp_receipt'
  AND COALESCE(pt.pay_method, '') = 'sbp_receipt'
  AND COALESCE(u.visitor_id, '') <> '';

INSERT INTO analytics_events (
    visitor_id,
    attribution_token,
    step,
    source,
    tier,
    promo_code,
    payment_id,
    metadata,
    created_at
)
SELECT
    pt.visitor_id,
    pt.attribution_token,
    'payment_confirmed',
    pt.source,
    pt.tier,
    pt.promo_code,
    pt.payment_id,
    json_build_object(
        'payment_id', pt.payment_id,
        'order_id', pt.order_id,
        'telegram_id', pt.telegram_id,
        'tier', pt.tier,
        'amount', pt.amount,
        'source', pt.source,
        'pay_method', pt.pay_method,
        'promo_code', pt.promo_code,
        'backfilled', true
    )::text,
    pt.created_at
FROM payment_transactions pt
WHERE pt.status = 'SUCCESS'
  AND COALESCE(pt.pay_method, '') NOT IN ('admin_manual', 'historical_access')
  AND COALESCE(pt.source, '') NOT IN ('manual', 'historical')
  AND NOT EXISTS (
      SELECT 1
      FROM analytics_events ae
      WHERE ae.payment_id = pt.payment_id
        AND ae.step = 'payment_confirmed'
  );

-- +goose Down
DELETE FROM analytics_events
WHERE step = 'payment_confirmed'
  AND metadata LIKE '%"backfilled": true%';
