'use client';

import { useEffect, useState } from 'react';
import { useGasFlowStore } from '../../../lib/store/gas-flow.store';
import { useTrackOrder } from '../../../lib/hooks/use-order-mutations';

const STEPS = ['Order confirmed', 'Rider assigned', 'On the way', 'Delivered'];

const STATUS_STEP: Record<string, number> = {
  pending: 0,
  confirmed: 0,
  driver_assigned: 1,
  ready_for_pickup: 1,
  in_transit: 2,
  delivered: 3,
};

export default function GasTrackingPage() {
  const { orderId, reset } = useGasFlowStore();
  const { data: liveOrder } = useTrackOrder(orderId);
  const [stepIndex, setStepIndex] = useState(1);
  const [etaMin, setEtaMin] = useState(4);

  const effectiveStepIndex = liveOrder ? (STATUS_STEP[liveOrder.status] ?? stepIndex) : stepIndex;

  useEffect(() => {
    if (liveOrder) return;
    const stepTimer = setInterval(() => setStepIndex((s) => Math.min(s + 1, STEPS.length - 1)), 8000);
    const etaTimer = setInterval(() => setEtaMin((m) => Math.max(m - 1, 0)), 15000);
    return () => {
      clearInterval(stepTimer);
      clearInterval(etaTimer);
    };
  }, [liveOrder]);

  useEffect(() => {
    if (effectiveStepIndex === STEPS.length - 1) reset();
  }, [effectiveStepIndex, reset]);

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
        <span className="font-display font-bold text-3xl text-lime">{etaMin} min</span>
        <span className="text-sm text-sand/70">On its way</span>
      </div>

      <div className="flex-1 px-5 py-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full flex flex-col gap-6">
        <div className="bg-white border border-line rounded-2xl p-4 flex items-center gap-3">
          <span className="w-11 h-11 rounded-full bg-tile flex items-center justify-center font-display font-semibold text-emerald">
            {liveOrder?.driverName?.[0] ?? 'R'}
          </span>
          <div className="flex-1 flex flex-col">
            <span className="font-semibold text-ink text-[14px]">{liveOrder?.driverName ?? 'Finding your rider…'}</span>
            <span className="text-[12px] text-mid">
              {liveOrder?.driverVehicle ?? ''}{liveOrder?.driverRating ? ` · ★ ${liveOrder.driverRating.toFixed(1)}` : ''}
            </span>
          </div>
          {liveOrder?.driverPhone && (
            <a href={`tel:${liveOrder.driverPhone}`} className="w-10 h-10 rounded-full bg-tile flex items-center justify-center" aria-label="Call rider">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 16.92v3a2 2 0 01-2.18 2 19.79 19.79 0 01-8.63-3.07 19.5 19.5 0 01-6-6 19.79 19.79 0 01-3.07-8.67A2 2 0 014.11 2h3a2 2 0 012 1.72c.127.96.361 1.903.7 2.81a2 2 0 01-.45 2.11L8.09 9.91a16 16 0 006 6l1.27-1.27a2 2 0 012.11-.45c.907.339 1.85.573 2.81.7A2 2 0 0122 16.92z" />
              </svg>
            </a>
          )}
        </div>

        <div className="flex flex-col gap-0">
          {STEPS.map((label, i) => {
            const done = i <= effectiveStepIndex;
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
