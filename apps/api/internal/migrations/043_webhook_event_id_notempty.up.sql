-- Migration 043: forbid an empty webhook_events.event_id.
--
-- webhook_events carries UNIQUE(provider, event_id) and the replay guard in
-- WalletService.ProcessWebhook short-circuits on any row it matches. A handler
-- bug made event_id '' for every Paystack and Flutterwave delivery (data.id
-- arrives as a JSON number and was type-asserted to string), so the first
-- stored row shadowed every later event: real payments were acked 200 OK and
-- never credited.
--
-- The handler now normalises numeric IDs and falls back to the transaction
-- reference. This constraint makes a regression fail loudly at the write
-- instead of silently swallowing money.

-- Clear any sentinel rows written while the bug was live. Deleting these is
-- safe: they were never a meaningful dedup key, and a redelivered event is
-- still caught by the payment_intents status guard (intent.Status = 'success')
-- before any credit is posted.
DELETE FROM webhook_events WHERE event_id = '';

ALTER TABLE webhook_events
    ADD CONSTRAINT webhook_events_event_id_not_empty CHECK (event_id <> '');
