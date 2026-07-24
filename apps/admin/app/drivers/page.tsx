'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type DriverRow } from '@speedplus/api-client';
import { Badge } from '@speedplus/ui';

const STATUS_FILTERS = ['', 'pending', 'under_review', 'approved', 'suspended'] as const;

export default function DriversPage() {
  const [drivers, setDrivers] = useState<DriverRow[]>([]);
  const [filter, setFilter] = useState<string>('pending');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    setError('');
    adminApi.listDrivers(filter || undefined)
      .then((d) => setDrivers(d.drivers))
      .catch((e: Error) => setError(e.message));
  }, [filter]);

  function handleStatus(id: string, status: 'approved' | 'suspended' | 'under_review') {
    const reason = status === 'suspended' ? window.prompt('Suspension reason:') ?? '' : '';
    startTransition(async () => {
      try {
        await adminApi.setDriverStatus(id, status, reason);
        setDrivers((prev) => prev.map((d) => (d.id === id ? { ...d, status } : d)));
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-8 py-7 flex flex-col gap-4">
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
      {error && <p className="text-sm text-red-600">{error}</p>}
      {drivers.map((d) => (
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
