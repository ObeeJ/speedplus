-- Migration 039: Velocity limits and AML counters
--
-- velocity_limits: per-tier, per-operation caps (admin-configurable).
-- velocity_counters: rolling 24h and 30d accumulators per user.
-- suspicious_activity: structuring-detection flags (write-only, never blocks).

CREATE TABLE velocity_limits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier            INT NOT NULL,           -- 0=TierNew, 1=TierRegular
    operation       TEXT NOT NULL,          -- 'transfer'|'cashout'|'withdraw'|'fund'|'gift_card'
    per_txn_kobo    BIGINT NOT NULL,        -- max single transaction
    daily_kobo      BIGINT NOT NULL,        -- max per rolling 24h window
    monthly_kobo    BIGINT NOT NULL,        -- max per rolling 30d window
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tier, operation)
);

-- Seed conservative defaults (₦ amounts in kobo: 1 NGN = 100 kobo)
-- TierNew (0): tighter limits
INSERT INTO velocity_limits (tier, operation, per_txn_kobo, daily_kobo, monthly_kobo) VALUES
    (0, 'transfer',   5000000,   10000000,   50000000),   -- ₦50k / ₦100k / ₦500k
    (0, 'cashout',    5000000,   10000000,   50000000),
    (0, 'withdraw',   5000000,   10000000,   50000000),
    (0, 'fund',      10000000,   20000000,  100000000),
    (0, 'gift_card',  2000000,    5000000,   20000000);

-- TierRegular (1): higher limits
INSERT INTO velocity_limits (tier, operation, per_txn_kobo, daily_kobo, monthly_kobo) VALUES
    (1, 'transfer',  20000000,   50000000,  200000000),   -- ₦200k / ₦500k / ₦2M
    (1, 'cashout',   20000000,   50000000,  200000000),
    (1, 'withdraw',  20000000,   50000000,  200000000),
    (1, 'fund',      50000000,  100000000,  500000000),
    (1, 'gift_card', 10000000,   20000000,  100000000);

CREATE TABLE velocity_counters (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation    TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,  -- truncated to hour; window = [window_start, window_start+window_size)
    window_size  INTERVAL NOT NULL,     -- '24 hours' or '30 days'
    amount_kobo  BIGINT NOT NULL DEFAULT 0,
    txn_count    INT NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, operation, window_start, window_size)
);
CREATE INDEX idx_velocity_counters_user_op ON velocity_counters(user_id, operation, window_start);

CREATE TABLE suspicious_activity (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id),
    operation    TEXT NOT NULL,
    reason       TEXT NOT NULL,          -- e.g. 'structuring_detected'
    amount_kobo  BIGINT NOT NULL,
    window_kobo  BIGINT NOT NULL,        -- rolling sum that triggered the flag
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_suspicious_activity_user ON suspicious_activity(user_id, created_at DESC);
