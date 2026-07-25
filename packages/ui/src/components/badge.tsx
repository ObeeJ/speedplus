import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

const badgeVariants = cva('inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-display font-semibold transition-colors', {
  variants: {
    variant: {
      gas:      'bg-[#E9F3D8] text-[#0A3D2C]',
      grocery:  'bg-[#E9F3D8] text-[#0A3D2C]',
      food:     'bg-[#E9F3D8] text-[#0A3D2C]',
      pharmacy: 'bg-[#E9F3D8] text-[#0A3D2C]',
      default:  'bg-[#E9F3D8] text-[#0A3D2C]',
      success:  'bg-[#E9F3D8] text-[#0A3D2C]',
      warning:  'bg-[#E8B14E]/20 text-[#0A3D2C]',
      error:    'bg-red-100 text-red-700',
    },
  },
  defaultVariants: { variant: 'default' },
});

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
