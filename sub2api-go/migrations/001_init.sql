-- Sub2API Database Schema
-- SQLite / PostgreSQL compatible

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT UNIQUE NOT NULL,
    key_prefix TEXT NOT NULL,
    user_id TEXT NOT NULL,
    name TEXT,
    balance REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active', -- active, disabled
    rate_limit INTEGER DEFAULT 60,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

-- Transactions table (billing records)
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    key_id TEXT NOT NULL,
    type TEXT NOT NULL, -- consume, topup, refund
    amount REAL NOT NULL,
    balance_before REAL NOT NULL,
    balance_after REAL NOT NULL,
    model TEXT,
    input_tokens INTEGER,
    output_tokens INTEGER,
    request_id TEXT,
    stripe_payment_id TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (key_id) REFERENCES api_keys(id)
);

CREATE INDEX IF NOT EXISTS idx_transactions_key_id ON transactions(key_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);

-- Users table (for dashboard auth, Phase 3)
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    name TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Model pricing table
CREATE TABLE IF NOT EXISTS model_pricing (
    model TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    input_price_per_k REAL NOT NULL,
    output_price_per_k REAL NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert default pricing
INSERT OR REPLACE INTO model_pricing (model, provider, input_price_per_k, output_price_per_k) VALUES
    ('claude-3-5-sonnet-20241022', 'anthropic', 0.003, 0.015),
    ('claude-3-5-haiku-20241022', 'anthropic', 0.001, 0.005),
    ('claude-3-opus-20240229', 'anthropic', 0.015, 0.075),
    ('gpt-4o', 'openai', 0.005, 0.015),
    ('gpt-4o-mini', 'openai', 0.00015, 0.0006),
    ('deepseek-chat', 'deepseek', 0.00014, 0.00028),
    ('deepseek-coder', 'deepseek', 0.00014, 0.00028);

-- Stripe payments table (Phase 4)
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    stripe_session_id TEXT UNIQUE,
    stripe_payment_intent TEXT,
    amount REAL NOT NULL,
    currency TEXT NOT NULL DEFAULT 'usd',
    status TEXT NOT NULL DEFAULT 'pending', -- pending, completed, failed
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_stripe_session_id ON payments(stripe_session_id);
