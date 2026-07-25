DROP INDEX IF EXISTS idx_cashout_merchant;
ALTER TABLE cashout_requests DROP COLUMN IF EXISTS merchant_id, DROP COLUMN IF EXISTS actor_type;
DROP TABLE IF EXISTS merchant_bank_accounts;
