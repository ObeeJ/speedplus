'use client';

import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import { type ButtonHTMLAttributes, forwardRef } from 'react';
import { motion } from 'framer-motion';

const buttonVariants = cva(
  [
    'relative inline-flex items-center justify-center gap-2 whitespace-nowrap',
    'font-display font-bold tracking-[-0.01em]',
    'transition-colors duration-150',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
    'disabled:pointer-events-none disabled:opacity-40',
    'select-none cursor-pointer',
  ].join(' '),
  {
    variants: {
      variant: {
        primary: [
          'bg-lime text-emerald',
          'hover:bg-lime-600',
          'active:bg-lime-700',
          'focus-visible:ring-emerald',
          'shadow-[0_1px_3px_rgba(0,0,0,0.12),0_4px_12px_rgba(198,242,78,0.25)]',
          'hover:shadow-[0_2px_8px_rgba(0,0,0,0.14),0_6px_20px_rgba(198,242,78,0.35)]',
        ].join(' '),
        secondary: [
          'bg-emerald text-sand',
          'hover:bg-emerald-600',
          'active:bg-emerald-900',
          'focus-visible:ring-emerald',
          'shadow-[0_1px_3px_rgba(0,0,0,0.15),0_4px_12px_rgba(10,61,44,0.20)]',
          'hover:shadow-[0_2px_8px_rgba(0,0,0,0.18),0_6px_20px_rgba(10,61,44,0.28)]',
        ].join(' '),
        outline: [
          'border border-line bg-transparent text-ink',
          'hover:bg-sand hover:border-mid/40',
          'focus-visible:ring-emerald',
          'shadow-[0_1px_2px_rgba(0,0,0,0.04)]',
        ].join(' '),
        ghost: [
          'bg-transparent text-mid',
          'hover:bg-sand hover:text-ink',
          'focus-visible:ring-emerald',
        ].join(' '),
        danger: [
          'bg-red-600 text-white',
          'hover:bg-red-700',
          'focus-visible:ring-red-500',
          'shadow-[0_1px_3px_rgba(0,0,0,0.12),0_4px_12px_rgba(220,38,38,0.20)]',
        ].join(' '),
      },
      size: {
        sm:   'h-9  px-4  text-[13px] rounded-[var(--radius-sm)]',
        md:   'h-11 px-5  text-[14px] rounded-[var(--radius-md)]',
        lg:   'h-12 px-6  text-[15px] rounded-[var(--radius-md)]',
        xl:   'h-14 px-8  text-[16px] rounded-[var(--radius-lg)]',
        icon: 'h-10 w-10 text-[14px] rounded-[var(--radius-md)]',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
);

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  isLoading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, isLoading, children, disabled, ...props }, ref) => (
    <motion.button
      ref={ref}
      className={cn(buttonVariants({ variant, size }), className)}
      disabled={disabled ?? isLoading}
      whileHover={{ y: -1.5, scale: 1.005 }}
      whileTap={{ scale: 0.97, y: 0 }}
      transition={{ type: 'spring', stiffness: 500, damping: 30 }}
      {...(props as React.ComponentProps<typeof motion.button>)}
    >
      {isLoading ? (
        <motion.span
          className="flex items-center gap-2"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.15 }}
        >
          <svg
            className="animate-spin h-4 w-4 flex-shrink-0"
            viewBox="0 0 24 24"
            fill="none"
            aria-hidden="true"
          >
            <circle className="opacity-20" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
            <path className="opacity-80" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          <span>{children}</span>
        </motion.span>
      ) : (
        children
      )}
    </motion.button>
  ),
);
Button.displayName = 'Button';

export { buttonVariants };
