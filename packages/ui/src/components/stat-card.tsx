import { cn } from '../lib/utils';
import type { ReactNode } from 'react';

export interface StatCardProps {
  label: string;
  value: string;
  sub?: string;
  icon?: ReactNode;
  className?: string;
}

export function StatCard({ label, value, sub, icon, className }: StatCardProps) {
  return (
    <div className={cn('bg-white border border-line rounded-2xl p-5 flex flex-col gap-3', className)}>
      <div className="flex items-center justify-between">
        <span className="text-[10.5px] font-semibold text-mid tracking-[0.6px] uppercase">{label}</span>
        {icon && (
          <div className="w-8 h-8 rounded-[9px] flex items-center justify-center bg-tile">
            {icon}
          </div>
        )}
      </div>
      <div>
        <span className="font-display font-bold text-[28px] text-ink leading-none">{value}</span>
        {sub && <p className="text-[11px] text-mid mt-1">{sub}</p>}
      </div>
    </div>
  );
}

export function StatCardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn('bg-white border border-line rounded-2xl p-5 h-[108px] animate-pulse', className)} />
  );
}
