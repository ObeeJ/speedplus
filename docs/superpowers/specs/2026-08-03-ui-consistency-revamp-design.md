# UI Consistency Revamp — Design

**Date:** 2026-08-03
**Status:** Approved

## Problem

SpeedPlus has four Next.js apps (customer, merchant, admin, driver) sharing a design system package, `@speedplus/ui` (emerald/lime/sand palette, Space Grotesk/Instrument Sans, token-driven components: Button, Input, Card, AuthShell, Avatar, Badge, Progress, SelectionCard, Separator, Skeleton, StatusSteps). An audit found the system itself is coherent and well-built — components consume CSS variables correctly, no hardcoded colors in the library. The actual problem is **adoption**: several screens were built before or outside the system and read as visually inconsistent with the rest of their app.

### Audit findings

| App | Pages | Using `@speedplus/ui` | Outliers |
|---|---|---|---|
| Customer | 41 | ~70% | 17× duplicated raw `rounded-xl border` card pattern instead of a shared component |
| Merchant | 10 | ~80% | No material outliers found |
| Admin | 20 | ~55% | `disputes/page.tsx`, `ledger/page.tsx`, dashboard `page.tsx`, `providers.tsx` — zero design-system imports; raw tables/divs |
| Driver | 3 pages / 6 files | 50% (auth screens only) | Main dashboard `app/page.tsx` (784 lines) is fully custom: hardcoded hex colors that don't even match canonical tokens (e.g. `#E9E6DD` vs. actual `--color-sand: #F7F5EF`), reinvented nav/badge/status-step markup that duplicates existing components |

Canonical tokens (from `packages/config/tailwind/tokens.css`):
```
--color-emerald: #0A3D2C   --color-emerald-600: #0D4E38   --color-emerald-900: #072D20
--color-lime: #C6F24E      --color-lime-600: #AEE032      --color-lime-700: #98C92B
--color-sand: #F7F5EF      --color-tile: #E9F3D8
--color-ink: #121216       --color-mid: #63636E           --color-line: #E4E0D6
--color-amber: #E8B14E
--color-icon-stroke: #1C3A2E   --color-icon-accent: #7BA05B
--font-display: 'Space Grotesk'   --font-body: 'Instrument Sans'
--radius-sm/md/lg/xl/full: 8/12/16/20/9999px
--shadow-xs/sm/md/lg/xl, --shadow-lime
```

## Goal

Bring all four apps to full, consistent use of `@speedplus/ui`. No new visual language, no breaking changes to existing component APIs. Fix the adoption gaps identified above and extract two new shared components to close the gaps that don't yet have a component.

## Non-goals

- No new color tokens, fonts, or visual direction change (existing emerald/lime/sand identity stays).
- No behavior or data-flow changes — this is a styling-layer migration only.
- No changes to the merchant app (already ~80% consistent; audit found no material outliers).
- No new automated test infrastructure.

## Scope

### 1. New shared components in `packages/ui`

- **`StatCard`** — token-based stat tile: label, value, optional trend/delta, optional icon. Replaces the local stat-tile reimplementation in `apps/admin/app/metrics/page.tsx` (or wherever admin's stat display currently lives) — the audit found this pattern should be promoted to a shared component rather than exist ad hoc in one app.
- **`ListCard`** (naming TBD at implementation time, e.g. bordered row/list-item component) — replaces the 17 raw `rounded-xl border` instances in the customer app with a single reusable, token-based component.

Both new components follow the existing pattern in `packages/ui/src/components/*.tsx`: `cva`-based variants, Tailwind token classes only (no hardcoded hex), exported from the package entry point alongside existing components.

### 2. Driver dashboard rebuild

`apps/driver/app/page.tsx` (784 lines): replace all hardcoded hex colors and custom-built nav/badge/status markup with token classes and existing components (`StatusSteps`, `Badge`, `SelectionCard`, `DuotoneIcon` variants already imported). This is a styling migration — all existing logic (websocket location polling via `buildWsUrl`/`buildWsProtocols`, badge metadata table, proof capture flow, react-query usage) is preserved unchanged. Driver's auth screens (`login`, `forgot-password`) already use `AuthShell`/`Button`/`Input` correctly and need no changes.

### 3. Admin legacy screens

Migrate `disputes/page.tsx`, `ledger/page.tsx`, dashboard `page.tsx`, and `providers.tsx` from raw tables/divs onto `Card`, `Badge`, the new `StatCard`, and token utility classes. Preserve existing data-fetching and interaction logic (sorting, filtering, routing) unchanged.

### 4. Customer card cleanup

Replace the 17 instances of the raw `rounded-xl border` pattern with the new `ListCard` component across the customer app.

## Order of work

1. `StatCard` and `ListCard` in `packages/ui` (unblocks everything else; independent of app-level work).
2. Driver dashboard rebuild (highest-severity outlier — non-canonical colors).
3. Admin legacy screens.
4. Customer card cleanup.

Steps 2–4 are independent of each other once step 1 lands, but will be delivered together in one pass per the user's preference (not phased into separate reviewable chunks).

## Testing / verification approach

Manual/visual verification, not new automated tests:
- Start each app's dev server and visually check migrated screens render correctly.
- Manually exercise preserved interactions: driver websocket location updates and proof capture flow, admin table sort/filter/routing, customer navigation through migrated list screens.
- Run existing `tsc` and lint checks across all four apps and `packages/ui` to catch prop/import errors introduced by the migration.

## Risks / open questions

- Exact final naming/API for `StatCard` and `ListCard` variant props will be decided during implementation, informed by the actual shapes of data being displayed in admin's metrics and customer's card lists.
- The driver dashboard file is large (784 lines) with non-trivial state (websocket, badges, proof capture) — the rebuild must be done carefully to avoid regressing behavior while changing only presentation.
