-- ── Merchant fill-status remediation ──────────────────────────────────────────
-- fill_accuracy_pct alone is a number nobody acts on. fill_status turns it into
-- a state a merchant can see and recover from:
--   good      — no concern
--   warned    — accuracy trending short; visible to the merchant, not customers
--   probation — offered less prominently; needs sustained improvement to clear
--   delisted  — blocked from new gas orders (enforced in OrderService.Create)
-- Recomputed nightly (service.RecomputeFillAccuracy) from a rolling window of
-- each merchant's most recent verified fills, not all-time history — so a
-- merchant that recalibrates a bad scale can earn back to 'good' over time
-- rather than being permanently marked by one bad patch.

ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS fill_status TEXT NOT NULL DEFAULT 'good'
        CHECK (fill_status IN ('good', 'warned', 'probation', 'delisted'));
