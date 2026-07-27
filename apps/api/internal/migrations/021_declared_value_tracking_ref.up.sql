-- declared_value_kobo: sender-stated value of the package contents.
-- Used for liability cap calculation and dispute resolution.
-- NULL = not declared (platform liability capped at ₦50,000 per policy).
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS declared_value_kobo BIGINT DEFAULT NULL;

-- tracking_ref: short human-readable reference the sender can share with
-- the recipient (e.g. "SPX-A3K9"). Generated at order creation.
-- Unique index ensures no two orders share a ref.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS tracking_ref VARCHAR(12) DEFAULT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_tracking_ref
    ON orders (tracking_ref)
    WHERE tracking_ref IS NOT NULL;

-- Stable cursor pagination: (created_at DESC, id DESC) composite index
-- prevents duplicate/missing rows when two orders share the same timestamp.
CREATE INDEX IF NOT EXISTS idx_orders_customer_cursor
    ON orders (customer_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_orders_merchant_cursor
    ON orders (merchant_id, created_at DESC, id DESC);
