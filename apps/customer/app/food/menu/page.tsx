'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { catalogApi } from '@speedplus/api-client';
import { Skeleton } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';

export default function FoodMerchantsPage() {
  const router = useRouter();
  const setMerchant = useFoodFlowStore((s) => s.setMerchant);

  const { data, isLoading, isError } = useQuery({
    queryKey: ['food-merchants'],
    queryFn: () => catalogApi.listMerchants('food'),
  });

  function choose(id: string, lat: number, lng: number) {
    setMerchant(id, lat, lng);
    router.push('/food/items');
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="What are you craving?" step={1} totalSteps={4} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-3 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        {isLoading && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-[62px] rounded-[13px]" />
            <Skeleton className="h-[62px] rounded-[13px]" />
            <Skeleton className="h-[62px] rounded-[13px]" />
          </div>
        )}

        {isError && (
          <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
            Couldn&apos;t load kitchens right now. Please try again.
          </div>
        )}

        {data?.merchants.length === 0 && (
          <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
            No kitchens are open in your area right now.
          </div>
        )}

        {data?.merchants.map((m) => (
          <button
            key={m.id}
            data-testid="merchant-card"
            onClick={() => choose(m.id, m.lat, m.lng)}
            disabled={!m.isOpen}
            className={`w-full flex items-center justify-between gap-3 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
              m.isOpen
                ? 'bg-white border-line hover:border-emerald/40'
                : 'bg-white/50 border-line opacity-50 cursor-not-allowed'
            }`}
          >
            <span className="flex flex-col gap-0.5">
              <span className="font-display font-semibold text-[15px] text-ink">{m.businessName}</span>
              <span className="text-[12.5px] text-mid">
                {m.isOpen ? `★ ${m.rating.toFixed(1)}` : 'Closed'}
              </span>
            </span>
          </button>
        ))}
      </div>
    </main>
  );
}
