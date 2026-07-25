'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { usePackageFlowStore, type PaymentMethod } from '../../../lib/store/package-flow.store';
import { useRequestQuote, useRequestMultiStopQuote, useCreateOrder, useWalletBalance } from '../../../lib/hooks/use-order-mutations';
import { useQuery } from '@tanstack/react-query';
import { cardApi } from '@speedplus/api-client';

// Weight in kg per category — used for quote request
const WEIGHT_KG: Record<string, number> = {
  light: 1.5,
  medium: 6,
  heavy: 17,
  very_heavy: 30,
};

function naira(n: number) {
  return `₦${(n / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

// SpeedPlus package merchant ID — the logistics vertical merchant
const PACKAGE_MERCHANT_ID = process.env['NEXT_PUBLIC_PACKAGE_MERCHANT_ID'] ?? '';

export default function PackagePricePage() {
  const router = useRouter();
  const {
    pickup, dropoff, size, weight,
    quote, paymentMethod, isMultiDrop, stops,
    setQuote, setPaymentMethod, setOrderId,
  } = usePackageFlowStore();

  const requestQuote = useRequestQuote();
  const requestMultiStopQuote = useRequestMultiStopQuote();
  const createOrder = useCreateOrder();
  const { data: walletData } = useWalletBalance();

  const { data: trustTier } = useQuery({
    queryKey: ['trust-tier'],
    queryFn: () => cardApi.getTrustTier(),
    staleTime: 60_000,
  });
  const canPayOnArrival = trustTier?.canPayOnArrival ?? false;
  const ordersToUnlock = Math.max(0, 3 - (trustTier?.completedOrders ?? 0));

  const [quoteError, setQuoteError] = useState('');
  const [consent, setConsent] = useState(false);

  // Fetch quote when the page mounts (or when size/weight/stops change).
  // Multi-drop prices via /quotes/multistop (route distance + per-stop fee);
  // single drop-off uses /quotes.
  useEffect(() => {
    if (!pickup || !size || !weight) return;
    const onSuccess = (q: Parameters<typeof setQuote>[0]) => setQuote(q);
    const onError = (err: unknown) =>
      setQuoteError(err instanceof Error ? err.message : 'Could not get a price. Try again.');
    setQuoteError('');

    if (isMultiDrop) {
      if (stops.length < 1) return;
      requestMultiStopQuote.mutate(
        {
          merchantId: PACKAGE_MERCHANT_ID,
          vertical: 'package',
          subtotalKobo: 0,
          originLat: pickup.lat,
          originLng: pickup.lng,
          stops: stops.map((s) => ({ lat: s.address.lat, lng: s.address.lng })),
          weightKg: WEIGHT_KG[weight] ?? 1.5,
          sizeCategory: size,
        },
        { onSuccess, onError },
      );
      return;
    }

    if (!dropoff) return;
    requestQuote.mutate(
      {
        merchantId: PACKAGE_MERCHANT_ID,
        vertical: 'package',
        subtotalKobo: 0, // package vertical: no product subtotal
        originLat: pickup.lat,
        originLng: pickup.lng,
        destLat: dropoff.lat,
        destLng: dropoff.lng,
        weightKg: WEIGHT_KG[weight] ?? 1.5,
        sizeCategory: size,
      },
      { onSuccess, onError },
    );
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pickup?.id, dropoff?.id, isMultiDrop, stops.length, size, weight]);

  const walletBalanceKobo = walletData?.balanceKobo ?? 0;
  const totalKobo = quote?.totalKobo ?? 0;
  const insufficientBalance = paymentMethod === 'wallet' && walletBalanceKobo < totalKobo;

  function handleConfirm() {
    if (!quote || !consent) return;
    if (isMultiDrop ? stops.length < 1 : !dropoff) return;
    createOrder.mutate(
      {
        merchantId: PACKAGE_MERCHANT_ID,
        quoteId: quote.id,
        vertical: 'package',
        deliveryAddressId: isMultiDrop && stops.length > 0
          ? stops[0]!.address.id
          : (dropoff?.id ?? ''),
        paymentMethod,
        items: [
          {
            productId: 'package-delivery',
            name: 'Package delivery',
            quantity: 1,
            unitPriceKobo: 0,
            weightKg: WEIGHT_KG[weight ?? 'light'],
            sizeCategory: size ?? 'small',
          },
        ],
        stops: isMultiDrop
          ? stops.map((s) => ({
              sequence: s.sequence,
              addressId: s.address.id,
              recipientName: s.recipientName,
              recipientPhone: s.recipientPhone,
              notes: s.notes || undefined,
            }))
          : undefined,
      } as Parameters<typeof createOrder.mutate>[0],
      {
        onSuccess: (order) => {
          setOrderId(order.id);
          router.push('/package/finding');
        },
        onError: (err) => {
          setQuoteError(err instanceof Error ? err.message : 'Could not place order. Try again.');
        },
      },
    );
  }

  const isLoading = requestQuote.isPending || requestMultiStopQuote.isPending;

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Here's the price" step={3} backHref="/package/what" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-5 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">

        {/* Price breakdown */}
        <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
          {isLoading ? (
            <>
              <div className="h-5 bg-line rounded animate-pulse w-3/4" />
              <div className="h-5 bg-line rounded animate-pulse w-1/2" />
              <div className="h-px bg-line my-1" />
              <div className="h-7 bg-line rounded animate-pulse w-1/3" />
            </>
          ) : quote ? (
            <>
              <div className="flex items-center justify-between text-[14px]">
                <span className="text-mid">Delivery ({quote.distanceKm.toFixed(1)} km)</span>
                <span className="text-ink font-medium">{naira(quote.deliveryKobo)}</span>
              </div>
              <div className="flex items-center justify-between text-[14px]">
                <span className="text-mid">Service fee</span>
                <span className="text-ink font-medium">{naira(quote.serviceKobo)}</span>
              </div>
              <div className="h-px bg-line my-1" />
              <div className="flex items-center justify-between">
                <span className="font-display font-semibold text-lg text-ink">Total</span>
                <span className="font-display font-bold text-2xl text-emerald">{naira(quote.totalKobo)}</span>
              </div>
              <div className="flex items-center gap-1.5 text-[11px] text-mid">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                ETA ~{quote.etaMinutes} min
              </div>
            </>
          ) : quoteError ? (
            <p className="text-sm text-[#DC2626]">{quoteError}</p>
          ) : null}
        </div>

        {/* Weather advisory */}
        {quote?.weatherAdvisory && (
          <div className="flex items-start gap-2.5 bg-[#FFF7E6] border border-[#F0DFB4] rounded-xl px-4 py-3">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#E8B14E" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="mt-0.5 flex-shrink-0">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <span className="text-[12px] text-[#8A6A1B]">{quote.weatherAdvisory}</span>
          </div>
        )}

        {/* Wallet balance */}
        <div className="flex items-center justify-between bg-white border border-line rounded-xl px-4 py-3">
          <span className="text-[13px] text-mid">Wallet balance</span>
          <span className={`text-[13px] font-semibold ${insufficientBalance ? 'text-[#DC2626]' : 'text-emerald'}`}>
            {naira(walletBalanceKobo)}
          </span>
        </div>

        {insufficientBalance && (
          <div className="flex items-center justify-between bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-4 py-3">
            <span className="text-[12px] text-[#DC2626]">
              You need {naira(totalKobo - walletBalanceKobo)} more to pay from wallet.
            </span>
            <button
              onClick={() => router.push('/wallet/fund')}
              className="text-[12px] font-semibold text-[#DC2626] underline ml-3 whitespace-nowrap"
            >
              Top up
            </button>
          </div>
        )}

        {/* Payment method */}
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">How do you want to pay?</span>
          <div className="flex flex-col gap-2">

            {/* Wallet — always available */}
            <button
              type="button"
              onClick={() => setPaymentMethod('wallet')}
              className={`w-full text-left rounded-[14px] border px-4 py-3 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald ${
                paymentMethod === 'wallet' ? 'border-emerald bg-tile' : 'border-line bg-white hover:border-emerald/40'
              }`}
            >
              <span className="block text-[13px] font-semibold text-ink">Pay from wallet</span>
              <span className="block text-[11px] text-mid mt-0.5">Deducted now, released on delivery</span>
            </button>

            {/* Pay on arrival — shown to everyone, enabled only when eligible */}
            {canPayOnArrival ? (
              <button
                type="button"
                onClick={() => setPaymentMethod('pay_on_arrival')}
                className={`w-full text-left rounded-[14px] border px-4 py-3 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald ${
                  paymentMethod === 'pay_on_arrival' ? 'border-emerald bg-tile' : 'border-line bg-white hover:border-emerald/40'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="block text-[13px] font-semibold text-ink">Pay on arrival</span>
                  <span className="text-[10px] font-bold text-emerald bg-tile rounded-full px-2 py-0.5">Unlocked</span>
                </div>
                <span className="block text-[11px] text-mid mt-0.5">
                  Rider scans your SpeedPlus card at the door. You enter your PIN to confirm.
                </span>
              </button>
            ) : (
              <div className="w-full text-left rounded-[14px] border border-line bg-white px-4 py-3">
                <div className="flex items-center justify-between">
                  <span className="block text-[13px] font-semibold text-ink">Pay on arrival</span>
                  <span className="text-[10px] font-bold text-mid bg-line rounded-full px-2 py-0.5">
                    {ordersToUnlock} order{ordersToUnlock !== 1 ? 's' : ''} to unlock
                  </span>
                </div>
                <span className="block text-[11px] text-mid mt-0.5">
                  Complete {ordersToUnlock} more paid order{ordersToUnlock !== 1 ? 's' : ''} to unlock this. We need to know you first.
                </span>
              </div>
            )}
          </div>
        </section>

        {/* Recipient consent — required before we store/share their name & phone */}
        <label className="flex items-start gap-2.5 text-[12px] text-mid">
          <input
            type="checkbox"
            checked={consent}
            onChange={(e) => setConsent(e.target.checked)}
            className="mt-0.5 w-4 h-4 accent-emerald flex-shrink-0"
          />
          <span>
            I confirm I have the recipient&apos;s permission to share their name and phone
            number with SpeedPlus for this delivery. Their details are encrypted and only
            shown to the assigned rider during delivery.
          </span>
        </label>

        <Button
          variant="primary"
          size="lg"
          disabled={!quote || !consent || insufficientBalance || createOrder.isPending}
          isLoading={createOrder.isPending}
          onClick={handleConfirm}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Confirm — find a rider
        </Button>

        {createOrder.isError && (
          <p className="text-xs text-[#DC2626]" role="alert">
            {createOrder.error instanceof Error ? createOrder.error.message : 'Something went wrong. Try again.'}
          </p>
        )}
      </div>
    </main>
  );
}
