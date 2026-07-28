'use client';

import { useRouter } from 'next/navigation';

interface FlowHeaderProps {
  title: string;
  step: 1 | 2 | 3;
  totalSteps?: number;
  backHref: string;
  subtitle?: string;
}

export function FlowHeader({ title, step, totalSteps = 3, backHref, subtitle }: FlowHeaderProps) {
  const router = useRouter();
  const progress = (step / totalSteps) * 100;

  return (
    <header className="sticky top-0 z-10 bg-[#0A3D2C] px-5 pt-safe-top">
      <div className="flex items-center gap-4 pt-4 pb-3">
        <button
          onClick={() => router.push(backHref)}
          className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors flex-shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/50"
          aria-label="Go back"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#F7F5EF" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
            <path d="M15 18l-6-6 6-6" />
          </svg>
        </button>

        <div className="flex-1 min-w-0">
          <p className="text-[10px] font-semibold text-white/50 tracking-[0.8px] uppercase mb-0.5">
            Step {step} of {totalSteps}
          </p>
          <h1 className="font-display font-semibold text-[18px] text-white leading-tight truncate">
            {title}
          </h1>
          {subtitle && (
            <p className="text-[11px] text-white/60 mt-0.5 truncate">{subtitle}</p>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="h-0.5 bg-white/10 -mx-5 mb-0">
        <div
          className="h-full bg-[#C6F24E] transition-all duration-500 ease-out"
          style={{ width: `${progress}%` }}
        />
      </div>
    </header>
  );
}
