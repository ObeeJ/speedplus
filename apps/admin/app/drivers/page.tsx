'use client';

import { useState, useTransition } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { adminApi, type DriverRow } from '@speedplus/api-client';
import { Badge, Skeleton } from '@speedplus/ui';

const STATUS_FILTERS = ['', 'pending', 'under_review', 'approved', 'suspended'] as const;

export default function DriversPage() {
  const [filter, setFilter] = useState<string>('pending');
  const [actionError, setActionError] = useState('');
  const [isPending, startTransition] = useTransition();
  const queryClient = useQueryClient();

  // useQuery rather than useEffect + setState. Beyond satisfying
  // react-hooks/set-state-in-effect (the synchronous setLoading(true) in the
  // effect body), this removes a real stale-response race: the old code had no
  // cancellation, so switching filters quickly could let a slower earlier
  // request resolve last and paint the wrong list under the current filter.
  const queryKey = ['admin-drivers', filter] as const;
  const { data, isPending: isLoading, error } = useQuery({
    queryKey,
    queryFn: () => adminApi.listDrivers(filter || undefined),
  });
  const drivers: DriverRow[] = data?.drivers ?? [];
  const errorMessage = actionError || (error instanceof Error ? error.message : '');

  function handleStatus(id: string, status: 'approved' | 'suspended' | 'under_review') {
    const reason = status === 'suspended' ? window.prompt('Suspension reason:') ?? '' : '';
    startTransition(async () => {
      try {
        setActionError('');
        await adminApi.setDriverStatus(id, status, reason);
        // Patch the cached list so the row updates without a refetch, keeping
        // the previous optimistic behaviour.
        queryClient.setQueryData<{ drivers: DriverRow[] }>(queryKey, (prev) =>
          prev ? { ...prev, drivers: prev.drivers.map((d) => (d.id === id ? { ...d, status } : d)) } : prev,
        );
      } catch (e: unknown) {
        setActionError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Drivers</h1>
      <div className="flex gap-2 flex-wrap">
        {STATUS_FILTERS.map((s) => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`text-[11px] font-bold rounded-full px-3 py-1 transition-colors ${
              filter === s ? 'bg-[#0A3D2C] text-sand' : 'bg-white border border-line text-mid hover:bg-sand'
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
      {!isLoading && drivers.map((d) => (
        <div key={d.id} className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4">
          <div className="flex-1 flex flex-col gap-0.5">
            <span className="text-[13.5px] font-semibold">{d.vehicleType} · {d.vehiclePlate}</span>
            <span className="text-[11px] text-mid">★ {d.rating.toFixed(1)} · {d.totalDeliveries} deliveries</span>
          </div>
          <Badge>{d.status}</Badge>
          <div className="flex gap-2">
            {d.status !== 'approved' && (
              <button
                disabled={isPending}
                onClick={() => handleStatus(d.id, 'approved')}
                className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
              >
                Approve
              </button>
            )}
            {d.status !== 'suspended' && (
              <button
                disabled={isPending}
                onClick={() => handleStatus(d.id, 'suspended')}
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
