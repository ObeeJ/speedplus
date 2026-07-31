-- Re-pause gas subscriptions (restore migration 010 state for gas).
UPDATE subscriptions SET status = 'paused' WHERE vertical = 'gas' AND status = 'active';

DROP RULE IF EXISTS no_delete_lpg_price ON lpg_price_index;
DROP RULE IF EXISTS no_update_lpg_price ON lpg_price_index;
DROP TABLE IF EXISTS lpg_price_index;

ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS avg_days_between_refills,
    DROP COLUMN IF EXISTS predicted_runout_at,
    DROP COLUMN IF EXISTS gas_mode,
    DROP COLUMN IF EXISTS cylinder_spec_id;
