'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi } from '@fourdat/api-client';
import { SubscriptionIcon } from '@fourdat/ui';

type SubscriptionRow = Awaited<ReturnType<typeof adminApi.listSubscriptions>>['subscriptions'][number];
type SubStatus = 'active' | 'paused' | 'cancelled';

const STATUS_META: Record<SubStatus, { label: string; bg: string; text: string; dot: string }> = {
  active:    { label: 'Active',    bg: '#C6F24E22', text: '#0A3D2C', dot: '#C6F24E' },
  paused:    { label: 'Paused',    bg: '#FFF7E6',   text: '#8A6A1B', dot: '#E8B14E' },
  cancelled: { label: 'Cancelled', bg: '#FEF2F2',   text: '#B4231F', dot: '#DC2626' },
};

function getStatusMeta(status: string): typeof STATUS_META[SubStatus] {
  return STATUS_META[status as SubStatus] ?? STATUS_META.active;
}

const VERTICAL_COLORS: Record<string, string> = {
  gas:      '#E9F3D8',
  food:     '#FFF7E6',
  pharmacy: '#EEF2FF',
  grocery:  '#F0FDF4',
  package:  '#F5F3FF',
};

const FILTERS: { value: string; label: string }[] = [
  { value: '',          label: 'All' },
  { value: 'active',    label: 'Active' },
  { value: 'paused',    label: 'Paused' },
  { value: 'cancelled', label: 'Cancelled' },
];

export default function SubscriptionsPage() {
  const [subs, setSubs] = useState<SubscriptionRow[]>([]);
  const [filter, setFilter] = useState('');
  const [error, setError] = useState('');
  const [, startTransition] = useTransition();

  useEffect(() => {
    startTransition(async () => {
      try {
        const d = await adminApi.listSubscriptions(filter || undefined);
        setError('');
        setSubs(d.subscriptions ?? []);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load subscriptions');
      }
    });
  }, [filter]);

  const visible = filter ? subs.filter((s) => s.status === filter) : subs;

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-[11px] flex items-center justify-center" style={{ background: '#E9F3D8' }}>
            <SubscriptionIcon size={18} color="#0A3D2C" accent="#7BA05B" />
          </div>
          <div>
            <h1 className="font-display font-semibold text-[22px] tracking-tight text-ink">Subscriptions</h1>
            <p className="text-[12px] text-mid">Recurring delivery plans</p>
          </div>
        </div>
        <span className="font-display text-[13px] font-semibold text-mid">
          {visible.length} subscription{visible.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Filters */}
      <div className="flex gap-2">
        {FILTERS.map(({ value, label }) => (
          <button
            key={value}
            onClick={() => setFilter(value)}
            className={`text-[11.5px] font-semibold rounded-full px-3.5 py-1.5 transition-colors border ${
              filter === value
                ? 'bg-[#0A3D2C] text-[#F7F5EF] border-[#0A3D2C]'
                : 'bg-white text-mid border-line hover:border-[#0A3D2C]/30 hover:text-ink'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {error && (
        <div className="bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-4 py-3 text-sm text-[#DC2626]">
          {error}
        </div>
      )}

      {visible.length === 0 && !error ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center" style={{ background: '#E9F3D8' }}>
            <SubscriptionIcon size={26} color="#0A3D2C" accent="#7BA05B" />
          </div>
          <p className="text-[13px] text-mid text-center max-w-[240px]">
            No subscriptions found. Customers create recurring plans from the app.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {visible.map((sub) => {
            const meta = getStatusMeta(sub.status);
            const vertBg = VERTICAL_COLORS[sub.vertical] ?? '#F7F5EF';
            return (
              <div
                key={sub.id}
                className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4"
              >
                {/* Vertical chip */}
                <div
                  className="shrink-0 rounded-[9px] px-2.5 py-1 text-[10.5px] font-bold text-ink capitalize"
                  style={{ background: vertBg }}
                >
                  {sub.vertical}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="font-display font-semibold text-[13.5px] text-ink truncate">
                      {sub.merchantName ?? sub.merchantId.slice(0, 8)}
                    </span>
                    <span
                      className="text-[10.5px] font-bold rounded-full px-2.5 py-0.5"
                      style={{ background: meta.bg, color: meta.text }}
                    >
                      {meta.label}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-[11.5px] text-mid">
                    <span className="capitalize">{sub.frequency}</span>
                    {sub.nextRunAt && (
                      <span>
                        Next:{' '}
                        {new Date(sub.nextRunAt).toLocaleDateString('en-NG', {
                          day: 'numeric', month: 'short',
                        })}
                      </span>
                    )}
                    <span className="font-mono text-[11px]">{sub.customerId.slice(0, 8)}</span>
                  </div>
                </div>

                {/* Status dot */}
                <div className="w-2 h-2 rounded-full shrink-0" style={{ background: meta.dot }} />
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
