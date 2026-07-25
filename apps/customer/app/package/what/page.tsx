'use client';

import { useRouter } from 'next/navigation';
import { Button, SelectionCard } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { usePackageFlowStore, type PackageSize, type PackageWeight } from '../../../lib/store/package-flow.store';

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">{children}</p>;
}

const SIZES: { id: PackageSize; label: string; description: string; icon: React.ReactNode }[] = [
  {
    id: 'small',
    label: 'Small',
    description: 'Fits in one hand — envelope, phone, small box',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
        <rect x="8" y="8" width="8" height="8" rx="1" />
        <path d="M3 3h4v4H3zM17 3h4v4h-4zM3 17h4v4H3zM17 17h4v4h-4z" opacity="0.3" />
      </svg>
    ),
  },
  {
    id: 'medium',
    label: 'Medium',
    description: 'Shoebox or bag — needs two hands to carry',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
        <rect x="5" y="5" width="14" height="14" rx="1" />
      </svg>
    ),
  },
  {
    id: 'large',
    label: 'Large',
    description: 'Suitcase or crate — bulky, needs a van',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
        <rect x="2" y="3" width="20" height="18" rx="1" />
        <line x1="2" y1="9" x2="22" y2="9" />
        <line x1="12" y1="9" x2="12" y2="21" />
      </svg>
    ),
  },
];

const WEIGHTS: { id: PackageWeight; label: string; description: string; kg: string }[] = [
  { id: 'light',     label: 'Light',      description: 'Under 3 kg',   kg: '< 3 kg' },
  { id: 'medium',    label: 'Medium',     description: '3 – 10 kg',    kg: '3–10 kg' },
  { id: 'heavy',     label: 'Heavy',      description: '10 – 25 kg',   kg: '10–25 kg' },
  { id: 'very_heavy',label: 'Very heavy', description: 'Over 25 kg',   kg: '25+ kg' },
];

export default function PackageWhatPage() {
  const router = useRouter();
  const { size, weight, setSize, setWeight } = usePackageFlowStore();
  const canContinue = Boolean(size && weight);

  const selectedSize = SIZES.find((s) => s.id === size);
  const selectedWeight = WEIGHTS.find((w) => w.id === weight);

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <FlowHeader title="What are we moving?" step={2} backHref="/package/where" />

      <div className="flex-1 px-5 py-6 flex flex-col gap-7 max-w-[600px] mx-auto w-full pb-32">

        {/* Size */}
        <section className="flex flex-col gap-3">
          <SectionLabel>Package size</SectionLabel>
          <div className="flex flex-col gap-2">
            {SIZES.map((s) => (
              <SelectionCard
                key={s.id}
                selected={size === s.id}
                label={s.label}
                description={s.description}
                icon={s.icon}
                onClick={() => setSize(s.id)}
              />
            ))}
          </div>
        </section>

        {/* Weight */}
        <section className="flex flex-col gap-3">
          <SectionLabel>Approximate weight</SectionLabel>
          <div className="grid grid-cols-2 gap-2">
            {WEIGHTS.map((w) => (
              <button
                key={w.id}
                type="button"
                onClick={() => setWeight(w.id)}
                className={`text-left rounded-2xl border-2 px-4 py-3.5 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#C6F24E] active:scale-[0.99] ${
                  weight === w.id
                    ? 'border-[#0A3D2C] bg-[#E9F3D8] shadow-[0_0_0_1px_#0A3D2C]'
                    : 'border-[#E4E0D6] bg-white hover:border-[#0A3D2C]/30 hover:bg-[#F7F5EF]'
                }`}
              >
                <p className={`font-display font-semibold text-[14px] ${weight === w.id ? 'text-[#0A3D2C]' : 'text-[#121216]'}`}>
                  {w.label}
                </p>
                <p className={`text-[11px] mt-0.5 ${weight === w.id ? 'text-[#0A3D2C]/70' : 'text-[#63636E]'}`}>
                  {w.kg}
                </p>
              </button>
            ))}
          </div>
        </section>

        {/* Summary */}
        {canContinue && selectedSize && selectedWeight && (
          <div
            className="bg-[#0A3D2C] rounded-2xl px-5 py-4 flex items-center gap-3"
            style={{ animation: 'fadeUp 0.25s cubic-bezier(0.16,1,0.3,1) both' }}
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <p className="text-[13px] text-white/90">
              <span className="font-semibold text-white">{selectedSize.label}</span> package,{' '}
              <span className="font-semibold text-white">{selectedWeight.label.toLowerCase()}</span> ({selectedWeight.kg})
            </p>
          </div>
        )}
      </div>

      {/* Sticky CTA */}
      <div className="fixed bottom-0 left-0 right-0 bg-[#F7F5EF]/95 backdrop-blur-sm border-t border-[#E4E0D6] px-5 py-4 pb-safe-bottom">
        <div className="max-w-[600px] mx-auto flex items-center justify-between gap-4">
          <p className="text-[11px] text-[#9A968D]">Price shown on next step</p>
          <Button
            variant="primary"
            size="lg"
            disabled={!canContinue}
            onClick={() => router.push('/package/price')}
            className="flex-shrink-0"
          >
            See price
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </Button>
        </div>
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(8px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
