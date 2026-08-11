'use client';

import { useEffect, useState } from 'react';
import { useFoodFlowStore } from '../../../lib/store/food-flow.store';
import { useTrackOrder } from '../../../lib/hooks/use-order-mutations';

const STEPS = ['Order confirmed', 'Kitchen preparing', 'On the way', 'Delivered'];

const STATUS_STEP: Record<string, number> = {
  pending: 0,
  confirmed: 0,
  preparing: 1,
  ready_for_pickup: 1,
  driver_assigned: 1,
  in_transit: 2,
  delivered: 3,
};

export default function FoodTrackingPage() {
  const { deliverToAddress, orderId, reset } = useFoodFlowStore();
  const { data: liveOrder } = useTrackOrder(orderId);
  const [now, setNow] = useState(() => Date.now());

  const stepIndex = liveOrder ? (STATUS_STEP[liveOrder.status] ?? 0) : 0;
  const etaMin = liveOrder?.estimatedDeliveryAt
    ? Math.max(0, Math.round((new Date(liveOrder.estimatedDeliveryAt).getTime() - now) / 60000))
    : null;

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 30_000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    if (stepIndex === STEPS.length - 1) reset();
  }, [stepIndex, reset]);

  const addressLabel = deliverToAddress?.label || deliverToAddress?.street || 'your address';

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="bg-emerald relative overflow-hidden px-5 py-8 min-[700px]:px-9 min-[700px]:py-12 flex flex-col items-center gap-3">
        <svg width="90" height="90" viewBox="0 0 90 90" className="mb-1">
          <circle cx="45" cy="45" r="6" fill="#C6F24E" />
          <circle cx="45" cy="45" r="6" fill="#C6F24E" opacity="0.4">
            <animate attributeName="r" values="6;20;6" dur="2s" repeatCount="indefinite" />
            <animate attributeName="opacity" values="0.4;0;0.4" dur="2s" repeatCount="indefinite" />
          </circle>
          <path d="M20 60 Q45 20 70 60" stroke="#C6F24E" strokeWidth="2" strokeDasharray="4 5" fill="none" />
        </svg>
        {etaMin !== null ? (
          <span className="font-display font-bold text-3xl text-lime">{etaMin} min</span>
        ) : (
          <span className="font-display font-bold text-3xl text-lime">On its way</span>
        )}
        <span className="text-sm text-sand/70">Arriving at {addressLabel}</span>
      </div>

      <div className="flex-1 px-5 py-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full flex flex-col gap-6">
        {liveOrder?.driverName && (
          <div className="bg-white border border-line rounded-2xl p-4 flex items-center gap-3">
            <span className="w-11 h-11 rounded-full bg-tile flex items-center justify-center font-display font-semibold text-emerald">
              {liveOrder.driverName[0]}
            </span>
            <div className="flex-1 flex flex-col">
              <span className="font-semibold text-ink text-[14px]">{liveOrder.driverName}</span>
              {liveOrder.driverVehicle && (
                <span className="text-[12px] text-mid">{liveOrder.driverVehicle}</span>
              )}
            </div>
          </div>
        )}

        <div className="flex flex-col gap-0">
          {STEPS.map((label, i) => {
            const done = i <= stepIndex;
            return (
              <div key={label} className="flex gap-3">
                <div className="flex flex-col items-center">
                  <span className={`w-3 h-3 rounded-full ${done ? 'bg-emerald' : 'bg-line'}`} />
                  {i < STEPS.length - 1 && <span className={`w-0.5 flex-1 min-h-[28px] ${done ? 'bg-emerald' : 'bg-line'}`} />}
                </div>
                <span className={`text-[14px] pb-7 ${done ? 'text-ink font-medium' : 'text-mid'}`}>{label}</span>
              </div>
            );
          })}
        </div>
      </div>
    </main>
  );
}
