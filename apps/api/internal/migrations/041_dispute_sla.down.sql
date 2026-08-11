ALTER TABLE escrow_holds
    DROP COLUMN IF EXISTS frozen_sla_deadline,
    DROP COLUMN IF EXISTS partial_refund_kobo,
    DROP COLUMN IF EXISTS auto_adjudicated;

DROP INDEX IF EXISTS idx_escrow_holds_sla;
