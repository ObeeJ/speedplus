-- ── Product ↔ cylinder spec link ──────────────────────────────────────────────
-- Lets chargeOne (and the catalog generally) resolve "which product is this
-- cylinder size" for a merchant, instead of guessing by name or defaulting to
-- the cheapest product regardless of the subscription's chosen size.

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS cylinder_spec_id UUID REFERENCES cylinder_specs(id);

UPDATE products SET cylinder_spec_id = '00000000-0000-0000-0000-000000000021'
    WHERE id = '00000000-0000-0000-0000-000000000011'; -- 3kg
UPDATE products SET cylinder_spec_id = '00000000-0000-0000-0000-000000000022'
    WHERE id = '00000000-0000-0000-0000-000000000012'; -- 6kg
UPDATE products SET cylinder_spec_id = '00000000-0000-0000-0000-000000000023'
    WHERE id = '00000000-0000-0000-0000-000000000013'; -- 12.5kg
UPDATE products SET cylinder_spec_id = '00000000-0000-0000-0000-000000000024'
    WHERE id = '00000000-0000-0000-0000-000000000014'; -- 25kg

-- ── Close dangling FK promises from migrations 024 and 026 ───────────────────
-- 024 left cylinder_custody_events.cylinder_id unconstrained pending Phase 1;
-- 026 never constrained subscriptions.cylinder_spec_id at all. Both target
-- tables now exist (customer_cylinders since 027, cylinder_specs since 028),
-- so both FKs can be added.

ALTER TABLE cylinder_custody_events
    ADD CONSTRAINT fk_custody_events_cylinder
        FOREIGN KEY (cylinder_id) REFERENCES customer_cylinders(id);

ALTER TABLE subscriptions
    ADD CONSTRAINT fk_subscriptions_cylinder_spec
        FOREIGN KEY (cylinder_spec_id) REFERENCES cylinder_specs(id);

-- ── Subscription pause provenance ─────────────────────────────────────────────
-- Distinguishes why a subscription is paused: 'customer' (self-service pause),
-- 'dunning' (3 failed charge attempts), or 'system' (migration-level pause,
-- e.g. 010's force-pause while chargeOne was unimplemented). Needed so future
-- migrations can reactivate system-paused rows without touching subscriptions
-- a customer deliberately stopped.

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS paused_reason TEXT DEFAULT NULL;
