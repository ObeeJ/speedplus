-- ── Virtual accounts (DVA) ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS virtual_accounts (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    account_number TEXT NOT NULL,
    bank_name      TEXT NOT NULL,
    bank_code      TEXT NOT NULL,
    provider       TEXT NOT NULL DEFAULT 'monnify',
    provider_ref   TEXT NOT NULL,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── SpeedPlus card (identity QR) ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_cards (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    payload    TEXT NOT NULL,   -- spd.card.v1.{userID}.{HMAC-SHA256}
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Trust tiers ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_trust_tiers (
    user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    tier             SMALLINT NOT NULL DEFAULT 0,
    completed_orders INT      NOT NULL DEFAULT 0,
    fraud_flags      INT      NOT NULL DEFAULT 0,
    frozen           BOOLEAN  NOT NULL DEFAULT FALSE,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Username column on users ──────────────────────────────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS username TEXT UNIQUE;

-- ── Cancellation rules seed data ─────────────────────────────────────────────
-- Existing rows are left untouched; INSERT ... ON CONFLICT DO NOTHING is safe
-- to re-run.

-- All verticals: free cancel within pending (3-min window enforced in app)
INSERT INTO cancellation_rules (id, vertical, order_status_at_cancel, full_refund, merchant_comp_pct, rider_comp_pct_of_delivery)
VALUES
    (gen_random_uuid(), '*',        'pending',          TRUE,  0,    0),
    (gen_random_uuid(), '*',        'confirmed',        TRUE,  0,    0),

-- Food
    (gen_random_uuid(), 'food',     'preparing',        FALSE, 0.50, 0),
    (gen_random_uuid(), 'food',     'ready_for_pickup', FALSE, 1.00, 0),
    (gen_random_uuid(), 'food',     'driver_assigned',  FALSE, 1.00, 0.50),
    (gen_random_uuid(), 'food',     'in_transit',       FALSE, 1.00, 1.00),

-- Gas (irreversible once dispensed)
    (gen_random_uuid(), 'gas',      'preparing',        FALSE, 1.00, 0),
    (gen_random_uuid(), 'gas',      'ready_for_pickup', FALSE, 1.00, 0),
    (gen_random_uuid(), 'gas',      'driver_assigned',  FALSE, 1.00, 0.50),
    (gen_random_uuid(), 'gas',      'in_transit',       FALSE, 1.00, 1.00),

-- Grocery
    (gen_random_uuid(), 'grocery',  'preparing',        FALSE, 0.50, 0),
    (gen_random_uuid(), 'grocery',  'ready_for_pickup', FALSE, 1.00, 0),
    (gen_random_uuid(), 'grocery',  'driver_assigned',  FALSE, 1.00, 0.50),
    (gen_random_uuid(), 'grocery',  'in_transit',       FALSE, 1.00, 1.00),

-- Pharmacy (always full refund — pills not dispensed until pickup)
    (gen_random_uuid(), 'pharmacy', 'preparing',        TRUE,  0,    0),
    (gen_random_uuid(), 'pharmacy', 'ready_for_pickup', TRUE,  0,    0),
    (gen_random_uuid(), 'pharmacy', 'driver_assigned',  TRUE,  0,    0),
    (gen_random_uuid(), 'pharmacy', 'in_transit',       TRUE,  0,    0),

-- Package
    (gen_random_uuid(), 'package',  'preparing',        FALSE, 0.50, 0),
    (gen_random_uuid(), 'package',  'ready_for_pickup', FALSE, 1.00, 0),
    (gen_random_uuid(), 'package',  'driver_assigned',  FALSE, 1.00, 0.50),
    (gen_random_uuid(), 'package',  'in_transit',       FALSE, 1.00, 1.00)

ON CONFLICT DO NOTHING;
