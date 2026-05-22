-- Persistence layer: account ledger, payments, request logs, admin audit, sync outbox
-- Applied automatically by SQLiteStore.migrate() on startup

CREATE TABLE IF NOT EXISTS account_ledger (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL,
    key_id              TEXT,
    type                TEXT NOT NULL,
    amount              REAL NOT NULL,
    balance_before      REAL NOT NULL,
    balance_after       REAL NOT NULL,
    model               TEXT,
    input_tokens        INTEGER,
    output_tokens       INTEGER,
    request_id          TEXT,
    stripe_payment_id   TEXT,
    payment_id          TEXT,
    note                TEXT,
    actor               TEXT NOT NULL DEFAULT 'system',
    created_at          DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ledger_user_id ON account_ledger(user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_type ON account_ledger(type);
CREATE INDEX IF NOT EXISTS idx_ledger_created_at ON account_ledger(created_at);

CREATE TABLE IF NOT EXISTS payments (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL,
    stripe_session_id       TEXT UNIQUE,
    stripe_payment_intent   TEXT,
    amount_usd              REAL NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'usd',
    status                  TEXT NOT NULL DEFAULT 'pending',
    failure_reason          TEXT,
    ledger_entry_id         TEXT,
    created_at              DATETIME NOT NULL,
    completed_at            DATETIME
);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);

CREATE TABLE IF NOT EXISTS request_logs (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    key_id          TEXT NOT NULL,
    request_id      TEXT,
    model           TEXT NOT NULL,
    stream          INTEGER NOT NULL DEFAULT 0,
    outcome         TEXT NOT NULL,
    input_tokens    INTEGER,
    output_tokens   INTEGER,
    charged_usd     REAL,
    ledger_entry_id TEXT,
    latency_ms      INTEGER,
    client_ip       TEXT,
    created_at      DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reqlog_user_id ON request_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_reqlog_key_id ON request_logs(key_id);
CREATE INDEX IF NOT EXISTS idx_reqlog_created_at ON request_logs(created_at);

CREATE TABLE IF NOT EXISTS admin_audit_log (
    id              TEXT PRIMARY KEY,
    admin_id        TEXT,
    target_user_id  TEXT NOT NULL,
    action          TEXT NOT NULL,
    before_json     TEXT NOT NULL,
    after_json      TEXT NOT NULL,
    ledger_entry_id TEXT,
    created_at      DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sync_outbox (
    id              TEXT PRIMARY KEY,
    entity_type     TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    payload_json    TEXT NOT NULL,
    retry_count     INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT,
    created_at      DATETIME NOT NULL,
    processed_at    DATETIME
);

-- Migrate legacy transactions into account_ledger (idempotent)
INSERT OR IGNORE INTO account_ledger (
    id, user_id, key_id, type, amount, balance_before, balance_after,
    model, input_tokens, output_tokens, request_id, stripe_payment_id, actor, created_at
)
SELECT
    id,
    COALESCE(user_id, ''),
    NULLIF(key_id, ''),
    type,
    amount,
    balance_before,
    balance_after,
    model,
    input_tokens,
    output_tokens,
    request_id,
    stripe_payment_id,
    'system',
    created_at
FROM transactions
WHERE COALESCE(user_id, '') != '';
