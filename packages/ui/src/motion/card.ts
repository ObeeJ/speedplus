import type { Variants } from 'framer-motion';

export const cardHover: Variants = {
  rest: {
    scale: 1,
    y: 0,
    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(255, 255, 255, 0.05)',
    transition: { duration: 0.2 },
  },
  hover: {
    scale: 1.005,
    y: -2,
    boxShadow: '0 12px 32px rgba(0, 0, 0, 0.24), 0 0 0 1px rgba(37, 99, 235, 0.25)',
    transition: { type: 'spring', stiffness: 350, damping: 25 },
  },
  tap: {
    scale: 0.99,
    y: 0,
    boxShadow: '0 2px 6px rgba(0, 0, 0, 0.12)',
    transition: { type: 'spring', stiffness: 500, damping: 30 },
  },
};
