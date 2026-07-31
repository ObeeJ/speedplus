-- ── Weight proof for gas orders ───────────────────────────────────────────────
-- Extends proof_media with a weight_photo kind and the measured_kg the rider
-- reads from the hanging scale. The table is already insert-only (migration 018
-- rules); measured_kg lives here so the evidence and the measurement are
-- tamper-proof together.

-- Drop whatever check constraint currently governs the kind column, by
-- looking it up rather than assuming Postgres's default-generated name
-- (proof_media_kind_check). If migration 018's constraint was ever renamed,
-- assuming the name here would make DROP IF EXISTS a no-op, then ADD a
-- second, narrower CHECK alongside the untouched original — CHECK
-- constraints AND together, so weight_photo inserts would still fail
-- against the old list even though this migration appears to have added it.
DO $$
DECLARE
    existing_check text;
BEGIN
    SELECT con.conname INTO existing_check
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    WHERE rel.relname = 'proof_media'
      AND con.contype = 'c'
      AND pg_get_constraintdef(con.oid) ILIKE '%kind%';
    IF existing_check IS NOT NULL THEN
        EXECUTE format('ALTER TABLE proof_media DROP CONSTRAINT %I', existing_check);
    END IF;
END $$;

ALTER TABLE proof_media
    ADD CONSTRAINT proof_media_kind_check
        CHECK (kind IN ('pickup_photo','pickup_video','dropoff_photo','dropoff_video','weight_photo'));

ALTER TABLE proof_media
    ADD COLUMN IF NOT EXISTS measured_kg NUMERIC(6,3) DEFAULT NULL;

-- ── Merchant fill accuracy ─────────────────────────────────────────────────────
-- Running fill-accuracy score recomputed nightly from weight_photo rows.
-- fill_accuracy_pct = avg(measured_kg / ordered_kg) across the last N fills.
-- NULL until the merchant has at least one verified fill.

ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS fill_accuracy_pct  NUMERIC(5,4) DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS fill_sample_count  INTEGER      NOT NULL DEFAULT 0;
