'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@speedplus/ui';
import { paymentLinksApi } from '@speedplus/api-client';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function PaymentLinksPage() {
  const router = useRouter();
  const [amount, setAmount] = useState('');
  const [note, setNote] = useState('');
  const [created, setCreated] = useState<{ url: string; slug: string; amountKobo: number; note?: string } | null>(null);
  const [copied, setCopied] = useState(false);

  const create = useMutation({
    mutationFn: () => paymentLinksApi.create(
      { amountKobo: Math.round(parseFloat(amount) * 100), note: note.trim() || undefined },
      `pl-${Date.now()}`,
    ),
    onSuccess: (data) => setCreated(data),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const kobo = Math.round(parseFloat(amount) * 100);
    if (!kobo || kobo < 10000) return;
    create.mutate();
  }

  function copyLink() {
    if (!created) return;
    navigator.clipboard?.writeText(created.url).then(() => { setCopied(true); setTimeout(() => setCopied(false), 2000); });
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-6">
        <div className="flex items-center gap-3">
          <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Payment link</h1>
        </div>
      </div>

      <div className="flex-1 px-5 py-6 max-w-[480px] mx-auto w-full flex flex-col gap-5">
        {created ? (
          <div className="bg-white rounded-2xl border border-[#E4E0D6] p-6 flex flex-col items-center gap-4">
            <div className="w-14 h-14 rounded-2xl bg-[#E9F3D8] flex items-center justify-center">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71" /><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71" /></svg>
            </div>
            <div className="text-center">
              <p className="font-display font-bold text-[20px] text-[#121216]">{naira(created.amountKobo)}</p>
              {created.note && <p className="text-[13px] text-[#63636E] mt-0.5">{created.note}</p>}
            </div>
            <div className="w-full bg-[#F7F5EF] rounded-xl px-4 py-3 flex items-center justify-between gap-3">
              <p className="text-[12px] text-[#63636E] truncate flex-1">{created.url}</p>
              <button
                onClick={copyLink}
                className={`text-[12px] font-semibold rounded-lg px-3 py-1.5 flex-shrink-0 transition-all ${copied ? 'bg-[#E9F3D8] text-[#0A3D2C]' : 'border border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30'}`}
              >
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
            <p className="text-[11px] text-[#9A968D] text-center">Share this link. Anyone can pay you — no SpeedPlus account needed.</p>
            <div className="flex gap-2 w-full">
              <Button variant="ghost" size="sm" onClick={() => { setCreated(null); setAmount(''); setNote(''); }} className="flex-1">New link</Button>
              <Button variant="primary" size="sm" onClick={() => router.push('/wallet')} className="flex-1">Done</Button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="bg-white rounded-2xl border border-[#E4E0D6] p-5 flex flex-col gap-4">
            <p className="text-[14px] font-semibold text-[#121216]">Create a payment link</p>
            <p className="text-[12px] text-[#63636E]">Share the link — anyone can pay you, even without a SpeedPlus account.</p>

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
                  className="w-full border border-[#E4E0D6] rounded-xl pl-8 pr-4 py-3 text-[15px] text-[#121216] placeholder-[#C5C2BB] focus:outline-none focus:border-[#0A3D2C] transition-colors"
                />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-[#63636E]">Note (optional)</label>
              <input
                type="text"
                maxLength={80}
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="e.g. For the groceries"
                className="w-full border border-[#E4E0D6] rounded-xl px-4 py-3 text-[14px] text-[#121216] placeholder-[#C5C2BB] focus:outline-none focus:border-[#0A3D2C] transition-colors"
              />
            </div>

            {create.isError && <p className="text-xs text-red-600" role="alert">{(create.error as Error).message}</p>}

            <Button type="submit" variant="primary" size="md" isLoading={create.isPending} disabled={!amount || parseFloat(amount) < 100} className="w-full">
              Generate link
            </Button>
          </form>
        )}
      </div>
    </main>
  );
}
