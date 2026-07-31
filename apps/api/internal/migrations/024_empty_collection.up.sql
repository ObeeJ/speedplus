-- ── Empty cylinder collection ─────────────────────────────────────────────────
-- Adds per-stop empty-cylinder collection tracking to order_stops.
-- empty_collected: rider checked the box confirming they took the empty.
-- empty_cylinder_serial: the serial number of the empty collected (optional;
--   populated for refill orders where the customer's own cylinder is tracked).

ALTER TABLE order_stops
    ADD COLUMN IF NOT EXISTS empty_collected       BOOLEAN DEFAULT FALSE NOT NULL,
    ADD COLUMN IF NOT EXISTS empty_cylinder_serial TEXT    DEFAULT NULL;

-- ── Cylinder custody events ───────────────────────────────────────────────────
-- Append-only audit trail: who held which cylinder, when, and where.
-- Used for refill orders where the customer's own cylinder is in our custody.
-- cylinder_id references customer_cylinders (Phase 1); nullable here so Phase 3
-- can ship independently — rows without a cylinder_id are still valid custody
-- events keyed by serial number alone.

CREATE TABLE IF NOT EXISTS cylinder_custody_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID        NOT NULL REFERENCES orders(id),
    stop_id         UUID        REFERENCES order_stops(id),
    cylinder_id     UUID        DEFAULT NULL,   -- FK to customer_cylinders added in Phase 1
    serial          TEXT        DEFAULT NULL,   -- human-readable serial, always populated
    event_type      TEXT        NOT NULL CHECK (event_type IN ('collected','at_plant','returned')),
    actor_id        UUID        NOT NULL REFERENCES users(id),
    captured_lat    DOUBLE PRECISION,
    captured_lng    DOUBLE PRECISION,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custody_events_order  ON cylinder_custody_events(order_id);
CREATE INDEX IF NOT EXISTS idx_custody_events_serial ON cylinder_custody_events(serial) WHERE serial IS NOT NULL;

-- Append-only: custody events are evidence — never editable or deletable.
CREATE RULE no_update_custody_events AS ON UPDATE TO cylinder_custody_events DO INSTEAD NOTHING;
CREATE RULE no_delete_custody_events AS ON DELETE TO cylinder_custody_events DO INSTEAD NOTHING;
