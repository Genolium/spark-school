-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(64),
    first_name VARCHAR(128) NOT NULL,
    last_name VARCHAR(128),
    photo_url TEXT,
    role VARCHAR(20) DEFAULT 'student',
    has_access BOOLEAN DEFAULT FALSE,
    access_granted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS referrals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    referred_by_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    parent_referrer_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS affiliate_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    referral_code VARCHAR(32) UNIQUE NOT NULL,
    current_balance NUMERIC(10, 2) DEFAULT 0.00 CHECK (current_balance >= 0),
    total_earned NUMERIC(10, 2) DEFAULT 0.00,
    total_withdrawn NUMERIC(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS referral_transactions (
    id BIGSERIAL PRIMARY KEY,
    affiliate_id BIGINT REFERENCES users(id),
    buyer_id BIGINT REFERENCES users(id),
    level INT NOT NULL CHECK (level IN (1, 2)),
    amount NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payout_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(10, 2) NOT NULL CHECK (amount >= 1000),
    payment_details TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS promo_codes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,
    discount_percent INT DEFAULT 5,
    owner_telegram_id BIGINT,
    owner_username VARCHAR(64),
    uses_count INT DEFAULT 0,
    reward_amount NUMERIC(10, 2) DEFAULT 1000.00,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_receipts (
    id BIGSERIAL PRIMARY KEY,
    file_hash VARCHAR(64) UNIQUE NOT NULL,
    file_unique_id VARCHAR(64),
    telegram_id BIGINT,
    username VARCHAR(64),
    file_id VARCHAR(255),
    tier VARCHAR(32) DEFAULT 'accelerator',
    amount NUMERIC(10, 2),
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed'
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 5,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_affiliate_referral_code ON affiliate_profiles(referral_code);
CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);
CREATE INDEX IF NOT EXISTS idx_payment_receipts_hash ON payment_receipts(file_hash);
CREATE INDEX IF NOT EXISTS idx_payment_receipts_unique_id ON payment_receipts(file_unique_id);
CREATE INDEX IF NOT EXISTS idx_payment_receipts_telegram_id ON payment_receipts(telegram_id);
CREATE INDEX IF NOT EXISTS idx_outbox_events_status ON outbox_events(status);

-- Seed initial promo codes
INSERT INTO promo_codes (code, discount_percent, owner_username, reward_amount, is_active)
VALUES 
    ('START5', 5, 'admin', 1000.00, true),
    ('SPARK5', 5, 'admin', 1000.00, true),
    ('IL5', 5, 'ilyan_vas', 1000.00, true),
    ('ILYA5', 5, 'ilyan_vas', 1000.00, true)
ON CONFLICT (code) DO NOTHING;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS payment_receipts;
DROP TABLE IF EXISTS promo_codes;
DROP TABLE IF EXISTS payout_requests;
DROP TABLE IF EXISTS referral_transactions;
DROP TABLE IF EXISTS affiliate_profiles;
DROP TABLE IF EXISTS referrals;
DROP TABLE IF EXISTS users;
