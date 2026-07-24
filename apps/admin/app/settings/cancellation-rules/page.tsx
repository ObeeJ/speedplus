'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type CancellationRule } from '@speedplus/api-client';

const EMPTY: Omit<CancellationRule, 'id'> = {
  vertical: '',
  orderStatusAtCancel: '',
  merchantCompKobo: 0,
  merchantCompPct: 0,
  riderCompPctOfDelivery: 0,
  fullRefund: false,
};

export default function CancellationRulesPage() {
  const [rules, setRules] = useState<CancellationRule[]>([]);
  const [form, setForm] = useState(EMPTY);
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  function load() {
    adminApi.listCancellationRules()
      .then((d) => setRules(d.rules))
      .catch((e: Error) => setError(e.message));
  }

  useEffect(load, []);

  function handleSave() {
    startTransition(async () => {
      try {
        await adminApi.upsertCancellationRule(form);
        setForm(EMPTY);
        load();
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  function handleDelete(id: string) {
    if (!window.confirm('Delete this rule?')) return;
    startTransition(async () => {
      try {
        await adminApi.deleteCancellationRule(id);
        setRules((prev) => prev.filter((r) => r.id !== id));
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-8 py-7 flex flex-col gap-6">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Cancellation Rules</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}

      {/* Upsert form */}
      <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
        <h2 className="font-semibold text-sm">Add / Update Rule</h2>
        <div className="grid grid-cols-2 gap-3">
          {(['vertical', 'orderStatusAtCancel'] as const).map((k) => (
            <div key={k} className="flex flex-col gap-1">
              <label className="text-[11px] font-semibold text-mid uppercase tracking-wide">{k}</label>
              <input
                value={form[k]}
                onChange={(e) => setForm((f) => ({ ...f, [k]: e.target.value }))}
                className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
              />
            </div>
          ))}
          {(['merchantCompKobo', 'merchantCompPct', 'riderCompPctOfDelivery'] as const).map((k) => (
            <div key={k} className="flex flex-col gap-1">
              <label className="text-[11px] font-semibold text-mid uppercase tracking-wide">{k}</label>
              <input
                type="number"
                value={form[k]}
                onChange={(e) => setForm((f) => ({ ...f, [k]: Number(e.target.value) }))}
                className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
              />
            </div>
          ))}
          <div className="flex items-center gap-2 col-span-2">
            <input
              type="checkbox"
              id="fullRefund"
              checked={form.fullRefund}
              onChange={(e) => setForm((f) => ({ ...f, fullRefund: e.target.checked }))}
              className="w-4 h-4 accent-emerald"
            />
            <label htmlFor="fullRefund" className="text-sm">Full refund</label>
          </div>
        </div>
        <button
          disabled={isPending || !form.vertical || !form.orderStatusAtCancel}
          onClick={handleSave}
          className="self-start font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] px-5 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
        >
          Save Rule
        </button>
      </div>

      {/* Rules table */}
      <div className="bg-white border border-line rounded-2xl overflow-hidden">
        <div className="grid px-5 py-3 border-b border-[#EFECE3] text-[10.5px] font-semibold text-mid tracking-[.5px]"
          style={{ gridTemplateColumns: '1fr 1fr 80px 80px 80px 60px 48px' }}>
          <span>VERTICAL</span><span>STATUS AT CANCEL</span>
          <span>COMP ₦</span><span>COMP %</span><span>RIDER %</span>
          <span>FULL</span><span />
        </div>
        {rules.map((r) => (
          <div key={r.id} className="grid px-5 py-3 text-[12.5px] items-center border-b border-[#EFECE3] last:border-0"
            style={{ gridTemplateColumns: '1fr 1fr 80px 80px 80px 60px 48px' }}>
            <span>{r.vertical}</span>
            <span>{r.orderStatusAtCancel}</span>
            <span>{r.merchantCompKobo / 100}</span>
            <span>{r.merchantCompPct}</span>
            <span>{r.riderCompPctOfDelivery}</span>
            <span>{r.fullRefund ? 'Yes' : 'No'}</span>
            <button
              disabled={isPending}
              onClick={() => handleDelete(r.id)}
              className="text-[11px] font-semibold text-red-600 hover:underline disabled:opacity-50"
            >
              Delete
            </button>
          </div>
        ))}
        {rules.length === 0 && <p className="px-5 py-4 text-sm text-mid">No rules configured.</p>}
      </div>
    </div>
  );
}
