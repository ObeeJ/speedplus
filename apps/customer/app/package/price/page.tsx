'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Skeleton, SelectionCard } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { usePackageFlowStore, type PaymentMethod } from '../../../lib/store/package-flow.store';
import { useRequestQuote, useRequestMultiStopQuote, useCreateOrder, useWalletBalance } from '../../../lib/hooks/use-order-mutations';
import { useQuery } from '@tanstack/react-query';
import { cardApi } from '@speedplus/api-client';

const WEIGHT_KG: Record<string, number> = { light: 1.5, medium: 6, heavy: 17, very_heavy: 30 };

function naira(n: number) {
  return `₦${(n / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">{children}</p>;
}

function LineItem({ label, value, bold }: { label: string; value: string; bold?: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span className={`text-[13px] ${bold ? 'font-semibold text-[#121216]' : 'text-[#63636E]'}`}>{label}</span>
      <span className={`text-[13px] ${bold ? 'font-display font-bold text-[#0A3D2C] text-[18px]' : 'font-medium text-[#121216]'}`}>{value}</span>
    </div>
  );
}

const PACKAGE_MERCHANT_ID = process.env['NEXT_PUBLIC_PACKAGE_MERCHANT_ID'] ?? '';

export default function PackagePricePage() {
  const router = useRouter();
  const { pickup, dropoff, size, weight, quote, paymentMethod, isMultiDrop, stops, setQuote, setPaymentMethod, setOrderId } = usePackageFlowStore();

  const requestQuote = useRequestQuote();
  const requestMultiStopQuote = useRequestMultiStopQuote();
  const createOrder = useCreateOrder();
  const { data: walletData } = useWalletBalance();

  const { data: trustTier } = useQuery({ queryKey: ['trust-tier'], queryFn: () => cardApi.getTrustTier(), staleTime: 60_000 });
  const canPayOnArrival = trustTier?.canPayOnArrival ?? false;
  const ordersToUnlock = Math.max(0, 3 - (trustTier?.completedOrders ?? 0));

  const [quoteError, setQuoteError] = useState('');
  const [consent, setConsent] = useState(false);

  useEffect(() => {
    if (!pickup || !size || !weight) return;
    setQuoteError('');
    const mutate = isMultiDrop && stops.length > 0 ? requestMultiStopQuote : requestQuote;
    const payload = isMultiDrop && stops.length > 0
      ? { merchantId: PACKAGE_MERCHANT_ID, vertical: 'package', subtotalKobo: 1, originLat: pickup.lat, originLng: pickup.lng, stops: stops.map((s) => ({ lat: s.address.lat, lng: s.address.lng })), weightKg: WEIGHT_KG[weight] ?? 1.5, sizeCategory: size }
      : { merchantId: PACKAGE_MERCHANT_ID, vertical: 'package', subtotalKobo: 1, originLat: pickup.lat, originLng: pickup.lng, destLat: dropoff?.lat ?? 0, destLng: dropoff?.lng ?? 0, weightKg: WEIGHT_KG[weight] ?? 1.5, sizeCategory: size };
    (mutate as typeof requestQuote).mutate(payload as Parameters<typeof requestQuote.mutate>[0], {
      onSuccess: (q) => setQuote(q),
      onError: (err) => setQuoteError(err instanceof Error ? err.message : 'Could not get a price. Try again.'),
    });
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pickup?.id, dropoff?.id, size, weight]);

  const walletKobo = walletData?.balanceKobo ?? 0;
  const totalKobo = quote?.totalKobo ?? 0;
  const insufficientBalance = paymentMethod === 'wallet' && walletKobo < totalKobo;
  const isLoading = requestQuote.isPending || requestMultiStopQuote.isPending;
  const canConfirm = Boolean(quote && !insufficientBalance && consent && !createOrder.isPending);

  function handleConfirm() {
    if (!quote || !dropoff) return;
    createOrder.mutate(
      {
        merchantId: PACKAGE_MERCHANT_ID,
        quoteId: quote.id,
        vertical: 'package',
        deliveryAddressId: isMultiDrop && stops.length > 0 ? stops[0]!.address.id : dropoff.id,
        paymentMethod,
        items: [{ productId: 'package-delivery', name: 'Package delivery', quantity: 1, unitPriceKobo: 0, weightKg: WEIGHT_KG[weight ?? 'light'], sizeCategory: size ?? 'small' }],
        stops: isMultiDrop ? stops.map((s) => ({ sequence: s.sequence, addressId: s.address.id, recipientName: s.recipientName, recipientPhone: s.recipientPhone, notes: s.notes || undefined })) : undefined,
      } as Parameters<typeof createOrder.mutate>[0],
      {
        onSuccess: (order) => { setOrderId(order.id); router.push('/package/finding'); },
        onError: (err) => setQuoteError(err instanceof Error ? err.message : 'Could not place order. Try again.'),
      },
    );
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <FlowHeader title="Price breakdown" step={3} backHref="/package/what" />

      <div className="flex-1 px-5 py-6 flex flex-col gap-6 max-w-[600px] mx-auto w-full pb-36">

        {/* Price card */}
        <div className="bg-white rounded-2xl border border-[#E4E0D6] overflow-hidden">
          {isLoading ? (
            <div className="p-5 flex flex-col gap-3">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-2/3" />
              <div className="h-px bg-[#E4E0D6] my-1" />
              <Skeleton className="h-7 w-1/3" />
            </div>
          ) : quote ? (
            <div className="p-5 flex flex-col gap-3">
              <LineItem label={`Delivery (${quote.distanceKm.toFixed(1)} km)`} value={naira(quote.deliveryKobo)} />
              {quote.serviceKobo > 0 && <LineItem label="Service fee" value={naira(quote.serviceKobo)} />}
              <div className="h-px bg-[#E4E0D6]" />
              <LineItem label="Total" value={naira(quote.totalKobo)} bold />
              <div className="flex items-center gap-1.5 text-[11px] text-[#63636E] mt-1">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" />
                </svg>
                ETA ~{quote.etaMinutes} min
              </div>
            </div>
          ) : quoteError ? (
            <div className="p-5 flex items-start gap-2.5">
              <svg className="flex-shrink-0 mt-0.5" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#DC2626" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
              <p className="text-[13px] text-[#DC2626]">{quoteError}</p>
            </div>
          ) : null}
        </div>

        {/* Weather advisory */}
        {quote?.weatherAdvisory && (
          <div className="flex items-start gap-3 bg-[#FFF7E6] border border-[#F0DFB4] rounded-2xl px-4 py-3.5">
            <svg className="flex-shrink-0 mt-0.5" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#E8B14E" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" /><line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
            <p className="text-[12px] text-[#8A6A1B] leading-relaxed">{quote.weatherAdvisory}</p>
          </div>
        )}

        {/* Wallet balance */}
        <section className="flex flex-col gap-3">
          <SectionLabel>Your wallet</SectionLabel>
          <div className={`rounded-2xl border px-4 py-3.5 flex items-center justify-between ${insufficientBalance ? 'bg-[#FEF2F2] border-[#FECACA]' : 'bg-white border-[#E4E0D6]'}`}>
            <div>
              <p className="text-[12px] text-[#63636E]">Available balance</p>
              <p className={`font-display font-bold text-[20px] mt-0.5 ${insufficientBalance ? 'text-[#DC2626]' : 'text-[#0A3D2C]'}`}>
                {naira(walletKobo)}
              </p>
            </div>
            {insufficientBalance ? (
              <button
                onClick={() => router.push('/wallet/fund')}
                className="flex items-center gap-1.5 text-[12px] font-semibold text-white bg-[#DC2626] rounded-xl px-3.5 py-2 hover:bg-[#B91C1C] transition-colors"
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
                </svg>
                Top up {naira(totalKobo - walletKobo)}
              </button>
            ) : (
              <button
                onClick={() => router.push('/wallet/fund')}
                className="text-[12px] font-semibold text-[#0A3D2C] border border-[#0A3D2C]/20 rounded-xl px-3.5 py-2 hover:bg-[#E9F3D8] transition-colors"
              >
                Add money
              </button>
            )}
          </div>
        </section>

        {/* Payment method */}
        <section className="flex flex-col gap-3">
          <SectionLabel>How do you want to pay?</SectionLabel>
          <div className="flex flex-col gap-2">
            <SelectionCard
              selected={paymentMethod === 'wallet'}
              label="Pay from wallet"
              description="Deducted now, released when delivered"
              icon={<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" /></svg>}
              onClick={() => setPaymentMethod('wallet')}
            />
            {canPayOnArrival ? (
              <SelectionCard
                selected={paymentMethod === 'pay_on_arrival'}
                label="Pay on arrival"
                description="Rider scans your SpeedPlus card at the door. You enter your PIN."
                badge="Unlocked"
                icon={<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0110 0v4" /></svg>}
                onClick={() => setPaymentMethod('pay_on_arrival')}
              />
            ) : (
              <SelectionCard
                selected={false}
                label="Pay on arrival"
                description="Complete your first 3 paid orders to unlock this."
                badge={`${ordersToUnlock} order${ordersToUnlock !== 1 ? 's' : ''} to unlock`}
                locked
                lockReason={`You need ${ordersToUnlock} more completed order${ordersToUnlock !== 1 ? 's' : ''}. We need to know you first.`}
                icon={<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0110 0v4" /></svg>}
                onClick={() => {}}
              />
            )}
          </div>
        </section>

        {/* Consent */}
        <label className="flex items-start gap-3 cursor-pointer group">
          <div className={`mt-0.5 w-5 h-5 rounded-md border-2 flex items-center justify-center flex-shrink-0 transition-all ${consent ? 'bg-[#0A3D2C] border-[#0A3D2C]' : 'bg-white border-[#E4E0D6] group-hover:border-[#0A3D2C]/40'}`}>
            {consent && (
              <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            )}
          </div>
          <input type="checkbox" checked={consent} onChange={(e) => setConsent(e.target.checked)} className="sr-only" />
          <p className="text-[12px] text-[#63636E] leading-relaxed">
            I confirm the recipient has agreed to share their name and phone number with SpeedPlus for this delivery. Their details are encrypted and only shown to the assigned rider.
          </p>
        </label>

        {createOrder.isError && (
          <div className="flex items-start gap-2.5 bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-3.5 py-2.5" role="alert">
            <svg className="flex-shrink-0 mt-0.5" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#DC2626" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
            <p className="text-xs text-[#DC2626]">{createOrder.error instanceof Error ? createOrder.error.message : 'Something went wrong.'}</p>
          </div>
        )}
      </div>

      {/* Sticky CTA */}
      <div className="fixed bottom-0 left-0 right-0 bg-[#F7F5EF]/95 backdrop-blur-sm border-t border-[#E4E0D6] px-5 py-4 pb-safe-bottom">
        <div className="max-w-[600px] mx-auto flex items-center justify-between gap-4">
          <div>
            {quote && <p className="font-display font-bold text-[20px] text-[#0A3D2C]">{naira(quote.totalKobo)}</p>}
            <p className="text-[11px] text-[#9A968D]">{paymentMethod === 'wallet' ? 'Charged from wallet' : 'Paid at the door'}</p>
          </div>
          <Button
            variant="primary"
            size="lg"
            disabled={!canConfirm}
            isLoading={createOrder.isPending}
            onClick={handleConfirm}
            className="flex-shrink-0"
          >
            Confirm & find rider
          </Button>
        </div>
      </div>
    </main>
  );
}
