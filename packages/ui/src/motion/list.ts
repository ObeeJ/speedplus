import type { Variants } from 'framer-motion';

export const staggerContainer: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.05, delayChildren: 0.04 },
  },
};

export const staggerItem: Variants = {
  hidden: { opacity: 0, y: 10 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { type: 'spring', stiffness: 300, damping: 28 },
  },
  exit: { opacity: 0, y: -6, transition: { duration: 0.15 } },
};

export const tableRowVariants: Variants = {
  hidden: { opacity: 0, x: -6 },
  visible: { opacity: 1, x: 0, transition: { duration: 0.2 } },
  exit: { opacity: 0, x: 6, transition: { duration: 0.15 } },
};
