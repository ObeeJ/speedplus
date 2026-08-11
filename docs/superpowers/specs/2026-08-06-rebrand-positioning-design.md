# SpeedPlus Rebrand — Positioning & Naming Design

**Status:** Positioning validated and adopted. Naming: open, unresolved —
see §5. This doc supersedes no other doc for legal/financial facts;
`docs/BUSINESS-MODEL.md` remains canonical for fees, unit economics, and
regulatory posture. This doc is canonical for **brand positioning** only.

## 1. Why this doc exists

`speedplus.com` is taken. That forced a rename, which forced a harder
question: was "SpeedPlus" — implicitly "fast delivery, Opay-for-logistics"
— ever the right positioning? This doc records the investigation, the
conclusion, and the constraints for whatever name eventually replaces
SpeedPlus. The naming search itself did not converge on a finalist in this
session and is treated as a separate, still-open follow-up (§5).

## 2. The problem, correctly stated

**Original hypothesis (founder, unvalidated):** compete on speed, the way
Opay won on fast transactions.

**Investigation:** `docs/BUSINESS-MODEL.md` already states the real pitch,
independent of this rebrand exercise — "the delivery network where your
money is only released when your goods arrive" (line 16). Market research
into Nigerian delivery-app sentiment (Glovo/Chowdeck, 2025–26) ranked
complaints: order accuracy, app experience, **payment/refund issues**,
delivery time, support. Delivery *speed* ranked fourth, not first.
Separately, Techpoint Africa demonstrated a fictitious restaurant could take
live orders on both Glovo and Chowdeck with zero identity verification — a
live trust crisis in a $1.1B market.

**Conclusion:** speed is hygiene, not a wedge — any funded competitor buys
it by hiring more riders. Money-certainty and identity-verified merchants/
riders are structurally harder to copy, because incumbents' growth models
depend on frictionless onboarding, which escrow and verification directly
slow down. A competitor cannot adopt this model without damaging the thing
that made them big. **That is the actual moat**, and it is not escrow alone
— it is the whole reinforcing stack: verified merchants + verified riders +
escrow + delivery-code release + immutable ledger + dispute evidence + fair
rider economics + transparent pricing. Any single piece is copyable; all of
them together, without redesigning a competitor's own economics, is not.

**Correction (external PR/brand review, accepted):** do not conflate the
customer's emotional pain ("I don't want to lose my money") with the
engineering solution (escrow, delivery codes). The brand sells the
**outcome** — "you'll get what you paid for, or your money stays protected"
— not the mechanism. Mechanism is *how*; it is not *why*, and it does not
belong in the brand voice or the name.

**Category correction (same review, accepted):** the closer comparison is
not Opay. It is Paystack + Stripe Connect + Uber + Escrow.com combined —
**trusted commerce infrastructure**, not "a delivery app." This matters
operationally: if SpeedPlus later expands into home services, repairs,
rentals, errands, or B2B logistics, all of that is still "trusted local
commerce." A brand locked to "delivery" or "dropping packages" would need a
second rename to follow that expansion. The brand adopted now should not
require that.

## 3. Adopted positioning

- **One-line promise:** *your money doesn't move until your order does.*
- **Brand sells:** outcome/certainty. Never the mechanism (don't put
  "escrow," "ledger," or "delivery code" in headline copy — those are
  proof points for the FAQ and trust page, not the tagline).
- **Speed:** hygiene only. Never the headline claim, never absent either.
- **Coverage (gas/pharmacy/package/food/grocery):** the acquisition hook —
  "why someone downloads." Not the core promise.
- **Rider economics (₦ above-market pay per `BUSINESS-MODEL.md`):** a
  recruiting/retention story for the supply side. Real, but not
  customer-facing brand content — customers don't feel it directly.

This positioning is considered settled pending real-world testing (ad copy,
landing page A/B, customer interviews). It is not to be re-litigated from
scratch in the naming follow-up session — start there, challenge only with
new evidence.

## 4. Naming constraints (binding on any future naming pass)

Established over two rounds of founder correction plus one external
PR/brand review:

1. Real, correctly-spelled word(s) only if going the two-word route
   (Paystack/Chowdeck/Doordash pattern) — no invented spelling, no
   letter-swaps (no K-for-C, no dropped vowels, no "-ify/-ly" filler
   dressing up an existing word).
2. Effortlessly pronounceable on first hearing — no decoding required.
3. Verbable in everyday Nigerian speech patterns — test against *"you fi
   ___ am"* and plain English *"I'll ___ it."* A name that doesn't survive
   being used as a casual verb between friends is weaker, though not
   automatically disqualified (not every strong brand is a verb).
4. Must travel — legible and adoptable to a Nigerian **and** to someone
   abroad with zero context. Pidgin grammar/usage patterns are fine to
   design around; Pidgin vocabulary itself is not, per founder correction.
5. Must claim a **`.com`**. DNS-absence (`dig +short NS <domain>` returning
   empty) is a useful cheap signal but is **not** proof of availability —
   every shortlisted name must be confirmed at an actual registrar
   (Namecheap/GoDaddy availability check) before being treated as secured,
   and cleared through an actual Nigerian trademark search (CAC/Trademarks
   Registry) before any spend commits to it.
6. Prefer names that encode the **outcome** (completion, confirmation,
   protection, fulfillment) over the **mechanism** (escrow, delivery code)
   or bare **movement** (drop, ship, ride, move, go) — per §2's PR
   correction and the rejected "Drop" family (see §5).
7. Avoid anything reading as "AI-generated generic" — no portmanteau with
   no real-word roots, no vowel-dropped startup-speak.
8. Must still make sense if the company expands beyond delivery into
   trusted local commerce generally (home services, repairs, rentals,
   errands, B2B). Avoid names that hard-lock to "package moved A to B."

**Scoring rubric** (external review, apply to every future candidate,
100 pts total):

| Category           | Weight |
|---------------------|-------:|
| Easy to pronounce    |    20 |
| Memorable            |    15 |
| Can become a verb    |    20 |
| International        |    10 |
| Nigerian-friendly     |    10 |
| Trust association    |    15 |
| Future-proof         |    10 |

## 5. Naming search: what was tried, and why nothing was finalized

Three lexical passes were run this session, using DNS nameserver-lookup as
a cheap (not authoritative) availability signal:

**Pass 1 — movement/logistics compounds** (sure/drop/land/code/point/link/
gate/hand): produced **SureDrop** (`suredrop.com`, DNS-clear) as the
leading candidate. Rejected on external PR review: reads as courier/dev-
tooling ("a package was dropped"), not as a money-certainty promise. Family
abandoned per Rule 6/8 above.

**Pass 2 — completion/protection/confirmation compounds** (trust/sure/true/
proof/claim/settle/seal/key + core/base/mark/wave/seal): produced
**TrustSeal** (`trustseal.com`, DNS-clear) as the leading candidate.
Rejected on further scrutiny: "TrustSeal" is an existing industry term for
website-security badges (Norton Secured Seal, McAfee SECURE), creating
category confusion and real trademark-collision risk. Broader concern
raised and accepted: literal-descriptor names ("Trust___") cap long-term
brand ceiling versus arbitrary/evocative names (Apple, Bolt) that earn
meaning through consistent behavior rather than stating it upfront.

**Pass 3 — one-word exploration:**
- Real single-word `.com`s (anchor, vault, haven, harbor, beacon, compass,
  halo, atlas, nimbus, landed, sealed, convey, vouch) are **all
  registered** — confirmed, not category-specific; essentially the entire
  common-English-word `.com` namespace is owned, typically by brokers or
  incumbents, and reclaiming one would cost a five-to-six-figure broker
  purchase. Off the table at current stage.
- All 4-letter `.com`s are registered as a structural fact (~456,976
  possible combinations, fully claimed industry-wide since roughly 2015),
  independent of word choice.
- 5–6 letter invented coinages (Kuda/Bolt/Grab/Opay pattern — short,
  natural-sounding, no literal meaning) were explored via random syllable
  generation. Two came back DNS-clear (`nuvora.com`, `tovako.com`) but
  neither has Nigerian warmth or passes the verb test convincingly, and
  random generation was judged a low-hit-rate method for this kind of
  name. **Not adopted; method rejected, not just the results.**
- One near-miss worth revisiting deliberately: **"Kolo"** — existing
  Nigerian slang for a home savings box/informal savings container ("money
  kept safe until needed"). Semantically close to the brand promise and
  linguistically native rather than borrowed, but `kolo.com` is registered.
  A deliberate (non-random) coinage session should explore this and similar
  meaningful-fragment directions rather than blind syllable combination.

**Decision:** naming is explicitly left open rather than forcing a pick
between two flawed finalists (SureDrop, TrustSeal) or a weak coinage found
by brute force. A follow-up session should run a **deliberate** coinage
exercise (not randomized DNS sweeps) against the constraints in §4, or
revisit the two-word route with different roots than drop/trust/seal.

## 6. Core brand values

Every value below is backed by something specific already built or decided
— not aspirational filler. Grouped for readability; not ranked.

**Trust & certainty**
1. Certainty of outcome — the money doesn't move until the order does
   (`BUSINESS-MODEL.md:16`).
2. Protection by design, not by promise — escrow, delivery-code release,
   dual-approval freeze; the guarantee is structural.
3. Verification over convenience — merchant/rider verification exists
   precisely where competitors let a fake vendor take live orders.
4. Accountability, provably — append-only double-entry ledger; money
   cannot leak without an invariant error; every fee change is versioned
   and audit-logged.
5. Tamper-evidence — HMAC-signed single-use quotes; suspicious GPS-stamped
   confirmations are flagged, never silently trusted.

**Honesty & transparency**
6. Say only what's true — unshipped features (subscriptions, POD checkout)
   are explicitly barred from marketing until real; the same discipline
   applies to brand voice.
7. No hidden charges — weather surcharge shown as a labelled line item,
   admin-controlled, default off.
8. No opportunistic repricing — fuel-index changes only suggest an
   adjustment, never auto-applied; in-flight orders settle at the rate in
   force when placed.
9. Plain-spoken regulatory posture — the same description of wallets/PSP
   rails is used with investors, regulators, and customers alike.

**Fairness & equity**
10. Riders paid above survival cost, on purpose — nets ~₦500–600/job,
    specifically because underpaying was a retention failure.
11. Merchants keep the majority — 92% take-home.
12. No cash-carrying risk for riders at night — escrow replaces COD's
    real danger.
13. Fast, no-fuss resolution on small disputes — refunds under ₦5k paid
    immediately, not investigated.
14. Fraud discipline protects the honest majority — growth spend stops if
    fraud loss exceeds 0.7% of GMV, rather than normalizing losses.

**Reliability & rigor**
15. Real-world accuracy over shortcuts — actual road distance via OSRM,
    ETA includes a real pickup buffer.
16. Operational discipline as habit — five numbers checked daily; a
    3-day breach is a founder-level incident.
17. Consistency under pressure — pricing/settlement don't shift mid-order
    even if fees change minutes later.

**Identity & belonging**
18. Built in Nigeria, priced for Nigerian reality — base fees derived
    from actual fuel cost and traffic conditions.
19. Woven into how people already talk — naming work chases a name that
    becomes part of ordinary speech, not just a logo.
20. Breadth as generosity, not just growth — five verticals show up for
    more of someone's life, not just their lunch order.

**Ambition & restraint**
21. Serious infrastructure, not a novelty app — self-concept as trusted
    commerce infrastructure (Paystack + Stripe Connect + Uber +
    Escrow.com), not merely "a delivery app."
22. Patience over hype — subscriptions, batching, fleet deals are
    explicitly sequenced "prove it first, then build it."
23. Humility as a working value — willing to correct a fee table that
    underpaid riders in the past; willing to leave naming unresolved
    rather than force a weak choice.

## 7. Explicitly open / not decided

- No brand name is chosen. SureDrop and TrustSeal remain candidates but are
  not recommended as-is.
- No domain has been purchased or registrar-confirmed. All "DNS-clear"
  signals in §5 require registrar verification before any name is treated
  as securable.
- No trademark search has been run in Nigeria (CAC/Trademarks Registry) on
  any candidate.
- Social-media handle availability was probe-checked via HTTP status codes
  and found **unreliable** (platforms return 200 for both live and empty
  profile shells) — requires manual verification, not automated.
- The positioning in §3 has not been tested against real customers, ad
  copy, or a landing page — it is the strongest-argued conclusion available
  tonight, not a market-validated one.
