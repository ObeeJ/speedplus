-- platform_settings: append-only key/value store for platform-level toggles.
-- Consistent with fee_configs: insert-only, audit-logged, latest row wins.
-- Postgres rules prevent UPDATE/DELETE so history is always preserved.

CREATE TABLE IF NOT EXISTS platform_settings (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key             TEXT        NOT NULL,
    value           TEXT        NOT NULL,
    updated_by      UUID        NOT NULL REFERENCES users(id),
    reason          TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_settings_key_time ON platform_settings(key, created_at DESC);

CREATE RULE no_update_platform_settings AS ON UPDATE TO platform_settings DO INSTEAD NOTHING;
CREATE RULE no_delete_platform_settings AS ON DELETE TO platform_settings DO INSTEAD NOTHING;

-- Seed defaults: surcharge disabled, amount ₦200 (20000 kobo).
INSERT INTO platform_settings (key, value, updated_by, reason)
SELECT 'weather_surcharge_enabled', 'false',
       '00000000-0000-0000-0000-000000000009',
       'initial default: surcharge off'
WHERE NOT EXISTS (SELECT 1 FROM platform_settings WHERE key = 'weather_surcharge_enabled');

INSERT INTO platform_settings (key, value, updated_by, reason)
SELECT 'weather_surcharge_kobo', '20000',
       '00000000-0000-0000-0000-000000000009',
       'initial default: ₦200'
WHERE NOT EXISTS (SELECT 1 FROM platform_settings WHERE key = 'weather_surcharge_kobo');

-- Add weather_surcharge_kobo to pricing_quotes so the amount is stored on the
-- quote at creation time and never recalculated at settlement.
ALTER TABLE pricing_quotes
    ADD COLUMN IF NOT EXISTS weather_surcharge_kobo BIGINT NOT NULL DEFAULT 0;
