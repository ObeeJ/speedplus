'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button } from '@fourdat/ui';
import { subscriptionsApi, usersApi, type Subscription } from '@fourdat/api-client';

// Seeded in migration 022 — same constant used in gas/price/page.tsx
const GAS_MERCHANT_ID = '00000000-0000-0000-0000-000000000004';

const CADENCE_LABELS: Record<Subscription['cadence'], string> = {
  weekly: 'Every week',
  biweekly: 'Every 2 weeks',
  monthly: 'Every month',
};

export default function SubscriptionsPage() {
  const router = useRouter();
  const qc = useQueryClient();

  const [creating, setCreating] = useState(false);
  const [cadence, setCadence] = useState<Subscription['cadence']>('monthly');
  const [addressId, setAddressId] = useState('');
  const [formError, setFormError] = useState('');

  // Fetch subscriptions from server; fall back to empty array on 404 (endpoint pending)
  const { data: subs = [], refetch: refetchSubs } = useQuery({
    queryKey: ['subscriptions'],
    queryFn: () => subscriptionsApi.list().catch(() => [] as Subscription[]),
    staleTime: 30_000,
  });

  const { data: addresses = [] } = useQuery({
    queryKey: ['addresses'],
    queryFn: () => usersApi.listAddresses(),
    staleTime: 30_000,
  });

  const create = useMutation({
    mutationFn: () => subscriptionsApi.create({
      merchantId: GAS_MERCHANT_ID,
      vertical: 'gas',
      cadence,
      addressId,
      paymentMethod: 'wallet',
    }),
    onSuccess: () => {
      setCreating(false);
      setFormError('');
      void refetchSubs();
      qc.invalidateQueries({ queryKey: ['addresses'] });
    },
    onError: (e: Error) => setFormError(e.message),
  });

  const pause = useMutation({
    mutationFn: (id: string) => subscriptionsApi.pause(id),
    onSuccess: () => void refetchSubs(),
  });

  const cancel = useMutation({
    mutationFn: (id: string) => subscriptionsApi.cancel(id),
    onSuccess: () => void refetchSubs(),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!addressId) { setFormError('Select a delivery address.'); return; }
    setFormError('');
    create.mutate();
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="px-5 pt-12 pb-4 flex items-center justify-between">
        <button onClick={() => router.back()} className="text-mid hover:text-ink transition-colors" aria-label="Back">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5M12 5l-7 7 7 7" /></svg>
        </button>
        <h1 className="font-display font-semibold text-[17px] text-ink">Auto-refill</h1>
        <div className="w-5" />
      </div>

      <div className="flex-1 px-5 py-4 flex flex-col gap-4 min-[700px]:max-w-[640px] min-[700px]:mx-auto min-[700px]:w-full">
        <p className="text-[13px] text-mid">Never run out of gas. We refill automatically on your schedule.</p>

        {subs.length === 0 && !creating && (
          <Button variant="primary" size="md" onClick={() => setCreating(true)} className="w-full">
            Set up auto-refill
          </Button>
        )}

        {subs.map((s) => (
          <div key={s.id} className="bg-white border border-line rounded-2xl px-4 py-3 flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-[13px] font-semibold text-ink capitalize">{CADENCE_LABELS[s.cadence]}</span>
              <span className={`text-[11px] font-semibold px-2 py-0.5 rounded-full ${s.status === 'active' ? 'bg-emerald/10 text-emerald' : 'bg-line text-mid'}`}>
                {s.status}
              </span>
            </div>
            <p className="text-[11px] text-mid">Next charge: {new Date(s.nextChargeAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric' })}</p>
            {s.status === 'active' && (
              <div className="flex gap-2 pt-1">
                <button onClick={() => pause.mutate(s.id)} disabled={pause.isPending} className="text-[12px] text-mid hover:text-ink underline">
                  Pause
                </button>
                <button onClick={() => cancel.mutate(s.id)} disabled={cancel.isPending} className="text-[12px] text-red-500 hover:underline">
                  Cancel
                </button>
              </div>
            )}
          </div>
        ))}

        {creating && (
          <form onSubmit={handleSubmit} className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <span className="text-[12px] font-semibold text-mid">How often?</span>
              <div className="flex flex-col gap-2">
                {(['weekly', 'biweekly', 'monthly'] as const).map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setCadence(c)}
                    className={`text-left rounded-xl border-2 px-3 py-2.5 text-[13px] transition-all ${cadence === c ? 'border-emerald bg-emerald/5 text-ink font-semibold' : 'border-line text-mid'}`}
                  >
                    {CADENCE_LABELS[c]}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <span className="text-[12px] font-semibold text-mid">Deliver to</span>
              {addresses.length === 0 ? (
                <p className="text-[12px] text-mid">No saved addresses. <a href="/profile" className="text-emerald underline">Add one first.</a></p>
              ) : (
                <select
                  aria-label="Deliver to address"
                  required
                  value={addressId}
                  onChange={(e) => setAddressId(e.target.value)}
                  className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink focus:outline-none focus:border-emerald"
                >
                  <option value="">Select address</option>
                  {addresses.map((a) => (
                    <option key={a.id} value={a.id}>{a.label || a.street}, {a.city}</option>
                  ))}
                </select>
              )}
            </div>

            {formError && <p className="text-xs text-red-600" role="alert">{formError}</p>}

            <div className="flex gap-2">
              <Button type="submit" variant="primary" size="sm" isLoading={create.isPending} className="flex-1">
                Confirm
              </Button>
              <Button type="button" variant="ghost" size="sm" onClick={() => { setCreating(false); setFormError(''); }} className="flex-1">
                Cancel
              </Button>
            </div>
          </form>
        )}
      </div>
    </main>
  );
}
