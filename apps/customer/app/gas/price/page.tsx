'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { useGasFlowStore, CYLINDER_KG } from '../../../lib/store/gas-flow.store';
import { useRequestQuote, useCreateOrder } from '../../../lib/hooks/use-order-mutations';

// Deterministic UUIDs seeded in migration 022.
const GAS_MERCHANT_ID = '00000000-0000-0000-0000-000000000004';
const CYLINDER_PRODUCT_ID: Record<string, string> = {
  '3':    '00000000-0000-0000-0000-000000000011',
  '6':    '00000000-0000-0000-0000-000000000012',
  '12.5': '00000000-0000-0000-0000-000000000013',
  '25':   '00000000-0000-0000-0000-000000000014',
};
// Gas merchant location (migration 022 seed).
const MERCHANT_LAT = 6.5244;
const MERCHANT_LNG = 3.3792;

// Cylinder price in kobo (placeholder; Phase 5 derives from LPG index).
const CYLINDER_PRICE_KOBO: Record<string, number> = {
  '3': 320000, '6': 610000, '12.5': 1180000, '25': 2300000,
};

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function GasPricePage() {
  const router = useRouter();
  const { cylinder, mode, deliverToId, deliverToAddress, quote, setQuote, setOrderId } = useGasFlowStore();
  const requestQuote = useRequestQuote();
  const createOrder = useCreateOrder();

  useEffect(() => {
    if (!cylinder || !deliverToId || quote) return;
    const destLat = deliverToAddress?.lat ?? MERCHANT_LAT;
    const destLng = deliverToAddress?.lng ?? MERCHANT_LNG;
    requestQuote.mutate(
      {
        merchantId: GAS_MERCHANT_ID,
        vertical: 'gas',
        subtotalKobo: CYLINDER_PRICE_KOBO[cylinder] ?? 0,
        originLat: MERCHANT_LAT,
        originLng: MERCHANT_LNG,
        destLat,
        destLng,
        weightKg: CYLINDER_KG[cylinder],
      },
      { onSuccess: setQuote },
    );
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cylinder, deliverToId]);

  function handleConfirm() {
    if (!quote || !cylinder || !deliverToId) return;
    createOrder.mutate(
      {
        merchantId: GAS_MERCHANT_ID,
        quoteId: quote.id,
        vertical: 'gas',
        gasMode: mode ?? 'swap',
        items: [{ productId: CYLINDER_PRODUCT_ID[cylinder], quantity: 1 }],
        deliveryAddressId: deliverToId,
        paymentMethod: 'wallet',
      },
      {
        onSuccess: (order) => {
          setOrderId(order.id);
          router.push('/gas/finding');
        },
      },
    );
  }

  const loading = requestQuote.isPending;
  const quoteError = requestQuote.isError ? (requestQuote.error as Error).message : null;
  const orderError = createOrder.isError ? (createOrder.error as Error).message : null;

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Here's the price" step={3} backHref="/gas/deliver" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        {loading && <span className="text-[13px] text-mid">Getting your price…</span>}

        {quoteError && (
          <span className="text-xs text-red-600" role="alert">Couldn't get a price: {quoteError}</span>
        )}

        {quote && (
          <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
            <div className="flex items-center justify-between text-[14px]">
              <span className="text-mid">Cylinder</span>
              <span className="text-ink font-medium">{naira(quote.subtotalKobo)}</span>
            </div>
            <div className="flex items-center justify-between text-[14px]">
              <span className="text-mid">Delivery ({quote.distanceKm.toFixed(1)} km)</span>
              <span className="text-ink font-medium">{naira(quote.deliveryKobo)}</span>
            </div>
            <div className="flex items-center justify-between text-[14px]">
              <span className="text-mid">Service fee</span>
              <span className="text-ink font-medium">{naira(quote.serviceKobo)}</span>
            </div>
            {quote.weatherAdvisory && (
              <div className="flex items-center justify-between text-[13px]">
                <span className="text-mid italic">{quote.weatherAdvisory}</span>
              </div>
            )}
            <div className="h-px bg-line my-1" />
            <div className="flex items-center justify-between">
              <span className="font-display font-semibold text-lg text-ink">Total</span>
              <span className="font-display font-bold text-2xl text-emerald">{naira(quote.totalKobo)}</span>
            </div>
          </div>
        )}

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
          Confirm — find a rider
        </Button>
      </div>
    </main>
  );
}
