# SpeedPlus — Auth & Onboarding Styling Reference

> **Scope:** Customer, Driver (Rider), and Merchant apps
> **Source:** `packages/ui` shared components + per-app pages
> **Last updated:** August 2026

---

## Table of Contents

1. [Design Tokens](#1-design-tokens)
2. [Global CSS Per App](#2-global-css-per-app)
3. [AuthShell — Shared Layout Component](#3-authshell--shared-layout-component)
4. [Input Component](#4-input-component)
5. [Button Component](#5-button-component)
6. [Error Banners](#6-error-banners)
7. [Per-Page Field Layout](#7-per-page-field-layout)
8. [Customer Onboarding / Home Page](#8-customer-onboarding--home-page)
9. [Summary](#9-summary-what-makes-the-auth-ui-feel-premium)

---

## 1. Design Tokens

**File:** `packages/config/tailwind/tokens.css`

These CSS custom properties are the single source of truth shared across all three apps.

### 1.1 Color Palette

| Token | Hex | Role |
|---|---|---|
| `--color-emerald` | `#0A3D2C` | Primary brand surface, secondary button bg, focus ring |
| `--color-emerald-600` | `#0D4E38` | Hover on emerald |
| `--color-emerald-900` | `#072D20` | Pressed on emerald |
| `--color-lime` | `#C6F24E` | **Primary action**, active states, live pulse dot |
| `--color-lime-600` | `#AEE032` | Hover on lime |
| `--color-lime-700` | `#98C92B` | Pressed on lime |
| `--color-sand` | `#F7F5EF` | App background (customer + merchant) |
| `--color-tile` | `#E9F3D8` | Icon tiles, chips, soft accents |
| `--color-ink` | `#121216` | Primary text |
| `--color-mid` | `#63636E` | Secondary text, labels, placeholders |
| `--color-line` | `#E4E0D6` | Borders / hairlines |
| `--color-amber` | `#E8B14E` | Waiting / in-review status |
| `--color-icon-stroke` | `#1C3A2E` | Icon primary stroke |
| `--color-icon-accent` | `#7BA05B` | Icon olive accent stroke |

### 1.2 Typography

| Token | Value | Usage |
|---|---|---|
| `--font-display` | `Space Grotesk, sans-serif` | Headings, buttons, numbers |
| `--font-body` | `Instrument Sans, sans-serif` | Body text, labels, captions |

Both fonts are loaded via `next/font/google` in each app's `layout.tsx` with weights 400 / 500 / 600 / 700.

### 1.3 Border Radius Scale

| Token | Value |
|---|---|
| `--radius-sm` | `8px` |
| `--radius-md` | `12px` |
| `--radius-lg` | `16px` |
| `--radius-xl` | `20px` |
| `--radius-full` | `9999px` |

### 1.4 Shadow Scale

| Token | Value |
|---|---|
| `--shadow-xs` | `0 1px 2px rgba(0,0,0,0.05), 0 0 0 1px rgba(0,0,0,0.04)` |
| `--shadow-sm` | `0 1px 3px rgba(0,0,0,0.08), 0 0 0 1px rgba(0,0,0,0.04)` |
| `--shadow-md` | `0 4px 12px rgba(0,0,0,0.10), 0 1px 3px rgba(0,0,0,0.06)` |
| `--shadow-lg` | `0 8px 24px rgba(0,0,0,0.12), 0 2px 6px rgba(0,0,0,0.06)` |
| `--shadow-xl` | `0 20px 48px rgba(0,0,0,0.16), 0 4px 12px rgba(0,0,0,0.08)` |
| `--shadow-lime` | `0 4px 16px rgba(198,242,78,0.28), 0 1px 3px rgba(0,0,0,0.10)` |

### 1.5 Animation Utilities

All easing uses `cubic-bezier(0.16, 1, 0.3, 1)` — a snappy spring-like ease-out.

| Utility class | Delay | Keyframe |
|---|---|---|
| `animate-fade-up` | 0ms | `fadeUp` — translateY(14px → 0) + opacity |
| `animate-fade-up-1` | 55ms | ↑ |
| `animate-fade-up-2` | 110ms | ↑ |
| `animate-fade-up-3` | 165ms | ↑ |
| `animate-fade-up-4` | 220ms | ↑ |
| `animate-fade-up-5` | 275ms | ↑ |
| `animate-fade-up-6` | 330ms | ↑ |
| `animate-slide-down` | — | `slideDown` — translateY(-12px → 0) + opacity, 0.3s |
| `animate-slide-up` | — | `slideUp` — translateY(12px → 0) + opacity, 0.3s |
| `animate-press` | — | `scale(0.97)` on :active, 0.1s ease |

---

## 2. Global CSS Per App

All three apps import Tailwind v4 and the design tokens:

```css
@import "tailwindcss";
@import "@speedplus/config/tailwind/tokens.css";
```

| App | Body background | Notable additions |
|---|---|---|
| **Customer** | `var(--color-sand)` = `#F7F5EF` | `scrollbar-none` utility; `scroll-behavior: smooth`; `-webkit-tap-highlight-color: transparent` |
| **Driver** | `#E9E6DD` (warmer than sand) | Minimal base styles |
| **Merchant** | `var(--color-sand)` = `#F7F5EF` | Minimal base styles |

All apps set `themeColor: '#0A3D2C'` (emerald) via `next/viewport`.

---

## 3. `AuthShell` — Shared Layout Component

**File:** `packages/ui/src/components/auth-shell.tsx`

All auth pages across all three portals wrap their form content in this component. It renders a **two-panel split-screen layout**.

### 3.1 Layout Overview

```
┌──────────────────────────────────────────────────────────────┐
│  Brand Panel                    │  Form Panel                │
│  lg:w-[520px] xl:w-[580px]      │  flex-1                    │
│  hidden on mobile               │  full width on mobile      │
│                                 │                            │
│  bg: linear-gradient(160deg,    │  bg: linear-gradient(      │
│    #0A3D2C → #061F16 → #030E0A) │    #F9F7F2 → #F4F1EA)      │
└──────────────────────────────────────────────────────────────┘
```

**Root:** `min-h-screen flex bg-[#060F09]`

### 3.2 Brand Panel (Left — Desktop Only)

| Property | Value |
|---|---|
| Background | `linear-gradient(160deg, #0A3D2C 0%, #061F16 60%, #030E0A 100%)` |
| Width | `lg:w-[520px] xl:w-[580px]` |
| Padding | `p-14` |
| Flex direction | `flex-col justify-between` |

**Animated Decoration (`DeliveryOrbs`):**

| Element | Details |
|---|---|
| Orb 1 | 480×480px radial gradient `rgba(198,242,78,0.12)`, pulsing scale 1→1.08→1, 8s loop |
| Orb 2 | 360×360px radial gradient `rgba(0,196,140,0.10)`, pulsing scale 1→1.12→1, 10s loop, 2s delay |
| Route line 1 | SVG dashed path, `rgba(198,242,78,0.15)`, 1.5px stroke, draws in over 2.5s |
| Route line 2 | SVG dashed path, `rgba(0,196,140,0.10)`, 1px stroke, draws in over 3s |
| Route dots | 5 lime circles `rgba(198,242,78,0.6)`, r=2.5–4, staggered pop-in |
| Travelling dot | Lime `#C6F24E` r=5, SVG glow filter, travels route via `offsetPath`, 4s repeat |
| Stat chip 1 | "2,400+ deliveries today" — `bg-white/[0.06] border border-white/10 rounded-xl backdrop-blur-sm`, bottom-left, fades in at 1.8s |
| Stat chip 2 | "4.9 avg rating" — same glassmorphism style, top-right, fades in at 2.1s |

**Headline:** `font-display font-bold leading-[1.05] text-white`, `clamp(36px, 4vw, 48px)`, accent word in `text-lime`

**Subtext:** `text-white/50 text-[15px] leading-relaxed max-w-[280px]`

**Copyright:** `text-white/20 text-[11px] tracking-wide`

**Per-portal headline copy:**

| App | Headline | Accent word |
|---|---|---|
| Customer — Login | "Faster. Cheaper. **Better.**" | Better |
| Customer — Register | "Join the **movement.**" | movement |
| Customer — Forgot PW | "Faster. Cheaper. **Better.**" | Better |
| Driver — Login | "Deliver. Earn. **Grow.**" | Grow |
| Driver — Forgot PW | "Deliver. Earn. **Grow.**" | Grow |
| Merchant — Login | "Your business, **amplified.**" | amplified |
| Merchant — Forgot PW | "Your business, **amplified.**" | amplified |

**Portal label** (Driver + Merchant only):
`text-[11px] font-bold text-emerald tracking-[0.08em] uppercase mb-3`
Values: `"Rider Portal"` / `"Partner Portal"`

### 3.3 Form Panel (Right)

| Property | Value |
|---|---|
| Background | `linear-gradient(180deg, #F9F7F2 0%, #F4F1EA 100%)` |
| Decorative corner gradient | `radial-gradient(circle at 100% 0%, rgba(10,61,44,0.04) 0%, transparent 60%)` |
| Form card max-width | `max-w-[400px] mx-auto px-6 py-10 lg:py-0` |
| Form heading | `font-display font-bold text-ink tracking-tight`, `clamp(26px, 3vw, 32px)`, `lineHeight: 1.08` |
| Form subheading | `text-mid text-[14px] mt-2 leading-snug` |
| Mobile logo | `lg:hidden mb-10 mt-12` (light theme logo) |

**Motion:** Stagger container (`staggerContainer` + `staggerItem` Framer Motion variants). Disabled when `useReducedMotion()` is true.

---

## 4. Input Component

**File:** `packages/ui/src/components/input.tsx`

### 4.1 Base Styles

| Property | Value |
|---|---|
| Height | `h-[52px]` |
| Background | `bg-white` (error: `bg-red-50/40`) |
| Text size / color | `text-[14px] text-ink` |
| Placeholder | `text-mid/40` |
| Border radius | `var(--radius-md)` = 12px |
| Padding | `px-4` |
| Outline | Removed; ring via animated `boxShadow` |

### 4.2 Label

| State | Color | Notes |
|---|---|---|
| Idle | `#63636E` (mid) | |
| Focused | `#0A3D2C` (emerald) | Animated via Framer Motion 0.15s |
| Error | `#DC2626` | |

Label style: `text-[12px] font-semibold tracking-[0.02em]`

### 4.3 Focus Ring (Spring-animated `boxShadow`)

| State | Shadow |
|---|---|
| Idle | `0 0 0 1px rgba(0,0,0,0.10), 0 1px 3px rgba(0,0,0,0.05)` |
| Focused | `0 0 0 1.5px rgba(10,61,44,0.6), 0 0 0 4px rgba(10,61,44,0.08)` |
| Success | `0 0 0 1.5px rgba(22,163,74,0.55), 0 0 0 4px rgba(22,163,74,0.08)` |
| Error | `0 0 0 1.5px rgba(220,38,38,0.5), 0 0 0 4px rgba(220,38,38,0.08)` |

Spring config: `stiffness: 400, damping: 32`

### 4.4 State Behaviours

| State | Behaviour |
|---|---|
| **Error** | Container shakes: `x: [0, -6, 6, -4, 4, -2, 2, 0]` over 0.38s |
| **Success** | Green checkmark icon spring-pops in at right |
| **Disabled** | `cursor-not-allowed opacity-40 bg-sand` |
| **Suffix** | `absolute inset-y-0 right-0 pr-3.5` — used for eye icon on password fields |

### 4.5 Error / Hint Messages

- **Error:** `text-[11px] font-medium text-red-600`, `y: -4 → 0` animated on enter/exit
- **Hint:** `text-[11px] text-mid`, fades in/out

---

## 5. Button Component

**File:** `packages/ui/src/components/button.tsx`

Built with `class-variance-authority`. All auth forms use `variant="primary" size="lg"`.

### 5.1 Variants

| Variant | Background | Text | Notes |
|---|---|---|---|
| `primary` | `#C6F24E` (lime) | `#0A3D2C` (emerald) | Lime glow shadow |
| `secondary` | `#0A3D2C` (emerald) | `#F7F5EF` (sand) | Emerald shadow |
| `outline` | Transparent | `text-ink` | `border border-line` |
| `ghost` | Transparent | `text-mid` | `bg-sand` on hover |
| `danger` | `bg-red-600` | White | Red shadow |

**Primary shadow:**
- Rest: `0 1px 3px rgba(0,0,0,0.12), 0 4px 12px rgba(198,242,78,0.25)`
- Hover: `0 2px 8px rgba(0,0,0,0.14), 0 6px 20px rgba(198,242,78,0.35)`

### 5.2 Size Scale

| Size | Height | Padding | Font | Radius |
|---|---|---|---|---|
| `sm` | 36px | `px-4` | 13px | 8px |
| `md` | 44px | `px-5` | 14px | 12px |
| `lg` | 48px | `px-6` | 15px | 12px |
| `xl` | 56px | `px-8` | 16px | 16px |
| `icon` | 40×40px | — | 14px | 12px |

### 5.3 Motion & Interactions

| Interaction | Transform |
|---|---|
| Hover | `y: -1.5, scale: 1.005` |
| Tap | `scale: 0.97, y: 0` |
| Spring | `stiffness: 500, damping: 30` |

**Loading state:** Spinning SVG circle + children text, spring-fades in.
**Font:** `font-display font-bold tracking-[-0.01em]` (Space Grotesk).

---

## 6. Error Banners

Standard pattern across all auth forms:

```tsx
<div
  className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-3.5 py-2.5"
  role="alert"
  aria-live="polite"
>
  <AlertCircleIcon color="#DC2626" />
  <p className="text-xs text-red-600">{error}</p>
</div>
```

| App | Animation |
|---|---|
| Customer Login | `motion.div` with `opacity: 0, y: -6, scale: 0.98` → `1, 0, 1` + `AnimatePresence` |
| Customer Register | Static (no motion wrapper) |
| Driver Login | Static |
| Merchant Login | `motion.div` with `opacity: 0, y: -6` → `1, 0` + `AnimatePresence`, `mt-5` |

---

## 7. Per-Page Field Layout

### 7.1 Customer — Login (`/login`)

```
Phone number
Password  [👁]
              Forgot password? (right-aligned, 12px)
[Error banner]
[Sign in]  ← primary lg, full width
──── or ────
New to SpeedPlus?  Create an account
```

### 7.2 Customer — Register (`/register`)

```
[First name]  [Last name]   ← 2-col grid, gap-3
Phone number
Password  [👁]
Referral code  [+₦500 bonus chip if filled]
[Error banner]
[Create account]  ← primary lg, full width, mt-1
Terms of Service · Privacy Policy  (12px, centered)
──────────────────────────────────
Already have an account?  Sign in
```

### 7.3 Driver — Login (`/login`)

```
Phone number
Password  [👁]
              Forgot password? (right-aligned)
[Error banner]
[Sign in]  ← primary lg, full width, mt-1
```

### 7.4 Merchant — Login (`/login`)

```
Phone number
                    ← mt-5 (20px gap)
Password  [👁]     ← animated eye icon (rotate+scale on hover, AnimatePresence swap)
              Forgot password? (right-aligned, 12px, mt-2.5)
[Error banner — mt-5]
[Sign in to Partner Portal]  ← primary lg, full width, mt-7 (28px gap)
```

### 7.5 All Apps — Forgot Password (`/forgot-password`)

**Default:**
```
Phone number
[Error banner]
[Send reset link]  ← primary lg, full width, mt-1
Back to sign in (centered link)
```

**After submission:**
```
┌─────────────────────────────────────────────────┐
│  bg-tile · border-emerald/20 · rounded-xl       │
│  Check your messages           (sm, bold)       │
│  We sent a reset link to {phone}.               │
│  It expires in 15 minutes.    (sm, text-mid)    │
└─────────────────────────────────────────────────┘
Back to sign in (centered link)
```

---

## 8. Customer Onboarding / Home Page

**File:** `apps/customer/app/page.tsx`

### 8.1 Hero Section

| Property | Value |
|---|---|
| Root background | `#080F0A` |
| Video | `/hero.mp4`, poster `/hero-poster.jpg`, `opacity: 0.25`, `mixBlendMode: luminosity` |
| Gradient overlay | `linear-gradient(180deg, rgba(8,15,10,0.5) 0%, rgba(8,15,10,0.2) 40%, rgba(8,15,10,0.95) 100%)` |
| Padding | `px-5 pt-14 pb-10` |

### 8.2 Top Bar

- Wallet button: `text-[12px] font-semibold text-white/70 bg-white/[0.08] border border-white/[0.08] rounded-full px-3.5 py-1.5`

### 8.3 Hero Copy

| Element | Style |
|---|---|
| H1 | `font-display font-bold text-white leading-[1.05] tracking-tight`, `clamp(32px, 8vw, 42px)` |
| Sub-copy | `text-white/40 mt-2 text-[13px] font-medium tracking-wide` |

### 8.4 Service Verticals

Icons: `stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"`.
All use `accent: '#C6F24E'` (lime). Services: Cooking Gas · Food (feature-flagged) · Pharmacy · Send Package.

### 8.5 Motion

Framer Motion `spring.smooth` with stagger containers. `useReducedMotion()` disables all animation.

---

## 9. Summary: What Makes the Auth UI Feel Premium

| Element | Technique |
|---|---|
| **Layout** | Split-screen: dark brand panel (left) + light form panel (right) |
| **Brand panel** | Deep green gradient + animated SVG delivery route + glassmorphism stat chips |
| **Inputs** | 52px tall, spring-animated focus ring, shake-on-error, success checkmark |
| **Primary button** | Lime (`#C6F24E`) with glowing lime shadow + lift on hover |
| **Typography** | Space Grotesk (display) + Instrument Sans (body) — variable Google Fonts |
| **Animations** | Framer Motion stagger entrance, `cubic-bezier(0.16,1,0.3,1)` throughout |
| **Error states** | Animated red banners with icon, ARIA `role="alert"` + `aria-live="polite"` |
| **Color accent** | Lime word highlight in brand panel headline — distinct copy per portal |
| **Accessibility** | `aria-live`, `aria-invalid`, `aria-label`, `aria-pressed`, `aria-describedby` everywhere |
| **Reduced motion** | `useReducedMotion()` check disables all Framer Motion animations |
