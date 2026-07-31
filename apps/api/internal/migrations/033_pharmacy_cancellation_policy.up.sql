-- ── Pharmacy cancellation policy: replace always-full-refund with dispensing-
-- aware compensation, mirroring food's policy ──────────────────────────────────
-- Migration 005 seeded pharmacy as full refund at every pre-delivery status,
-- reasoning "pills not dispensed until pickup." That model is wrong: a pharmacy
-- counts, labels and allocates stock to a named patient the moment it accepts
-- the order (at `preparing`) — the same point food starts being non-resellable.
-- Once dispensed, prescription medicine cannot legally re-enter pharmacy stock
-- in Nigeria, so a 100%-refund policy has the pharmacy eat the full cost of
-- every cancellation from `preparing` onward. That is not commercially viable
-- for a pharmacy merchant and would block onboarding.
--
-- This mirrors food's canonical (005) policy shape 1:1, since a filled
-- prescription is at least as unsellable as prepared food at the same stage:
--   preparing         → 50% merchant comp
--   ready_for_pickup  → 100% merchant comp
--   driver_assigned   → 100% merchant comp + 50% rider comp
--   in_transit        → 100% merchant comp + 100% rider comp
-- pending/confirmed remain full refund via the existing '*' rule (pre-dispense).

UPDATE cancellation_rules
SET full_refund = FALSE, merchant_comp_pct = 0.50, rider_comp_pct_of_delivery = 0
WHERE vertical = 'pharmacy' AND order_status_at_cancel = 'preparing';

UPDATE cancellation_rules
SET full_refund = FALSE, merchant_comp_pct = 1.00, rider_comp_pct_of_delivery = 0
WHERE vertical = 'pharmacy' AND order_status_at_cancel = 'ready_for_pickup';

UPDATE cancellation_rules
SET full_refund = FALSE, merchant_comp_pct = 1.00, rider_comp_pct_of_delivery = 0.50
WHERE vertical = 'pharmacy' AND order_status_at_cancel = 'driver_assigned';

UPDATE cancellation_rules
SET full_refund = FALSE, merchant_comp_pct = 1.00, rider_comp_pct_of_delivery = 1.00
WHERE vertical = 'pharmacy' AND order_status_at_cancel = 'in_transit';
