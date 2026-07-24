'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';
import { useCreateOrder } from '../../../lib/hooks/use-order-mutations';

function naira(n: number) {
  return `₦${n.toLocaleString('en-NG')}`;
}

export default function FoodPricePage() {
  const router = useRouter();
  const { priceBreakdown, km, meal, deliverTo, setOrderId } = useFoodFlowStore();
  const { base, distance, item, total } = priceBreakdown();
  const m = meal();
  const createOrder = useCreateOrder();

  function handleConfirm() {
    createOrder.mutate(
      {
        merchantId: m?.kitchen ?? 'speedplus-food',
        vertical: 'food',
        items: [{ productId: m?.id ?? 'meal', quantity: 1 }],
        deliveryAddressId: deliverTo ?? 'demo-address',
      },
      {
        onSuccess: (order) => setOrderId(order.id),
        onSettled: () => router.push('/food/finding'),
      },
    );
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Here's the price" step={3} backHref="/food/deliver" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
          <div className="flex items-center justify-between text-[14px]">
            <span className="text-mid">Delivery ({km().toFixed(1)} km)</span>
            <span className="text-ink font-medium">{naira(distance)}</span>
          </div>
          <div className="flex items-center justify-between text-[14px]">
            <span className="text-mid">{m?.name ?? 'Meal'}</span>
            <span className="text-ink font-medium">{naira(item)}</span>
          </div>
          <div className="flex items-center justify-between text-[14px]">
            <span className="text-mid">Base fare</span>
            <span className="text-ink font-medium">{naira(base)}</span>
          </div>
          <div className="h-px bg-line my-1" />
          <div className="flex items-center justify-between">
            <span className="font-display font-semibold text-lg text-ink">Total</span>
            <span className="font-display font-bold text-2xl text-emerald">{naira(total)}</span>
          </div>
        </div>

        <span className="text-[13px] text-mid">
          You pay {naira(total)} when it arrives — cash or card, your choice.
        </span>

        <Button variant="primary" size="lg" isLoading={createOrder.isPending} onClick={handleConfirm} className="w-full min-[700px]:max-w-[380px]">
          Confirm order
        </Button>
        {createOrder.isError && (
          <span className="text-xs text-mid">Couldn’t reach the server just now — continuing with your order locally.</span>
        )}
      </div>
    </main>
  );
}
