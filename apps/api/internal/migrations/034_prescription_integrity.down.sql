DROP INDEX IF EXISTS idx_prescriptions_merchant_status;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS fk_orders_prescription;
ALTER TABLE prescriptions DROP CONSTRAINT IF EXISTS chk_prescriptions_status;
ALTER TABLE prescriptions ALTER COLUMN merchant_id DROP NOT NULL;
ALTER TABLE prescriptions
    DROP COLUMN IF EXISTS consumed_order_id,
    DROP COLUMN IF EXISTS expires_at;
