-- Multi-drop package orders price by (route distance + per-stop fee). The
-- stop count must be part of the signed quote so a tampered stop count can't
-- shrink the price after the route was computed.
ALTER TABLE pricing_quotes
    ADD COLUMN IF NOT EXISTS stop_count INTEGER NOT NULL DEFAULT 1;

-- per_stop_kobo: admin-tunable per-vertical fee for each stop beyond the first.
ALTER TABLE fee_configs
    ADD COLUMN IF NOT EXISTS per_stop_kobo BIGINT NOT NULL DEFAULT 0 CHECK (per_stop_kobo >= 0);
