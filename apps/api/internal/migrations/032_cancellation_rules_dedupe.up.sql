-- ── Cancellation rules: remove stale duplicate seeds, enforce uniqueness ───────
-- Migration 003 seeded a first-pass cancellation policy per vertical. Migration
-- 005 (dva_card_trust_cancellation) intentionally superseded it with a more
-- detailed, per-status policy — but never deleted 003's overlapping rows, so
-- both survived. FindCancellationRule (repo/ledger.go) does a bare `First()`
-- with no tiebreak between two rows of the same (vertical, status), so which
-- policy applied on cancellation was whichever row Postgres happened to return
-- first — confirmed nondeterministic across food/gas/grocery/pharmacy, with gas
-- alone swinging between a 10% and a 100% merchant clawback.
--
-- This migration deletes the specific stale 003-origin rows (matched by their
-- exact original values, not a generic "keep newest" heuristic, so a reseed
-- inserted after this migration can't accidentally get deleted too), then adds
-- a uniqueness constraint so the duplication class cannot recur.

DELETE FROM cancellation_rules
WHERE (vertical, order_status_at_cancel, merchant_comp_pct, full_refund) IN (
    ('food',     'preparing', 0.30, FALSE),
    ('grocery',  'preparing', 0.20, FALSE),
    ('pharmacy', 'preparing', 0.20, FALSE),
    ('gas',      'preparing', 0.10, FALSE)
);

-- '*' pending/confirmed duplicates have identical values (both full_refund=TRUE,
-- 0 comp) — no economic ambiguity, just redundant rows. Keep the oldest by id.
DELETE FROM cancellation_rules a
USING cancellation_rules b
WHERE a.vertical = '*'
  AND a.order_status_at_cancel IN ('pending', 'confirmed')
  AND b.vertical = a.vertical
  AND b.order_status_at_cancel = a.order_status_at_cancel
  AND b.id < a.id;

ALTER TABLE cancellation_rules
    ADD CONSTRAINT uq_cancellation_rules_vertical_status
        UNIQUE (vertical, order_status_at_cancel);
