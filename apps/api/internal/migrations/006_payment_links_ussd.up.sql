-- Payment links (P2P) and USSD funding intents.

CREATE TABLE IF NOT EXISTS payment_links (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            VARCHAR(16) UNIQUE NOT NULL,
    creator_id      UUID NOT NULL REFERENCES users(id),
    amount_kobo     BIGINT NOT NULL CHECK (amount_kobo > 0),
    note            TEXT,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending|paid|cancelled|expired
    paid_by_user_id UUID REFERENCES users(id),
    paid_by_email   VARCHAR(100),
    provider_ref    VARCHAR(100),
    paid_at         TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    journal_id      UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payment_links_creator ON payment_links(creator_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payment_links_status  ON payment_links(status) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS ussd_intents (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id),
    amount_kobo       BIGINT NOT NULL CHECK (amount_kobo > 0),
    provider          VARCHAR(20) NOT NULL,
    bank_code         VARCHAR(10) NOT NULL,
    bank_name         VARCHAR(100),
    ussd_code         VARCHAR(64) NOT NULL,
    provider_ref      VARCHAR(100) UNIQUE NOT NULL,
    payment_intent_id UUID REFERENCES payment_intents(id),
    status            VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending|paid|expired|failed
    expires_at        TIMESTAMPTZ NOT NULL,
    paid_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ussd_intents_user   ON ussd_intents(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ussd_intents_status ON ussd_intents(status) WHERE status = 'pending';
