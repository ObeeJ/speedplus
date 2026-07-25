'use client';

import { useRouter } from 'next/navigation';

function BackIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M15 18l-6-6 6-6" />
    </svg>
  );
}

export function FlowHeader({ title, step, backHref }: { title: string; step: 1 | 2 | 3; backHref: string }) {
  const router = useRouter();
  const pips = [1, 2, 3];

  return (
    <>
      {/* Mobile: emerald header */}
      <div className="min-[700px]:hidden bg-emerald px-5 pt-[18px] pb-5 flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <button
            onClick={() => router.push(backHref)}
            className="w-10 h-10 rounded-full bg-sand/[.12] flex items-center justify-center"
            aria-label="Back"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#F7F5EF" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 18l-6-6 6-6" />
            </svg>
          </button>
          <div className="flex gap-1.5">
            {pips.map((p) => (
              <span key={p} className={`h-[5px] w-[26px] rounded-full ${p <= step ? 'bg-lime' : 'bg-sand/20'}`} />
            ))}
          </div>
        </div>
        <span className="text-[11px] font-semibold text-sand/55">Step {step} of 3</span>
        <span className="font-display font-semibold text-2xl text-sand tracking-tight">{title}</span>
      </div>

      {/* Tablet/desktop: sand header, back chip + eyebrow + pips */}
      <div className="hidden min-[700px]:flex flex-col gap-4 w-full">
        <div className="flex items-center justify-between">
          <button
            onClick={() => router.push(backHref)}
            className="w-10 h-10 rounded-full bg-tile flex items-center justify-center hover:bg-tile/70 transition-colors"
            aria-label="Back"
          >
            <BackIcon />
          </button>
          <div className="flex gap-1.5">
            {pips.map((p) => (
              <span key={p} className={`h-[5px] w-[26px] rounded-full ${p <= step ? 'bg-emerald' : 'bg-line'}`} />
            ))}
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className="text-[11px] font-semibold text-[#9A968D] tracking-[.6px]">
            STEP {step} OF 3 · PRICE SHOWN BEFORE YOU PAY
          </span>
          <span className="font-display font-semibold text-[25px] text-ink tracking-tight">{title}</span>
        </div>
      </div>
    </>
  );
}
