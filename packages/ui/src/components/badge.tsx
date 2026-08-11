import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

const badgeVariants = cva('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-display font-semibold transition-colors', {
  variants: {
    variant: {
      gas:      'bg-tile text-emerald',
      grocery:  'bg-tile text-emerald',
      food:     'bg-tile text-emerald',
      pharmacy: 'bg-tile text-emerald',
      default:  'bg-tile text-emerald',
      success:  'bg-tile text-emerald',
      warning:  'bg-amber/20 text-emerald',
      error:    'bg-red-100 text-red-700',
    },
  },
  defaultVariants: { variant: 'default' },
});

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
