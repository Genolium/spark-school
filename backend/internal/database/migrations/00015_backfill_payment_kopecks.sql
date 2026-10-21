-- +goose Up
-- Historical reconciliation used to insert SUCCESS transactions with rubles
-- only. The admin revenue rollup sums integer kopecks, so those rows showed
-- up as purchases with zero revenue. Backfill them once, idempotently.
UPDATE payment_transactions
SET amount_kopecks = ROUND(amount * 100)
WHERE amount_kopecks = 0 AND amount <> 0;

-- +goose Down
SELECT 1;
