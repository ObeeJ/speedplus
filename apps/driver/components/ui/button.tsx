'use client';

import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap font-display font-bold tracking-[-0.01em] transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-40 select-none cursor-pointer active:scale-[0.97]',
  {
    variants: {
      variant: {
        primary: 'bg-[var(--color-lime)] text-[var(--color-emerald)] hover:bg-[var(--color-lime-600)] shadow-[0_1px_3px_rgba(0,0,0,0.12),0_4px_12px_rgba(198,242,78,0.25)] hover:shadow-[0_2px_8px_rgba(0,0,0,0.14),0_6px_20px_rgba(198,242,78,0.35)] hover:-translate-y-0.5 focus-visible:ring-[var(--color-emerald)]',
        secondary: 'bg-[var(--color-emerald)] text-[var(--color-sand)] hover:bg-[var(--color-emerald-600)] focus-visible:ring-[var(--color-emerald)]',
        outline: 'border border-[var(--color-line)] bg-transparent text-ink hover:bg-[var(--color-sand)]',
        ghost: 'bg-transparent text-mid hover:bg-[var(--color-sand)]',
      },
      size: {
        sm: 'h-9 px-4 text-[13px] rounded-[var(--radius-sm)]',
        md: 'h-11 px-5 text-[14px] rounded-[var(--radius-md)]',
        lg: 'h-12 px-6 text-[15px] rounded-[var(--radius-md)]',
        xl: 'h-14 px-8 text-[16px] rounded-[var(--radius-lg)]',
      },
    },
    defaultVariants: {
      variant: 'primary',
      size: 'lg',
    },
  }
);

interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  isLoading?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, isLoading, children, disabled, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        disabled={disabled || isLoading}
        {...props}
      >
        {isLoading ? (
          <>
            <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
            </svg>
            {children}
          </>
        ) : children}
      </Comp>
    );
  }
);
Button.displayName = 'Button';

export { Button, buttonVariants };
