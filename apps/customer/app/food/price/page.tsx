'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Skeleton } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';
import { useRequestQuote, useCreateOrder } from '../../../lib/hooks/use-order-mutations';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function FoodPricePage() {
  const router = useRouter();
  const {
    merchantId, merchantLat, merchantLng,
    productId, productPriceKobo,
    deliverToId, deliverToAddress,
    quote, setQuote, setOrderId,
  } = useFoodFlowStore();

  const requestQuote = useRequestQuote();
  const createOrder = useCreateOrder();

  const canQuote = Boolean(merchantId && deliverToId && deliverToAddress && productId);

  useEffect(() => {
    if (!canQuote || quote) return;
    requestQuote.mutate(
      {
        merchantId: merchantId!,
        vertical: 'food',
        subtotalKobo: productPriceKobo ?? 0,
        originLat: merchantLat!,
        originLng: merchantLng!,
        destLat: deliverToAddress!.lat,
        destLng: deliverToAddress!.lng,
      },
      { onSuccess: setQuote },
    );
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [merchantId, deliverToId, productId]);

  if (!canQuote) {
    router.replace('/food/deliver');
    return null;
  }

  function handleConfirm() {
    if (!quote || !merchantId || !productId || !deliverToId) return;
    createOrder.mutate(
      {
        payload: {
          merchantId,
          quoteId: quote.id,
          vertical: 'food',
          items: [{ productId, quantity: 1 }],
          deliveryAddressId: deliverToId,
          paymentMethod: 'wallet',
        },
        idempotencyKey: crypto.randomUUID(),
      },
      {
        onSuccess: (order) => {
          setOrderId(order.id);
          router.push('/food/finding');
        },
      },
    );
  }

  const loading = requestQuote.isPending;
  const quoteError = requestQuote.isError ? (requestQuote.error as Error).message : null;
  const orderError = createOrder.isError ? (createOrder.error as Error).message : null;

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Here's the price" step={4} totalSteps={4} backHref="/food/deliver" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        {loading && (
          <div className="flex flex-col gap-3">
            <Skeleton className="h-[120px] rounded-2xl" />
          </div>
        )}

        {quoteError && (
          <span className="text-xs text-red-600" role="alert">
            Couldn&apos;t get a price: {quoteError}
          </span>
        )}

        {quote && (
          <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
            <div className="flex items-center justify-between text-[14px]">
              <span className="text-mid">Meal</span>
              <span className="text-ink font-medium">{naira(quote.subtotalKobo)}</span>
            </div>
            <div className="flex items-center justify-between text-[14px]">
              <span className="text-mid">Delivery ({quote.distanceKm.toFixed(1)} km)</span>
              <span className="text-ink font-medium">{naira(quote.deliveryKobo)}</span>
            </div>
            {quote.serviceKobo > 0 && (
              <div className="flex items-center justify-between text-[14px]">
                <span className="text-mid">Service fee</span>
                <span className="text-ink font-medium">{naira(quote.serviceKobo)}</span>
              </div>
            )}
            <div className="h-px bg-line my-1" />
            <div className="flex items-center justify-between">
              <span className="font-display font-semibold text-lg text-ink">Total</span>
              <span className="font-display font-bold text-2xl text-emerald">{naira(quote.totalKobo)}</span>
            </div>
          </div>
        )}

        <span className="text-[13px] text-mid">
          ₦{quote ? (quote.totalKobo / 100).toLocaleString('en-NG') : '—'} will be deducted from your wallet.
        </span>

        {orderError && (
          <span className="text-xs text-red-600" role="alert">{orderError}</span>
        )}

        <Button
          variant="primary"
          size="lg"
          disabled={!quote || createOrder.isPending}
          isLoading={createOrder.isPending}
          onClick={handleConfirm}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Confirm order
        </Button>
      </div>
    </main>
  );
}
