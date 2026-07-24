-- Proof-of-delivery media chain of custody. r2_key is the only pointer to
-- the actual file (private bucket, presigned access only). sha256 lets a
-- dispute prove the file wasn't swapped after capture; seal_serial pairs a
-- pickup/dropoff photo for the same tamper-evident seal number.
CREATE TABLE IF NOT EXISTS proof_media (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID        NOT NULL REFERENCES orders(id),
    stop_id       UUID        REFERENCES order_stops(id), -- NULL for single-drop orders
    kind          TEXT        NOT NULL CHECK (kind IN ('pickup_photo','pickup_video','dropoff_photo','dropoff_video')),
    r2_key        TEXT        NOT NULL,
    sha256        TEXT        NOT NULL,
    seal_serial   TEXT,
    captured_lat  DOUBLE PRECISION,
    captured_lng  DOUBLE PRECISION,
    captured_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    captured_by   UUID        NOT NULL REFERENCES users(id), -- the driver
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_proof_media_order ON proof_media(order_id);
CREATE INDEX IF NOT EXISTS idx_proof_media_stop  ON proof_media(stop_id);

-- Append-only: proof media is evidence — never editable or deletable once written.
CREATE RULE no_update_proof_media AS ON UPDATE TO proof_media DO INSTEAD NOTHING;
CREATE RULE no_delete_proof_media AS ON DELETE TO proof_media DO INSTEAD NOTHING;
