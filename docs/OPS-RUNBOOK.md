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
