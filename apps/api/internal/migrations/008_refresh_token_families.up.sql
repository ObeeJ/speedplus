-- Refresh token rotation needs family tracking to detect reuse of a revoked
-- token (a signal of theft/replay). All tokens issued from the same
-- login/registration/rotation chain share a family_id; on detected reuse the
-- whole family is revoked, not just the single presented token.
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS family_id UUID;
UPDATE refresh_tokens SET family_id = id WHERE family_id IS NULL;
ALTER TABLE refresh_tokens ALTER COLUMN family_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family ON refresh_tokens(family_id);
