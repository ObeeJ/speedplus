import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

interface AvatarProps extends HTMLAttributes<HTMLSpanElement> {
  initials: string;
  size?: 'sm' | 'md' | 'lg';
  variant?: 'emerald' | 'sand' | 'neutral';
}

const sizeMap = { sm: 'w-8 h-8 text-xs', md: 'w-11 h-11 text-sm', lg: 'w-14 h-14 text-base' };
const variantMap = {
  emerald: 'bg-[#0A3D2C] text-[#C6F24E]',
  sand:    'bg-[#E9F3D8] text-[#0A3D2C]',
  neutral: 'bg-[#E4E0D6] text-[#63636E]',
};

export function Avatar({ initials, size = 'md', variant = 'emerald', className, ...props }: AvatarProps) {
  return (
    <span
      className={cn(
        'rounded-full flex items-center justify-center font-display font-semibold flex-shrink-0 select-none',
        sizeMap[size],
        variantMap[variant],
        className,
      )}
      aria-label={initials}
      {...props}
    >
      {initials.slice(0, 2).toUpperCase()}
    </span>
  );
}
