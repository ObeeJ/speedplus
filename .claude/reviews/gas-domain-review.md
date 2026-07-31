# Code Review: Gas Domain Implementation (migrations 022–028 + repo extraction)

**Reviewed:** 2026-07-30
**Scope:** local uncommitted changes — 57 modified files (+2,894 / −1,412), ~40 new untracked files
**Decision:** **REQUEST CHANGES** — 5 HIGH, 0 CRITICAL

## Summary

The architecture is sound and several genuinely subtle things are right — the shortfall
journal balances correctly, the `chargeOne` atomicity warning was honoured, the state
machine extension doesn't disturb the other four verticals, and the repo extraction
preserved transaction propagation. The problems are concentrated in three places:
**the weight-proof guard covers only one of three settlement paths**, **`chargeOne`
charges for the wrong product and weight**, and **the new tests are tautological, so
none of the gas money path is actually covered**. Migration 026 then unpauses live
subscriptions on top of the `chargeOne` bugs, which is what turns them into real money
moving incorrectly.

## Validation Results

| Check | Result |
|---|---|
| `go build ./...` | **Pass** |
| `go vet ./...` | **Pass** |
| `go test ./internal/...` | **Pass** — but see HIGH-5; DB-backed tests skip without `DATABASE_URL` (service pkg ran in 0.008s) |
| `pnpm typecheck` (repo) | **Fail** — pre-existing, not this changeset (see Note) |
| `tsc --noEmit` (apps/customer) | **Pass** |

---

## HIGH

### HIGH-1 — Weight-proof guard covers only 1 of 3 settlement paths

`apps/api/internal/service/paycode.go`

Three paths call `s.ledger.Settle`:

| Path | Line | Settle at | Weight guard |
|---|---|---|---|
| `Confirm` (QR paycode) | 160 | 192 | **none** |
| `ConfirmByCode` (6-digit) | 262 | 300 | ✅ 281–286 |
| `ConfirmByCard` (offline) | 349 | 376 | **none** |

A rider skips the mandatory weigh-and-photo entirely by confirming via QR or card;
the order settles at full price and the customer gets no shortfall refund. This voids
the trust wedge that the whole feature exists to deliver. It is also the same
bypass-path class the codebase has already been bitten by around `Transition`.

**Fix:** hoist the `CountWeightProof` check into a shared guard called by all three
paths — ideally at the top of `Settle` itself, so no future fourth path can miss it.

### HIGH-2 — `chargeOne` charges for the wrong product and the wrong weight

`apps/api/internal/service/subscription.go:85-91`

```go
productID := uuid.MustParse("00000000-0000-0000-0000-000000000013") // 12.5kg
weightKg := 12.5
if sub.CylinderSpecID != nil {
    if prod, err := s.repo.FindCheapestProduct(ctx, sub.MerchantID); err == nil {
        productID = prod.ID
    }
}
```

Two defects:

1. When the subscription **does** specify a cylinder spec, the code looks up the
   *cheapest* product and ignores `sub.CylinderSpecID` completely. A 25 kg subscriber
   is charged for a 3 kg cylinder.
2. `weightKg` stays `12.5` regardless of which product was chosen. Gas is now priced
   per-kg (`pricing.go:119`) and `vehicleClassFor` derives vehicle class from weight —
   so a 25 kg order is both mispriced and dispatched to a **car instead of a van**.

The `if err == nil` also silently swallows a lookup failure and falls back to the
hardcoded 12.5 kg product.

**Fix:** resolve the product from `sub.CylinderSpecID` via `cylinder_specs`, derive
`weightKg` from `spec.size_kg`, and propagate the lookup error instead of falling back.

### HIGH-3 — Migration 026 unpauses subscriptions the customer paused

`apps/api/internal/migrations/026_gas_subscriptions.up.sql`

```sql
UPDATE subscriptions SET status='active', next_charge_at=NOW()+INTERVAL '1 day', dunning_count=0
WHERE vertical='gas' AND status='paused';
```

`status='paused'` has at least three distinct causes: migration 010's system pause,
`SubscriptionService.Pause` (customer-initiated), and `ProcessDue`'s auto-pause after
3 dunning failures. This statement cannot tell them apart, so it reactivates
subscriptions users deliberately stopped and starts billing them within 24 hours.

Combined with HIGH-2, those reactivated customers are charged for the wrong cylinder.

**Fix:** add a `paused_reason` column (or restrict to rows whose `updated_at` matches
the 010 window) and unpause only system-paused rows. Given the blast radius, prefer
leaving them paused and letting customers opt back in.

### HIGH-4 — `weightProof` swallows DB errors, making the caller's error check dead code

`apps/api/internal/service/ledger.go:126-138`

```go
err := tx.WithContext(ctx).Where("order_id = ? AND kind = 'weight_photo'", orderID).
    Order("captured_at DESC").First(&proof).Error
if err != nil {
    return 0, nil // no proof row — caller handles
}
```

Every error — including a genuine DB failure — becomes `0, nil`. So the caller's guard
at line 170, `if err == nil && measuredKg > 0`, has an `err` that is *always* nil. A
transient DB error is indistinguishable from "no proof row", and the result is **no
shortfall refund and a full merchant payout**, silently.

This sits directly above line 192's `// FIX #3: propagate all account/wallet lookup
errors — no more _, _ discards`, which is the same lesson being re-learned.

**Fix:** return `gorm.ErrRecordNotFound` distinctly from other errors; propagate the
rest and let `Settle` fail rather than under-refund.

### HIGH-5 — New tests are tautological; the gas money path has no real coverage

`apps/api/internal/service/ledger_gas_test.go` (and `phase1/3/4/5/6_test.go`, 830 lines total)

Three separate problems:

1. **The test reimplements production logic and asserts against its own copy.**
   `shortfallKobo` (line 11) is a private duplicate of the formula in `Settle`, with the
   comment `// mirrors the production formula so we can assert it without a DB`. If
   `Settle` drifts, these tests still pass.

2. **`journalSumsToZero` cannot fail.** Line 33 is `sum += -shortfall + shortfall`, and
   `platformTotal` is defined by the test (line 101) as exactly the remainder. Substituting
   in, the expression reduces to `0` identically for all inputs. `TestSettlementJournalZeroSum`
   and `TestSettlementJournalZeroSumNoShortfall` would pass with `Settle` deleted.

3. **No test in the changeset invokes `Settle`, `Create`, `chargeOne`, or `ConfirmByCode`.**
   Confirmed by grep. `phase4_test.go:28` is representative — it asserts a constant is in
   a range and that "the struct accepts the fields", which the compiler already guarantees.

The pre-existing DB-backed tests (`ledger_money_test.go` et al.) do call real `Settle`,
but they skip without `DATABASE_URL` — the service package completed in 0.008 s.

**Fix:** add DB-backed tests that call `Settle` on a real gas order and assert the
posted ledger entries, and one per settlement path asserting the weight guard (HIGH-1).
Delete `shortfallKobo` and `journalSumsToZero` rather than keeping them.

---

## MEDIUM

### MED-1 — `pricePerKg` misprices `new_cylinder` orders
`ledger.go:174` — `pricePerKg := float64(order.SubtotalKobo) / orderedKg`. In
`new_cylinder` mode the subtotal includes the cylinder body (₦25–40k), inflating
₦/kg several-fold. A 300 g short-fill could consume the merchant's entire payout
(capped at `merchantShareKobo`, so 100% of it). Price the shortfall off the LPG
index, not the basket total.

### MED-2 — No plausibility bound on `measured_kg`
`proof_media.go:92` validates only `> 0`. A rider typing `1.25` instead of `12.5`
triggers a near-total automatic merchant clawback. Reject or flag readings outside
a sane band (e.g. 0.5×–1.5× ordered) for human review instead of auto-refunding.

### MED-3 — Migration 022's fee-config insert silently no-ops
`022_gas_fee_correction.up.sql:8-30` — `INSERT ... SELECT ... FROM users WHERE
role = 'admin' ... LIMIT 1`. No migration ever seeds an admin user (only 001's CHECK
constraint mentions `'admin'`). On a fresh database this inserts **zero rows with no
error**, so the headline Phase 0 pricing fix never lands in `fee_configs`. It works
today only because `pricing.go:64` carries the correct fallback. Seed an admin, use a
sentinel `updated_by`, or make the migration fail loudly.

### MED-4 — Dangling FK promises
- `024`: `cylinder_id UUID DEFAULT NULL, -- FK to customer_cylinders added in Phase 1`
  — migration 028 (Phase 1) never adds it.
- `026`'s `subscriptions.cylinder_spec_id` has no FK to `cylinder_specs` either.

Unenforced references on an append-only custody/evidence table are exactly where you
want referential integrity most.

### MED-5 — Error-string matching for authorization control flow
`handler/gas.go:101-108` switches on `err.Error()` for `"cylinder not found"` and
`"forbidden"`. Rewording a message in the service silently downgrades a 403 to a 422.
Use sentinel errors and `errors.Is`.

### MED-6 — `err.Error()` echoed to clients
`handler/gas.go:86` and `:107` put raw internal error text in the response body,
unlike the `internalError(c, err)` helper used elsewhere. Information disclosure and
inconsistent with the rest of the handler layer.

### MED-7 — Identity parse errors discarded
`handler/gas.go:33, 58, 99` — `customerID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))`
yields `uuid.Nil` and proceeds. Auth middleware should make this unreachable, but
discarding an error on an *identity* value is the pattern that becomes IDOR the moment
middleware changes. Return 401 instead.

---

## LOW

| # | Finding |
|---|---|
| LOW-1 | `025_zones_runs.up.sql` stores window times as **UTC** minutes-since-midnight (default 480 = "08:00 UTC"). Nigeria is UTC+1 with no DST, so customer-facing windows land an hour off. Store WAT or name the columns `*_utc_minutes` and convert in one place. |
| LOW-2 | `025` `active_days` comment mixes conventions in one sentence: "bit 0 = Sunday … bit 6 = Saturday (ISO: Mon=1)". Pick one and encode it in a helper. |
| LOW-3 | `023` drops the CHECK by Postgres's auto-generated name `proof_media_kind_check` (018 created it inline). Works, but couples to an implementation detail. |
| LOW-4 | `subscription.go:85` hardcodes seed UUID `…013` via `uuid.MustParse` — couples service logic to a migration seed row. Derive from `cylinder_specs`. |
| LOW-5 | No rate limit on `POST /api/v1/cylinders`; unbounded distinct-serial registration per user. |
| LOW-6 | Over-fill (`measured > ordered`) is silently ignored. Worth flagging as possible scale tampering even if not refunded. |

---

## Note — pre-existing, blocks validation

`pnpm typecheck` fails repo-wide: `Could not resolve workspace. Missing
devEngines.packageManager or legacy packageManager field in package.json`. `package.json`
is **not** modified in this changeset and HEAD lacks the field too, so this is
pre-existing — but it means the 4-app type gate is currently non-functional in CI. I
validated `apps/customer` directly with `tsc --noEmit` (clean). Worth a separate fix.

---

## What was done well

Called out because these are the parts that are easy to get wrong and weren't:

- **Shortfall accounting is correct.** `merchantShareKobo -= S` (`ledger.go:180`) flows
  into `platformTotal` (`:186`), so the extra revenue/customer entry pair at `:232-233`
  nets the platform to zero and the journal still balances. Net effect: merchant pays,
  customer is refunded, platform neutral. That is the right allocation and the algebra
  works out — now make a real test prove it stays true (HIGH-5).
- **The `chargeOne` atomicity warning was honoured.** Rather than adding a standalone
  wallet debit, it delegates to `OrderService.Create`, which holds escrow inside its own
  transaction. The old stub's warning was explicit about this and it was respected.
- **State machine extension is clean.** Swap keeps `driver_assigned → in_transit`
  untouched; refill adds a properly gated `awaiting_collection → empty_collected →
  at_plant` chain; every new state can reach `cancelled`. The other four verticals are
  unaffected.
- **The root-cause fix is right.** `vehicleClassFor` (weight-derived) plus the widened
  `vehicleFilter` (`IN ('car','van')`) correctly resolves the van-only/flat-fee inversion,
  and `pricing.go:119` extends weight pricing to gas.
- **Repo extraction preserved transaction propagation.** `tx *gorm.DB` is threaded through
  all money methods (`repo/ledger.go`: 30 funcs / 32 tx params) and no repo opens its own
  transaction. This is the thing that usually breaks in this refactor.
- **Migration ordering was handled deliberately.** The phase numbers don't match the
  migration numbers (027 creates `customer_cylinders`, 028 extends it), and rather than
  shipping a broken chain the author made 027/028 additive and idempotent. Worth
  renumbering for readability, but the chain is sound.
- **Append-only discipline is consistent** — RULES on `cylinder_custody_events`,
  `lpg_price_index`, and `cylinder_handover_checklists` all follow the 018 pattern.

---

## Must fix before merge

1. HIGH-1 — guard all three settlement paths (move the check into `Settle`)
2. HIGH-2 — resolve product and weight from `CylinderSpecID`
3. HIGH-3 — do not unpause customer-paused subscriptions
4. HIGH-4 — propagate `weightProof` errors
5. HIGH-5 — real DB-backed tests for `Settle` + the weight guard; delete the mirrored helpers

MED-1 and MED-2 should follow immediately after — both let a single bad reading zero out
a merchant's payout.
