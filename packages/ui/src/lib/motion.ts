import type { Variants, Transition } from 'framer-motion';

// ── Spring configs ────────────────────────────────────────────────────────────

export const spring = {
  snappy:  { type: 'spring', stiffness: 500, damping: 35, mass: 0.8 } as Transition,
  smooth:  { type: 'spring', stiffness: 300, damping: 30, mass: 1   } as Transition,
  gentle:  { type: 'spring', stiffness: 200, damping: 28, mass: 1   } as Transition,
  slow:    { type: 'spring', stiffness: 120, damping: 20, mass: 1   } as Transition,
  bounce:  { type: 'spring', stiffness: 400, damping: 20, mass: 0.8 } as Transition,
} as const;

export const ease = {
  out:     [0.16, 1, 0.3, 1]  as [number, number, number, number],
  inOut:   [0.4, 0, 0.2, 1]   as [number, number, number, number],
  sharp:   [0.4, 0, 0.6, 1]   as [number, number, number, number],
} as const;

// ── Page / container variants ─────────────────────────────────────────────────

export const fadeUp: Variants = {
  hidden:  { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0, transition: { ...spring.smooth } },
  exit:    { opacity: 0, y: -8, transition: { duration: 0.2, ease: ease.sharp } },
};

export const fadeIn: Variants = {
  hidden:  { opacity: 0 },
  visible: { opacity: 1, transition: { duration: 0.3, ease: ease.out } },
  exit:    { opacity: 0, transition: { duration: 0.15 } },
};

export const scaleIn: Variants = {
  hidden:  { opacity: 0, scale: 0.96 },
  visible: { opacity: 1, scale: 1, transition: { ...spring.snappy } },
  exit:    { opacity: 0, scale: 0.96, transition: { duration: 0.15 } },
};

export const slideInRight: Variants = {
  hidden:  { opacity: 0, x: 24 },
  visible: { opacity: 1, x: 0, transition: { ...spring.smooth } },
  exit:    { opacity: 0, x: -16, transition: { duration: 0.2 } },
};

// ── Stagger container ─────────────────────────────────────────────────────────

export const staggerContainer: Variants = {
  hidden:  { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.06, delayChildren: 0.05 },
  },
};

export const staggerItem: Variants = {
  hidden:  { opacity: 0, y: 12 },
  visible: { opacity: 1, y: 0, transition: { ...spring.smooth } },
};

// ── Interactive element variants ──────────────────────────────────────────────

export const buttonTap = { scale: 0.97 };
export const buttonHover = { scale: 1.01, y: -1 };

export const cardHover: Variants = {
  rest:  { scale: 1,    y: 0,  boxShadow: '0 1px 3px rgba(0,0,0,0.08), 0 0 0 1px rgba(0,0,0,0.04)' },
  hover: { scale: 1.01, y: -2, boxShadow: '0 8px 24px rgba(0,0,0,0.12), 0 0 0 1px rgba(0,0,0,0.06)' },
  tap:   { scale: 0.99, y: 0,  boxShadow: '0 1px 3px rgba(0,0,0,0.08)' },
};

export const inputFocus: Variants = {
  rest:    { boxShadow: '0 1px 2px rgba(0,0,0,0.05), 0 0 0 1px rgba(0,0,0,0.03)' },
  focused: { boxShadow: '0 0 0 3px rgba(10,61,44,0.12), 0 1px 2px rgba(0,0,0,0.05)' },
};

// ── Error shake ───────────────────────────────────────────────────────────────

export const shakeVariants: Variants = {
  idle:  { x: 0 },
  shake: {
    x: [0, -6, 6, -4, 4, -2, 2, 0],
    transition: { duration: 0.4, ease: 'easeInOut' },
  },
};

// ── Reduced motion helper ─────────────────────────────────────────────────────
// Use this to wrap any variant set — returns empty variants when user prefers
// reduced motion. Call at component level with useReducedMotion().

export function respectMotion<T extends Variants>(variants: T, reduced: boolean | null): T | Record<string, object> {
  if (reduced) return { hidden: {}, visible: {}, exit: {}, rest: {}, hover: {}, tap: {}, idle: {}, shake: {}, focused: {} };
  return variants;
}
