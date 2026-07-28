DROP INDEX IF EXISTS idx_users_referral_code;
ALTER TABLE users DROP COLUMN IF EXISTS referral_code;
ALTER TABLE pricing_quotes DROP COLUMN IF EXISTS weather_advisory;
