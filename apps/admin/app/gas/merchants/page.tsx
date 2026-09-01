'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type GasMerchantRow, type FillStatus } from '@fourdat/api-client';

const FILL_FILTERS: Array<{ value: FillStatus | ''; label: string }> = [
  { value: '',           label: 'All'        },
  { value: 'good',       label: 'Good'       },
  { value: 'warned',     label: 'Warned'     },
  { value: 'probation',  label: 'Probation'  },
  { value: 'delisted',   label: 'Delisted'   },
];

const FILL_COLORS: Record<FillStatus, { bg: string; text: string }> = {
  good:      { bg: '#EEFADE', text: '#3A7D0A' },
  warned:    { bg: '#FEF3C7', text: '#92400E' },
  probation: { bg: '#FEE2E2', text: '#991B1B' },
  delisted:  { bg: '#F3F4F6', text: '#6B7280' },
};

const OVERRIDE_OPTIONS: FillStatus[] = ['good', 'warned', 'probation', 'delisted'];

export default function GasMerchantsPage() {
  const [merchants, setMerchants] = useState<GasMerchantRow[]>([]);
  const [filter, setFilter] = useState<FillStatus | ''>('');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    adminApi.listGasMerchants(filter || undefined)
      .then((d) => { setError(''); setMerchants(d.merchants); })
      .catch((e: Error) => setError(e.message));
  }, [filter]);

  function handleOverride(id: string, current: FillStatus) {
    const next = window.prompt(
      `Override fill status (current: ${current})\nOptions: ${OVERRIDE_OPTIONS.join(', ')}`,
    )?.trim() as FillStatus | undefined;
    if (!next || !OVERRIDE_OPTIONS.includes(next) || next === current) return;
    const reason = window.prompt('Reason (required):')?.trim() ?? '';
    if (!reason) return;
    startTransition(async () => {
      try {
        await adminApi.setMerchantFillStatus(id, next, reason);
        setMerchants((prev) =>
          prev.map((m) => (m.id === id ? { ...m, fillStatus: next } : m)),
        );
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Gas Merchants — Fill Accuracy</h1>

      <div className="flex gap-2 flex-wrap">
        {FILL_FILTERS.map(({ value, label }) => (
          <button
            key={value}
            onClick={() => setFilter(value)}
            className={`text-[11px] font-bold rounded-full px-3 py-1 transition-colors ${
              filter === value
                ? 'bg-[#0A3D2C] text-sand'
                : 'bg-white border border-line text-mid hover:bg-sand'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <div className="flex flex-col gap-2">
        {merchants.length === 0 && !error && (
          <p className="text-sm text-mid">No merchants found.</p>
        )}
        {merchants.map((m) => {
          const colors = FILL_COLORS[m.fillStatus];
          const accuracy = m.fillAccuracyPct != null
            ? `${(m.fillAccuracyPct * 100).toFixed(1)}%`
            : '—';
          return (
            <div
              key={m.id}
              className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4"
            >
              <div className="flex-1 flex flex-col gap-0.5">
                <span className="text-[13.5px] font-semibold">{m.businessName}</span>
                <span className="text-[11px] text-mid">
                  Accuracy: {accuracy} · {m.fillSampleCount} sample{m.fillSampleCount !== 1 ? 's' : ''}
                </span>
              </div>
              <span
                className="text-[11px] font-bold rounded-full px-3 py-1"
                style={{ background: colors.bg, color: colors.text }}
              >
                {m.fillStatus}
              </span>
              <button
                disabled={isPending}
                onClick={() => handleOverride(m.id, m.fillStatus)}
                className="font-display text-xs font-semibold rounded-[10px] px-4 py-2 border-[1.5px] border-line text-mid hover:border-[#0A3D2C] hover:text-[#0A3D2C] transition-colors disabled:opacity-50"
              >
                Override
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
