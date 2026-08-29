'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Button, Skeleton } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { useGasFlowStore, CYLINDER_KG } from '../../../lib/store/gas-flow.store';
import { useRequestQuote, useCreateOrder } from '../../../lib/hooks/use-order-mutations';
import { gasApi } from '@speedplus/api-client';

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

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function GasPricePage() {
  const router = useRouter();
  const { cylinder, mode, deliverToId, deliverToAddress, quote, setQuote, setOrderId } = useGasFlowStore();
  const requestQuote = useRequestQuote();
  const createOrder = useCreateOrder();
  // Stable per mount — regenerated only when the component unmounts and remounts
  // (i.e. the user navigates away and back, starting a fresh checkout attempt).
  // Calling crypto.randomUUID() inline in handleConfirm would generate a new key
  // on every button press, defeating idempotency on slow-connection double-taps.
  const [idempotencyKey] = useState(() => crypto.randomUUID());

  // Live LPG price index — subtotal must come from the backend, not hard-coded constants.
  const { data: lpgPrice, isLoading: lpgLoading, isError: lpgError } = useQuery({
    queryKey: ['lpg-price-index'],
    queryFn: () => gasApi.getPriceIndex('Lagos'),
    staleTime: 5 * 60 * 1000, // 5 min — price index changes infrequently
  });

  useEffect(() => {
    if (!cylinder || !deliverToId || quote || !lpgPrice) return;
    const weightKg = CYLINDER_KG[cylinder];
    const subtotalKobo = Math.round(lpgPrice.pricePerKgKobo * weightKg);
    const destLat = deliverToAddress?.lat ?? MERCHANT_LAT;
    const destLng = deliverToAddress?.lng ?? MERCHANT_LNG;
    requestQuote.mutate(
      {
        merchantId: GAS_MERCHANT_ID,
        vertical: 'gas',
        subtotalKobo,
        originLat: MERCHANT_LAT,
        originLng: MERCHANT_LNG,
        destLat,
        destLng,
        weightKg,
      },
      { onSuccess: setQuote },
    );
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cylinder, deliverToId, lpgPrice]);

  function handleConfirm() {
    if (!quote || !cylinder || !deliverToId) return;
    createOrder.mutate(
      {
        payload: {
          merchantId: GAS_MERCHANT_ID,
          quoteId: quote.id,
          vertical: 'gas',
          gasMode: mode ?? 'swap',
          items: [{ productId: CYLINDER_PRODUCT_ID[cylinder], quantity: 1 }],
          deliveryAddressId: deliverToId,
          paymentMethod: 'wallet',
        },
        idempotencyKey,
      },
      {
        onSuccess: (order) => {
          setOrderId(order.id);
          router.push('/gas/finding');
        },
      },
    );
  }

  const loading = lpgLoading || requestQuote.isPending;
  const quoteError = lpgError
    ? 'Could not load current gas prices. Please try again.'
    : requestQuote.isError ? (requestQuote.error as Error).message : null;
  const orderError = createOrder.isError ? (createOrder.error as Error).message : null;

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Here's the price" step={3} backHref="/gas/deliver" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        {loading && (
          <div className="flex flex-col gap-3">
            <Skeleton className="h-[120px] rounded-2xl" />
          </div>
        )}

        {quoteError && (
          <span className="text-xs text-red-600" role="alert">Couldn&apos;t get a price: {quoteError}</span>
        )}

        {quote && (
          <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
            <div className="flex items-center justify-between text-[14px]">
              <span className="text-mid">Cylinder ({cylinder} kg)</span>
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
            {quote.weatherSurchargeKobo > 0 && (
              <div className="flex items-center justify-between text-[14px]">
                <span className="text-mid">Weather surcharge</span>
                <span className="text-ink font-medium">{naira(quote.weatherSurchargeKobo)}</span>
              </div>
            )}
            {quote.weatherAdvisory && (
              <div className="flex items-center gap-2 bg-[#FFF7E6] border border-[#F0DFB4] rounded-xl px-3 py-2.5 text-[12px] text-[#8A6A1B]">
                <span>⚠️</span>
                <span>{quote.weatherAdvisory}</span>
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
