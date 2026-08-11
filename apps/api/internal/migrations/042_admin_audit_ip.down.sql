ALTER TABLE admin_audit_logs
    DROP COLUMN IF EXISTS ip,
    DROP COLUMN IF EXISTS user_agent;
