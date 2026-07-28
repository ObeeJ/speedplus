# SpeedPlus Business Model — Canonical (2026-07)

This document supersedes all prior strategy drafts. Every number here matches
what the code charges (`apps/api/internal/service/pricing.go` +
`fee_configs` table). If code and this doc disagree, one of them is a bug.

## What SpeedPlus is

An asset-light, multi-vertical last-mile logistics marketplace (food, grocery,
pharmacy, gas, package) for Nigerian cities. We own no fleet; independent
verified riders provide transport. Our product is **trust infrastructure**:
escrowed payments released by delivery code, a wallet on licensed PSP rails
(Paystack / Flutterwave / Monnify), and dispatch/pricing/tracking.

**One-sentence pitch:** the delivery network where your money is only released
when your goods arrive.

The problem, per party: customers don't trust that paying upfront gets them
their goods; merchants and riders don't trust they'll get paid; cash-on-delivery
"solves" this by making riders carry cash at night. Every feature must map to
one of those three; anything else is scope creep.

## Canonical fee table (live in code)

| Vertical | Base fee | Per km | Per kg | Service fee | Merchant commission | Rider share of delivery |
|---|---|---|---|---|---|---|
| Food     | ₦900   | ₦150 | –      | 5% | 8% | 80% |
| Grocery  | ₦900   | ₦150 | –      | 5% | 8% | 80% |
| Pharmacy | ₦1,000 | ₦150 | –      | 5% | 8% | 80% |
| Gas      | ₦1,500 flat | ₦0 | –  | 3% | 8% | 80% |
| Package  | ₦900   | ₦170 | ₦70/kg | 4% | 8% | 80% |

Additional charges, all live in code:

- **Weather surcharge: ₦200 flat** on rain/storm (WMO 51–99) or >38 °C heat,
  via Open-Meteo at quote time. Compensates riders for the riskiest hours.
- **Package size surcharge:** medium ₦150, large ₦400 (on top of weight).
- Distance is **road distance from OSRM**, not straight-line; ETA comes from
  OSRM duration + 5 min pickup buffer.

Rationale for the base fees: at ₦1,400/L petrol, ~30 km/L in traffic, dead
kilometres included, a rider's hard cost per average job is ₦550–650. The old
₦500 + ₦100/km table paid riders below cost; this table nets a rider
~₦500–600/job (~₦160–180k/month at 12 jobs/day), which is the retention story.

**All of these are admin-tunable without deploys** via
`PUT /admin/settings/fees` (admin app → Settings → Delivery Fees & Commissions).
Changes are versioned, append-only, audit-logged, and take effect on new quotes
within ~1 minute. Each config row carries a fuel reference price; when an admin
updates it by >10%, the system *suggests* a proportional per-km adjustment —
never auto-applied. In-flight orders always settle at the rates in force when
the order was created.

## Unit economics (worked example)

Food order, 4 km, ₦5,000 basket, fair weather:

| Line | Amount |
|---|---|
| Basket (subtotal) | ₦5,000 |
| Delivery (900 + 4×150) | ₦1,500 |
| Service fee (5%) | ₦250 |
| **Customer pays** | **₦6,750** |
| Merchant receives (92%) | ₦4,600 |
| Rider receives (80% of delivery) | ₦1,200 |
| Platform: commission ₦400 + service ₦250 + delivery share ₦300 | **₦950 gross/order** |

After PSP fees (~1.4–1.5% capped, ≈ ₦100) and a 0.3–0.5% GMV fraud reserve
(≈ ₦25): **≈ ₦825 net contribution per order**.

**Break-even** at ≈ ₦6M/month bootstrap opex (cloud, support, marketing,
admin, legal): 6,000,000 / 825 ≈ **7,300 orders/month ≈ 245 orders/day**.
Revenue per order is the KPI that matters; track it weekly alongside:
orders/day, MAU, repeat rate, AOV, GMV, delivery time, rider acceptance,
CAC/LTV, merchant & rider retention, reconciliation delta (must be ₦0),
dispute rate (<2%), refund rate (<4%), fraud loss (<0.5% GMV).

## Implemented vs. claimed — say only what's true

**Built and live in the API today:**

- Double-entry ledger, append-only entries, balanced journals (money cannot
  leak without an invariant error); escrow hold/release with dual-approval
  admin freeze.
- Delivery-code release: 6-digit bcrypt-hashed code, 2 h expiry, 5 attempts
  then lockout; GPS logged at code entry — confirmations >500 m from the
  dropoff (or with no location) are flagged for review, never blocked.
- HMAC-signed single-use pricing quotes (10-min expiry, tamper-evident).
- Admin pricing engine (versioned `fee_configs` + admin UI + fuel-index
  suggestions) — this claim is now true.
- Trust-tier ladder: Tier 1 at 3 completed orders + clean record unlocks
  pay-on-arrival *eligibility*, capped at ₦10,000/order, one active POD at a
  time; fraud flags demote, POD failure demotes, freeze is permanent.
- Wallet + USSD top-up (Monnify), payment links, scan-to-pay/QR paycodes,
  gift cards, loyalty points, driver earnings + EWA cashout.
- Referral: ₦500 to referrer after referee's first order of ≥₦2,000;
  self-referral (same user/phone) and frozen accounts rejected.
- Webhook signature verification on all three PSPs; idempotency keys and
  Redis rate limits on all money paths.

**Not yet true — do not claim until shipped:**

- **Pay-on-arrival checkout**: eligibility and caps are enforced, but POD
  orders are declined at checkout until POD settlement (wallet debit at the
  door + failure handling) is implemented in the ledger. Clients see the cap
  via the trust-tier endpoint (`podCapKobo`).
- **Subscriptions**: force-paused (migration 010); the charge path is
  deliberately unimplemented. Remove from marketing until built.
- **Referral link at signup**: reward payout is wired to order completion,
  but registration does not yet accept a referral code — `Record` has no
  caller. Small API + signup-form change required.
- **Batching / two-tier rider network / fleet-anchor deals**: roadmap, not
  product. Sequence: prove 150+ orders/day pure-independent first, then sign
  small fleet anchors (5–15 bikes) on guaranteed minimums.
- BVN dedup on referrals, device fingerprinting, rider in-shift selfie
  checks: roadmap fraud items.

## Payments posture (regulatory)

SpeedPlus never holds customer funds on its own balance sheet. Wallets are a
ledger over licensed PSP infrastructure (Paystack + Monnify virtual accounts;
Flutterwave as redundancy). Escrow is order-state logic on that ledger. Say it
exactly this way to investors and regulators. Refunds always return to the
source instrument. Stablecoin/diaspora rails: parked until demand shows
(month 9+), it is not a launch feature.

## Risk register

| Risk | Status | Mitigation |
|---|---|---|
| OSRM outage kills quoting (SPOF) | **Open** | No haversine fallback today; quote returns error. Add fallback or self-host redundancy before scale. |
| Fuel price spikes erode rider margin | Mitigated | Fuel reference on every fee config; >10% delta triggers an admin repricing suggestion. |
| Mid-flight fee changes reallocating money | Closed | Settlement pins to the fee config effective at order creation. |
| Rider code-phishing ("read me the code") | Partially mitigated | GPS flag on far-away confirmations; customer-education copy still needed in app. |
| POD ghosting | Mitigated by design | Tier gate + ₦10k cap + one active POD + demotion on failure (checkout itself still disabled pending settlement work). |
| Referral farming | Mitigated | ₦2,000 minimum qualifying order, self-referral checks, frozen-account exclusion. |
| Weather surcharge unexplained on receipts | Open | Surface "rain surcharge ₦200" as a labelled line item in the customer apps. |
| Subscriptions advertised but dead | Open | Keep out of marketing; unpause only with the charge path + tests. |

## Operating discipline

Five numbers checked daily: reconciliation delta (₦0), dispute rate (<2%),
POD failure rate (<3%), refund rate (<4%), fraud loss % of GMV (<0.5%). Any
breach for 3 consecutive days is a founder-level incident. Losses under ₦5k:
refund fast, don't investigate. If fraud loss exceeds 0.7% of GMV, stop
growth spend and fix controls first — fraud scales faster than revenue.

Milestones: (1) one city, 50–100 merchants, 100–200 riders, 250 deliveries/day;
(2) operational break-even, repeat-order growth; (3) new cities, business
accounts, external capital only after unit economics are proven.
