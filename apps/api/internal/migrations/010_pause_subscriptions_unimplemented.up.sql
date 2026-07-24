-- subscription.chargeOne has no real order-creation implementation yet
-- (see internal/service/subscription.go). Any subscription left 'active'
-- would otherwise silently cycle in ProcessDue with no charge and no
-- delivery until the dunning counter caught it after ~3 days. Pause every
-- currently-active subscription immediately rather than waiting on dunning.
UPDATE subscriptions SET status = 'paused' WHERE status = 'active';
