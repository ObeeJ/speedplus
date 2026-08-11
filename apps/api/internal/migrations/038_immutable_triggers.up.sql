-- Migration 038: Replace silent DO INSTEAD NOTHING rules with BEFORE triggers
-- that RAISE EXCEPTION. The old rules silently swallowed UPDATE/DELETE
-- statements, making bugs invisible. Triggers surface them as hard errors.
--
-- Tables covered (7 pairs → 7 triggers):
--   ledger_entries, admin_audit_logs, fee_configs, proof_media,
--   cylinder_custody_events, lpg_price_index, cylinder_handover_checklists

-- ── 1. Drop the old silent rules ─────────────────────────────────────────────

DROP RULE IF EXISTS no_update_ledger          ON ledger_entries;
DROP RULE IF EXISTS no_delete_ledger          ON ledger_entries;

DROP RULE IF EXISTS no_update_admin_audit     ON admin_audit_logs;
DROP RULE IF EXISTS no_delete_admin_audit     ON admin_audit_logs;

DROP RULE IF EXISTS no_update_fee_configs     ON fee_configs;
DROP RULE IF EXISTS no_delete_fee_configs     ON fee_configs;

DROP RULE IF EXISTS no_update_proof_media     ON proof_media;
DROP RULE IF EXISTS no_delete_proof_media     ON proof_media;

DROP RULE IF EXISTS no_update_custody_events  ON cylinder_custody_events;
DROP RULE IF EXISTS no_delete_custody_events  ON cylinder_custody_events;

DROP RULE IF EXISTS no_update_lpg_price       ON lpg_price_index;
DROP RULE IF EXISTS no_delete_lpg_price       ON lpg_price_index;

DROP RULE IF EXISTS no_update_handover_checklists ON cylinder_handover_checklists;
DROP RULE IF EXISTS no_delete_handover_checklists ON cylinder_handover_checklists;

DROP RULE IF EXISTS no_update_platform_settings ON platform_settings;
DROP RULE IF EXISTS no_delete_platform_settings ON platform_settings;

-- ── 2. Shared trigger function ────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION raise_immutable()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'table % is append-only: % is not permitted',
        TG_TABLE_NAME, TG_OP;
END;
$$;

-- ── 3. Wire the trigger onto each table ──────────────────────────────────────

CREATE TRIGGER immutable_ledger_entries
    BEFORE UPDATE OR DELETE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_admin_audit_logs
    BEFORE UPDATE OR DELETE ON admin_audit_logs
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_fee_configs
    BEFORE UPDATE OR DELETE ON fee_configs
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_proof_media
    BEFORE UPDATE OR DELETE ON proof_media
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_cylinder_custody_events
    BEFORE UPDATE OR DELETE ON cylinder_custody_events
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_lpg_price_index
    BEFORE UPDATE OR DELETE ON lpg_price_index
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_cylinder_handover_checklists
    BEFORE UPDATE OR DELETE ON cylinder_handover_checklists
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();

CREATE TRIGGER immutable_platform_settings
    BEFORE UPDATE OR DELETE ON platform_settings
    FOR EACH ROW EXECUTE FUNCTION raise_immutable();
