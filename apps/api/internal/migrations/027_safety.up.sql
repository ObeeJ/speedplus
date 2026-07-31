-- ── Customer cylinders ────────────────────────────────────────────────────────
-- Minimal cylinder registry needed for Phase 6 safety features.
-- Phase 1 (gas domain) will ADD the remaining columns (spec_id, tare_kg,
-- valve_type, manufacture_year) via ALTER TABLE.

CREATE TABLE IF NOT EXISTS customer_cylinders (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    serial          TEXT        NOT NULL,
    last_recert_at  DATE        DEFAULT NULL,   -- last recertification date; NULL = unknown
    status          TEXT        NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active','retired','in_custody')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, serial)
);

CREATE INDEX IF NOT EXISTS idx_customer_cylinders_user   ON customer_cylinders(user_id);
CREATE INDEX IF NOT EXISTS idx_customer_cylinders_recert ON customer_cylinders(last_recert_at)
    WHERE last_recert_at IS NOT NULL;

-- ── Cylinder handover checklist ───────────────────────────────────────────────
-- Append-only safety checklist recorded by the rider at handover.
-- Items: valve_seated, no_hiss, regulator_fitted — all boolean.
-- Stored as an order event so it is part of the immutable audit trail.

CREATE TABLE IF NOT EXISTS cylinder_handover_checklists (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID        NOT NULL REFERENCES orders(id),
    driver_id           UUID        NOT NULL REFERENCES users(id),
    valve_seated        BOOLEAN     NOT NULL DEFAULT FALSE,
    no_hiss             BOOLEAN     NOT NULL DEFAULT FALSE,
    regulator_fitted    BOOLEAN     NOT NULL DEFAULT FALSE,
    notes               TEXT        DEFAULT NULL,
    captured_lat        DOUBLE PRECISION,
    captured_lng        DOUBLE PRECISION,
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_handover_checklists_order ON cylinder_handover_checklists(order_id);

-- Append-only: safety evidence — never editable or deletable.
CREATE RULE no_update_handover_checklists AS ON UPDATE TO cylinder_handover_checklists DO INSTEAD NOTHING;
CREATE RULE no_delete_handover_checklists AS ON DELETE TO cylinder_handover_checklists DO INSTEAD NOTHING;

-- ── Hazmat certification on driver profiles ───────────────────────────────────
-- Required for 25kg+ gas runs (Phase 4 dispatch gate).

ALTER TABLE driver_profiles
    ADD COLUMN IF NOT EXISTS hazmat_certified BOOLEAN NOT NULL DEFAULT FALSE;
