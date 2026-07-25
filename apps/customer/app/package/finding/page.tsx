'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { usePackageFlowStore } from '../../../lib/store/package-flow.store';

const WS_URL = process.env['NEXT_PUBLIC_WS_URL'] ?? 'ws://localhost:8000/api/v1/ws';
const TIMEOUT_MS = 90_000;

export default function PackageFindingPage() {
  const router = useRouter();
  const { orderId, reset } = usePackageFlowStore();
  const [timedOut, setTimedOut] = useState(false);
  const [dots, setDots] = useState(1);
  const wsRef = useRef<WebSocket | null>(null);

  // Animated dots
  useEffect(() => {
    const t = setInterval(() => setDots((d) => (d % 3) + 1), 600);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    if (!orderId) { router.replace('/package/where'); return; }

    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;
    ws.onopen = () => ws.send(JSON.stringify({ action: 'subscribe', channel: `order:${orderId}` }));
    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data as string) as { event: string };
        if (msg.event === 'driver_assigned') router.push('/package/tracking');
      } catch { /* ignore */ }
    };

    const timeout = setTimeout(() => setTimedOut(true), TIMEOUT_MS);
    return () => { clearTimeout(timeout); ws.close(); };
  }, [orderId, router]);

  if (timedOut) {
    return (
      <main className="min-h-screen bg-[#F7F5EF] flex flex-col items-center justify-center px-5 gap-6">
        <div className="w-16 h-16 rounded-2xl bg-[#FEF2F2] flex items-center justify-center">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#DC2626" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
        </div>
        <div className="text-center flex flex-col gap-2">
          <h2 className="font-display font-bold text-[22px] text-[#121216]">No riders available</h2>
          <p className="text-[14px] text-[#63636E] max-w-xs leading-relaxed">
            All riders in your area are busy right now. Your order is saved — try again in a few minutes.
          </p>
        </div>
        <button
          onClick={() => { reset(); router.replace('/'); }}
          className="font-display text-sm font-semibold text-[#0A3D2C] border-2 border-[#0A3D2C] rounded-2xl px-8 py-3 hover:bg-[#E9F3D8] transition-colors"
        >
          Back to home
        </button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#0A3D2C] flex flex-col items-center justify-center px-5 gap-8">
      {/* Animated rings */}
      <div className="relative flex items-center justify-center">
        {[80, 56, 36].map((size, i) => (
          <span
            key={size}
            className="absolute rounded-full border border-[#C6F24E]/20"
            style={{
              width: size,
              height: size,
              animation: `pulse 2s cubic-bezier(0.4,0,0.6,1) ${i * 300}ms infinite`,
            }}
          />
        ))}
        <span className="w-14 h-14 rounded-full bg-[#C6F24E] flex items-center justify-center relative z-10">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
            <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z" />
            <circle cx="12" cy="10" r="3" />
          </svg>
        </span>
      </div>

      <div className="text-center flex flex-col gap-2">
        <h2 className="font-display font-bold text-[24px] text-white">
          Finding your rider{'.'.repeat(dots)}
        </h2>
        <p className="text-[14px] text-white/50">This usually takes under a minute</p>
      </div>

      {/* Progress indicator */}
      <div className="flex gap-1.5">
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className="w-1.5 h-1.5 rounded-full bg-white/30"
            style={{ animation: `bounce 1.2s ease-in-out ${i * 200}ms infinite` }}
          />
        ))}
      </div>

      <style>{`
        @keyframes pulse {
          0%, 100% { opacity: 0.4; transform: scale(1); }
          50% { opacity: 0.1; transform: scale(1.15); }
        }
        @keyframes bounce {
          0%, 100% { transform: translateY(0); opacity: 0.3; }
          50% { transform: translateY(-4px); opacity: 1; }
        }
      `}</style>
    </main>
  );
}
