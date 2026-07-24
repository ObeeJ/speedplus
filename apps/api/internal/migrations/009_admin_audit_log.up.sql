-- Admin audit log — append-only record of every admin action that mutates
-- user, order, or escrow state. Never update or delete rows.
CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id     UUID        NOT NULL REFERENCES users(id),
    action       TEXT        NOT NULL,
    target_type  TEXT        NOT NULL,
    target_id    UUID        NOT NULL,
    reason       TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_admin_audit_admin    ON admin_audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_target   ON admin_audit_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_created  ON admin_audit_logs(created_at DESC);

-- Enforce append-only at the DB layer
CREATE RULE no_update_admin_audit AS ON UPDATE TO admin_audit_logs DO INSTEAD NOTHING;
CREATE RULE no_delete_admin_audit AS ON DELETE TO admin_audit_logs DO INSTEAD NOTHING;
