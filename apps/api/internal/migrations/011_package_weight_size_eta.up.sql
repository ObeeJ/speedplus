-- pricing_quotes: ETA from OSRM, weight and size for package vertical
ALTER TABLE pricing_quotes
    ADD COLUMN IF NOT EXISTS eta_minutes   INTEGER      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS weight_kg     NUMERIC(8,3) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS size_category VARCHAR(10)  NOT NULL DEFAULT '';

-- order_items: store weight/size per item for package vertical
ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS weight_kg     NUMERIC(8,3) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS size_category VARCHAR(10)  NOT NULL DEFAULT '';
