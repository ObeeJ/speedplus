'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { useFoodFlowStore, MEALS } from '../../../lib/store/food-flow.store';

function naira(n: number) {
  return `₦${n.toLocaleString('en-NG')}`;
}

export default function FoodMenuPage() {
  const router = useRouter();
  const { mealId, setMealId } = useFoodFlowStore();
  const canContinue = Boolean(mealId);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="What are you craving?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-4 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <div className="flex flex-col gap-2">
          {MEALS.map((m) => {
            const selected = mealId === m.id;
            return (
              <button
                key={m.id}
                onClick={() => setMealId(m.id)}
                className={`w-full flex items-center justify-between gap-3 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
                  selected ? 'bg-emerald border-lime' : 'bg-white border-line hover:border-emerald/40'
                }`}
              >
                <span className="flex flex-col gap-0.5">
                  <span className={`font-display font-semibold text-[15px] ${selected ? 'text-lime' : 'text-ink'}`}>
                    {selected ? `✓ ${m.name}` : m.name}
                  </span>
                  <span className={`text-[12.5px] ${selected ? 'text-sand/70' : 'text-mid'}`}>
                    {m.kitchen} · {m.prepTime}
                  </span>
                </span>
                <span className={`font-display font-semibold text-[14px] ${selected ? 'text-lime' : 'text-ink'}`}>{naira(m.price)}</span>
              </button>
            );
          })}
        </div>

        {canContinue && (
          <span className="text-[13px] text-mid">
            ✓ <b className="text-emerald">{MEALS.find((m) => m.id === mealId)!.name}</b> from{' '}
            <b className="text-emerald">{MEALS.find((m) => m.id === mealId)!.kitchen}</b>. Next: where do we bring it?
          </span>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!canContinue}
          onClick={() => router.push('/food/deliver')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
