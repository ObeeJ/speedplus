-- ── Zone launch status ────────────────────────────────────────────────────────
-- Rollout is per-LGA/zone, not company-wide: a zone starts 'piloting' (batching
-- and marketing not yet safe), is promoted to 'live' once route density and
-- rider economics are proven there, and can be 'paused' if it regresses.
-- Gas subscription creation checks this per the customer's zone (service.go),
-- so "no gas marketing ahead of readiness" is enforced per zone, not globally.

ALTER TABLE service_zones
    ADD COLUMN IF NOT EXISTS launch_status TEXT NOT NULL DEFAULT 'piloting'
        CHECK (launch_status IN ('piloting', 'live', 'paused'));
