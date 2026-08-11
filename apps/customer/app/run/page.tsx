'use client';

import { useSearchParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { runsApi } from '@speedplus/api-client';
import { Skeleton, ListCard } from '@speedplus/ui';

const STATUS_COLORS: Record<string, { bg: string; text: string; label: string }> = {
  assembling:  { bg: '#FEF3C7', text: '#92400E', label: 'Assembling' },
  dispatched:  { bg: '#DBEAFE', text: '#1E40AF', label: 'Dispatched' },
  in_progress: { bg: '#E9F3D8', text: '#0A3D2C', label: 'In progress' },
  completed:   { bg: '#F0FDF4', text: '#166534', label: 'Completed' },
  cancelled:   { bg: '#FEF2F2', text: '#991B1B', label: 'Cancelled' },
};

function fmt(iso: string) {
  return new Date(iso).toLocaleTimeString('en-NG', { hour: '2-digit', minute: '2-digit' });
}

export default function RunPage() {
  const router = useRouter();
  const params = useSearchParams();
  const runId = params.get('id');

  const { data: run, isLoading, isError } = useQuery({
    queryKey: ['run', runId],
    queryFn: () => runsApi.get(runId!),
    enabled: !!runId,
    refetchInterval: (q) => q.state.data?.status === 'in_progress' ? 10_000 : false,
  });

  const sc = STATUS_COLORS[run?.status ?? ''] ?? STATUS_COLORS.assembling;

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-6">
        <div className="flex items-center gap-3 mb-4">
          <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Delivery run</h1>
        </div>
        {run && (
          <div className="flex items-center gap-3">
            <span className="text-[12px] font-bold rounded-full px-3 py-1" style={{ background: sc.bg, color: sc.text }}>{sc.label}</span>
            <span className="text-[12px] text-white/60">{fmt(run.windowStart)} – {fmt(run.windowEnd)}</span>
          </div>
        )}
      </div>

      <div className="flex-1 px-5 py-5 max-w-[600px] mx-auto w-full flex flex-col gap-4">
        {!runId && (
          <ListCard className="px-5 py-10 text-center">
            <p className="text-[13px] text-[#63636E]">No run ID provided.</p>
          </ListCard>
        )}

        {isLoading && (
          <div className="flex flex-col gap-3">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 rounded-2xl" />)}
          </div>
        )}

        {isError && (
          <ListCard className="px-5 py-10 text-center">
            <p className="text-[13px] text-red-600">Run not found.</p>
          </ListCard>
        )}

        {run && (
          <>
            <div className="grid grid-cols-3 gap-3">
              {[
                { label: 'Orders', value: run.orderCount },
                { label: 'Distance', value: `${run.totalDistanceKm.toFixed(1)} km` },
                { label: 'Window', value: `${fmt(run.windowStart)}–${fmt(run.windowEnd)}` },
              ].map((s) => (
                <ListCard key={s.label} className="p-3.5 flex flex-col gap-1">
                  <p className="text-[10px] font-semibold text-[#9A968D] uppercase tracking-[0.5px]">{s.label}</p>
                  <p className="font-display font-bold text-[16px] text-[#121216]">{s.value}</p>
                </ListCard>
              ))}
            </div>

            <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">Run ID</p>
            <ListCard className="px-4 py-3">
              <p className="font-mono text-[12px] text-[#63636E] break-all">{run.id}</p>
            </ListCard>
          </>
        )}
      </div>
    </main>
  );
}
