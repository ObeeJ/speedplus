'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@speedplus/ui';
import { giftCardsApi } from '@speedplus/api-client';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

const PRESETS = [500000, 1000000, 2000000, 5000000]; // ₦5k, ₦10k, ₦20k, ₦50k

export default function GiftCardsPage() {
  const router = useRouter();
  const [tab, setTab] = useState<'redeem' | 'issue'>('redeem');
  const [redeemCode, setRedeemCode] = useState('');
  const [issueAmount, setIssueAmount] = useState('');
  const [issued, setIssued] = useState<{ code: string; amountKobo: number } | null>(null);
  const [copied, setCopied] = useState(false);

  const redeem = useMutation({
    mutationFn: (code: string) => giftCardsApi.redeem(code),
    onSuccess: () => { setRedeemCode(''); },
  });

  const issue = useMutation({
    mutationFn: (kobo: number) => giftCardsApi.issue(kobo),
    onSuccess: (data) => setIssued({ code: data.code, amountKobo: data.card.amountKobo }),
  });

  function handleRedeem(e: React.FormEvent) {
    e.preventDefault();
    if (!redeemCode.trim()) return;
    redeem.mutate(redeemCode.trim().toUpperCase());
  }

  function handleIssue(e: React.FormEvent) {
    e.preventDefault();
    const kobo = Math.round(parseFloat(issueAmount) * 100);
    if (!kobo || kobo < 50000) return;
    issue.mutate(kobo);
  }

  function copyCode() {
    if (!issued) return;
    navigator.clipboard?.writeText(issued.code).then(() => { setCopied(true); setTimeout(() => setCopied(false), 2000); });
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-6">
        <div className="flex items-center gap-3 mb-4">
          <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Gift cards</h1>
        </div>
        <div className="flex gap-1 bg-white/10 rounded-xl p-1">
          {(['redeem', 'issue'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`flex-1 rounded-lg py-2 text-[13px] font-semibold transition-all ${tab === t ? 'bg-white text-[#0A3D2C]' : 'text-white/60 hover:text-white'}`}
            >
              {t === 'redeem' ? 'Redeem a card' : 'Send a card'}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 px-5 py-6 max-w-[480px] mx-auto w-full flex flex-col gap-5">
        {tab === 'redeem' ? (
          <>
            <div className="bg-white rounded-2xl border border-[#E4E0D6] p-5 flex flex-col gap-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-[#E9F3D8] flex items-center justify-center">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="1" y="4" width="22" height="16" rx="2" ry="2" /><line x1="1" y1="10" x2="23" y2="10" /></svg>
                </div>
                <div>
                  <p className="text-[14px] font-semibold text-[#121216]">Enter gift card code</p>
                  <p className="text-[12px] text-[#63636E]">Balance added to your wallet instantly</p>
                </div>
              </div>
              <form onSubmit={handleRedeem} className="flex flex-col gap-3">
                <input
                  value={redeemCode}
                  onChange={(e) => setRedeemCode(e.target.value.toUpperCase())}
                  placeholder="XXXX-XXXX-XXXX-XXXX"
                  className="w-full border border-[#E4E0D6] rounded-xl px-4 py-3 text-[15px] font-mono text-[#121216] placeholder-[#C5C2BB] tracking-widest focus:outline-none focus:border-[#0A3D2C] transition-colors"
                  maxLength={24}
                  aria-label="Gift card code"
                />
                {redeem.isError && <p className="text-xs text-red-600" role="alert">{(redeem.error as Error).message}</p>}
                {redeem.isSuccess && <p className="text-xs text-[#0A3D2C] font-semibold" role="status">✓ Card redeemed — balance added to your wallet</p>}
                <Button type="submit" variant="primary" size="md" isLoading={redeem.isPending} disabled={!redeemCode.trim()} className="w-full">
                  Redeem
                </Button>
              </form>
            </div>
          </>
        ) : (
          <>
            {issued ? (
              <div className="bg-white rounded-2xl border border-[#E4E0D6] p-5 flex flex-col gap-4">
                <div className="flex flex-col items-center gap-3 py-4">
                  <div className="w-14 h-14 rounded-2xl bg-[#E9F3D8] flex items-center justify-center">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                  </div>
                  <p className="font-display font-bold text-[20px] text-[#121216]">{naira(issued.amountKobo)} gift card</p>
                  <p className="text-[12px] text-[#63636E]">Share this code with the recipient</p>
                  <div className="w-full bg-[#F7F5EF] rounded-xl px-4 py-3 flex items-center justify-between gap-3">
                    <p className="font-mono text-[16px] font-bold text-[#121216] tracking-widest">{issued.code}</p>
                    <button onClick={copyCode} className={`text-[12px] font-semibold rounded-lg px-3 py-1.5 transition-all ${copied ? 'bg-[#E9F3D8] text-[#0A3D2C]' : 'border border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30'}`}>
                      {copied ? 'Copied!' : 'Copy'}
                    </button>
                  </div>
                </div>
                <Button variant="ghost" size="md" onClick={() => { setIssued(null); setIssueAmount(''); }} className="w-full">
                  Send another
                </Button>
              </div>
            ) : (
              <div className="bg-white rounded-2xl border border-[#E4E0D6] p-5 flex flex-col gap-4">
                <p className="text-[14px] font-semibold text-[#121216]">Choose amount</p>
                <div className="grid grid-cols-2 gap-2">
                  {PRESETS.map((p) => (
                    <button
                      key={p}
                      onClick={() => setIssueAmount(String(p / 100))}
                      className={`rounded-xl border-2 py-3 text-[14px] font-semibold transition-all ${issueAmount === String(p / 100) ? 'border-[#0A3D2C] bg-[#E9F3D8] text-[#0A3D2C]' : 'border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30'}`}
                    >
                      {naira(p)}
                    </button>
                  ))}
                </div>
                <div className="flex items-center gap-2">
                  <div className="h-px flex-1 bg-[#E4E0D6]" />
                  <span className="text-[11px] text-[#9A968D]">or enter amount</span>
                  <div className="h-px flex-1 bg-[#E4E0D6]" />
                </div>
                <form onSubmit={handleIssue} className="flex flex-col gap-3">
                  <div className="relative">
                    <span className="absolute left-4 top-1/2 -translate-y-1/2 text-[15px] font-semibold text-[#9A968D]">₦</span>
                    <input
                      type="number"
                      min="500"
                      step="100"
                      value={issueAmount}
                      onChange={(e) => setIssueAmount(e.target.value)}
                      placeholder="500"
                      className="w-full border border-[#E4E0D6] rounded-xl pl-8 pr-4 py-3 text-[15px] text-[#121216] placeholder-[#C5C2BB] focus:outline-none focus:border-[#0A3D2C] transition-colors"
                    />
                  </div>
                  {issue.isError && <p className="text-xs text-red-600" role="alert">{(issue.error as Error).message}</p>}
                  <Button type="submit" variant="primary" size="md" isLoading={issue.isPending} disabled={!issueAmount || parseFloat(issueAmount) < 500} className="w-full">
                    Generate card
                  </Button>
                </form>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  );
}
