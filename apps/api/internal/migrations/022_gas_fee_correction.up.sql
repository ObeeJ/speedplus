-- ── System user for fee-config attribution ────────────────────────────────────
-- fee_configs.updated_by is NOT NULL with an FK to users(id). No prior
-- migration seeds any admin user, so the previous
--   SELECT id FROM users WHERE role='admin' LIMIT 1
-- pattern matched zero rows on a fresh database and silently inserted
-- nothing — the headline gas pricing fix never landed. Seed a deterministic
-- system user so this migration's fee correction always applies.

INSERT INTO users (id, role, first_name, last_name, phone, password_hash, is_verified, is_active, referral_code)
VALUES (
    '00000000-0000-0000-0000-000000000009',
    'admin',
    'SpeedPlus',
    'System',
    '+2340000000009',
    '$argon2id$v=19$m=65536,t=1,p=4$73797374656d$73797374656d73797374656d73797374656d73797374656d73797374656d3132',
    true,
    true,
    'SPEEDSYS'
)
ON CONFLICT (id) DO NOTHING;

-- ── Gas fee correction ────────────────────────────────────────────────────────
-- Replaces the flat ₦1,500 / PerKm=0 gas config with distance- and
-- weight-sensitive rates that make a single un-batched delivery survivable
-- and a batched run profitable. Append-only: insert, never update.

-- fee_configs is append-only (migration 012's no_update/no_delete RULEs), and
-- Postgres rejects ON CONFLICT entirely on any table that has rules — so this
-- must be a plain conditional INSERT, not INSERT ... ON CONFLICT DO NOTHING.
INSERT INTO fee_configs (
    id, vertical,
    base_fee_kobo, per_km_kobo, per_kg_kobo, per_stop_kobo,
    service_pct, merchant_take_rate, driver_take_rate, platform_take_rate,
    fuel_price_ref_kobo, effective_at, updated_by, reason
)
SELECT
    '00000000-0000-0000-0000-000000000010',
    'gas',
    80000, 22000, 2000, 25000,
    0.03, 0.92, 0.80, 0.20,
    140000,
    NOW(),
    '00000000-0000-0000-0000-000000000009',
    'Phase 0: distance+weight pricing replaces flat ₦1500/PerKm=0; enables batched routes'
WHERE NOT EXISTS (
    SELECT 1 FROM fee_configs WHERE id = '00000000-0000-0000-0000-000000000010'
);

-- ── SpeedPlus Gas merchant ─────────────────────────────────────────────────────
-- Platform-owned gas merchant, mirroring the package merchant in migration 015.

INSERT INTO users (id, role, first_name, last_name, phone, password_hash, is_verified, is_active, referral_code)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'merchant',
    'SpeedPlus',
    'Gas',
    '+2340000000003',
    '$argon2id$v=19$m=65536,t=1,p=4$73797374656d$73797374656d73797374656d73797374656d73797374656d73797374656d3132',
    true,
    true,
    'SPEEDGAS'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO merchants (id, user_id, business_name, vertical, status, is_open, lat, lng)
VALUES (
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000003',
    'SpeedPlus Gas',
    'gas',
    'active',
    true,
    6.5244,
    3.3792
)
ON CONFLICT (id) DO NOTHING;

-- ── Cylinder products ──────────────────────────────────────────────────────────
-- One product per standard cylinder size. unit_price_kobo = ₦/kg × kg.
-- These are placeholder prices; the LPG index (Phase 5) will drive live rates.

INSERT INTO products (id, merchant_id, name, description, price_kobo, is_available)
VALUES
    ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000004',
     '3kg LPG cylinder',   '3 kg refillable LPG cylinder',  320000,  true),
    ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000004',
     '6kg LPG cylinder',   '6 kg refillable LPG cylinder',  610000,  true),
    ('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000004',
     '12.5kg LPG cylinder','12.5 kg refillable LPG cylinder',1180000, true),
    ('00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000004',
     '25kg LPG cylinder',  '25 kg refillable LPG cylinder', 2300000, true)
ON CONFLICT (id) DO NOTHING;
