-- Reverse migration 043.
--
-- Deliberately does NOT restore the event_id = '' rows the up-migration
-- deleted. They were corrupt sentinels produced by the webhook dedup bug;
-- recreating one would re-arm the failure where the first stored row shadows
-- every later event and payments are acked without being credited.

ALTER TABLE webhook_events
    DROP CONSTRAINT IF EXISTS webhook_events_event_id_not_empty;
