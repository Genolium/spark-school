-- +goose Up
-- Telegram "chat not found" is permanent: the bot cannot initiate a dialog
-- with that account. Stop counting these dead letters as active failures.
UPDATE outbox_events
SET status = 'ignored',
    last_error = COALESCE(NULLIF(last_error, ''), 'permanent: chat not found')
WHERE status = 'failed'
  AND LOWER(last_error) LIKE '%chat not found%';

-- +goose Down
SELECT 1;
