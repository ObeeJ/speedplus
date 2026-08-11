import type { Variants } from 'framer-motion';

export const backdropVariants: Variants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { duration: 0.2, ease: [0.16, 1, 0.3, 1] } },
  exit: { opacity: 0, transition: { duration: 0.15 } },
};

export const dialogContentVariants: Variants = {
  hidden: { opacity: 0, scale: 0.95, y: 10 },
  visible: {
    opacity: 1,
    scale: 1,
    y: 0,
    transition: { type: 'spring', stiffness: 380, damping: 28 },
  },
  exit: {
    opacity: 0,
    scale: 0.96,
    y: 6,
    transition: { duration: 0.15, ease: [0.4, 0, 0.6, 1] },
  },
};
