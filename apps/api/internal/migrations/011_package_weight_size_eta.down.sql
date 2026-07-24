ALTER TABLE pricing_quotes
    DROP COLUMN IF EXISTS eta_minutes,
    DROP COLUMN IF EXISTS weight_kg,
    DROP COLUMN IF EXISTS size_category;

ALTER TABLE order_items
    DROP COLUMN IF EXISTS weight_kg,
    DROP COLUMN IF EXISTS size_category;
