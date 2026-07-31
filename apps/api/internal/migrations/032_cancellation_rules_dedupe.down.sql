ALTER TABLE cancellation_rules DROP CONSTRAINT IF EXISTS uq_cancellation_rules_vertical_status;

-- Restore the 003-origin rows this migration deleted, so the down migration is
-- a true rollback of 032's up.sql (not a rollback of 003/005 themselves).
INSERT INTO cancellation_rules (vertical, order_status_at_cancel, merchant_comp_pct, full_refund) VALUES
    ('food', 'preparing', 0.30, FALSE),
    ('grocery', 'preparing', 0.20, FALSE),
    ('pharmacy', 'preparing', 0.20, FALSE),
    ('gas', 'preparing', 0.10, FALSE);
