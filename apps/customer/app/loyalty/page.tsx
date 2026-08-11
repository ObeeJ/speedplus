'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { loyaltyApi, type LoyaltyEvent } from '@speedplus/api-client';
import { Skeleton, ListCard } from '@speedplus/ui';

const EVENT_LABELS: Record<string, { label: string; positive: boolean }> = {
  order_completed:  { label: 'Order completed',   positive: true  },
  referral_bonus:   { label: 'Referral bonus',     positive: true  },
  signup_bonus:     { label: 'Welcome bonus',      positive: true  },
  redeemed:         { label: 'Points redeemed',    positive: false },
};

function fmt(iso: string) {
  return new Date(iso).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric' });
}

export default function LoyaltyPage() {
  const router = useRouter();

  const { data: balance, isLoading: balLoading } = useQuery({
    queryKey: ['loyalty-balance'],
    queryFn: () => loyaltyApi.getBalance(),
    staleTime: 30_000,
  });

  const { data: historyData, isLoading: histLoading } = useQuery({
    queryKey: ['loyalty-history'],
    queryFn: () => loyaltyApi.getHistory(),
    staleTime: 30_000,
  });

  const events = historyData?.events ?? [];

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      {/* Header */}
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-8 relative overflow-hidden">
        <div className="absolute inset-0 opacity-[0.06]" style={{ backgroundImage: 'radial-gradient(circle at 70% 30%, #C6F24E 0%, transparent 55%)' }} />
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-6">
            <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
            </button>
            <h1 className="font-display font-semibold text-white text-[18px]">Loyalty points</h1>
          </div>

          <div className="flex items-end gap-3">
            <div>
              <p className="text-[11px] font-semibold text-white/50 tracking-[0.8px] uppercase mb-1">Your points</p>
              {balLoading ? (
                <Skeleton className="h-14 w-32 bg-white/10" />
              ) : (
                <p className="font-display font-bold text-[56px] text-[#C6F24E] leading-none">
                  {(balance?.points ?? 0).toLocaleString()}
                </p>
              )}
            </div>
            <div className="mb-2 flex items-center gap-1.5 bg-white/10 rounded-xl px-3 py-1.5">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="#C6F24E" stroke="none"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" /></svg>
              <span className="text-[12px] font-semibold text-white/80">SpeedPlus Rewards</span>
            </div>
          </div>

          <p className="text-[12px] text-white/50 mt-2">Earn points on every order. Redeem for discounts.</p>
        </div>
      </div>

      {/* How to earn */}
      <div className="px-5 py-5 max-w-[600px] mx-auto w-full">
        <div className="grid grid-cols-3 gap-3 mb-6">
          {[
            { icon: '📦', label: 'Place an order', pts: '+10 pts' },
            { icon: '👥', label: 'Refer a friend', pts: '+50 pts' },
            { icon: '⭐', label: 'Leave a review', pts: '+5 pts' },
          ].map((item) => (
            <ListCard key={item.label} className="p-3.5 flex flex-col items-center gap-2 text-center">
              <span className="text-2xl" role="img" aria-label={item.label}>{item.icon}</span>
              <p className="text-[11px] text-[#63636E] leading-snug">{item.label}</p>
              <span className="text-[11px] font-bold text-[#0A3D2C] bg-[#E9F3D8] rounded-full px-2 py-0.5">{item.pts}</span>
            </ListCard>
          ))}
        </div>

        {/* History */}
        <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase mb-3">History</p>
        <ListCard noPadding>
          {histLoading && (
            <div className="p-4 flex flex-col gap-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-3">
                  <Skeleton className="w-9 h-9 rounded-xl" />
                  <div className="flex-1 flex flex-col gap-1.5"><Skeleton className="h-3.5 w-32" /><Skeleton className="h-3 w-20" /></div>
                  <Skeleton className="h-4 w-12" />
                </div>
              ))}
            </div>
          )}
          {!histLoading && events.length === 0 && (
            <div className="px-5 py-10 flex flex-col items-center gap-2 text-center">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#D5D2C8" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" /></svg>
              <p className="text-[13px] font-semibold text-[#121216]">No points yet</p>
              <p className="text-[12px] text-[#63636E]">Place your first order to start earning.</p>
            </div>
          )}
          {events.map((ev: LoyaltyEvent, i) => {
            const meta = EVENT_LABELS[ev.eventType] ?? { label: ev.eventType.replace(/_/g, ' '), positive: ev.points > 0 };
            return (
              <div key={ev.id} className={`flex items-center gap-3 px-4 py-3.5 ${i < events.length - 1 ? 'border-b border-[#F7F5EF]' : ''}`}>
                <div className={`w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 ${meta.positive ? 'bg-[#E9F3D8]' : 'bg-[#FEF2F2]'}`}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke={meta.positive ? '#0A3D2C' : '#DC2626'} strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                    {meta.positive
                      ? <><line x1="12" y1="19" x2="12" y2="5" /><polyline points="5 12 12 5 19 12" /></>
                      : <><line x1="12" y1="5" x2="12" y2="19" /><polyline points="19 12 12 19 5 12" /></>}
                  </svg>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-[13px] font-medium text-[#121216] capitalize">{meta.label}</p>
                  <p className="text-[11px] text-[#9A968D] mt-0.5">{fmt(ev.createdAt)}</p>
                </div>
                <p className={`text-[13px] font-bold flex-shrink-0 ${meta.positive ? 'text-[#0A3D2C]' : 'text-[#DC2626]'}`}>
                  {meta.positive ? '+' : ''}{ev.points} pts
                </p>
              </div>
            );
          })}
        </ListCard>
      </div>
    </main>
  );
}
