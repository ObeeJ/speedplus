import type { Variants } from 'framer-motion';

export const buttonTap = { scale: 0.98 };

export const buttonHover = {
  scale: 1.01,
  y: -1,
  transition: { type: 'spring', stiffness: 400, damping: 25 },
};

export const buttonVariants: Variants = {
  rest: { scale: 1, y: 0 },
  hover: { scale: 1.01, y: -1, transition: { type: 'spring', stiffness: 400, damping: 25 } },
  tap: { scale: 0.98, y: 0, transition: { type: 'spring', stiffness: 500, damping: 30 } },
};

export const loadingMorphVariants: Variants = {
  initial: { opacity: 0, scale: 0.8 },
  animate: { opacity: 1, scale: 1, transition: { duration: 0.2 } },
  exit: { opacity: 0, scale: 0.8, transition: { duration: 0.15 } },
};
