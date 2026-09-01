'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { Button, ListCard } from '@fourdat/ui';
import { walletApi } from '@fourdat/api-client';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

const PRESETS = [500, 1000, 2000, 5000];

export default function TransferPage() {
  const router = useRouter();
  const [phone, setPhone] = useState('');
  const [amount, setAmount] = useState('');
  const [pin, setPin] = useState('');
  const [done, setDone] = useState<{ recipientName: string; amountKobo: number } | null>(null);

  const transfer = useMutation({
    mutationFn: () => walletApi.transfer(
      { phone: phone.trim(), amountKobo: Math.round(parseFloat(amount) * 100), pin },
      `transfer-${Date.now()}`,
    ),
    onSuccess: (data) => setDone({ recipientName: data.recipientName, amountKobo: Math.round(parseFloat(amount) * 100) }),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const kobo = Math.round(parseFloat(amount) * 100);
    if (!phone.trim() || !kobo || kobo < 10000 || pin.length < 4) return;
    transfer.mutate();
  }

  if (done) {
    return (
      <main className="min-h-screen bg-sand flex flex-col items-center justify-center px-5 gap-5">
        <div className="w-16 h-16 rounded-2xl bg-tile flex items-center justify-center">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
        </div>
        <div className="text-center">
          <p className="font-display font-bold text-[22px] text-ink">{naira(done.amountKobo)} sent</p>
          <p className="text-[13px] text-mid mt-1">to {done.recipientName}</p>
        </div>
        <Button variant="primary" size="md" onClick={() => router.push('/wallet')} className="w-full max-w-[320px]">Back to wallet</Button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="bg-emerald px-5 pt-12 pb-6">
        <div className="flex items-center gap-3">
          <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Send money</h1>
        </div>
      </div>

      <div className="flex-1 px-5 py-6 max-w-[480px] mx-auto w-full flex flex-col gap-5">
        <ListCard className="flex flex-col gap-4">

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-semibold text-mid">Recipient phone number</label>
            <input
              type="tel"
              inputMode="tel"
              required
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="08012345678"
              className="w-full border border-line rounded-xl px-4 py-3 text-[15px] text-ink placeholder-[#C5C2BB] focus:outline-none focus:border-emerald transition-colors" aria-label="Recipient phone number"/>
          </div>

          <div className="flex flex-col gap-2">
            <label className="text-[12px] font-semibold text-mid">Amount</label>
            <div className="grid grid-cols-4 gap-2">
              {PRESETS.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setAmount(String(p))}
                  className={`rounded-xl border-2 py-2 text-[13px] font-semibold transition-all ${amount === String(p) ? 'border-emerald bg-tile text-emerald' : 'border-line text-mid hover:border-emerald/30'}`}
                >
                  ₦{p.toLocaleString()}
                </button>
              ))}
            </div>
            <div className="relative">
              <span className="absolute left-4 top-1/2 -translate-y-1/2 text-[15px] font-semibold text-[#9A968D]">₦</span>
              <input
                type="number"
                min="100"
                step="50"
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
                placeholder="100"
                className="w-full border border-line rounded-xl pl-8 pr-4 py-3 text-[15px] text-ink placeholder-[#C5C2BB] focus:outline-none focus:border-emerald transition-colors" aria-label="100"/>
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-semibold text-mid">Wallet PIN</label>
            <input
              type="password"
              inputMode="numeric"
              maxLength={4}
              required
              value={pin}
              onChange={(e) => setPin(e.target.value.replace(/\D/g, ''))}
              placeholder="••••"
              className="w-full border border-line rounded-xl px-4 py-3 text-[20px] text-ink placeholder-[#C5C2BB] tracking-widest focus:outline-none focus:border-emerald transition-colors" aria-label="Wallet PIN"/>
          </div>

          {transfer.isError && <p className="text-xs text-red-600" role="alert">{(transfer.error as Error).message}</p>}

          <Button
            type="submit"
            variant="primary"
            size="md"
            isLoading={transfer.isPending}
            disabled={!phone.trim() || !amount || parseFloat(amount) < 100 || pin.length < 4}
            className="w-full"
          >
            Send {amount ? `₦${parseFloat(amount).toLocaleString('en-NG')}` : 'money'}
          </Button>
        </ListCard>

        <p className="text-[11px] text-[#9A968D] text-center">
          Transfers are instant and irreversible. Double-check the phone number before sending.
        </p>
      </div>
    </main>
  );
}
