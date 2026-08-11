DROP RULE IF EXISTS no_update_platform_settings ON platform_settings;
DROP RULE IF EXISTS no_delete_platform_settings ON platform_settings;
DROP TABLE IF EXISTS platform_settings;
ALTER TABLE pricing_quotes DROP COLUMN IF EXISTS weather_surcharge_kobo;
