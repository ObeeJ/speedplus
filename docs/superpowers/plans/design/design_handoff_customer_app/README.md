# Handoff: SpeedPlus Customer Web App (Home + 4 Service Flows)

## Overview
SpeedPlus is a multi-vertical logistics platform for Nigeria (Lagos first): one network moving **packages, LPG gas, food, and pharmacy orders** (OTC + pharmacist-validated prescriptions). B2C and B2B2C. This handoff covers the **customer web app**: the responsive home and the four end-to-end ordering flows, plus the locked design system.

## About the Design Files
The files in this bundle are **design references created in HTML** — interactive prototypes showing intended look and behavior, NOT production code to copy directly. Recreate them in the target codebase: the existing **speedplus monorepo** (github.com/ObeeJ/speedplus, `monorepo` branch) — Turborepo with Next.js 15 / React 19 / Tailwind v4 apps (`apps/customer`, `apps/driver`, `apps/merchant`, `apps/admin`) and shared packages (`packages/ui`, `packages/types`, `packages/api-client`, `packages/config`). This design targets `apps/customer`.

## Fidelity
**High-fidelity.** Colors, type, spacing, radii, copy, and interactions are final. Recreate pixel-perfectly using the monorepo's patterns (Tailwind v4 tokens, cva variants in `packages/ui`).

## ⚠ Design tokens: REPLACE the ones in the repo
`packages/config/tailwind/tokens.css` currently holds an OLD brand (`--color-primary: #00C48C`, Inter / Plus Jakarta Sans). **This design supersedes it.** Replace the `@theme` block with:

```css
@theme {
  /* Brand */
  --color-emerald:      #0A3D2C;  /* primary brand surface / secondary button */
  --color-emerald-600:  #0D4E38;  /* hover on emerald */
  --color-emerald-900:  #072D20;  /* pressed on emerald */
  --color-lime:         #C6F24E;  /* primary action, active state, live pulse */
  --color-lime-600:     #AEE032;  /* hover on lime */
  --color-lime-700:     #98C92B;  /* pressed on lime */
  --color-sand:         #F7F5EF;  /* app background */
  --color-tile:         #E9F3D8;  /* icon tiles, soft accents, chips */
  --color-ink:          #121216;  /* primary text */
  --color-mid:          #63636E;  /* secondary text */
  --color-line:         #E4E0D6;  /* borders / hairlines */
  --color-amber:        #E8B14E;  /* waiting / in-review status */
  --color-icon-stroke:  #1C3A2E;  /* icon primary stroke */
  --color-icon-accent:  #7BA05B;  /* icon olive accent stroke */

  /* Typography */
  --font-display: 'Space Grotesk', sans-serif;  /* headings, buttons, numbers */
  --font-body:    'Instrument Sans', sans-serif; /* body, labels, captions */

  /* Radius */
  --radius-sm: 9px; --radius-md: 13px; --radius-lg: 16px; --radius-xl: 20px; --radius-full: 9999px;
}
```

Update `packages/ui/src/components/button.tsx` cva variants accordingly:
- `primary`: bg lime `#C6F24E`, text emerald `#0A3D2C`, hover `#AEE032`, active `#98C92B`
- `secondary`: bg emerald `#0A3D2C`, text `#F7F5EF`, hover `#0D4E38`, active `#072D20`
- `ghost`: transparent, text emerald, hover `rgba(10,61,44,0.07)`
- Font: Space Grotesk 600. Radius 12–13px. Keep `active:scale-[0.98]`.

## Layouts — desktop, tablet, and mobile are SEPARATE designs
Do not scale one into the other.

### Desktop (≥1024px in production; prototype uses ≥760px so it demos in narrow panes)
- **Top bar** (64px, emerald `#0A3D2C`): logo `speed+` (lime plus), nav: Home / My orders / My money, area selector, avatar.
- **Home = "command center"**: full-bleed emerald map canvas (faint city-grid, dimmed), centered: 46px Space Grotesk headline **"What do you need moved?"**, 640px spotlight search bar (sand bg, lime Send button), 4 service pills (Package active by default = lime pill; Gas refill / Food / Medicine = translucent outline pills), caption "Priced by weight, size and distance — you see it before you confirm". Bottom-left: live-status pill; when an order is active, route + rider card overlay the map.
- **Service flows = full-width page (checkout pattern)**: sand page, content centered in an 860px column. Header: pale-green back chip (40px, `#E9F3D8`), eyebrow `STEP n OF 3 · PRICE SHOWN BEFORE YOU PAY` (11px, `#9A968D`), 25px ink title, 3 emerald progress pips (26×5px) right. Buttons max-width 380px, left-aligned. **No map, no modal, no side panel on flow pages.**
- **Tracking**: full-width; live map band (dark emerald, lime dashed route, pulsing origin marker), rider card (avatar, name, `Bike · ★ 4.9`, call button), 4-step vertical timeline, ETA counts down.

### Tablet (700–1023px)
Same as desktop treatment (top bar + command-center home + full-width flow pages).

### Mobile (<700px)
- **Home**: emerald hero (logo, area, avatar; 23px headline; sand search bar with lime Send; caption "…or just tap what you need below 👇") → 4 duotone service tiles (pale-green `#E9F3D8` tiles) with confirmation sentence under → "YOUR ORDERS RIGHT NOW" feed (emerald live card + white status rows) → reorder chips → bottom nav **Home / My orders / My money / Me** (nav only — services never live in the bottom bar).
- **Flows**: emerald flow header (back chip, title, "Step n of 3", 3 lime progress bars), full-width lime CTA buttons.

## The Four Flows (all: price ALWAYS shown before confirming; "Nothing is paid yet" microcopy on every pre-price step)
1. **Package** — Where (pickup + dropoff, one-tap shortcuts "🏠 My home", "📍 Where I am now", saved destinations w/ km) → What (size: Small ✉️ "Fits one hand" / Medium 📦 "A shoe box or bag" / Large 🧺 "Needs two hands"; weight: Light <3kg / Medium 3–10kg / Heavy 10–25kg / Very heavy >25kg; tells the vehicle: bike / tricycle / van) → Price → Finding rider → Tracking.
2. **Gas** — Cylinder (3 / 6 / 12.5 / 25 kg with plain descriptions) + mode (**Refill mine**: "We take your cylinder, fill it, bring it back" / **Swap it**: "We bring a full one, take your empty — faster", +₦500) → Deliver to → Price → Tracking.
3. **Food** — Meal list from nearby kitchens (name, kitchen, prep time, price) → Deliver to → Price → Tracking ("Kitchen cooked it fresh").
4. **Pharmacy** — Two tabs: **Everyday medicine** (OTC items, no prescription) / **I have a prescription** (photo upload → amber "Pharmacist is checking it now… Adaeze at HealthPlus Lekki" → green "✅ Approved — … Checked by Adaeze O. (PCN licensed)") → Deliver to → Price → Tracking ("Pharmacist packed and sealed it"). Rx state maps to repo type `RxStatus` (`uploaded → under_review → approved`).

## Pricing model (demo values in prototype)
`total = ₦100 base + km × ₦45 + item/size fee`. Package size fees: S ₦100 / M ₦250 / L ₦600; weight: 0 / 100 / 300 / 700. Gas: 3kg ₦3,200 / 6kg ₦6,100 / 12.5kg ₦11,800 / 25kg ₦23,000 (+₦500 swap). Breakdown always itemized: Delivery (km) / item line / Base fare. Payment: "You pay ₦X when it arrives — cash or card, your choice."

## Selection & active states (the "zero-confusion" rules)
1. Every screen asks ONE question at the top, in plain words.
2. Selected item = deep emerald tile + lime border(2px) + lime text + ✓ label. Rest state = white bg, `#E4E0D6` border.
3. After every choice, a confirmation sentence: "✓ A **12.5 kg** cylinder, **swapped for a full one**. Next: where do we come?"
4. Price before paying, always. 5. Plain names: "My orders", "My money", "Medicine" — no jargon.
6. Disabled CTA = `#E4E0D6` bg / `#9A968D` text until the step is complete.

## Icons
Custom stroked set, 24-grid, 1.7–1.8px stroke, round caps/joins, one olive (`#7BA05B`) accent stroke per glyph. On tiles: pale-green `#E9F3D8` squircle (radius 14). Active: emerald tile, lime glyph. All SVGs are inline in the prototype files — lift the exact paths. (Package = cube, Gas = cylinder, Food = cloche, Medicine = rounded-square +, plus pin/wallet/user/cart/bell/home.)

## Interactions & state
- Flow state machine: `home → [gas|food|pharm] → where → (what) → price → finding(2.2s) → tracking`; back at every step.
- ETA counts down (15s tick in prototype). State persists (`localStorage` in prototype → Zustand + server state in production).
- Hovers: cards lift `translateY(-2/-3px)` + shadow; lime buttons darken; list rows tint `#FBFAF5`.
- Live pulse animation on active-order dots (scale 1→1.6 + fade, 2s loop).
- Repo `OrderStatus` mapping: confirmed → "Order confirmed", driver_assigned/picked up, in_transit → "On the way", delivered.

## Files
- `SpeedPlus Prototype.dc.html` — **primary reference**: interactive, all 4 flows, all 3 layouts (Tweaks: layout force, panel width, accent).
- `SpeedPlus Home.dc.html` — static responsive home (mobile / tablet / desktop layouts).
- `SpeedPlus Driver.dc.html` — rider app (`apps/driver`): online toggle, job offer, delivery stages + proof-of-delivery, earnings, profile/documents.
- `SpeedPlus Merchant.dc.html` — partner portal (`apps/merchant`): dashboard, order pipeline (confirm→preparing→ready), Rx review queue (approve/reject under PCN licence), products, earnings, verification/onboarding.
- `SpeedPlus Admin.dc.html` — ops portal (`apps/admin`): live dispatch + manual rider assignment, orders table, rider/partner approval queues, analytics.
- `SpeedPlus Ideation.dc.html` — full design-decision history (turns 1–11); icon spec at #11b, token sheet at #7c/#8c.
Open each in a browser. They are self-contained except Google Fonts (Space Grotesk, Instrument Sans).
