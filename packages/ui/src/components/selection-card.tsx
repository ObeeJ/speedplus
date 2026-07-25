'use client';

import { cn } from '../lib/utils';
import type { ReactNode, ButtonHTMLAttributes } from 'react';

interface SelectionCardProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  selected: boolean;
  label: string;
  description?: string;
  icon?: ReactNode;
  badge?: string;
  locked?: boolean;
  lockReason?: string;
}

export function SelectionCard({
  selected,
  label,
  description,
  icon,
  badge,
  locked,
  lockReason,
  className,
  ...props
}: SelectionCardProps) {
  return (
    <button
      type="button"
      disabled={locked}
      className={cn(
        'group w-full text-left rounded-2xl border-2 px-4 py-3.5 transition-all duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-[#C6F24E]',
        selected
          ? 'border-[#0A3D2C] bg-[#E9F3D8] shadow-[0_0_0_1px_#0A3D2C]'
          : locked
          ? 'border-[#E4E0D6] bg-white/60 cursor-not-allowed opacity-70'
          : 'border-[#E4E0D6] bg-white hover:border-[#0A3D2C]/40 hover:bg-[#F7F5EF] active:scale-[0.99]',
        className,
      )}
      {...props}
    >
      <div className="flex items-start gap-3">
        {icon && (
          <span className={cn(
            'mt-0.5 flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center transition-colors',
            selected ? 'bg-[#0A3D2C]/10' : 'bg-[#F7F5EF] group-hover:bg-[#E9F3D8]',
          )}>
            {icon}
          </span>
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className={cn(
              'font-display font-semibold text-[14px] leading-snug',
              selected ? 'text-[#0A3D2C]' : 'text-[#121216]',
            )}>
              {label}
            </span>
            {badge && (
              <span className={cn(
                'text-[10px] font-bold rounded-full px-2 py-0.5 flex-shrink-0',
                selected ? 'bg-[#0A3D2C] text-[#C6F24E]' : 'bg-[#E9F3D8] text-[#0A3D2C]',
              )}>
                {badge}
              </span>
            )}
          </div>
          {description && (
            <span className={cn(
              'block text-[12px] mt-0.5 leading-relaxed',
              selected ? 'text-[#0A3D2C]/70' : 'text-[#63636E]',
            )}>
              {description}
            </span>
          )}
          {locked && lockReason && (
            <span className="block text-[11px] mt-1 text-[#9A968D]">{lockReason}</span>
          )}
        </div>
        {selected && (
          <svg className="flex-shrink-0 mt-0.5" width="18" height="18" viewBox="0 0 24 24" fill="none">
            <circle cx="12" cy="12" r="10" fill="#0A3D2C" />
            <path d="M8 12l3 3 5-5" stroke="#C6F24E" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </div>
    </button>
  );
}
