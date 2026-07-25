'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { SelectTile } from '../../components/select-tile';
import { usePackageFlowStore } from '../../../lib/store/package-flow.store';

const PICKUP_SHORTCUTS = [
  { label: '🏠 My home', value: 'Home — 14 Admiralty Way, Lekki Phase 1' },
  { label: '📍 Where I am now', value: 'Current location' },
];

const DROPOFF_SHORTCUTS = [
  { label: 'Yaba', value: 'Yaba, Lagos', km: '11.2 km' },
  { label: 'Surulere', value: 'Surulere, Lagos', km: '8.6 km' },
  { label: 'Ikeja', value: 'Ikeja, Lagos', km: '14.9 km' },
];

export default function PackageWherePage() {
  const router = useRouter();
  const { pickup, dropoff, setPickup, setDropoff } = usePackageFlowStore();
  const canContinue = Boolean(pickup && dropoff);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Where's it going?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Pickup</span>
          <div className="flex flex-col gap-2">
            {PICKUP_SHORTCUTS.map((s) => (
              <SelectTile key={s.value} label={s.label} selected={pickup === s.value} onClick={() => setPickup(s.value)} />
            ))}
          </div>
        </section>

        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Drop-off</span>
          <div className="flex flex-col gap-2">
            {DROPOFF_SHORTCUTS.map((s) => (
              <SelectTile
                key={s.value}
                label={s.label}
                description={`${s.km} away`}
                selected={dropoff === s.value}
                onClick={() => setDropoff(s.value)}
              />
            ))}
          </div>
        </section>

        {canContinue && (
          <span className="text-[13px] text-mid">
            ✓ Picking up from <b className="text-emerald">{pickup}</b>, dropping off at <b className="text-emerald">{dropoff}</b>. Next: what's the package?
          </span>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!canContinue}
          onClick={() => router.push('/package/what')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
