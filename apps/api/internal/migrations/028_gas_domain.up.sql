-- ── Cylinder specs ────────────────────────────────────────────────────────────
-- Canonical catalog of LPG cylinder sizes. Seeded with the four standard
-- Nigerian household sizes; new sizes can be inserted without a deploy.

CREATE TABLE IF NOT EXISTS cylinder_specs (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    size_kg     NUMERIC(4,1)   NOT NULL,
    tare_kg     NUMERIC(4,1)   NOT NULL,
    valve_type  TEXT           NOT NULL DEFAULT 'standard', -- standard|POL|ACME
    label       TEXT           NOT NULL,                    -- "3kg — small household"
    is_active   BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (size_kg)
);

INSERT INTO cylinder_specs (id, size_kg, tare_kg, valve_type, label) VALUES
    ('00000000-0000-0000-0000-000000000021', 3,    5.0,  'standard', '3 kg — small household'),
    ('00000000-0000-0000-0000-000000000022', 6,    7.5,  'standard', '6 kg — standard household'),
    ('00000000-0000-0000-0000-000000000023', 12.5, 14.5, 'standard', '12.5 kg — most popular'),
    ('00000000-0000-0000-0000-000000000024', 25,   26.0, 'POL',      '25 kg — heavy use / business')
ON CONFLICT (id) DO NOTHING;

-- ── Extend customer_cylinders ─────────────────────────────────────────────────
-- Phase 6 created the table with minimal columns. Add the full domain fields.

ALTER TABLE customer_cylinders
    ADD COLUMN IF NOT EXISTS spec_id          UUID    REFERENCES cylinder_specs(id),
    ADD COLUMN IF NOT EXISTS manufacture_year SMALLINT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS valve_type       TEXT     DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS tare_kg          NUMERIC(4,1) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS notes            TEXT     DEFAULT NULL;

-- ── Gas mode + cylinder FK on orders ─────────────────────────────────────────
-- gas_mode: swap (merchant-stocked) | refill (own cylinder) | new_cylinder
-- cylinder_id: FK to the customer's registered cylinder (refill mode only)

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS gas_mode    TEXT DEFAULT NULL
        CHECK (gas_mode IS NULL OR gas_mode IN ('swap','refill','new_cylinder')),
    ADD COLUMN IF NOT EXISTS cylinder_id UUID DEFAULT NULL
        REFERENCES customer_cylinders(id);

-- ── Cylinder spec on order items ──────────────────────────────────────────────
ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS cylinder_spec_id UUID DEFAULT NULL
        REFERENCES cylinder_specs(id);

-- ── Gas plant fields on merchants ─────────────────────────────────────────────
-- is_gas_plant: true for filling plants (can do refill mode); false for swap-only shops
-- plant_capacity_kg: total daily fill capacity in kg (informational)
-- float_count: number of filled cylinders the plant holds as swap float

ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS is_gas_plant       BOOLEAN  NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS plant_capacity_kg  INTEGER  DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS float_count        INTEGER  DEFAULT NULL;

-- Mark the seeded SpeedPlus Gas merchant as a gas plant
UPDATE merchants
SET is_gas_plant = TRUE
WHERE id = '00000000-0000-0000-0000-000000000004';
