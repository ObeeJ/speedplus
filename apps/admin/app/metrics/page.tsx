'use client';

import { useEffect, useState } from 'react';
import { adminApi, type OperationalMetrics } from '@speedplus/api-client';
import {
  MetricsIcon,
  ReceiptIcon,
  WalletIcon,
  DriverIcon,
  StoreIcon,
  DisputeIcon,
  StatCard,
  StatCardSkeleton,
  iconColors,
} from '@speedplus/ui';

function naira(kobo: number) {
  if (kobo >= 100_000_00) return `₦${(kobo / 100_000_00).toFixed(1)}M`;
  if (kobo >= 100_000)    return `₦${(kobo / 100_000).toFixed(1)}K`;
  return `₦${(kobo / 100).toLocaleString('en-NG', { maximumFractionDigits: 0 })}`;
}

type StatItem = {
  label: string;
  value: string;
  sub?: string;
  iconBg: string;
  iconColor: string;
  iconAccent: string;
  Icon: typeof MetricsIcon;
};

export default function MetricsPage() {
  const [m, setM] = useState<OperationalMetrics | null>(null);
  const [error, setError] = useState('');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  function load() {
    adminApi.getMetrics()
      .then((data) => { setM(data); setLastUpdated(new Date()); setError(''); })
      .catch((e: Error) => setError(e.message));
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 60_000);
    return () => clearInterval(id);
  }, []);

  const cards: StatItem[] = m ? [
    {
      label: 'Orders today',
      value: m.ordersToday.toLocaleString(),
      sub: 'across all verticals',
      iconBg: 'bg-tile',
      Icon: ReceiptIcon,
      iconColor: iconColors.emerald,
      iconAccent: iconColors.accent,
    },
    {
      label: 'GMV',
      value: naira(m.gmvKobo),
      sub: 'gross merchandise value',
      iconBg: 'bg-lime/10',
      Icon: WalletIcon,
      iconColor: iconColors.emerald,
      iconAccent: iconColors.accent,
    },
    {
      label: 'Revenue',
      value: naira(m.revenueKobo),
      sub: 'platform service fees',
      iconBg: 'bg-tile',
      Icon: MetricsIcon,
      iconColor: iconColors.accent,
      iconAccent: iconColors.accent,
    },
    {
      label: 'Active drivers',
      value: m.activeDrivers.toLocaleString(),
      sub: 'online right now',
      iconBg: 'bg-amber/20',
      Icon: DriverIcon,
      iconColor: iconColors.amberDeep,
      iconAccent: iconColors.amberDeep,
    },
    {
      label: 'Active merchants',
      value: m.activeMerchants.toLocaleString(),
      sub: 'open right now',
      iconBg: 'bg-indigo-50',
      Icon: StoreIcon,
      iconColor: iconColors.indigo,
      iconAccent: iconColors.indigo,
    },
    {
      label: 'Cancellation rate',
      value: `${(m.cancellationRate * 100).toFixed(1)}%`,
      sub: `${m.cancellations} cancelled today`,
      iconBg: m.cancellationRate > 0.1 ? 'bg-red-50' : 'bg-sand',
      Icon: DisputeIcon,
      iconColor: m.cancellationRate > 0.1 ? iconColors.danger : iconColors.mid,
      iconAccent: m.cancellationRate > 0.1 ? iconColors.danger : iconColors.mid,
    },
  ] : [];

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-[11px] flex items-center justify-center bg-tile">
            <MetricsIcon size={18} color={iconColors.emerald} accent={iconColors.accent} />
          </div>
          <div>
            <h1 className="font-display font-semibold text-[22px] tracking-tight text-ink">Operations</h1>
            <p className="text-[12px] text-mid">Live metrics — today</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {lastUpdated && (
            <span className="text-[11px] text-mid">
              Updated {lastUpdated.toLocaleTimeString('en-NG', { hour: '2-digit', minute: '2-digit' })}
            </span>
          )}
          <button
            onClick={load}
            className="font-display text-[12px] font-semibold text-emerald border border-emerald/25 rounded-[10px] px-3.5 py-1.5 hover:bg-emerald/[.06] transition-colors"
          >
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-xl px-4 py-3 text-sm text-red-600">
          {error}
        </div>
      )}

      {/* Stat grid */}
      {m ? (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          {cards.map((c) => (
            <StatCard
              key={c.label}
              label={c.label}
              value={c.value}
              sub={c.sub}
              icon={
                <div className={`w-8 h-8 rounded-[9px] flex items-center justify-center ${c.iconBg}`}>
                  <c.Icon size={15} color={c.iconColor} accent={c.iconAccent} />
                </div>
              }
            />
          ))}
        </div>
      ) : !error ? (
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => <StatCardSkeleton key={i} />)}
        </div>
      ) : null}

      {/* Failed payments callout */}
      {m && m.failedPayments > 0 && (
        <div className="flex items-center gap-3 rounded-2xl px-5 py-4 border bg-red-50 border-red-200">
          <div className="w-8 h-8 rounded-[9px] flex items-center justify-center bg-red-100">
            <WalletIcon size={15} color={iconColors.danger} accent={iconColors.dangerAlt} />
          </div>
          <div>
            <p className="font-display font-semibold text-[13.5px] text-red-700">
              {m.failedPayments} failed payment{m.failedPayments !== 1 ? 's' : ''} today
            </p>
            <p className="text-[11.5px] text-red-700/70">Check the ledger for provider errors.</p>
          </div>
        </div>
      )}
    </div>
  );
}
