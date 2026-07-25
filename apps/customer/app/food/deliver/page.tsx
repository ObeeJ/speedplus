'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { SelectTile } from '../../components/select-tile';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';

const DELIVER_SHORTCUTS = [
  { label: '🏠 My home', value: 'Home — 14 Admiralty Way, Lekki Phase 1' },
  { label: '📍 Where I am now', value: 'Current location' },
  { label: 'Office', value: 'Office — 22 Adeola Odeku, Victoria Island' },
];

export default function FoodDeliverPage() {
  const router = useRouter();
  const { deliverTo, setDeliverTo } = useFoodFlowStore();
  const canContinue = Boolean(deliverTo);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Where should we bring it?" step={2} backHref="/food/menu" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Delivery address</span>
          <div className="flex flex-col gap-2">
            {DELIVER_SHORTCUTS.map((s) => (
              <SelectTile key={s.value} label={s.label} selected={deliverTo === s.value} onClick={() => setDeliverTo(s.value)} />
            ))}
          </div>
        </section>

        {canContinue && (
          <span className="text-[13px] text-mid">
            ✓ Delivering to <b className="text-emerald">{deliverTo}</b>. Next: see the price.
          </span>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!canContinue}
          onClick={() => router.push('/food/price')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
