'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi } from '@fourdat/api-client';
import { RunIcon, DriverIcon } from '@fourdat/ui';

type RunStatus = 'assembling' | 'dispatched' | 'in_progress' | 'completed' | 'cancelled';
type DeliveryRun = Awaited<ReturnType<typeof adminApi.listRuns>>['runs'][number];

const STATUS_META: Record<RunStatus, { label: string; bg: string; text: string; dot: string }> = {
  assembling:  { label: 'Assembling',  bg: '#FFF7E6', text: '#8A6A1B', dot: '#E8B14E' },
  dispatched:  { label: 'Dispatched',  bg: '#E9F3D8', text: '#0A3D2C', dot: '#7BA05B' },
  in_progress: { label: 'In Progress', bg: '#C6F24E22', text: '#0A3D2C', dot: '#C6F24E' },
  completed:   { label: 'Completed',   bg: '#EFECE3', text: '#63636E', dot: '#BDBAB2' },
  cancelled:   { label: 'Cancelled',   bg: '#FEF2F2', text: '#B4231F', dot: '#DC2626' },
};

function getRunMeta(status: string) {
  return STATUS_META[status as RunStatus] ?? STATUS_META.assembling;
}

const FILTERS: { value: string; label: string }[] = [
  { value: '',            label: 'All' },
  { value: 'assembling',  label: 'Assembling' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'dispatched',  label: 'Dispatched' },
  { value: 'completed',   label: 'Completed' },
];

function fmt(iso: string) {
  return new Date(iso).toLocaleTimeString('en-NG', { hour: '2-digit', minute: '2-digit' });
}
function fmtDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-NG', { day: 'numeric', month: 'short' });
}

export default function RunsPage() {
  const [runs, setRuns] = useState<DeliveryRun[]>([]);
  const [filter, setFilter] = useState('');
  const [error, setError] = useState('');
  const [, startTransition] = useTransition();

  useEffect(() => {
    startTransition(async () => {
      try {
        const d = await adminApi.listRuns(filter || undefined);
        setError('');
        setRuns(d.runs ?? []);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load runs');
      }
    });
  }, [filter]);

  const visible = filter ? runs.filter((r) => r.status === filter) : runs;

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-[11px] flex items-center justify-center" style={{ background: '#E9F3D8' }}>
            <RunIcon size={18} color="#0A3D2C" accent="#7BA05B" />
          </div>
          <div>
            <h1 className="font-display font-semibold text-[22px] tracking-tight text-ink">Delivery Runs</h1>
            <p className="text-[12px] text-mid">Zone-batched dispatch windows</p>
          </div>
        </div>
        <span className="font-display text-[13px] font-semibold text-mid">
          {visible.length} run{visible.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Filters */}
      <div className="flex gap-2 flex-wrap">
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

      {/* Runs list */}
      {visible.length === 0 && !error ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center" style={{ background: '#E9F3D8' }}>
            <RunIcon size={26} color="#0A3D2C" accent="#7BA05B" />
          </div>
          <p className="text-[13px] text-mid text-center max-w-[240px]">
            No delivery runs found. Runs are created automatically when orders are batched into a zone window.
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {visible.map((run) => {
            const meta = getRunMeta(run.status);
            return (
              <div
                key={run.id}
                className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4"
              >
                {/* Status dot */}
                <div
                  className="w-2 h-2 rounded-full shrink-0"
                  style={{ background: meta.dot }}
                />

                {/* Main info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="font-display font-semibold text-[13.5px] text-ink truncate">
                      Run {run.id.slice(0, 8).toUpperCase()}
                    </span>
                    <span
                      className="text-[10.5px] font-bold rounded-full px-2.5 py-0.5"
                      style={{ background: meta.bg, color: meta.text }}
                    >
                      {meta.label}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-[11.5px] text-mid">
                    <span>{fmtDate(run.windowStart)} · {fmt(run.windowStart)} – {fmt(run.windowEnd)}</span>
                    {run.orderCount != null && (
                      <span>{run.orderCount} order{run.orderCount !== 1 ? 's' : ''}</span>
                    )}
                    {run.totalDistanceKm > 0 && (
                      <span>{run.totalDistanceKm.toFixed(1)} km</span>
                    )}
                  </div>
                </div>

                {/* Driver */}
                <div className="flex items-center gap-1.5 text-[11.5px] text-mid shrink-0">
                  <DriverIcon size={14} color="#63636E" accent="#9A968D" />
                  {run.driverId ? (
                    <span className="font-mono">{run.driverId.slice(0, 8)}</span>
                  ) : (
                    <span className="text-[#E8B14E] font-semibold">Unassigned</span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
