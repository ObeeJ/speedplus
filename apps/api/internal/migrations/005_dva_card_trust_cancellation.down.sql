DELETE FROM cancellation_rules WHERE vertical IN ('*','food','gas','grocery','pharmacy','package');
DROP TABLE IF EXISTS user_trust_tiers;
DROP TABLE IF EXISTS user_cards;
DROP TABLE IF EXISTS virtual_accounts;
ALTER TABLE users DROP COLUMN IF EXISTS username;
