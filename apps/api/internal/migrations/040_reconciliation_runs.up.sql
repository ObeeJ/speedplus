-- Migration 040: Provider reconciliation runs
--
-- reconciliation_runs: one row per daily provider reconciliation job.
-- Drift > 0 means the provider settled more than we recorded (or vice versa).
-- Any non-zero drift triggers an alert.

CREATE TABLE reconciliation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        TEXT NOT NULL,          -- 'paystack'|'flutterwave'|'monnify'
    run_date        DATE NOT NULL,          -- the settlement date being reconciled
    provider_total  BIGINT NOT NULL,        -- sum from provider settlement report (kobo)
    ledger_total    BIGINT NOT NULL,        -- sum from payment_intents + ledger (kobo)
    drift_kobo      BIGINT NOT NULL,        -- provider_total - ledger_total
    status          TEXT NOT NULL DEFAULT 'clean', -- 'clean'|'drift_detected'|'error'
    error_detail    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider, run_date)
);
CREATE INDEX idx_reconciliation_runs_drift ON reconciliation_runs(run_date DESC) WHERE drift_kobo != 0;
