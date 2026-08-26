'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Skeleton } from '@speedplus/ui';
import { cylindersApi, gasApi, type RegisterCylinderInput } from '@speedplus/api-client';

const BLANK: RegisterCylinderInput = { specId: '', serial: '', manufactureYear: new Date().getFullYear() };

export default function CylindersPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState<RegisterCylinderInput>(BLANK);
  const [formError, setFormError] = useState('');

  const { data: cylinders = [], isLoading } = useQuery({
    queryKey: ['cylinders'],
    queryFn: () => cylindersApi.list(),
    staleTime: 30_000,
  });

  const { data: specs = [] } = useQuery({
    queryKey: ['gas-specs'],
    queryFn: () => gasApi.listSpecs(),
    staleTime: Infinity,
  });

  const register = useMutation({
    mutationFn: (input: RegisterCylinderInput) => cylindersApi.register(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cylinders'] });
      setAdding(false);
      setForm(BLANK);
      setFormError('');
    },
    onError: (e: Error) => setFormError(e.message),
  });

  const retire = useMutation({
    mutationFn: (id: string) => cylindersApi.retire(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cylinders'] }),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.specId || !form.serial.trim() || !form.manufactureYear) {
      setFormError('All fields are required.');
      return;
    }
    setFormError('');
    register.mutate({ ...form, serial: form.serial.trim() });
  }

  const specLabel = (specId: string) => specs.find((s) => s.id === specId)?.label ?? specId;

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="px-5 pt-12 pb-4 flex items-center justify-between">
        <button onClick={() => router.back()} className="text-mid hover:text-ink transition-colors" aria-label="Back">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5M12 5l-7 7 7 7" /></svg>
        </button>
        <h1 className="font-display font-semibold text-[17px] text-ink">My cylinders</h1>
        <div className="w-5" />
      </div>

      <div className="flex-1 px-5 py-4 flex flex-col gap-4 min-[700px]:max-w-[640px] min-[700px]:mx-auto min-[700px]:w-full">
        <div className="flex items-center justify-between">
          <span className="text-[13px] font-semibold text-mid">Registered cylinders</span>
          {!adding && (
            <button onClick={() => setAdding(true)} className="text-[12px] font-semibold text-emerald hover:underline">
              + Register
            </button>
          )}
        </div>

        {isLoading && (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-[58px] rounded-2xl" />
            <Skeleton className="h-[58px] rounded-2xl" />
          </div>
        )}

        {!isLoading && cylinders.length === 0 && !adding && (
          <p className="text-[13px] text-mid">No cylinders registered. Register one to use refill mode.</p>
        )}

        {cylinders.map((c) => (
          <div key={c.id} className="bg-white border border-line rounded-2xl px-4 py-3 flex items-center justify-between gap-3">
            <div>
              <p className="text-[13px] font-semibold text-ink">{specLabel(c.specId)} · {c.serial}</p>
              <p className="text-[11px] text-mid capitalize">{c.manufactureYear} · {c.status.replace('_', ' ')}</p>
            </div>
            {c.status === 'active' && (
              <button
                onClick={() => retire.mutate(c.id)}
                disabled={retire.isPending}
                className="text-[11px] text-red-500 hover:underline shrink-0"
              >
                Retire
              </button>
            )}
          </div>
        ))}

        {adding && (
          <form onSubmit={handleSubmit} className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-3">
            <select
              aria-label="Cylinder size"
              required
              value={form.specId}
              onChange={(e) => setForm((f) => ({ ...f, specId: e.target.value }))}
              className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink focus:outline-none focus:border-emerald"
            >
              <option value="">Select size *</option>
              {specs.map((s) => (
                <option key={s.id} value={s.id}>{s.label}</option>
              ))}
            </select>
            <input
              placeholder="Serial number *"
              required
              value={form.serial}
              onChange={(e) => setForm((f) => ({ ...f, serial: e.target.value }))}
              className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Serial number *"/>
            <input
              placeholder="Manufacture year *"
              type="number"
              required
              min={1990}
              max={new Date().getFullYear()}
              value={form.manufactureYear}
              onChange={(e) => setForm((f) => ({ ...f, manufactureYear: parseInt(e.target.value) || 0 }))}
              className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Manufacture year *"/>
            <input
              placeholder="Last recertification date (optional)"
              type="date"
              value={form.lastRecertAt ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, lastRecertAt: e.target.value || undefined }))}
              className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] text-ink focus:outline-none focus:border-emerald" aria-label="Last recertification date (optional)"/>
            {formError && <p className="text-xs text-red-600" role="alert">{formError}</p>}
            <div className="flex gap-2">
              <Button type="submit" variant="primary" size="sm" isLoading={register.isPending} className="flex-1">
                Register
              </Button>
              <Button type="button" variant="ghost" size="sm" onClick={() => { setAdding(false); setForm(BLANK); setFormError(''); }} className="flex-1">
                Cancel
              </Button>
            </div>
          </form>
        )}
      </div>
    </main>
  );
}
