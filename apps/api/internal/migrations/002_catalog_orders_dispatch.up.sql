-- Phase 2: Catalog & Pricing

CREATE TABLE merchants (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    business_name  TEXT NOT NULL,
    vertical       VARCHAR(20) NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'pending',
    rating         DOUBLE PRECISION NOT NULL DEFAULT 5.0,
    is_open        BOOLEAN NOT NULL DEFAULT FALSE,
    lat            DOUBLE PRECISION,
    lng            DOUBLE PRECISION,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_merchants_vertical ON merchants(vertical);
CREATE INDEX idx_merchants_status ON merchants(status);

CREATE TABLE products (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id  UUID NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT,
    price_kobo   BIGINT NOT NULL CHECK (price_kobo >= 0),
    category     TEXT,
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    search_vec   TSVECTOR,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_products_merchant ON products(merchant_id);
CREATE INDEX idx_products_search ON products USING GIN(search_vec);

-- Auto-update tsvector on insert/update
CREATE OR REPLACE FUNCTION products_search_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vec := to_tsvector('english', COALESCE(NEW.name,'') || ' ' || COALESCE(NEW.description,'') || ' ' || COALESCE(NEW.category,''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER products_search_trigger
BEFORE INSERT OR UPDATE ON products
FOR EACH ROW EXECUTE FUNCTION products_search_update();

CREATE TABLE pricing_quotes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id    UUID NOT NULL REFERENCES users(id),
    merchant_id    UUID NOT NULL REFERENCES merchants(id),
    distance_km    DOUBLE PRECISION,
    subtotal_kobo  BIGINT NOT NULL,
    delivery_kobo  BIGINT NOT NULL,
    service_kobo   BIGINT NOT NULL,
    total_kobo     BIGINT NOT NULL,
    quote_hash     TEXT NOT NULL UNIQUE,
    expires_at     TIMESTAMPTZ NOT NULL,
    used_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Phase 3: Orders

CREATE TABLE orders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id         UUID NOT NULL REFERENCES users(id),
    merchant_id         UUID NOT NULL REFERENCES merchants(id),
    driver_id           UUID REFERENCES users(id),
    quote_id            UUID NOT NULL REFERENCES pricing_quotes(id),
    vertical            VARCHAR(20) NOT NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'pending',
    subtotal_kobo       BIGINT NOT NULL,
    delivery_kobo       BIGINT NOT NULL,
    service_kobo        BIGINT NOT NULL,
    tip_kobo            BIGINT NOT NULL DEFAULT 0,
    total_kobo          BIGINT NOT NULL,
    delivery_address_id UUID NOT NULL REFERENCES addresses(id),
    prescription_id     UUID,
    scheduled_for       TIMESTAMPTZ,
    estimated_at        TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    cancel_reason       TEXT,
    idempotency_key     TEXT NOT NULL UNIQUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_orders_merchant ON orders(merchant_id);
CREATE INDEX idx_orders_driver ON orders(driver_id);
CREATE INDEX idx_orders_status ON orders(status);

CREATE TABLE order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id),
    name                TEXT NOT NULL,
    quantity            INT NOT NULL CHECK (quantity > 0),
    unit_price_kobo     BIGINT NOT NULL,
    total_kobo          BIGINT NOT NULL,
    customizations      TEXT,
    substitution_pref   VARCHAR(10)
);
CREATE INDEX idx_order_items_order ON order_items(order_id);

CREATE TABLE order_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    from_status  VARCHAR(30),
    to_status    VARCHAR(30) NOT NULL,
    actor_id     UUID NOT NULL REFERENCES users(id),
    actor_role   TEXT NOT NULL,
    note         TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_order_events_order ON order_events(order_id);

CREATE TABLE order_stops (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    sequence      INT NOT NULL,
    address_id    UUID NOT NULL REFERENCES addresses(id),
    qr_code       TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    confirmed_at  TIMESTAMPTZ,
    UNIQUE(order_id, sequence)
);

CREATE TABLE prescriptions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id  UUID NOT NULL REFERENCES users(id),
    merchant_id  UUID REFERENCES merchants(id),
    r2_key       TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    reviewer_id  UUID REFERENCES users(id),
    review_note  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_prescriptions_customer ON prescriptions(customer_id);

-- Phase 4: Dispatch

CREATE TABLE driver_locations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id   UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    location    GEOMETRY(Point, 4326) NOT NULL,
    heading     DOUBLE PRECISION,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_driver_locations_gist ON driver_locations USING GIST(location);
CREATE INDEX idx_driver_locations_updated ON driver_locations(updated_at);

CREATE TABLE delivery_offers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    driver_id   UUID REFERENCES users(id),
    status      TEXT NOT NULL DEFAULT 'pending',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_delivery_offers_order ON delivery_offers(order_id);
CREATE INDEX idx_delivery_offers_driver ON delivery_offers(driver_id);
