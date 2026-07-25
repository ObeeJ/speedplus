-- Phase 5: Wallet & Ledger

CREATE TABLE ledger_accounts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(20) NOT NULL,
    currency   VARCHAR(10) NOT NULL DEFAULT 'NGN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ledger_accounts_owner ON ledger_accounts(owner_id);
CREATE UNIQUE INDEX idx_ledger_accounts_owner_type ON ledger_accounts(owner_id, type) WHERE owner_id IS NOT NULL;

CREATE TABLE ledger_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id   UUID NOT NULL,
    account_id   UUID NOT NULL REFERENCES ledger_accounts(id),
    amount_kobo  BIGINT NOT NULL,
    description  TEXT NOT NULL,
    ref_type     TEXT,
    ref_id       UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ledger_entries_account ON ledger_entries(account_id);
CREATE INDEX idx_ledger_entries_journal ON ledger_entries(journal_id);
CREATE INDEX idx_ledger_entries_created ON ledger_entries(created_at DESC);
-- Append-only: no UPDATE or DELETE allowed on this table (enforced via policy)
CREATE RULE no_update_ledger AS ON UPDATE TO ledger_entries DO INSTEAD NOTHING;
CREATE RULE no_delete_ledger AS ON DELETE TO ledger_entries DO INSTEAD NOTHING;

CREATE TABLE wallet_balances (
    account_id    UUID PRIMARY KEY REFERENCES ledger_accounts(id),
    balance_kobo  BIGINT NOT NULL DEFAULT 0 CHECK (balance_kobo >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payment_intents (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id),
    amount_kobo      BIGINT NOT NULL,
    currency         VARCHAR(10) NOT NULL DEFAULT 'NGN',
    provider         TEXT NOT NULL,
    provider_ref     TEXT UNIQUE,
    status           TEXT NOT NULL DEFAULT 'pending',
    idempotency_key  TEXT NOT NULL UNIQUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_payment_intents_user ON payment_intents(user_id);

CREATE TABLE provider_transactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      TEXT NOT NULL,
    provider_ref  TEXT NOT NULL UNIQUE,
    amount_kobo   BIGINT NOT NULL,
    status        TEXT NOT NULL,
    raw_payload   JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider      TEXT NOT NULL,
    event_id      TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    payload       JSONB NOT NULL,
    processed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, event_id)
);

CREATE TABLE idempotency_keys (
    key          TEXT PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id),
    status_code  INT,
    response     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_idempotency_expires ON idempotency_keys(expires_at);

-- Phase 6: Escrow, Paycodes, Earnings, Splits

CREATE TABLE escrow_holds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL UNIQUE REFERENCES orders(id),
    account_id      UUID NOT NULL REFERENCES ledger_accounts(id),
    amount_kobo     BIGINT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'held',
    released_at     TIMESTAMPTZ,
    released_by     UUID,
    frozen_at       TIMESTAMPTZ,
    frozen_reason   TEXT,
    approval_one    UUID REFERENCES users(id),
    approval_two    UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_escrow_holds_order ON escrow_holds(order_id);
CREATE INDEX idx_escrow_holds_status ON escrow_holds(status);

CREATE TABLE paycodes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL REFERENCES orders(id),
    nonce         TEXT NOT NULL UNIQUE,
    payload       TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    scanned_by    UUID REFERENCES users(id),
    confirmed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_paycodes_order ON paycodes(order_id);

CREATE TABLE driver_earnings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id    UUID NOT NULL REFERENCES users(id),
    order_id     UUID NOT NULL REFERENCES orders(id),
    amount_kobo  BIGINT NOT NULL,
    tip_kobo     BIGINT NOT NULL DEFAULT 0,
    paid_out_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_driver_earnings_driver ON driver_earnings(driver_id);
CREATE INDEX idx_driver_earnings_unpaid ON driver_earnings(driver_id) WHERE paid_out_at IS NULL;

CREATE TABLE cashout_requests (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id        UUID NOT NULL REFERENCES users(id),
    amount_kobo      BIGINT NOT NULL,
    fee_kobo         BIGINT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    provider_ref     TEXT,
    idempotency_key  TEXT NOT NULL UNIQUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cashout_driver ON cashout_requests(driver_id);

CREATE TABLE order_splits (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         UUID NOT NULL REFERENCES orders(id),
    participant_id   UUID NOT NULL REFERENCES users(id),
    share_kobo       BIGINT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'pending',
    funded_at        TIMESTAMPTZ,
    refunded_at      TIMESTAMPTZ,
    idempotency_key  TEXT NOT NULL UNIQUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_order_splits_order ON order_splits(order_id);

CREATE TABLE cancellation_rules (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vertical                    TEXT NOT NULL DEFAULT '*',
    order_status_at_cancel      TEXT NOT NULL,
    merchant_comp_kobo          BIGINT NOT NULL DEFAULT 0,
    merchant_comp_pct           DOUBLE PRECISION NOT NULL DEFAULT 0,
    rider_comp_pct_of_delivery  DOUBLE PRECISION NOT NULL DEFAULT 0,
    full_refund                 BOOLEAN NOT NULL DEFAULT FALSE
);

-- Seed default cancellation rules
INSERT INTO cancellation_rules (vertical, order_status_at_cancel, full_refund) VALUES
    ('*', 'pending', TRUE),
    ('*', 'confirmed', TRUE);

INSERT INTO cancellation_rules (vertical, order_status_at_cancel, merchant_comp_pct, full_refund) VALUES
    ('food', 'preparing', 0.30, FALSE),
    ('grocery', 'preparing', 0.20, FALSE),
    ('pharmacy', 'preparing', 0.20, FALSE),
    ('gas', 'preparing', 0.10, FALSE);

INSERT INTO cancellation_rules (vertical, order_status_at_cancel, merchant_comp_pct, rider_comp_pct_of_delivery, full_refund) VALUES
    ('*', 'driver_assigned', 0.10, 0.50, FALSE),
    ('*', 'in_transit', 0.10, 0.75, FALSE);
