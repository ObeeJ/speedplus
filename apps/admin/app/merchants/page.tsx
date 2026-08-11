'use client';

import { useState, useTransition } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { adminApi, type MerchantRow } from '@speedplus/api-client';
import { Badge, Skeleton } from '@speedplus/ui';

const STATUS_FILTERS = ['', 'pending', 'active', 'suspended'] as const;

export default function MerchantsPage() {
  const [filter, setFilter] = useState<string>('pending');
  const [actionError, setActionError] = useState('');
  const [isPending, startTransition] = useTransition();
  const queryClient = useQueryClient();

  // useQuery rather than useEffect + setState — see drivers/page.tsx for the
  // reasoning: satisfies react-hooks/set-state-in-effect and removes the
  // uncancelled stale-response race on rapid filter switching.
  const queryKey = ['admin-merchants', filter] as const;
  const { data, isPending: isLoading, error } = useQuery({
    queryKey,
    queryFn: () => adminApi.listMerchants(filter || undefined),
  });
  const merchants: MerchantRow[] = data?.merchants ?? [];
  const errorMessage = actionError || (error instanceof Error ? error.message : '');

  function handleStatus(id: string, status: 'active' | 'suspended') {
    const reason = status === 'suspended' ? window.prompt('Suspension reason:') ?? '' : '';
    startTransition(async () => {
      try {
        setActionError('');
        await adminApi.setMerchantStatus(id, status, reason);
        queryClient.setQueryData<{ merchants: MerchantRow[] }>(queryKey, (prev) =>
          prev ? { ...prev, merchants: prev.merchants.map((m) => (m.id === id ? { ...m, status } : m)) } : prev,
        );
      } catch (e: unknown) {
        setActionError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Merchants</h1>
      <div className="flex gap-2">
        {STATUS_FILTERS.map((s) => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`text-[11px] font-bold rounded-full px-3 py-1 transition-colors ${
              filter === s
                ? 'bg-[#0A3D2C] text-sand'
                : 'bg-white border border-line text-mid hover:bg-sand'
            }`}
          >
            {s || 'All'}
          </button>
        ))}
      </div>
      {errorMessage && <p className="text-sm text-red-600">{errorMessage}</p>}
      {isLoading && (
        <div className="flex flex-col gap-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-[72px] rounded-2xl" />)}
        </div>
      )}
      {!isLoading && merchants.map((m) => (
        <div key={m.id} className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4">
          <div className="flex-1 flex flex-col gap-0.5">
            <span className="text-[13.5px] font-semibold">{m.businessName}</span>
            <span className="text-[11px] text-mid">{m.vertical} · ★ {m.rating.toFixed(1)}</span>
          </div>
          <Badge>{m.status}</Badge>
          <div className="flex gap-2">
            {m.status !== 'active' && (
              <button
                disabled={isPending}
                onClick={() => handleStatus(m.id, 'active')}
                className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
              >
                Approve
              </button>
            )}
            {m.status !== 'suspended' && (
              <button
                disabled={isPending}
                onClick={() => handleStatus(m.id, 'suspended')}
                className="font-display text-xs font-semibold rounded-[10px] px-4 py-2 border-[1.5px] transition-colors disabled:opacity-50"
                style={{ color: '#B4231F', borderColor: '#E5B5B3' }}
              >
                Suspend
              </button>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
