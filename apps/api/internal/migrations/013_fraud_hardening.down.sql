DROP INDEX IF EXISTS idx_orders_active_pod;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_method;
ALTER TABLE delivery_codes
    DROP COLUMN IF EXISTS confirm_lat,
    DROP COLUMN IF EXISTS confirm_lng,
    DROP COLUMN IF EXISTS confirm_distance_m,
    DROP COLUMN IF EXISTS location_flagged;
