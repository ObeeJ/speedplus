'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { usePackageFlowStore } from '../../../lib/store/package-flow.store';

const WS_URL = process.env['NEXT_PUBLIC_WS_URL'] ?? 'ws://localhost:8000/api/v1/ws';
const SEARCH_TIMEOUT_MS = 90_000; // 90s before showing "no riders" state

export default function PackageFindingPage() {
  const router = useRouter();
  const { orderId, reset } = usePackageFlowStore();
  const [timedOut, setTimedOut] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!orderId) {
      router.replace('/package/where');
      return;
    }

    // Subscribe to the order channel via WS
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;

    ws.onopen = () => {
      ws.send(JSON.stringify({ action: 'subscribe', channel: `order:${orderId}` }));
    };

    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data as string) as { event: string };
        if (msg.event === 'driver_assigned' || msg.event === 'searching_driver') {
          if (msg.event === 'driver_assigned') {
            router.push('/package/tracking');
          }
        }
      } catch {
        // malformed message — ignore
      }
    };

    // Timeout: if no driver found in 90s, show error state
    const timeout = setTimeout(() => setTimedOut(true), SEARCH_TIMEOUT_MS);

    return () => {
      clearTimeout(timeout);
      ws.close();
    };
  }, [orderId, router]);

  if (timedOut) {
    return (
      <main className="min-h-screen bg-sand flex flex-col items-center justify-center gap-5 px-5">
        <div className="w-16 h-16 rounded-full bg-[#FEF2F2] flex items-center justify-center">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#DC2626" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <span className="font-display font-semibold text-xl text-ink text-center">No riders available right now</span>
        <span className="text-sm text-mid text-center max-w-xs">
          All riders in your area are busy. Your order is saved — try again in a few minutes.
        </span>
        <button
          onClick={() => { reset(); router.replace('/'); }}
          className="mt-2 text-sm font-semibold text-emerald underline"
        >
          Back to home
        </button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-emerald flex flex-col items-center justify-center gap-5 px-5">
      {/* Animated pulse rings */}
      <div className="relative flex items-center justify-center w-20 h-20">
        <span className="absolute w-20 h-20 rounded-full border-2 border-lime/30 animate-ping" />
        <span className="absolute w-14 h-14 rounded-full border-2 border-lime/50 animate-ping [animation-delay:300ms]" />
        <span className="w-8 h-8 rounded-full bg-lime" />
      </div>
      <span className="font-display font-semibold text-xl text-sand text-center">Finding you a rider…</span>
      <span className="text-sm text-sand/60 text-center">This usually takes under a minute</span>
    </main>
  );
}
