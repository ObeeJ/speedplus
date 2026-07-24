-- Fraud hardening: GPS evidence on delivery-code confirmation, and a
-- payment_method column gating the pay-on-arrival (POD) trust-ladder path.

ALTER TABLE delivery_codes
    ADD COLUMN IF NOT EXISTS confirm_lat        DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS confirm_lng        DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS confirm_distance_m NUMERIC(10,1),
    ADD COLUMN IF NOT EXISTS location_flagged   BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20) NOT NULL DEFAULT 'wallet'; -- wallet|pay_on_arrival

-- One active POD order per customer, checked via this partial index.
CREATE INDEX IF NOT EXISTS idx_orders_active_pod ON orders(customer_id)
    WHERE payment_method = 'pay_on_arrival' AND status NOT IN ('delivered','cancelled','refunded');
