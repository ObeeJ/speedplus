ALTER TABLE order_stops
    DROP COLUMN IF EXISTS recipient_name_enc,
    DROP COLUMN IF EXISTS recipient_phone_enc,
    ADD COLUMN IF NOT EXISTS recipient_name  TEXT,
    ADD COLUMN IF NOT EXISTS recipient_phone TEXT;

ALTER TABLE orders
    DROP COLUMN IF EXISTS recipient_name_enc,
    DROP COLUMN IF EXISTS recipient_phone_enc,
    ADD COLUMN IF NOT EXISTS recipient_name  TEXT,
    ADD COLUMN IF NOT EXISTS recipient_phone TEXT;
