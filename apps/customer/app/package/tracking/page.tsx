'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { usePackageFlowStore } from '../../../lib/store/package-flow.store';
import { useTrackOrder } from '../../../lib/hooks/use-order-mutations';
import { ordersApi } from '@speedplus/api-client';

const STEPS = ['Order confirmed', 'Rider assigned', 'On the way', 'Delivered'];

const STATUS_STEP: Record<string, number> = {
  pending: 0,
  confirmed: 0,
  driver_assigned: 1,
  ready_for_pickup: 1,
  in_transit: 2,
  delivered: 3,
};

function naira(n: number) {
  return `₦${(n / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

export default function PackageTrackingPage() {
  const router = useRouter();
  const { pickup, dropoff, orderId, quote, reset } = usePackageFlowStore();
  const { data: order, isError } = useTrackOrder(orderId);

  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState('');

  const stepIndex = order ? (STATUS_STEP[order.status] ?? 0) : 0;
  const etaMinutes = quote?.etaMinutes ?? 0;

  // Reset flow when delivered
  useEffect(() => {
    if (order?.status === 'delivered') {
      const t = setTimeout(() => { reset(); router.replace('/'); }, 4000);
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

  // Derive rider info from order (driverId present when assigned)
  const driverAssigned = order?.driverId != null;
  const driverInitial = driverAssigned ? 'R' : '?';

  if (isError) {
    return (
      <main className="min-h-screen bg-sand flex flex-col items-center justify-center gap-4 px-5">
        <span className="text-sm text-mid text-center">Could not load order status. Check your connection.</span>
        <button onClick={() => router.replace('/')} className="text-sm font-semibold text-emerald underline">
          Back to home
        </button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      {/* Map placeholder / ETA hero */}
      <div className="bg-emerald relative overflow-hidden px-5 py-8 min-[700px]:px-9 min-[700px]:py-12 flex flex-col items-center gap-3">
        <svg width="90" height="90" viewBox="0 0 90 90" aria-hidden="true">
          <circle cx="45" cy="45" r="6" fill="#C6F24E" />
          <circle cx="45" cy="45" r="6" fill="#C6F24E" opacity="0.4">
            <animate attributeName="r" values="6;22;6" dur="2s" repeatCount="indefinite" />
            <animate attributeName="opacity" values="0.4;0;0.4" dur="2s" repeatCount="indefinite" />
          </circle>
          <path d="M20 60 Q45 20 70 60" stroke="#C6F24E" strokeWidth="2" strokeDasharray="4 5" fill="none" />
        </svg>
        <span className="font-display font-bold text-3xl text-lime">
          {order?.status === 'delivered' ? 'Delivered!' : `~${etaMinutes} min`}
        </span>
        <span className="text-sm text-sand/70">
          {dropoff ? `Arriving at ${dropoff.street}` : 'On its way'}
        </span>
      </div>

      <div className="flex-1 px-5 py-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full flex flex-col gap-5">

        {/* Rider card */}
        {driverAssigned ? (
          <div className="bg-white border border-line rounded-2xl p-4 flex items-center gap-3">
            <span className="w-11 h-11 rounded-full bg-tile flex items-center justify-center font-display font-semibold text-emerald">
              {driverInitial}
            </span>
            <div className="flex-1 flex flex-col">
              <span className="font-semibold text-ink text-[14px]">Your rider</span>
              <span className="text-[12px] text-mid">Bike · On the way</span>
            </div>
            <a
              href={`tel:`}
              className="w-10 h-10 rounded-full bg-tile flex items-center justify-center hover:bg-[#DCEDC2] transition-colors"
              aria-label="Call rider"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07 19.5 19.5 0 01-6-6 19.79 19.79 0 01-3.07-8.67A2 2 0 014.11 2h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L8.09 9.91a16 16 0 006 6l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z" />
              </svg>
            </a>
          </div>
        ) : (
          <div className="bg-white border border-line rounded-2xl p-4 flex items-center gap-3">
            <span className="w-11 h-11 rounded-full bg-line animate-pulse" />
            <div className="flex-1 flex flex-col gap-1.5">
              <span className="h-3.5 bg-line rounded animate-pulse w-24" />
              <span className="h-3 bg-line rounded animate-pulse w-16" />
            </div>
          </div>
        )}

        {/* Route summary */}
        {pickup && dropoff && (
          <span className="text-[13px] text-mid">
            From <b className="text-ink">{pickup.street}</b> to <b className="text-ink">{dropoff.street}</b>
          </span>
        )}

        {/* Status steps */}
        <div className="flex flex-col gap-0">
          {STEPS.map((label, i) => {
            const done = i <= stepIndex;
            const current = i === stepIndex;
            return (
              <div key={label} className="flex gap-3">
                <div className="flex flex-col items-center">
                  <span
                    className={`w-3 h-3 rounded-full transition-colors duration-300 ${done ? 'bg-emerald' : 'bg-line'}`}
                  />
                  {i < STEPS.length - 1 && (
                    <span className={`w-0.5 flex-1 min-h-[28px] transition-colors duration-300 ${done ? 'bg-emerald' : 'bg-line'}`} />
                  )}
                </div>
                <span className={`text-[14px] pb-7 transition-colors duration-300 ${done ? 'text-ink font-medium' : 'text-mid'} ${current ? 'text-emerald' : ''}`}>
                  {label}
                </span>
              </div>
            );
          })}
        </div>

        {/* Order total */}
        {order && (
          <div className="flex items-center justify-between bg-white border border-line rounded-xl px-4 py-3">
            <span className="text-[13px] text-mid">Order total</span>
            <span className="text-[13px] font-semibold text-ink">{naira(order.total.amount)}</span>
          </div>
        )}

        {/* Cancel — only available before in_transit */}
        {order && !['in_transit', 'delivered', 'cancelled', 'refunded'].includes(order.status) && (
          <div className="flex flex-col gap-1.5">
            <button
              onClick={handleCancel}
              disabled={cancelling}
              className="text-sm font-semibold text-[#DC2626] hover:underline disabled:opacity-50 text-left"
            >
              {cancelling ? 'Cancelling…' : 'Cancel order'}
            </button>
            {cancelError && <p className="text-xs text-[#DC2626]">{cancelError}</p>}
          </div>
        )}
      </div>
    </main>
  );
}
