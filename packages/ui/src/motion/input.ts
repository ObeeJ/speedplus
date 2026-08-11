import type { Variants } from 'framer-motion';

export const inputFocus: Variants = {
  rest: { boxShadow: '0 1px 2px rgba(0,0,0,0.05), 0 0 0 1px rgba(255,255,255,0.08)' },
  focused: { boxShadow: '0 0 0 3px rgba(37,99,235,0.25), 0 0 0 1px rgba(37,99,235,0.6)' },
};

export const shakeVariants: Variants = {
  idle: { x: 0 },
  shake: {
    x: [0, -6, 6, -4, 4, -2, 2, 0],
    transition: { duration: 0.35, ease: 'easeInOut' },
  },
};

export const labelFloat: Variants = {
  resting: { y: 0, scale: 1, color: 'rgba(255,255,255,0.5)' },
  active: { y: -18, scale: 0.85, color: '#3B82F6', transition: { duration: 0.15 } },
};
