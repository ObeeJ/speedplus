ALTER TABLE driver_profiles DROP COLUMN IF EXISTS hazmat_certified;

DROP RULE IF EXISTS no_delete_handover_checklists ON cylinder_handover_checklists;
DROP RULE IF EXISTS no_update_handover_checklists ON cylinder_handover_checklists;
DROP TABLE IF EXISTS cylinder_handover_checklists;
DROP TABLE IF EXISTS customer_cylinders;
