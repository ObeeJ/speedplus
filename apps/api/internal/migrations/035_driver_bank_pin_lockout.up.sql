-- Driver bank accounts (mirrors merchant_bank_accounts).
-- account_name is always resolved via the payment provider — never trusted from the client.
CREATE TABLE IF NOT EXISTS driver_bank_accounts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id      UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    bank_code      TEXT        NOT NULL,
    bank_name      TEXT        NOT NULL,
    account_number TEXT        NOT NULL,
    account_name   TEXT        NOT NULL, -- provider-resolved, not user-supplied
    provider       TEXT        NOT NULL DEFAULT 'paystack',
    is_verified    BOOLEAN     NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- PIN lockout: track failed attempts and temporary lock per user.
ALTER TABLE pins
    ADD COLUMN IF NOT EXISTS failed_attempts INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until    TIMESTAMPTZ;
