# SpeedPlus — Remaining Scope (B2–B7)

Status as of this handoff: **B1 is done and verified** (USSD funding via Monnify + P2P Payment Links, backend + typed api-client, `go build`/`go vet` clean, `tsc --noEmit` clean).

Stack: pnpm + Turborepo monorepo. Go 1.26 API (Gin, GORM, golang-migrate, asynq, go-redis/v9, golang-jwt/v5, sentry-go). 4 Next.js 15 apps: `apps/customer`, `apps/driver`, `apps/merchant`, `apps/admin`. Shared packages: `packages/types`, `packages/api-client`, `packages/ui`, `packages/utils`, `packages/config`.

Money is always int64 kobo. `ApiResponse` envelope: `{success:true,data:T} | {success:false,error:{code,message,field}}`. Idempotency-Key header required on money-moving POSTs. Escrow release is receiver-gated only (paycode confirm), never bypassed. Webhooks: verify signature → dedupe by (provider,event_id) → provider verify-API call → credit. Never trust webhook payload alone.

Reference the live code before writing anything — do not assume field/route names, grep them:
- `apps/api/cmd/server/main.go` — all route registration and service/handler wiring (read this first every session, it has grown past 267 lines since B1)
- `apps/api/internal/model/models.go` + sibling files (`payment_link.go`, `ussd_intent.go`, `user.go`) — GORM models
- `apps/api/internal/service/*.go`, `apps/api/internal/handler/*.go` — existing patterns to copy (`ledger.go`, `wallet.go`, `paycode.go` are the best examples of the money-safety patterns: `SELECT FOR UPDATE`, idempotency check, journal + adjustBalance)
- `packages/api-client/src/endpoints/*.ts` — typed client pattern (see `wallet.ts`, `payment-links.ts` for the idiom: `apiClient.post<ApiResponse<T>>(...)`, throw on `!data.success`)
- `packages/types/src/*.ts` — shared TS types, extend rather than duplicate

---

## B2 — Admin app (highest priority, currently a bare stub)

`apps/admin/app/` currently has only `layout.tsx`, `page.tsx`, `providers.tsx` — no route subdirectories. Backend admin endpoints already exist and are registered under `authed.Group("/admin")` + `RequireRole("admin")` in `main.go`: `kyc/queue`, `kyc/:id/approve`, `kyc/:id/reject`, `dispatch/:orderId/assign`. Grep `main.go` for the current full list before starting.

Build these Next.js routes (App Router, one folder per route under `apps/admin/app/`):

1. **KYC queue** (`/kyc`) — list pending KYC submissions (`GET /admin/kyc/queue`), approve/reject actions per row. Backend: `internal/handler/kyc.go` (`AdminQueue`, `Approve`, `Reject`).
2. **Merchant approvals** (`/merchants`) — list merchants by status, approve/suspend. Check `internal/model/models.go` `Merchant` struct (`Status MerchantStatus`) and `internal/handler/*` for any existing merchant-admin handler; if none exists, add one following the KYC handler pattern (list + status-transition endpoint, admin-only, log the actor).
3. **Driver approvals** (`/drivers`) — same pattern against `DriverProfile` (in `user.go`).
4. **Order search / detail** (`/orders`) — search by ID/customer/status, view full order timeline (`OrderEvent`), state machine visualization from `Order.ValidTransitions` (see `models.go`). Read-only for now; add a manual "admin assign driver" action wired to the existing `dispatch/:orderId/assign`.
5. **Disputes** (`/disputes`) — freeze/release escrow on a disputed order. Check whether an escrow freeze/release admin endpoint exists (`escrow_holds` table, `EscrowHold` model); if not, this needs a new handler — **do not let admin release escrow directly to a party, route it through the same ledger-journal + adjustBalance pattern used in `wallet.go`/`payment_link.go`, and require a reason field logged to an audit table.**
6. **Cancellation-rule editor** (`/settings/cancellation-rules`) — CRUD over `CancellationRule` model (already exists in `models.go`). Simple admin table editor.
7. **Ledger viewer** (`/ledger`) — read-only view of `ledger_entries`/`ledger_accounts` for a given account or user, paginated. Reuse `LedgerService.GetTransactions` pattern from `wallet.go`.

For each: add the corresponding `packages/api-client/src/endpoints/admin.ts` module (typed, following existing idiom) and export it from `packages/api-client/src/index.ts`. Use `@speedplus/ui` components (Button, Card, Badge, Input already exist in `packages/ui/src/components/`).

**Security note:** every new admin handler must re-check `RequireRole("admin")` at the route group level (copy the existing `admin := authed.Group("/admin"); admin.Use(middleware.RequireRole("admin"))` pattern) — do not add unauthenticated or customer-role-accessible admin routes.

---

## B3 — WhatsApp channel (Meta Cloud API direct)

New Go package `apps/api/internal/whatsapp/`. This is the largest remaining backend block.

- Webhook handler: `POST /webhooks/whatsapp` — verify Meta's `X-Hub-Signature-256` HMAC, handle the GET verification challenge (`hub.mode=subscribe&hub.verify_token=...`) separately.
- Session store: Redis-backed conversation state keyed by WhatsApp phone number (map to `User` via phone lookup; if no account exists, run through an in-band OTP registration flow using the same OTP service already built for the webapp — do not build a second OTP system).
- Message router: parse incoming message type (text, interactive button/list reply, WhatsApp Flow response) → dispatch to the same service layer the webapp uses (`OrderService`, `WalletService`, `PaycodeService`, etc.) — WhatsApp is a UI, not a separate business-logic path.
- 14 T1 ops to support: auth (OTP), reorder, gas order, pharmacy refill, balance, fund wallet (show DVA number + USSD code), send money (wallet transfer), payment link (create + share), track order, cancel order, report issue, referral, loyalty balance, rate order.
- Explicitly OUT of WhatsApp scope (webapp-only): new prescription upload, KYC document upload, live map tracking, full catalog browse, admin surfaces.
- Use WhatsApp Flows (JSON-defined multi-step forms) for anything needing >1 field of structured input (e.g., gas order: cylinder size + address + quantity). Keep flows thin — validate server-side using the same validators the webapp forms use, never trust client-side Flow validation alone.
- Auth: WhatsApp number must be verified via OTP before any money-moving op; store a WhatsApp-specific short-lived session token, not a full 30-day refresh token, to limit blast radius if the Redis session store is compromised.

This block needs the Meta WhatsApp Business Account (WABA) ID, phone number ID, and permanent access token configured in env before webhook testing is possible — set that up in Meta's App Dashboard first if not already done.

---

## B4 — Customer frontend polish

`apps/customer/app/` already has 20+ routes across 4 verticals (food/gas/pharmacy/package) with Zustand stores. Add:

- **Visible escrow UI** — on the order tracking page, show "₦X held in escrow, releases when you confirm delivery" with a live status indicator (pending → held → released). Pull from `EscrowHold` via a new or existing read endpoint.
- **Trust-tier progress card** — on the wallet/profile tab, show current tier (0 New / 1 Regular / 2 Trusted / 3 VIP) and progress to next tier (order count / BVN status), using `GET /users/me/trust-tier` (already exists, see `card.go` handler).
- **DVA onboarding screen** — first-time wallet funding flow that surfaces the user's dedicated virtual account number (bank name + account number) prominently, with a copy-to-clipboard and "or use USSD" toggle into the B1 USSD flow.
- **Guest checkout path** — allow an unauthenticated user to complete a food/gas order paying by card only (no wallet), collecting just phone + delivery address. Needs a guest-order ledger account pattern similar to `PaymentLinkService.InitiateGuestPayment` in `payment_link.go` — reuse that pattern, don't invent a new one.
- **SpeedPlus Card in wallet tab** — render the QR (`spd.card.v1.{userID}.{HMAC}` format, already generated server-side via `GET /users/me/card`) as a scannable code in the customer wallet screen.

---

## B5 — Growth engines

Models already exist in `models.go`: `Referral`, `GiftCard`, `LoyaltyEvent`, `LoyaltyBalance`, `Subscription`, `Campaign`, `CampaignContribution`, `OrderSplit`. Services already exist: `loyalty.go`, `referral.go`, `gift_card.go`, `subscription.go` — **check what's already implemented in each before adding new code**, several may already be partially wired.

Remaining work likely needed:
- Gas subscription cron: asynq periodic task that creates a new order automatically on the subscription's cadence, with a dunning retry (3 attempts, then pause subscription) if wallet balance is insufficient — wire into `worker/scheduler.go`.
- Split-pay orchestration: `OrderSplit` model exists; need the service method that holds a single order in a "collecting" state until all split participants have paid their share into escrow, then transitions to normal dispatch — model this as a state machine addition to `OrderService`, not a separate flow.
- Multi-stop order handlers: `OrderStop` model exists; verify the dispatch/pricing services already iterate stops (`pricing.go`, `dispatch.go`) — if not, that's the gap.

---

## B6 — Driver + Merchant app wiring

Both `apps/driver/app/` and `apps/merchant/app/` are currently complete stubs (layout/page/providers only), same as admin was before B2. Backend driver/merchant surfaces already exist:
- Driver: `POST /drivers/location`, `/drivers/offers/:id/accept`, `/drivers/offers/:id/reject`, `GET /users/me/driver-profile`, earnings/EWA cashout endpoints, paycode scan-to-confirm (`paycodes/scan-card`, `/resolve`, `/:id/confirm`).
- Merchant: `GET /users/me/merchant-profile` exists; check for order-acceptance and menu-management endpoints — likely need new handlers if not present (list incoming orders, accept/reject, mark ready).

Build the corresponding pages: driver needs an offer-accept screen with a countdown (15s TTL per the dispatch design), an earnings/EWA cashout screen, and the paycode-scan UI (camera QR reader → `scan-card`/`resolve`/`confirm`). Merchant needs an incoming-orders queue, product/menu CRUD, and open/closed toggle (`Merchant.IsOpen`).

---

## B7 — Hardening

- k6 load tests against the dispatch offer-cascade and wallet-transfer paths (these are the concurrency-sensitive ones — `SELECT FOR UPDATE` correctness under load).
- Chaos tests: webhook retry storms (duplicate webhook delivery — verify dedupe by `(provider, event_id)` holds), split-pay race (two participants paying simultaneously).
- CI: add `govulncheck ./...` and `gitleaks detect` as required checks (`.gitleaks.toml` already exists at repo root — verify it's wired into `.github/workflows/`).
- OpenAPI spec generation from the Gin routes (or hand-written, covering all `/api/v1/*` routes) for external consumption / WhatsApp Flow contract validation.
- Runbooks: incident response for (a) payment provider outage/failover Paystack↔Flutterwave↔Monnify, (b) webhook backlog, (c) dispatch offer-cascade stall.

---

## Cross-cutting reminders for whoever picks this up

- Never use GORM AutoMigrate in production — new tables go through `golang-migrate` SQL files in `apps/api/migrations/`, numbered sequentially (`007_...`, `008_...`), with matching `.down.sql`.
- All money fields are `int64` kobo, never float, except at the payment-provider boundary where Monnify/Paystack/Flutterwave APIs expect Naira (divide/multiply by 100 only at that boundary, nowhere else).
- Every new money-moving POST needs `Idempotency-Key` header enforcement (see `middleware.Idempotency(rdb, ttl)` usage in `main.go`) and a DB idempotency-key row inside the same transaction as the ledger write.
- Every new admin/driver/merchant-role endpoint needs `middleware.RequireRole(...)` at the group level plus a row-level ownership check inside the handler (anti-IDOR) — a prior session already fixed one IDOR bug in `paycodes.go`, don't reintroduce the pattern.
- Run `go build ./... && go vet ./...` in `apps/api/` and `tsc --noEmit` in each touched package/app before calling any block done.
