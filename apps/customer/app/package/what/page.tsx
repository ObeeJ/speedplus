'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { SelectTile } from '../../components/select-tile';
import { usePackageFlowStore, type PackageSize, type PackageWeight } from '../../../lib/store/package-flow.store';

const SIZES: { id: PackageSize; label: string; description: string }[] = [
  { id: 'small', label: '✉️ Small', description: 'Fits one hand' },
  { id: 'medium', label: '📦 Medium', description: 'A shoe box or bag' },
  { id: 'large', label: '🧺 Large', description: 'Needs two hands' },
];

const WEIGHTS: { id: PackageWeight; label: string; description: string }[] = [
  { id: 'light', label: 'Light', description: 'Under 3kg' },
  { id: 'medium', label: 'Medium', description: '3–10kg' },
  { id: 'heavy', label: 'Heavy', description: '10–25kg' },
  { id: 'very_heavy', label: 'Very heavy', description: 'Over 25kg' },
];

export default function PackageWhatPage() {
  const router = useRouter();
  const { size, weight, setSize, setWeight } = usePackageFlowStore();
  const canContinue = Boolean(size && weight);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="What are we moving?" step={2} backHref="/package/where" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Size</span>
          <div className="grid grid-cols-1 min-[700px]:grid-cols-3 gap-2">
            {SIZES.map((s) => (
              <SelectTile key={s.id} label={s.label} description={s.description} selected={size === s.id} onClick={() => setSize(s.id)} />
            ))}
          </div>
        </section>

        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Weight</span>
          <div className="grid grid-cols-1 min-[700px]:grid-cols-2 gap-2">
            {WEIGHTS.map((w) => (
              <SelectTile key={w.id} label={w.label} description={w.description} selected={weight === w.id} onClick={() => setWeight(w.id)} />
            ))}
          </div>
        </section>

        {canContinue && (
          <span className="text-[13px] text-mid">
            ✓ A <b className="text-emerald">{SIZES.find((s) => s.id === size)!.label.replace(/^\S+\s/, '')}</b>,{' '}
            <b className="text-emerald">{WEIGHTS.find((w) => w.id === weight)!.label}</b> package. Next: see the price.
          </span>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!canContinue}
          onClick={() => router.push('/package/price')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
