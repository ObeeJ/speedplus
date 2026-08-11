ALTER TABLE pins DROP COLUMN IF EXISTS failed_attempts, DROP COLUMN IF EXISTS locked_until;
DROP TABLE IF EXISTS driver_bank_accounts;
