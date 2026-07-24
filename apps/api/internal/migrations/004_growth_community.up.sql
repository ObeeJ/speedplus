-- Phase 7: Growth

CREATE TABLE referrals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id     UUID NOT NULL REFERENCES users(id),
    referee_id      UUID NOT NULL UNIQUE REFERENCES users(id),
    reward_paid_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);

CREATE TABLE gift_cards (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash    TEXT NOT NULL UNIQUE,
    amount_kobo  BIGINT NOT NULL,
    issuer_id    UUID NOT NULL REFERENCES users(id),
    redeemed_by  UUID REFERENCES users(id),
    redeemed_at  TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE loyalty_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    event_type  TEXT NOT NULL,
    points      INT NOT NULL,
    ref_id      UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_loyalty_events_user ON loyalty_events(user_id);
CREATE INDEX idx_loyalty_events_user_day ON loyalty_events(user_id, DATE(created_at));

CREATE TABLE loyalty_balances (
    user_id     UUID PRIMARY KEY REFERENCES users(id),
    points      INT NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id      UUID NOT NULL REFERENCES users(id),
    merchant_id      UUID NOT NULL REFERENCES merchants(id),
    vertical         TEXT NOT NULL,
    cadence          TEXT NOT NULL,
    address_id       UUID NOT NULL REFERENCES addresses(id),
    payment_method   TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'active',
    next_charge_at   TIMESTAMPTZ NOT NULL,
    dunning_count    INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_subscriptions_customer ON subscriptions(customer_id);
CREATE INDEX idx_subscriptions_next_charge ON subscriptions(next_charge_at) WHERE status = 'active';

-- Phase 8: Community

CREATE TABLE campaigns (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id          UUID NOT NULL REFERENCES users(id),
    title               TEXT NOT NULL,
    description         TEXT,
    goal_kobo           BIGINT NOT NULL,
    escrow_account_id   UUID NOT NULL REFERENCES ledger_accounts(id),
    status              TEXT NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE campaign_contributions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     UUID NOT NULL REFERENCES campaigns(id),
    contributor_id  UUID NOT NULL REFERENCES users(id),
    amount_kobo     BIGINT NOT NULL,
    journal_id      UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_campaign_contributions_campaign ON campaign_contributions(campaign_id);

CREATE TABLE catering_rfqs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id  UUID NOT NULL REFERENCES users(id),
    description  TEXT NOT NULL,
    event_date   TIMESTAMPTZ,
    status       TEXT NOT NULL DEFAULT 'open',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catering_quotes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rfq_id       UUID NOT NULL REFERENCES catering_rfqs(id),
    merchant_id  UUID NOT NULL REFERENCES merchants(id),
    amount_kobo  BIGINT NOT NULL,
    note         TEXT,
    status       TEXT NOT NULL DEFAULT 'pending',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_catering_quotes_rfq ON catering_quotes(rfq_id);
