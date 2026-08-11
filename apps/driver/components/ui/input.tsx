'use client';

import * as React from 'react';
import { cn } from '@/lib/utils';

const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          'flex h-[52px] w-full rounded-[var(--radius-md)] bg-white px-4 text-[14px] text-ink',
          'border border-[var(--color-line)] shadow-[var(--shadow-xs)]',
          'placeholder:text-mid/40 outline-none',
          'transition-all duration-150',
          'focus:border-[var(--color-emerald)] focus:shadow-[0_0_0_1.5px_rgba(10,61,44,0.6),0_0_0_4px_rgba(10,61,44,0.08)]',
          'disabled:cursor-not-allowed disabled:opacity-40 disabled:bg-[var(--color-sand)]',
          className
        )}
        ref={ref}
        {...props}
      />
    );
  }
);
Input.displayName = 'Input';

export { Input };
