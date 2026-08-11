import type { Variants } from 'framer-motion';

export const toastVariants: Variants = {
  initial: { opacity: 0, y: -16, scale: 0.95 },
  animate: { opacity: 1, y: 0, scale: 1, transition: { type: 'spring', stiffness: 400, damping: 28 } },
  exit: { opacity: 0, y: -12, scale: 0.95, transition: { duration: 0.15 } },
};

export const toastProgressVariants: Variants = {
  initial: { width: '100%' },
  animate: (durationMs: number) => ({
    width: '0%',
    transition: { duration: durationMs / 1000, ease: 'linear' },
  }),
};
