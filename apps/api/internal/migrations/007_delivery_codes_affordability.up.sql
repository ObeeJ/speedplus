-- ── Delivery codes (one-time 6-digit code per order) ─────────────────────────
-- Generated when order transitions to in_transit.
-- Customer shares with whoever is collecting. Rider enters on their app.
-- SpeedPlus card is the fallback when customer has no phone signal.
CREATE TABLE IF NOT EXISTS delivery_codes (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID        NOT NULL UNIQUE REFERENCES orders(id) ON DELETE CASCADE,
    code_hash   TEXT        NOT NULL,
    attempts    INT         NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Affordability query index ─────────────────────────────────────────────────
-- Supports: SELECT vertical, AVG(total_kobo) FROM orders
--           WHERE merchant_id IN (...nearby merchants...)
--             AND status = 'delivered'
--             AND created_at > NOW() - INTERVAL '30 days'
--           GROUP BY vertical
CREATE INDEX IF NOT EXISTS idx_orders_affordability
    ON orders (vertical, status, created_at DESC)
    WHERE status = 'delivered';
