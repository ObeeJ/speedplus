-- Recipient name/phone were added as plaintext TEXT in migration 015. This is
-- third-party PII (the recipient never consented to our terms) and must be
-- encrypted at rest. Pre-launch: no production rows exist yet, so this is a
-- straight column swap rather than an encrypt-in-place backfill. If this ever
-- runs against a database with real rows, encrypt them via the application
-- (internal/crypto) before applying, not in raw SQL.

ALTER TABLE orders
    DROP COLUMN IF EXISTS recipient_name,
    DROP COLUMN IF EXISTS recipient_phone,
    ADD COLUMN IF NOT EXISTS recipient_name_enc  TEXT,
    ADD COLUMN IF NOT EXISTS recipient_phone_enc TEXT;

ALTER TABLE order_stops
    DROP COLUMN IF EXISTS recipient_name,
    DROP COLUMN IF EXISTS recipient_phone,
    ADD COLUMN IF NOT EXISTS recipient_name_enc  TEXT,
    ADD COLUMN IF NOT EXISTS recipient_phone_enc TEXT;
