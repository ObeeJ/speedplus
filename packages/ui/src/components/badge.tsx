import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

const badgeVariants = cva('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors', {
  variants: {
    variant: {
      gas:      'bg-[#E0FBF4] text-[#008F66]',
      grocery:  'bg-green-100 text-green-700',
      food:     'bg-amber-100 text-amber-700',
      pharmacy: 'bg-teal-100 text-teal-700',
      default:  'bg-[#E0FBF4] text-[#008F66]',
      success:  'bg-green-100 text-green-700',
      warning:  'bg-amber-100 text-amber-800',
      error:    'bg-red-100 text-red-700',
    },
  },
  defaultVariants: { variant: 'default' },
});

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
