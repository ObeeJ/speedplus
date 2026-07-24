-- Add referral_code to users (unique, generated on registration)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS referral_code TEXT NOT NULL DEFAULT '';

-- Back-fill existing rows with a random 7-char code so the unique index can be created.
UPDATE users
SET referral_code = upper(substring(md5(random()::text || id::text) for 7))
WHERE referral_code = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);

-- Add weather_advisory to pricing_quotes (informational only, no fee)
ALTER TABLE pricing_quotes
    ADD COLUMN IF NOT EXISTS weather_advisory TEXT NOT NULL DEFAULT '';
