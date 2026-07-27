DROP INDEX IF EXISTS idx_orders_merchant_cursor;
DROP INDEX IF EXISTS idx_orders_customer_cursor;
DROP INDEX IF EXISTS idx_orders_tracking_ref;
ALTER TABLE orders DROP COLUMN IF EXISTS tracking_ref;
ALTER TABLE orders DROP COLUMN IF EXISTS declared_value_kobo;
