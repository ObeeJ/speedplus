# SpeedPlus Ops Runbook

## A. Daily metrics (check every morning)

| Metric | Target | Where |
|--------|--------|-------|
| Reconciliation delta | ₦0 | Admin → Ledger → Reconcile escrow |
| Dispute rate | < 2% | Admin → Disputes |
| Refund rate | < 4% | Admin → Orders (filter: refunded) |
| Fraud loss % of GMV | < 0.5% | Manual review of reversed/disputed orders |
| POD failure rate | < 3% | Admin → Orders (filter: pay_on_arrival + cancelled) |

Any metric breaching its target for 3 consecutive days is a founder-level incident. Stop growth spend; fix controls first.

---

## B. Freeze / release escrow

**When to freeze:** fraud flag, open dispute, rider report of non-delivery, or admin investigation.

**How:**
1. Admin → Disputes → find the order → Freeze escrow.
2. Enter a reason (required; written to audit log).
3. Only two admins may approve a manual release (dual-approval enforced in code).

**How to release:**
1. Resolve the dispute (refund to customer or release to merchant/driver).
2. Admin → Disputes → Release escrow → enter reason.
3. Settlement runs immediately; ledger entries are created atomically.

**Rule:** Never release without a documented reason. The audit log is append-only.

---

## C. Fee-config change

1. Admin → Settings → Delivery Fees & Commissions.
2. Edit the vertical. Enter a reason (required).
3. Save. New quotes pick up the change within ~1 minute (60 s cache TTL).
4. In-flight orders always settle at the rates in force when the order was created.
5. If the fuel reference price moves > 10%, the system suggests a proportional per-km adjustment — review and apply manually if appropriate.

---

## D. LPG price-index update

1. Admin → Gas → Price index → Record new price.
2. Enter region (default: Lagos), price per kg in ₦, source, reason.
3. Rows are append-only. The latest row effective at or before now is the live price.
4. If the move is > 10%, review cylinder product prices and subscription subtotals.
5. Never edit or delete historical rows.

---

## E. Shortfall dispute

A customer claims they received less gas than ordered.

1. Check Admin → Orders → [order] → Proof media → weight_photo.
2. Compare `measured_kg` (scale reading) vs ordered weight (order items).
3. If `measured_kg < ordered_kg × 0.98` (beyond 2% tolerance): settlement already refunded the shortfall from the merchant's payout. Show the customer the refund ledger entry.
4. If the weight photo was missing or implausible at settle time (Settle rejected it): the order will not have settled. Investigate the proof chain, then either re-settle with corrected data or issue a manual refund via escrow release.
5. Manual adjustments require dual-admin approval and a written reason.

---

## F. OSRM (routing engine)

**Risk:** OSRM is a single point of failure for all quote requests. If it is unreachable, the pricing service falls back to a haversine estimate (1.4× road-tortuosity factor, 30 km/h average speed). Quotes still succeed but distances may be 20–30% off in dense Lagos traffic.

**Production requirement:** Self-host at least two OSRM instances behind a load balancer. Set `OSRM_URL` to the load-balanced endpoint.

**If OSRM goes down:** Quotes degrade to haversine (customers see slightly inaccurate ETAs and prices). Orders can still be placed. Monitor `osrm_fallback_total` metric. Restore within 30 minutes to avoid systematic under/over-pricing.

---

## G. Database backup / restore

**Backup schedule:** Postgres (with PostGIS) is the system of record for orders, ledger, and disputes. Production must run:
1. Automated daily full snapshot (Railway managed Postgres snapshot, or `pg_dump -Fc` to object storage if self-hosting), retained 30 days.
2. Continuous WAL archiving / point-in-time recovery (PITR) if the hosting provider supports it — required for money-path data, since a daily snapshot alone can lose up to 24h of ledger entries.

**Manual backup (ad hoc, before risky operations e.g. migrations):**
```
pg_dump -Fc "$DATABASE_URL" -f speedplus_$(date +%Y%m%d_%H%M).dump
```

**Restore:**
1. Never restore directly onto the production database. Restore into a fresh instance first.
2. `pg_restore -d "$RESTORE_DATABASE_URL" --clean --if-exists speedplus_<timestamp>.dump`
3. Reconcile: run the escrow reconciliation check (Admin → Ledger → Reconcile escrow) against the restored data before cutting traffic over — a restore that lands mid-settlement can leave orphaned escrow holds.
4. Cutover requires two-admin sign-off (same dual-approval bar as a manual escrow release) because a bad restore is a money-safety incident, not just a data incident.

**Rule:** A restore that isn't from an automated snapshot or a verified `pg_dump` is not a restore, it's a guess. Don't cut over to unverified data.

---

## H. Incident response

**Severity levels:**
- **SEV1** — money is moving incorrectly (double-charge, failed webhook causing lost payment, escrow released in error) or the API/DB is fully down. Page immediately, all hands.
- **SEV2** — a single subsystem degraded (OSRM down, one payment gateway failing, elevated error rate on one app) but orders can still be placed and money is safe. Fix within the hour, no page required outside working hours unless it persists.
- **SEV3** — cosmetic or non-money-path bug. Normal ticket flow.

**First response (any SEV1/SEV2):**
1. Confirm scope: check `/healthz` on the API, check the daily-metrics table (Section A) for the affected metric, check Sentry (`SENTRY_DSN`) for the error spike.
2. If a specific deploy caused it: roll back first, investigate after. Don't debug forward in production during a SEV1.
3. If a payment webhook is implicated: freeze the affected escrow (Section B) before doing anything else — money-safety takes priority over root-causing.
4. Post a timeline entry (time, symptom, action taken) as you go — this becomes the incident writeup, don't reconstruct it from memory afterward.

**Rollback:**
1. API: redeploy the previous known-good image/tag (Railway keeps prior deploys — use "redeploy" on the last green build rather than reverting commits under pressure).
2. DB migrations: this repo's migrations must be additive/backward-compatible by convention — if a bad migration shipped, roll back the API deploy first (old code should still run against the new schema); only run a down-migration if the schema change is actively causing incorrect data, and get a second admin to confirm before running it.
3. Frontend apps (customer/merchant/driver/admin): redeploy previous build; these are stateless, safe to roll back independently of the API.

**After the incident:** write it up (what broke, blast radius, root cause, fix, follow-up items) and add any new failure mode discovered to this runbook.
