'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type ZoneRow, type LaunchStatus } from '@speedplus/api-client';

const LAUNCH_FILTERS: Array<{ value: LaunchStatus | ''; label: string }> = [
  { value: '',         label: 'All'      },
  { value: 'piloting', label: 'Piloting' },
  { value: 'live',     label: 'Live'     },
  { value: 'paused',   label: 'Paused'   },
];

const LAUNCH_COLORS: Record<LaunchStatus, { bg: string; text: string }> = {
  piloting: { bg: '#EFF6FF', text: '#1D4ED8' },
  live:     { bg: '#EEFADE', text: '#3A7D0A' },
  paused:   { bg: '#FEF3C7', text: '#92400E' },
};

const OVERRIDE_OPTIONS: LaunchStatus[] = ['piloting', 'live', 'paused'];

function fmtWindow(minutesSinceMidnight: number): string {
  const h = Math.floor(minutesSinceMidnight / 60).toString().padStart(2, '0');
  const m = (minutesSinceMidnight % 60).toString().padStart(2, '0');
  return `${h}:${m}`;
}

export default function GasZonesPage() {
  const [zones, setZones] = useState<ZoneRow[]>([]);
  const [filter, setFilter] = useState<LaunchStatus | ''>('');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    adminApi.listZones(filter || undefined)
      .then((d) => { setError(''); setZones(d.zones); })
      .catch((e: Error) => setError(e.message));
  }, [filter]);

  function handleOverride(id: string, current: LaunchStatus) {
    const next = window.prompt(
      `Override launch status (current: ${current})\nOptions: ${OVERRIDE_OPTIONS.join(', ')}`,
    )?.trim() as LaunchStatus | undefined;
    if (!next || !OVERRIDE_OPTIONS.includes(next) || next === current) return;
    const reason = window.prompt('Reason (required):')?.trim() ?? '';
    if (!reason) return;
    startTransition(async () => {
      try {
        await adminApi.setZoneLaunchStatus(id, next, reason);
        setZones((prev) =>
          prev.map((z) => (z.id === id ? { ...z, launchStatus: next } : z)),
        );
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Gas Zones — Launch Status</h1>

      <div className="flex gap-2 flex-wrap">
        {LAUNCH_FILTERS.map(({ value, label }) => (
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
        {zones.length === 0 && !error && (
          <p className="text-sm text-mid">No zones found.</p>
        )}
        {zones.map((z) => {
          const colors = LAUNCH_COLORS[z.launchStatus];
          return (
            <div
              key={z.id}
              className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4"
            >
              <div className="flex-1 flex flex-col gap-0.5">
                <span className="text-[13.5px] font-semibold">{z.name}</span>
                <span className="text-[11px] text-mid">
                  Window: {fmtWindow(z.windowStart)}–{fmtWindow(z.windowEnd)} UTC
                  {!z.isActive && ' · inactive'}
                </span>
              </div>
              <span
                className="text-[11px] font-bold rounded-full px-3 py-1"
                style={{ background: colors.bg, color: colors.text }}
              >
                {z.launchStatus}
              </span>
              <button
                disabled={isPending}
                onClick={() => handleOverride(z.id, z.launchStatus)}
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
