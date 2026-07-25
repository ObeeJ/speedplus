-- Merchant saved bank account for withdrawals.
-- A merchant can have one active bank account at a time.
-- account_name is fetched from the provider's account-resolution API
-- (Paystack /bank/resolve) before saving — never trusted from the request.
CREATE TABLE IF NOT EXISTS merchant_bank_accounts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id    UUID        NOT NULL UNIQUE REFERENCES merchants(id) ON DELETE CASCADE,
    bank_code      TEXT        NOT NULL,
    bank_name      TEXT        NOT NULL,
    account_number TEXT        NOT NULL,
    account_name   TEXT        NOT NULL, -- resolved by provider, not user-supplied
    provider       TEXT        NOT NULL DEFAULT 'paystack',
    is_verified    BOOLEAN     NOT NULL DEFAULT true, -- set true after provider resolution
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add merchant_id to cashout_requests so the same table covers both
-- driver EWA cashouts and merchant withdrawals.
ALTER TABLE cashout_requests
    ADD COLUMN IF NOT EXISTS merchant_id UUID REFERENCES merchants(id),
    ADD COLUMN IF NOT EXISTS actor_type  TEXT NOT NULL DEFAULT 'driver'; -- driver|merchant

CREATE INDEX IF NOT EXISTS idx_cashout_merchant ON cashout_requests(merchant_id) WHERE merchant_id IS NOT NULL;
