-- Migration 041: Dispute SLA tracking
-- Adds frozen_sla_deadline to escrow_holds so the worker can alert on overdue disputes.
-- Also adds a partial_refund_kobo column for tier-1 auto-adjudication results.

ALTER TABLE escrow_holds
    ADD COLUMN IF NOT EXISTS frozen_sla_deadline TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS partial_refund_kobo  BIGINT,
    ADD COLUMN IF NOT EXISTS auto_adjudicated     BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_escrow_holds_sla
    ON escrow_holds(frozen_sla_deadline)
    WHERE status = 'frozen' AND frozen_sla_deadline IS NOT NULL;
