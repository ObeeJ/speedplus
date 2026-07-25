'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { walletApi, cardApi } from '@speedplus/api-client';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

function TxRow({ tx }: { tx: { description: string; amountKobo: number; refType?: string; createdAt: string } }) {
  const positive = tx.amountKobo > 0;
  return (
    <div className="flex items-center gap-3 px-4 py-3 border-b border-[#EFECE3] last:border-0">
      <div className={`w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0 ${positive ? 'bg-tile' : 'bg-[#FEF2F2]'}`}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke={positive ? '#0A3D2C' : '#DC2626'} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
          {positive
            ? <path d="M12 19V5M5 12l7-7 7 7" />
            : <path d="M12 5v14M5 12l7 7 7-7" />}
        </svg>
      </div>
      <div className="flex-1 flex flex-col gap-0.5 min-w-0">
        <span className="text-[13px] font-medium text-ink truncate">{tx.description}</span>
        <span className="text-[11px] text-mid">{new Date(tx.createdAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })}</span>
      </div>
      <span className={`text-[13px] font-semibold flex-shrink-0 ${positive ? 'text-emerald' : 'text-[#DC2626]'}`}>
        {positive ? '+' : ''}{naira(tx.amountKobo)}
      </span>
    </div>
  );
}

export default function WalletPage() {
  const router = useRouter();

  const { data: balance, isLoading: balLoading } = useQuery({
    queryKey: ['wallet-balance'],
    queryFn: () => walletApi.getBalance(),
    staleTime: 15_000,
  });

  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: ['wallet-transactions'],
    queryFn: () => walletApi.getTransactions(),
    staleTime: 30_000,
  });

  const { data: dva } = useQuery({
    queryKey: ['virtual-account'],
    queryFn: () => cardApi.getVirtualAccount(),
    staleTime: Infinity,
  });

  const transactions = (txData?.transactions ?? []) as Array<{
    id: string;
    description: string;
    amountKobo: number;
    refType?: string;
    createdAt: string;
  }>;

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      {/* Header */}
      <div className="bg-emerald px-5 pt-12 pb-8 flex flex-col gap-1">
        <span className="text-[11px] font-semibold text-sand/60 tracking-widest uppercase">Wallet balance</span>
        {balLoading ? (
          <div className="h-10 w-36 bg-sand/10 rounded-xl animate-pulse" />
        ) : (
          <span
            className="font-display font-bold text-4xl text-lime"
            style={{ animation: 'fadeUp 0.3s cubic-bezier(0.16,1,0.3,1) both' }}
          >
            {naira(balance?.balanceKobo ?? 0)}
          </span>
        )}
        <button
          onClick={() => router.push('/wallet/fund')}
          className="mt-4 self-start font-display text-sm font-semibold text-emerald bg-lime rounded-[13px] px-6 py-2.5 hover:bg-lime-600 transition-colors"
        >
          Add money
        </button>
      </div>

      <div className="flex-1 px-5 py-5 flex flex-col gap-5 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:w-full">

        {/* DVA — bank transfer details */}
        {dva && (
          <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2">
            <span className="text-[11px] font-semibold text-mid tracking-widest uppercase">Fund via bank transfer</span>
            <div className="flex items-center justify-between">
              <div className="flex flex-col gap-0.5">
                <span className="font-display font-bold text-xl text-ink tracking-wide">{dva.accountNumber}</span>
                <span className="text-[12px] text-mid">{dva.bankName}</span>
              </div>
              <button
                onClick={() => navigator.clipboard?.writeText(dva.accountNumber)}
                className="text-[11px] font-semibold text-emerald border border-emerald/30 rounded-[10px] px-3 py-1.5 hover:bg-tile transition-colors"
              >
                Copy
              </button>
            </div>
            <span className="text-[11px] text-mid">Transfer any amount — credited instantly.</span>
          </div>
        )}

        {/* Transaction history */}
        <div className="flex flex-col gap-2">
          <span className="text-[11px] font-semibold text-mid tracking-widest uppercase">Transactions</span>
          <div className="bg-white border border-line rounded-2xl overflow-hidden">
            {txLoading && (
              <div className="flex flex-col gap-3 p-4">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="h-10 bg-line rounded-xl animate-pulse" />
                ))}
              </div>
            )}
            {!txLoading && transactions.length === 0 && (
              <p className="px-4 py-6 text-sm text-mid text-center">No transactions yet.</p>
            )}
            {transactions.map((tx) => <TxRow key={tx.id} tx={tx} />)}
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
