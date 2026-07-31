ALTER TABLE subscriptions DROP COLUMN IF EXISTS paused_reason;
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS fk_subscriptions_cylinder_spec;
ALTER TABLE cylinder_custody_events DROP CONSTRAINT IF EXISTS fk_custody_events_cylinder;
ALTER TABLE products DROP COLUMN IF EXISTS cylinder_spec_id;
