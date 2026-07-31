-- ── Service zones ─────────────────────────────────────────────────────────────
-- A service zone is a named PostGIS polygon with a weekly delivery schedule.
-- active_days is a bitmask matching Go's time.Weekday() (Sunday=0 .. Saturday=6):
-- bit 0 = Sunday … bit 6 = Saturday. NOT ISO 8601 (which starts Monday=1) —
-- run.go computes it as 1 << now.Weekday(), so this must match that, not ISO.
-- window_start/end are stored as UTC minutes-since-midnight. Nigeria is a fixed
-- UTC+1 with no DST, so a business window of 08:00-17:00 WAT is 07:00-16:00 UTC
-- = 420-960, not the "08:00/17:00" numbers a naive WAT->minutes conversion
-- would suggest.

CREATE TABLE IF NOT EXISTS service_zones (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL,
    boundary        GEOMETRY(Polygon, 4326) NOT NULL,
    active_days     SMALLINT    NOT NULL DEFAULT 127, -- all days
    window_start    SMALLINT    NOT NULL DEFAULT 420, -- 08:00 WAT = 07:00 UTC
    window_end      SMALLINT    NOT NULL DEFAULT 960,  -- 17:00 WAT = 16:00 UTC
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (window_start >= 0 AND window_start < 1440),
    CHECK (window_end   > window_start AND window_end <= 1440)
);

CREATE INDEX IF NOT EXISTS idx_service_zones_boundary ON service_zones USING GIST (boundary);

-- ── Delivery runs ─────────────────────────────────────────────────────────────
-- A delivery run is one rider, one zone, one window, N orders.
-- optimized_sequence is the OSRM-ordered array of order IDs.
-- status: assembling → dispatched → in_progress → completed | cancelled

CREATE TABLE IF NOT EXISTS delivery_runs (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    zone_id             UUID        NOT NULL REFERENCES service_zones(id),
    driver_id           UUID        REFERENCES users(id),
    window_start        TIMESTAMPTZ NOT NULL,
    window_end          TIMESTAMPTZ NOT NULL,
    status              TEXT        NOT NULL DEFAULT 'assembling'
                            CHECK (status IN ('assembling','dispatched','in_progress','completed','cancelled')),
    optimized_sequence  JSONB       DEFAULT '[]',
    total_distance_km   DOUBLE PRECISION DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_delivery_runs_zone    ON delivery_runs(zone_id, window_start);
CREATE INDEX IF NOT EXISTS idx_delivery_runs_driver  ON delivery_runs(driver_id) WHERE driver_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_delivery_runs_status  ON delivery_runs(status);

-- ── Run orders ────────────────────────────────────────────────────────────────
-- Join table: which orders belong to which run, and in what sequence.

CREATE TABLE IF NOT EXISTS run_orders (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      UUID    NOT NULL REFERENCES delivery_runs(id) ON DELETE CASCADE,
    order_id    UUID    NOT NULL REFERENCES orders(id),
    sequence    INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (run_id, order_id),
    UNIQUE (run_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_run_orders_run   ON run_orders(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_run_orders_order ON run_orders(order_id);
