'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Button, ListCard } from '@speedplus/ui';
import { ussdApi, type USSDIntent } from '@speedplus/api-client';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function USSDBankPage() {
  const router = useRouter();
  const [amount, setAmount] = useState('');
  const [bankCode, setBankCode] = useState('');
  const [intent, setIntent] = useState<USSDIntent | null>(null);

  const { data: banksData, isLoading: banksLoading } = useQuery({
    queryKey: ['ussd-banks'],
    queryFn: () => ussdApi.getBanks(),
    staleTime: Infinity,
  });

  const { data: statusData } = useQuery({
    queryKey: ['ussd-intent', intent?.id],
    queryFn: () => ussdApi.getIntentStatus(intent!.id),
    enabled: !!intent && intent.status === 'pending',
    refetchInterval: 4000,
  });

  const initiate = useMutation({
    mutationFn: ({ bankCode, amountKobo }: { bankCode: string; amountKobo: number }) =>
      ussdApi.initiate({ bankCode, amountKobo, email: '' }, `ussd-${Date.now()}`),
    onSuccess: (data) => setIntent(data),
  });

  const banks = banksData?.banks ?? [];
  const liveStatus = statusData?.status ?? intent?.status;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const kobo = Math.round(parseFloat(amount) * 100);
    if (!kobo || kobo < 10000 || !bankCode) return;
    initiate.mutate({ bankCode, amountKobo: kobo });
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-6">
        <div className="flex items-center gap-3">
          <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Fund via USSD</h1>
        </div>
      </div>

      <div className="flex-1 px-5 py-6 max-w-[480px] mx-auto w-full flex flex-col gap-5">
        {intent && liveStatus !== 'expired' && liveStatus !== 'failed' ? (
          <ListCard className="p-6 flex flex-col items-center gap-4">
            {liveStatus === 'paid' ? (
              <>
                <div className="w-14 h-14 rounded-2xl bg-[#E9F3D8] flex items-center justify-center">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                </div>
                <p className="font-display font-bold text-[20px] text-[#121216]">Payment confirmed</p>
                <p className="text-[13px] text-[#63636E]">{naira(intent.amountKobo)} added to your wallet.</p>
                <Button variant="primary" size="md" onClick={() => router.push('/wallet')} className="w-full">Back to wallet</Button>
              </>
            ) : (
              <>
                <div className="w-14 h-14 rounded-2xl bg-[#E9F3D8] flex items-center justify-center">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="5" y="2" width="14" height="20" rx="2" ry="2" /><line x1="12" y1="18" x2="12.01" y2="18" /></svg>
                </div>
                <p className="font-display font-bold text-[20px] text-[#121216]">Dial this code</p>
                <div className="bg-[#F7F5EF] rounded-xl px-6 py-4 w-full text-center">
                  <p className="font-mono text-[22px] font-bold text-[#0A3D2C] tracking-wider">{intent.ussdCode}</p>
                </div>
                <p className="text-[12px] text-[#63636E] text-center">
                  Dial on your phone · {intent.bankName} · {naira(intent.amountKobo)}<br />
                  Expires {new Date(intent.expiresAt).toLocaleTimeString('en-NG', { hour: '2-digit', minute: '2-digit' })}
                </p>
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
                  <span className="text-[12px] text-[#63636E]">Waiting for payment…</span>
                </div>
                <Button variant="ghost" size="sm" onClick={() => setIntent(null)} className="w-full">Start over</Button>
              </>
            )}
          </ListCard>
        ) : (
          <ListCard className="flex flex-col gap-4">
            <p className="text-[14px] font-semibold text-[#121216]">Fund your wallet with USSD</p>
            <p className="text-[12px] text-[#63636E]">No internet needed on your phone. Dial the code we give you.</p>

            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-[#63636E]">Your bank</label>
              {banksLoading ? (
                <div className="h-12 bg-[#F7F5EF] rounded-xl animate-pulse" />
              ) : (
                <select
                  required
                  value={bankCode}
                  onChange={(e) => setBankCode(e.target.value)}
                  className="w-full border border-[#E4E0D6] rounded-xl px-4 py-3 text-[14px] text-[#121216] focus:outline-none focus:border-[#0A3D2C] transition-colors bg-white"
                >
                  <option value="">Select bank</option>
                  {banks.map((b) => <option key={b.code} value={b.code}>{b.name}</option>)}
                </select>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-[#63636E]">Amount (₦)</label>
              <div className="relative">
                <span className="absolute left-4 top-1/2 -translate-y-1/2 text-[15px] font-semibold text-[#9A968D]">₦</span>
                <input
                  type="number"
                  min="100"
                  step="50"
                  required
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  placeholder="100"
                  className="w-full border border-[#E4E0D6] rounded-xl pl-8 pr-4 py-3 text-[15px] text-[#121216] placeholder-[#C5C2BB] focus:outline-none focus:border-[#0A3D2C] transition-colors" aria-label="Amount (₦)"/>
              </div>
            </div>

            {initiate.isError && <p className="text-xs text-red-600" role="alert">{(initiate.error as Error).message}</p>}

            <Button type="submit" variant="primary" size="md" isLoading={initiate.isPending} disabled={!bankCode || !amount} className="w-full">
              Get USSD code
            </Button>
          </ListCard>
        )}
      </div>
    </main>
  );
}
