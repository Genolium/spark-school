-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS drip_leads (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(64),
    first_name VARCHAR(128),
    stage INT DEFAULT 0,
    calculated_score INT DEFAULT 0,
    next_trigger_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drip_leads_next_trigger_at ON drip_leads(next_trigger_at);
CREATE INDEX IF NOT EXISTS idx_drip_leads_is_completed ON drip_leads(is_completed);

CREATE TABLE IF NOT EXISTS payment_transactions (
    id BIGSERIAL PRIMARY KEY,
    payment_id VARCHAR(64) UNIQUE NOT NULL,
    order_id VARCHAR(100),
    telegram_id BIGINT,
    username VARCHAR(64),
    tier VARCHAR(32) DEFAULT 'accelerator',
    amount NUMERIC(10, 2) NOT NULL,
    original_amount NUMERIC(10, 2),
    fee NUMERIC(10, 2),
    status VARCHAR(32) NOT NULL,
    currency VARCHAR(10) DEFAULT 'RUB',
    pay_method VARCHAR(32),
    product_name VARCHAR(128),
    promo_code VARCHAR(32),
    customer_email VARCHAR(128),
    customer_phone VARCHAR(64),
    raw_callback TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_transactions_telegram_id ON payment_transactions(telegram_id);
CREATE INDEX IF NOT EXISTS idx_payment_transactions_order_id ON payment_transactions(order_id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS payment_transactions;
DROP TABLE IF EXISTS drip_leads;
