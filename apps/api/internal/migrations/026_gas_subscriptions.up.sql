-- ── Gas subscription fields ───────────────────────────────────────────────────
-- cylinder_spec_id: which cylinder size this subscription refills.
-- gas_mode: swap (merchant-stocked) or refill (own cylinder).
-- predicted_runout_at: estimated date the customer will run out of gas.
-- avg_days_between_refills: rolling average from delivered order history.

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS cylinder_spec_id          UUID    DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS gas_mode                  TEXT    DEFAULT NULL
        CHECK (gas_mode IS NULL OR gas_mode IN ('swap','refill','new_cylinder')),
    ADD COLUMN IF NOT EXISTS predicted_runout_at       TIMESTAMPTZ DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS avg_days_between_refills  NUMERIC(6,2) DEFAULT NULL;

-- ── LPG price index ───────────────────────────────────────────────────────────
-- Append-only: each row is a new price observation. The row with the greatest
-- effective_at <= now is live. Admin edits insert; never update or delete.
-- A >10% delta triggers a suggestion (same discipline as fee_configs fuel ref).

CREATE TABLE IF NOT EXISTS lpg_price_index (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    region              TEXT        NOT NULL DEFAULT 'Lagos',
    price_per_kg_kobo   BIGINT      NOT NULL CHECK (price_per_kg_kobo > 0),
    source              TEXT        NOT NULL,   -- e.g. "NMDPRA", "manual"
    effective_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          UUID        NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (region, effective_at)
);

CREATE INDEX IF NOT EXISTS idx_lpg_price_lookup ON lpg_price_index(region, effective_at DESC);

CREATE RULE no_update_lpg_price AS ON UPDATE TO lpg_price_index DO INSTEAD NOTHING;
CREATE RULE no_delete_lpg_price AS ON DELETE TO lpg_price_index DO INSTEAD NOTHING;

-- ── Deliberately NOT auto-unpausing subscriptions here ────────────────────────
-- Migration 010 paused every active subscription because chargeOne was
-- unimplemented, using the same status='paused' value as a customer's own
-- self-service pause and dunning's auto-pause after 3 failed charges. With no
-- marker distinguishing the three, a blanket
--   UPDATE subscriptions SET status='active' WHERE vertical='gas' AND status='paused'
-- would reactivate subscriptions a customer deliberately stopped and start
-- billing them again without consent. Migration 029 adds paused_reason so
-- future pauses are attributable; reactivating the historical 010 cohort is
-- left to a deliberate, reviewable admin action once paused_reason data is
-- available to distinguish them, not an automatic migration step.
