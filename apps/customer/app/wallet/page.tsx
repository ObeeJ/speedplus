'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { walletApi, cardApi } from '@speedplus/api-client';
import { Skeleton } from '@speedplus/ui';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

function TxIcon({ positive }: { positive: boolean }) {
  return (
    <div className={`w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 ${positive ? 'bg-[#E9F3D8]' : 'bg-[#FEF2F2]'}`}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke={positive ? '#0A3D2C' : '#DC2626'} strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
        {positive
          ? <><line x1="12" y1="19" x2="12" y2="5" /><polyline points="5 12 12 5 19 12" /></>
          : <><line x1="12" y1="5" x2="12" y2="19" /><polyline points="19 12 12 19 5 12" /></>}
      </svg>
    </div>
  );
}

export default function WalletPage() {
  const router = useRouter();
  const [copied, setCopied] = useState(false);

  const { data: balance, isLoading: balLoading } = useQuery({ queryKey: ['wallet-balance'], queryFn: () => walletApi.getBalance(), staleTime: 15_000 });
  const { data: txData, isLoading: txLoading } = useQuery({ queryKey: ['wallet-transactions'], queryFn: () => walletApi.getTransactions(), staleTime: 30_000 });
  const { data: dva } = useQuery({ queryKey: ['virtual-account'], queryFn: () => cardApi.getVirtualAccount(), staleTime: Infinity });

  const transactions = (txData?.transactions ?? []) as Array<{ id: string; description: string; amountKobo: number; refType?: string; createdAt: string }>;

  function copyAccount() {
    if (!dva) return;
    navigator.clipboard?.writeText(dva.accountNumber).then(() => { setCopied(true); setTimeout(() => setCopied(false), 2000); });
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      {/* Header */}
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-8 relative overflow-hidden">
        <div className="absolute inset-0 opacity-[0.05]" style={{ backgroundImage: 'radial-gradient(circle at 80% 20%, #C6F24E 0%, transparent 60%)' }} />
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-6">
            <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
            </button>
            <h1 className="font-display font-semibold text-white text-[18px]">Wallet</h1>
          </div>

          <p className="text-[11px] font-semibold text-white/50 tracking-[0.8px] uppercase mb-1">Available balance</p>
          {balLoading ? (
            <Skeleton className="h-12 w-40 bg-white/10" />
          ) : (
            <p className="font-display font-bold text-[48px] text-[#C6F24E] leading-none" style={{ animation: 'fadeUp 0.3s cubic-bezier(0.16,1,0.3,1) both' }}>
              {naira(balance?.balanceKobo ?? 0)}
            </p>
          )}

          <div className="mt-5 flex gap-2 flex-wrap">
            <button
              onClick={() => router.push('/wallet/fund')}
              className="flex items-center gap-2 bg-[#C6F24E] text-[#0A3D2C] font-display font-semibold text-[13px] rounded-xl px-4 py-2.5 hover:bg-[#AEE032] transition-colors"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" /></svg>
              Add money
            </button>
            <button
              onClick={() => router.push('/wallet/transfer')}
              className="flex items-center gap-2 bg-white/10 text-white font-semibold text-[13px] rounded-xl px-4 py-2.5 hover:bg-white/20 transition-colors"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><line x1="22" y1="2" x2="11" y2="13" /><polygon points="22 2 15 22 11 13 2 9 22 2" /></svg>
              Send
            </button>
            <button
              onClick={() => router.push('/wallet/ussd')}
              className="flex items-center gap-2 bg-white/10 text-white font-semibold text-[13px] rounded-xl px-4 py-2.5 hover:bg-white/20 transition-colors"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><rect x="5" y="2" width="14" height="20" rx="2" ry="2" /><line x1="12" y1="18" x2="12.01" y2="18" /></svg>
              USSD
            </button>
            <button
              onClick={() => router.push('/wallet/payment-links')}
              className="flex items-center gap-2 bg-white/10 text-white font-semibold text-[13px] rounded-xl px-4 py-2.5 hover:bg-white/20 transition-colors"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71" /><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71" /></svg>
              Pay link
            </button>
          </div>
        </div>
      </div>

      <div className="flex-1 px-5 py-5 flex flex-col gap-5 max-w-[600px] mx-auto w-full">

        {/* DVA */}
        {dva && (
          <div className="bg-white rounded-2xl border border-[#E4E0D6] p-5 flex flex-col gap-3">
            <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">Fund via bank transfer</p>
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="font-display font-bold text-[22px] text-[#121216] tracking-wider">{dva.accountNumber}</p>
                <p className="text-[12px] text-[#63636E] mt-0.5">{dva.bankName}</p>
              </div>
              <button
                onClick={copyAccount}
                className={`flex items-center gap-1.5 text-[12px] font-semibold rounded-xl px-3.5 py-2 transition-all ${copied ? 'bg-[#E9F3D8] text-[#0A3D2C]' : 'border border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30 hover:text-[#0A3D2C]'}`}
              >
                {copied ? (
                  <><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>Copied</>
                ) : (
                  <><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2" /><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" /></svg>Copy</>
                )}
              </button>
            </div>
            <p className="text-[11px] text-[#9A968D]">Transfer any amount — credited instantly to your wallet.</p>
          </div>
        )}

        {/* Transactions */}
        <div className="flex flex-col gap-3">
          <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">Transactions</p>
          <div className="bg-white rounded-2xl border border-[#E4E0D6] overflow-hidden">
            {txLoading && (
              <div className="p-4 flex flex-col gap-3">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="flex items-center gap-3">
                    <Skeleton className="w-9 h-9 rounded-xl" />
                    <div className="flex-1 flex flex-col gap-1.5">
                      <Skeleton className="h-3.5 w-36" />
                      <Skeleton className="h-3 w-20" />
                    </div>
                    <Skeleton className="h-4 w-16" />
                  </div>
                ))}
              </div>
            )}
            {!txLoading && transactions.length === 0 && (
              <div className="px-5 py-10 flex flex-col items-center gap-2 text-center">
                <div className="w-10 h-10 rounded-xl bg-[#F7F5EF] flex items-center justify-center text-[#9A968D]">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" /></svg>
                </div>
                <p className="text-[13px] font-semibold text-[#121216]">No transactions yet</p>
                <p className="text-[12px] text-[#63636E]">Add money to get started.</p>
              </div>
            )}
            {transactions.map((tx, i) => {
              const positive = tx.amountKobo > 0;
              return (
                <div key={tx.id} className={`flex items-center gap-3 px-4 py-3.5 ${i < transactions.length - 1 ? 'border-b border-[#F7F5EF]' : ''}`}>
                  <TxIcon positive={positive} />
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-medium text-[#121216] truncate">{tx.description}</p>
                    <p className="text-[11px] text-[#9A968D] mt-0.5">
                      {new Date(tx.createdAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}
                    </p>
                  </div>
                  <p className={`text-[13px] font-semibold flex-shrink-0 ${positive ? 'text-[#0A3D2C]' : 'text-[#DC2626]'}`}>
                    {positive ? '+' : ''}{naira(tx.amountKobo)}
                  </p>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(8px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
