'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Avatar, StatusSteps, Skeleton } from '@speedplus/ui';
import { usePackageFlowStore } from '../../../lib/store/package-flow.store';
import { useTrackOrder } from '../../../lib/hooks/use-order-mutations';
import { ordersApi } from '@speedplus/api-client';

function naira(n: number) {
  return `₦${(n / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

const STEPS = [
  { label: 'Order confirmed', sublabel: 'We received your order' },
  { label: 'Rider assigned', sublabel: 'A rider is heading to pickup' },
  { label: 'On the way', sublabel: 'Your package is in transit' },
  { label: 'Delivered', sublabel: 'Package handed over' },
];

const STATUS_STEP: Record<string, number> = {
  pending: 0, confirmed: 0,
  driver_assigned: 1, ready_for_pickup: 1,
  in_transit: 2,
  delivered: 3,
};

export default function PackageTrackingPage() {
  const router = useRouter();
  const { pickup, dropoff, orderId, quote, reset } = usePackageFlowStore();
  const { data: order, isError } = useTrackOrder(orderId);

  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState('');
  const [deliveryCode, setDeliveryCode] = useState<string | null>(null);

  // WS listener for delivery code
  useEffect(() => {
    if (!orderId) return;
    const ws = new WebSocket(process.env['NEXT_PUBLIC_WS_URL'] ?? 'ws://localhost:8000/api/v1/ws');
    ws.onopen = () => ws.send(JSON.stringify({ action: 'subscribe', channel: `order:${orderId}` }));
    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data as string) as { event: string; data: { code?: string } };
        if (msg.event === 'delivery_code' && msg.data.code) setDeliveryCode(msg.data.code);
      } catch { /* ignore */ }
    };
    return () => ws.close();
  }, [orderId]);

  const stepIndex = order ? (STATUS_STEP[order.status] ?? 0) : 0;
  const etaMinutes = quote?.etaMinutes ?? 0;

  useEffect(() => {
    if (order?.status === 'delivered') {
      const t = setTimeout(() => { reset(); router.replace('/'); }, 5000);
      return () => clearTimeout(t);
    }
  }, [order?.status, reset, router]);

  async function handleCancel() {
    if (!orderId) return;
    setCancelling(true);
    setCancelError('');
    try {
      await ordersApi.cancel(orderId, 'Customer cancelled before pickup');
      reset();
      router.replace('/');
    } catch (err) {
      setCancelError(err instanceof Error ? err.message : 'Could not cancel. Try again.');
    } finally {
      setCancelling(false);
    }
  }

  const driverAssigned = order?.driverId != null;
  const driverName = (order as (typeof order & { driverName?: string }) | undefined)?.driverName ?? '';
  const driverRating = (order as (typeof order & { driverRating?: number }) | undefined)?.driverRating;
  const driverVehicle = (order as (typeof order & { driverVehicle?: string }) | undefined)?.driverVehicle ?? 'Bike';
  const driverPhone = (order as (typeof order & { driverPhone?: string }) | undefined)?.driverPhone;

  if (isError) {
    return (
      <main className="min-h-screen bg-[#F7F5EF] flex flex-col items-center justify-center gap-4 px-5">
        <p className="text-[14px] text-[#63636E] text-center">Could not load order status. Check your connection.</p>
        <button onClick={() => router.replace('/')} className="text-[13px] font-semibold text-[#0A3D2C] underline">Back to home</button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      {/* ETA hero */}
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-8 flex flex-col items-center gap-2 relative overflow-hidden">
        {/* Subtle background rings */}
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
          {[200, 280, 360].map((s) => (
            <span key={s} className="absolute rounded-full border border-white/[0.04]" style={{ width: s, height: s }} />
          ))}
        </div>

        <div className="relative z-10 flex flex-col items-center gap-2">
          {order?.status === 'delivered' ? (
            <>
              <div className="w-16 h-16 rounded-full bg-[#C6F24E] flex items-center justify-center mb-1">
                <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </div>
              <p className="font-display font-bold text-[32px] text-[#C6F24E]">Delivered!</p>
              <p className="text-white/60 text-[14px]">Your package has arrived</p>
            </>
          ) : (
            <>
              <p className="text-white/50 text-[12px] font-semibold tracking-widest uppercase">Estimated arrival</p>
              <p className="font-display font-bold text-[52px] text-[#C6F24E] leading-none">
                {etaMinutes}<span className="text-[24px] text-white/60 ml-1">min</span>
              </p>
              {dropoff && <p className="text-white/60 text-[13px]">{dropoff.street}</p>}
            </>
          )}
        </div>
      </div>

      <div className="flex-1 px-5 py-5 flex flex-col gap-4 max-w-[600px] mx-auto w-full">

        {/* Rider card */}
        {driverAssigned ? (
          <div className="bg-white rounded-2xl border border-[#E4E0D6] p-4 flex items-center gap-3">
            <Avatar initials={driverName ? driverName.charAt(0) : 'R'} size="md" variant="emerald" />
            <div className="flex-1 min-w-0">
              <p className="font-semibold text-[14px] text-[#121216] truncate">{driverName || 'Your rider'}</p>
              <p className="text-[12px] text-[#63636E] capitalize">
                {driverVehicle}{driverRating ? ` · ★ ${driverRating.toFixed(1)}` : ''}
              </p>
            </div>
            {driverPhone && (
              <a
                href={`tel:${driverPhone}`}
                className="w-10 h-10 rounded-xl bg-[#E9F3D8] flex items-center justify-center hover:bg-[#D4EAC0] transition-colors flex-shrink-0"
                aria-label="Call rider"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                  <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07 19.5 19.5 0 01-6-6 19.79 19.79 0 01-3.07-8.67A2 2 0 014.11 2h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L8.09 9.91a16 16 0 006 6l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z" />
                </svg>
              </a>
            )}
          </div>
        ) : (
          <div className="bg-white rounded-2xl border border-[#E4E0D6] p-4 flex items-center gap-3">
            <Skeleton className="w-11 h-11 rounded-full" />
            <div className="flex-1 flex flex-col gap-2">
              <Skeleton className="h-3.5 w-28" />
              <Skeleton className="h-3 w-20" />
            </div>
          </div>
        )}

        {/* Delivery code */}
        {deliveryCode && order?.status === 'in_transit' && (
          <div
            className="bg-[#0A3D2C] rounded-2xl px-5 py-4 flex flex-col gap-1.5"
            style={{ animation: 'slideUp 0.3s cubic-bezier(0.16,1,0.3,1) both' }}
          >
            <p className="text-[10px] font-semibold text-white/50 tracking-[0.8px] uppercase">Your delivery code</p>
            <p className="font-display font-bold text-[40px] text-[#C6F24E] tracking-[14px] leading-none">{deliveryCode}</p>
            <p className="text-[11px] text-white/50 mt-1">Share this with whoever is collecting the package.</p>
          </div>
        )}

        {/* Route */}
        {pickup && dropoff && (
          <div className="flex items-center gap-2 text-[13px] text-[#63636E]">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" />
            </svg>
            <span className="font-medium text-[#121216] truncate">{pickup.street}</span>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#9A968D" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
            <span className="font-medium text-[#121216] truncate">{dropoff.street}</span>
          </div>
        )}

        {/* Status steps */}
        <div className="bg-white rounded-2xl border border-[#E4E0D6] p-5">
          <StatusSteps steps={STEPS} currentIndex={stepIndex} />
        </div>

        {/* Order total */}
        {order && (
          <div className="bg-white rounded-2xl border border-[#E4E0D6] px-4 py-3.5 flex items-center justify-between">
            <p className="text-[13px] text-[#63636E]">Order total</p>
            <p className="font-display font-semibold text-[16px] text-[#121216]">{naira(order.total.amount)}</p>
          </div>
        )}

        {/* Cancel */}
        {order && !['in_transit', 'delivered', 'cancelled', 'refunded'].includes(order.status) && (
          <div className="flex flex-col gap-1.5">
            <button
              onClick={handleCancel}
              disabled={cancelling}
              className="text-[13px] font-semibold text-[#DC2626] hover:underline disabled:opacity-50 text-left"
            >
              {cancelling ? 'Cancelling…' : 'Cancel order'}
            </button>
            {cancelError && <p className="text-[12px] text-[#DC2626]">{cancelError}</p>}
          </div>
        )}
      </div>

      <style>{`
        @keyframes slideUp {
          from { opacity: 0; transform: translateY(12px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
