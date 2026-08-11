-- Rollback: drop triggers and function, restore the old silent rules.
-- NOTE: rolling back means UPDATE/DELETE silently no-ops again.
-- Only use this if you need to unblock a hotfix; re-apply 038 immediately after.

DROP TRIGGER IF EXISTS immutable_ledger_entries              ON ledger_entries;
DROP TRIGGER IF EXISTS immutable_admin_audit_logs            ON admin_audit_logs;
DROP TRIGGER IF EXISTS immutable_fee_configs                 ON fee_configs;
DROP TRIGGER IF EXISTS immutable_proof_media                 ON proof_media;
DROP TRIGGER IF EXISTS immutable_cylinder_custody_events     ON cylinder_custody_events;
DROP TRIGGER IF EXISTS immutable_lpg_price_index             ON lpg_price_index;
DROP TRIGGER IF EXISTS immutable_cylinder_handover_checklists ON cylinder_handover_checklists;
DROP TRIGGER IF EXISTS immutable_platform_settings           ON platform_settings;

DROP FUNCTION IF EXISTS raise_immutable();

-- Restore silent rules
CREATE RULE no_update_ledger          AS ON UPDATE TO ledger_entries          DO INSTEAD NOTHING;
CREATE RULE no_delete_ledger          AS ON DELETE TO ledger_entries          DO INSTEAD NOTHING;
CREATE RULE no_update_admin_audit     AS ON UPDATE TO admin_audit_logs        DO INSTEAD NOTHING;
CREATE RULE no_delete_admin_audit     AS ON DELETE TO admin_audit_logs        DO INSTEAD NOTHING;
CREATE RULE no_update_fee_configs     AS ON UPDATE TO fee_configs             DO INSTEAD NOTHING;
CREATE RULE no_delete_fee_configs     AS ON DELETE TO fee_configs             DO INSTEAD NOTHING;
CREATE RULE no_update_proof_media     AS ON UPDATE TO proof_media             DO INSTEAD NOTHING;
CREATE RULE no_delete_proof_media     AS ON DELETE TO proof_media             DO INSTEAD NOTHING;
CREATE RULE no_update_custody_events  AS ON UPDATE TO cylinder_custody_events DO INSTEAD NOTHING;
CREATE RULE no_delete_custody_events  AS ON DELETE TO cylinder_custody_events DO INSTEAD NOTHING;
CREATE RULE no_update_lpg_price       AS ON UPDATE TO lpg_price_index         DO INSTEAD NOTHING;
CREATE RULE no_delete_lpg_price       AS ON DELETE TO lpg_price_index         DO INSTEAD NOTHING;
CREATE RULE no_update_handover_checklists AS ON UPDATE TO cylinder_handover_checklists DO INSTEAD NOTHING;
CREATE RULE no_delete_handover_checklists AS ON DELETE TO cylinder_handover_checklists DO INSTEAD NOTHING;
CREATE RULE no_update_platform_settings AS ON UPDATE TO platform_settings     DO INSTEAD NOTHING;
CREATE RULE no_delete_platform_settings AS ON DELETE TO platform_settings     DO INSTEAD NOTHING;
