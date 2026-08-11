-- Composite indexes for the admin metrics queries and order history pages.
-- orders(created_at) alone covers the "today" range scans.
-- orders(status, created_at) covers the filtered counts (cancelled today, etc.).
-- payment_intents(status, created_at) covers the failed-payments count.
-- All use IF NOT EXISTS so re-running is safe.

CREATE INDEX IF NOT EXISTS idx_orders_created_at
    ON orders(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_orders_status_created_at
    ON orders(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payment_intents_status_created_at
    ON payment_intents(status, created_at DESC);
