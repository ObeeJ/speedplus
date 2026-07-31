ALTER TABLE merchants
    DROP COLUMN IF EXISTS fill_sample_count,
    DROP COLUMN IF EXISTS fill_accuracy_pct;

ALTER TABLE proof_media
    DROP COLUMN IF EXISTS measured_kg;

ALTER TABLE proof_media
    DROP CONSTRAINT IF EXISTS proof_media_kind_check;

ALTER TABLE proof_media
    ADD CONSTRAINT proof_media_kind_check
        CHECK (kind IN ('pickup_photo','pickup_video','dropoff_photo','dropoff_video'));
