-- order_reviews: one review per order per reviewer (customer reviews driver/merchant after delivery)
CREATE TABLE IF NOT EXISTS order_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID NOT NULL REFERENCES orders(id),
    reviewer_id     UUID NOT NULL,                          -- customer user ID
    reviewee_id     UUID NOT NULL,                          -- driver user ID or merchant ID
    reviewee_type   VARCHAR(10) NOT NULL,                   -- 'driver' | 'merchant'
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (order_id, reviewee_type)                        -- one review per role per order
);

CREATE INDEX idx_order_reviews_reviewee ON order_reviews (reviewee_id, reviewee_type);
CREATE INDEX idx_order_reviews_order    ON order_reviews (order_id);

-- driver_badges: awarded automatically by the platform based on milestones
CREATE TABLE IF NOT EXISTS driver_badges (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id           UUID NOT NULL,
    badge_type          VARCHAR(30) NOT NULL,               -- 'first_delivery'|'10_deliveries'|'50_deliveries'|'100_deliveries'|'top_rated'|'zero_complaints'
    order_count_at_award INT NOT NULL DEFAULT 0,
    awarded_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (driver_id, badge_type)
);

CREATE INDEX idx_driver_badges_driver ON driver_badges (driver_id);
