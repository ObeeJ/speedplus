-- fee_configs: versioned, append-only pricing configuration per vertical.
-- Each change inserts a new row; the row with the greatest effective_at <= now
-- is live. Empty table => code falls back to DefaultFeeTable. Settlement pins
-- to the config effective at order creation time.
CREATE TABLE IF NOT EXISTS fee_configs (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    vertical            TEXT         NOT NULL CHECK (vertical IN ('food','grocery','pharmacy','gas','package')),
    base_fee_kobo       BIGINT       NOT NULL CHECK (base_fee_kobo >= 0),
    per_km_kobo         BIGINT       NOT NULL DEFAULT 0 CHECK (per_km_kobo >= 0),
    per_kg_kobo         BIGINT       NOT NULL DEFAULT 0 CHECK (per_kg_kobo >= 0),
    service_pct         NUMERIC(6,4) NOT NULL CHECK (service_pct >= 0 AND service_pct <= 0.5),
    merchant_take_rate  NUMERIC(6,4) NOT NULL CHECK (merchant_take_rate > 0.5 AND merchant_take_rate <= 1),
    driver_take_rate    NUMERIC(6,4) NOT NULL CHECK (driver_take_rate >= 0.5 AND driver_take_rate <= 1),
    platform_take_rate  NUMERIC(6,4) NOT NULL,
    fuel_price_ref_kobo BIGINT       NOT NULL DEFAULT 0 CHECK (fuel_price_ref_kobo >= 0),
    effective_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by          UUID         NOT NULL REFERENCES users(id),
    reason              TEXT         NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CHECK (driver_take_rate + platform_take_rate = 1.0),
    UNIQUE (vertical, effective_at)
);

CREATE INDEX IF NOT EXISTS idx_fee_configs_lookup ON fee_configs(vertical, effective_at DESC);

-- Append-only at the DB layer (same pattern as admin_audit_logs)
CREATE RULE no_update_fee_configs AS ON UPDATE TO fee_configs DO INSTEAD NOTHING;
CREATE RULE no_delete_fee_configs AS ON DELETE TO fee_configs DO INSTEAD NOTHING;
