import type { Variants, Transition } from 'framer-motion';

export const spring = {
  snappy: { type: 'spring', stiffness: 400, damping: 30, mass: 0.8 } as Transition,
  smooth: { type: 'spring', stiffness: 300, damping: 30, mass: 1 } as Transition,
  gentle: { type: 'spring', stiffness: 200, damping: 28, mass: 1 } as Transition,
  slow: { type: 'spring', stiffness: 120, damping: 20, mass: 1 } as Transition,
  bounce: { type: 'spring', stiffness: 400, damping: 20, mass: 0.8 } as Transition,
} as const;

export const ease = {
  out: [0.16, 1, 0.3, 1] as [number, number, number, number],
  inOut: [0.4, 0, 0.2, 1] as [number, number, number, number],
  sharp: [0.4, 0, 0.6, 1] as [number, number, number, number],
} as const;

export const pageVariants: Variants = {
  initial: { opacity: 0, y: 12 },
  animate: { opacity: 1, y: 0, transition: { duration: 0.25, ease: ease.out } },
  exit: { opacity: 0, y: -8, transition: { duration: 0.15, ease: ease.sharp } },
};

export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 16 },
  visible: { opacity: 1, y: 0, transition: { ...spring.smooth } },
  exit: { opacity: 0, y: -8, transition: { duration: 0.15, ease: ease.sharp } },
};

export const fadeIn: Variants = {
  hidden: { opacity: 0 },
  visible: { opacity: 1, transition: { duration: 0.25, ease: ease.out } },
  exit: { opacity: 0, transition: { duration: 0.15 } },
};
