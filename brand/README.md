# Fourdat — visual identity

**Status:** first proposal, not a cleared identity. Nothing here should be
printed, embroidered or registered until the checks in §7 are done.

Full presentation: [`fourdat-identity.html`](./fourdat-identity.html).

This document is canonical for **how the identity is drawn and used**. It does
not supersede `docs/superpowers/specs/2026-08-06-rebrand-positioning-design.md`,
which stays canonical for positioning and naming constraints, or
`docs/BUSINESS-MODEL.md`, which stays canonical for fees and unit economics.

---

## 1. The idea

Three directions were drawn. **Direction 01, The Aperture**, is the recommended
one and the only one developed into a system.

| # | Direction | Argument | Why not |
|---|-----------|----------|---------|
| 01 | **The Aperture** | A closed protective form cut once by a diagonal opening, the opening divided into four channels | — recommended |
| 02 | The Handover | Two identical hooks rotated into each other; the channel between them is the exchange | Reads as a logistics mark first, and the channel fills in below 20px |
| 03 | The Seal | A soft square with one corner turned back, like a sealed parcel | Closest to what a courier brand would already own |

The Aperture holds protection and release in the same shape, which is what the
adopted positioning actually sells — *your money doesn't move until your order
does.* The four segments are a fact about the company, not a diagram of it, so a
fifth vertical needs no redraw. The silhouette borrows from a cowrie shell
opened along its aperture: West African currency, and the oldest visual
shorthand on this continent for value that can be counted and handed over.

## 2. Assets

| File | Use |
|------|-----|
| `svg/mark.svg` | Primary symbol, four segments |
| `svg/mark-compact.svg` | 24px and below, favicon, avatars — solid aperture |
| `svg/mark-outline.svg` | Embroidery, engraving, single-weight stamps |
| `svg/wordmark.svg` | FOURDAT, drawn outlines, no font dependency |
| `svg/wordmark-open.svg` | Expressive cut — the O becomes the symbol |
| `svg/lockup-horizontal.svg` | Default: headers, documents, vehicle panels, signage |
| `svg/lockup-stacked.svg` | Square formats: bags, decals |

Every file paints with `currentColor`, so colour is set by the parent's CSS
`color` and one file serves light, dark and one-colour use.

## 3. Construction

The symbol is a 100u square with a 30u corner radius. The aperture is a lens on
the 45° axis, from `21,79` to `79,21`, drawn with two 76u arcs and broken by
three 5u bridges into four segments.

The wordmark is drawn, not set: 100u cap height, 20u stem, 14u letter gap,
constant across all seven letters. Its ownable moves are the flat-apex **A**,
the squircle **O** and **D** that share the symbol's corner logic, and even
monolinear colour with no optical thinning at the joins.

**Clear space** is the corner radius, `x`, on every side. Nothing enters it.

**Minimum sizes**

| Lockup | Screen | Print |
|--------|-------:|------:|
| Horizontal | 120 px | 30 mm |
| Stacked | 96 px | 24 mm |
| Wordmark only | 88 px | 22 mm |
| Symbol only | 16 px | 8 mm |

Switch to `mark-compact.svg` at 24px and below, in code, automatically.

## 4. Colour

Tokens live in `packages/config/tailwind/tokens.css` and were added without
removing anything, so no app breaks on this commit.

| Name | Hex | Role |
|------|-----|------|
| Palm | `#0A3D2C` | The brand. Large surfaces, wordmark, symbol |
| Palm Deep | `#062A1E` | Pressed states, deepest edge |
| Ink | `#14170F` | Text — a warm near-black, never pure `#000` |
| Clay | `#C25E3A` | Accent only. Section marks, one word in a headline, Food |
| Chalk | `#F4F2EB` | The light ground |
| Signal | `#C6F24E` | **Live states only.** Never brand furniture |

The one real change to the existing palette: the acid lime stops being a brand
colour and becomes a status. It means *live* — a rider moving, an order
tracked, a payment released — and appears at most once per view. Clay replaces
it as the warmth in the identity, and stays an accent.

Rough proportion on any surface: Chalk 52, Palm 34, Ink 9, Clay 4, Signal 1.

## 5. Typography

| Role | Face | Weights | Job |
|------|------|---------|-----|
| Display | Bricolage Grotesque | 600–800, tracked −3% | Headlines, campaign lines |
| Text | Familjen Grotesk | 400–600 | Every screen, button, paragraph |
| Data | Martian Mono | 300–500 | Tracking codes, delivery codes, receipts |

The mono is not decoration. Fourdat runs on codes people read aloud to a rider
at a gate, and distinct zeroes and ones are error prevention.

New tokens `--font-brand`, `--font-text` and `--font-code` are in place.
`--font-display` and `--font-body` still point at DM Sans and Instrument Sans so
nothing shifts under the apps; migrate them per app when each is ready.

## 6. Sub-brands

Always **Fourdat + the plain English noun**. Never Fourdat Rx, never FourdatGo,
never a compound containing the four. The symbol stays Palm and the wordmark
stays as drawn; only the chip colour and glyph change.

| Vertical | Chip |
|----------|------|
| Fourdat Food | Clay `#C25E3A` |
| Fourdat Gas | Ochre `#C08A2E` |
| Fourdat Pharmacy | Mint `#2F8A72` |
| Fourdat Packages | Plum `#4E4459` |

## 7. Before anything is printed

Per the naming constraints already binding in the rebrand spec:

- Clear **FOURDAT** and this mark through a CAC and Trademarks Registry search
  in Nigeria.
- Confirm the `.com` at an actual registrar. DNS absence is not proof.

No spend on signage, vests or bike boxes until both come back clean.

## 8. Rules

**Do** — let Palm hold large areas; switch to the compact symbol at 24px in
code; keep Signal to one live element per view; set every delivery and tracking
code in Martian Mono.

**Don't** — rotate the symbol; add a gradient, bevel, glow or drop shadow to the
mark; re-set FOURDAT in a font; put the four verticals inside the logo.
