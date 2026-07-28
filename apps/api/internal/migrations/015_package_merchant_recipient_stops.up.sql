-- ── SpeedPlus Logistics merchant (package vertical) ──────────────────────────
-- Platform-owned merchant for all package delivery orders.
-- Deterministic UUID so it can be referenced in env vars and tests.

INSERT INTO users (id, role, first_name, last_name, phone, password_hash, is_verified, is_active, referral_code)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'merchant',
    'SpeedPlus',
    'Logistics',
    '+2340000000001',
    '$argon2id$v=19$m=65536,t=1,p=4$73797374656d$73797374656d73797374656d73797374656d73797374656d73797374656d3132',
    true,
    true,
    'SPEEDPLUS'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO merchants (id, user_id, business_name, vertical, status, is_open, lat, lng)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001',
    'SpeedPlus Logistics',
    'package',
    'active',
    true,
    6.5244,
    3.3792
)
ON CONFLICT (id) DO NOTHING;

-- ── Recipient fields on orders (package vertical) ─────────────────────────────
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS recipient_name  TEXT,
    ADD COLUMN IF NOT EXISTS recipient_phone TEXT;

-- payment_method already added by migration 013; guard with IF NOT EXISTS
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payment_method  VARCHAR(20) NOT NULL DEFAULT 'wallet';

-- ── order_stops: add recipient fields per stop ────────────────────────────────
ALTER TABLE order_stops
    ADD COLUMN IF NOT EXISTS recipient_name  TEXT,
    ADD COLUMN IF NOT EXISTS recipient_phone TEXT,
    ADD COLUMN IF NOT EXISTS notes           TEXT;

CREATE INDEX IF NOT EXISTS idx_order_stops_order ON order_stops(order_id, sequence);
