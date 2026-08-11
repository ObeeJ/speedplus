'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Skeleton } from '@speedplus/ui';
import { usersApi, type SavedAddress } from '@speedplus/api-client';
import { FlowHeader } from '../../components/flow-header';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';

export default function FoodDeliverPage() {
  const router = useRouter();
  const { deliverToId, setDeliverTo } = useFoodFlowStore();
  const [addresses, setAddresses] = useState<SavedAddress[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    usersApi.listAddresses()
      .then(setAddresses)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const selected = addresses.find((a) => a.id === deliverToId);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Where should we bring it?" step={3} totalSteps={4} backHref="/food/items" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Delivery address</span>
          {loading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-[58px] rounded-2xl" />
              <Skeleton className="h-[58px] rounded-2xl" />
            </div>
          ) : addresses.length === 0 ? (
            <span className="text-[13px] text-mid">No saved addresses. Add one in your profile first.</span>
          ) : (
            <div className="flex flex-col gap-2">
              {addresses.map((a) => (
                <button
                  key={a.id}
                  type="button"
                  onClick={() => setDeliverTo(a)}
                  className={`w-full text-left rounded-2xl border-2 px-4 py-3.5 transition-all ${
                    deliverToId === a.id ? 'border-emerald bg-emerald/10' : 'border-line bg-white hover:border-emerald/40'
                  }`}
                >
                  <p className="text-[13px] font-semibold text-ink">{a.label || a.street}</p>
                  <p className="text-[11px] text-mid">{a.street}, {a.city}</p>
                </button>
              ))}
            </div>
          )}
        </section>

        {selected && (
          <span className="text-[13px] text-mid">
            ✓ Delivering to <b className="text-emerald">{selected.label || selected.street}</b>. Next: see the price.
          </span>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!deliverToId}
          onClick={() => router.push('/food/price')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
