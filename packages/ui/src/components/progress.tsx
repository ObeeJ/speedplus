import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

interface ProgressProps extends HTMLAttributes<HTMLDivElement> {
  value: number; // 0–100
  variant?: 'lime' | 'emerald';
}

export function Progress({ value, variant = 'lime', className, ...props }: ProgressProps) {
  return (
    <div
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={100}
      className={cn('h-1.5 w-full rounded-full bg-[#E4E0D6] overflow-hidden', className)}
      {...props}
    >
      <div
        className={cn(
          'h-full rounded-full transition-all duration-500 ease-out',
          variant === 'lime' ? 'bg-[#C6F24E]' : 'bg-[#0A3D2C]',
        )}
        style={{ width: `${Math.min(100, Math.max(0, value))}%` }}
      />
    </div>
  );
}
