-- ── Prescription integrity ─────────────────────────────────────────────────────
-- The Rx flow had no server-side integrity: merchant_id was nullable (so a
-- prescription submitted without a chosen pharmacy could never be reviewed —
-- ReviewPrescription rejects nil MerchantID), status had no CHECK, and
-- orders.prescription_id had no FK, so a dangling/garbage ID could sit on an
-- order forever. This migration closes all three, and adds the columns needed
-- to make an approved Rx single-use and time-bound (enforced in order.go).
--
-- prescriptions table is empty in every environment this has been checked
-- against (0 rows) because of the merchant_id bug above — no prescription
-- could ever complete review, so nothing to backfill.

ALTER TABLE prescriptions
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS consumed_order_id UUID REFERENCES orders(id);

ALTER TABLE prescriptions
    ALTER COLUMN merchant_id SET NOT NULL;

ALTER TABLE prescriptions
    ADD CONSTRAINT chk_prescriptions_status
        CHECK (status IN ('pending', 'approved', 'rejected', 'consumed', 'expired'));

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_prescription
        FOREIGN KEY (prescription_id) REFERENCES prescriptions(id);

CREATE INDEX IF NOT EXISTS idx_prescriptions_merchant_status
    ON prescriptions (merchant_id, status);
