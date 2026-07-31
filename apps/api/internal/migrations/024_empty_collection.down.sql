DROP RULE IF EXISTS no_delete_custody_events ON cylinder_custody_events;
DROP RULE IF EXISTS no_update_custody_events ON cylinder_custody_events;
DROP TABLE IF EXISTS cylinder_custody_events;

ALTER TABLE order_stops
    DROP COLUMN IF EXISTS empty_cylinder_serial,
    DROP COLUMN IF EXISTS empty_collected;
