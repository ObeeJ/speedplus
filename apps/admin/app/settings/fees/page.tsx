'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type FeeConfig, type FuelSuggestion } from '@fourdat/api-client';

const VERTICALS = ['food', 'grocery', 'pharmacy', 'gas', 'package'] as const;

type EditState = {
  baseFeeNaira: number;
  perKmNaira: number;
  perKgNaira: number;
  servicePct: number; // percent, e.g. 5
  merchantCommissionPct: number; // percent, e.g. 8 => merchantTakeRate 0.92
  driverSharePct: number; // percent, e.g. 80
  fuelPriceNaira: number; // ₦/litre reference
  reason: string;
};

function toEdit(c: FeeConfig): EditState {
  return {
    baseFeeNaira: c.baseFeeKobo / 100,
    perKmNaira: c.perKmKobo / 100,
    perKgNaira: c.perKgKobo / 100,
    servicePct: Math.round(c.servicePct * 10000) / 100,
    merchantCommissionPct: Math.round((1 - c.merchantTakeRate) * 10000) / 100,
    driverSharePct: Math.round(c.driverTakeRate * 10000) / 100,
    fuelPriceNaira: c.fuelPriceRefKobo / 100,
    reason: '',
  };
}

export default function FeeConfigsPage() {
  const [configs, setConfigs] = useState<FeeConfig[]>([]);
  const [edits, setEdits] = useState<Record<string, EditState>>({});
  const [suggestions, setSuggestions] = useState<Record<string, FuelSuggestion>>({});
  const [error, setError] = useState('');
  const [saved, setSaved] = useState('');
  const [isPending, startTransition] = useTransition();

  function load() {
    adminApi.listFeeConfigs()
      .then((d) => {
        setConfigs(d.configs);
        const next: Record<string, EditState> = {};
        for (const c of d.configs) next[c.vertical] = toEdit(c);
        setEdits(next);
      })
      .catch((e: Error) => setError(e.message));
  }

  useEffect(load, []);

  function setField(vertical: string, patch: Partial<EditState>) {
    setEdits((prev) => {
      const cur = prev[vertical];
      if (!cur) return prev;
      return { ...prev, [vertical]: { ...cur, ...patch } };
    });
  }

  function handleSave(vertical: string) {
    const e = edits[vertical];
    if (!e) return;
    setError('');
    setSaved('');
    startTransition(async () => {
      try {
        const res = await adminApi.upsertFeeConfig({
          vertical,
          baseFeeKobo: Math.round(e.baseFeeNaira * 100),
          perKmKobo: Math.round(e.perKmNaira * 100),
          perKgKobo: Math.round(e.perKgNaira * 100),
          servicePct: e.servicePct / 100,
          merchantTakeRate: 1 - e.merchantCommissionPct / 100,
          driverTakeRate: e.driverSharePct / 100,
          platformTakeRate: 1 - e.driverSharePct / 100,
          fuelPriceRefKobo: Math.round(e.fuelPriceNaira * 100),
          reason: e.reason,
        });
        if (res.fuelSuggestion) {
          setSuggestions((prev) => ({ ...prev, [vertical]: res.fuelSuggestion! }));
        } else {
          setSuggestions((prev) => {
            const { [vertical]: _drop, ...rest } = prev;
            return rest;
          });
        }
        setSaved(vertical);
        load();
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : 'Failed');
      }
    });
  }

  function applySuggestion(vertical: string) {
    const s = suggestions[vertical];
    if (!s) return;
    // Pre-fill only — the admin still reviews and saves to make it live.
    setField(vertical, { perKmNaira: s.suggestedPerKmKobo / 100 });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="font-display font-semibold text-[26px] tracking-tight">Delivery Fees & Commissions</h1>
        <p className="text-sm text-mid">
          Changes are versioned and take effect on new quotes within ~1 minute. In-flight orders settle at the
          rates in force when they were created.
        </p>
      </div>
      {error && <p className="text-sm text-red-600">{error}</p>}

      {VERTICALS.map((vertical) => {
        const cfg = configs.find((c) => c.vertical === vertical);
        const e = edits[vertical];
        const s = suggestions[vertical];
        if (!e) return null;
        return (
          <div key={vertical} className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-3">
            <div className="flex items-baseline justify-between">
              <h2 className="font-semibold text-sm capitalize">{vertical}</h2>
              {cfg?.id ? (
                <span className="text-[11px] text-mid">
                  effective {new Date(cfg.effectiveAt).toLocaleString()} · by {cfg.updatedBy.slice(0, 8)}
                  {cfg.reason ? ` · "${cfg.reason}"` : ''}
                </span>
              ) : (
                <span className="text-[11px] text-mid">platform defaults — never changed</span>
              )}
            </div>

            {s && (
              <div className="flex items-center justify-between bg-sand border border-line rounded-xl px-4 py-3">
                <p className="text-[12.5px]">
                  Fuel reference moved ₦{s.prevFuelKobo / 100} → ₦{s.newFuelKobo / 100} (&gt;10%). Suggested
                  per-km: <strong>₦{s.suggestedPerKmKobo / 100}</strong> (currently ₦{s.currentPerKmKobo / 100}).
                </p>
                <button
                  onClick={() => applySuggestion(vertical)}
                  className="text-[11.5px] font-semibold text-emerald hover:underline shrink-0 ml-3"
                >
                  Pre-fill per-km
                </button>
              </div>
            )}

            <div className="grid grid-cols-4 gap-3">
              {(
                [
                  ['baseFeeNaira', 'Base fee (₦)'],
                  ['perKmNaira', 'Per km (₦)'],
                  ['perKgNaira', 'Per kg (₦)'],
                  ['servicePct', 'Service fee (%)'],
                  ['merchantCommissionPct', 'Merchant commission (%)'],
                  ['driverSharePct', 'Rider share of delivery (%)'],
                  ['fuelPriceNaira', 'Fuel reference (₦/L)'],
                ] as const
              ).map(([k, label]) => (
                <div key={k} className="flex flex-col gap-1">
                  <label className="text-[11px] font-semibold text-mid uppercase tracking-wide">{label}</label>
                  <input
                    type="number"
                    min={0}
                    value={e[k]}
                    aria-label={label}
                    onChange={(ev) => setField(vertical, { [k]: Number(ev.target.value) } as Partial<EditState>)}
                    className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                  />
                </div>
              ))}
              <div className="flex flex-col gap-1">
                <label className="text-[11px] font-semibold text-mid uppercase tracking-wide">Reason (required)</label>
                <input
                  value={e.reason}
                  placeholder="e.g. fuel price adjustment"
                  onChange={(ev) => setField(vertical, { reason: ev.target.value })}
                  className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="Reason (required)"/>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <button
                disabled={isPending || !e.reason.trim()}
                onClick={() => handleSave(vertical)}
                className="self-start font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] px-5 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
              >
                Save {vertical} fees
              </button>
              {saved === vertical && !isPending && (
                <span className="text-[12px] text-emerald font-semibold">Saved ✓</span>
              )}
            </div>
          </div>
        );
      })}
      {configs.length === 0 && !error && <p className="text-sm text-mid">Loading…</p>}

      {/* ── Weather surcharge ─────────────────────────────────────────────── */}
      <WeatherSurchargeCard />
    </div>
  );
}

function WeatherSurchargeCard() {
  const [enabled, setEnabled] = useState(false);
  const [amountNaira, setAmountNaira] = useState('200');
  const [reason, setReason] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    adminApi.getWeatherSurcharge()
      .then((d) => { setEnabled(d.enabled); setAmountNaira(String(d.amountKobo / 100)); })
      .catch(() => {});
  }, []);

  async function handleSave() {
    if (!reason.trim()) return;
    setSaving(true);
    setError('');
    try {
      await adminApi.setWeatherSurcharge({
        enabled,
        amountKobo: Math.round(parseFloat(amountNaira) * 100),
        reason: reason.trim(),
      });
      setSaved(true);
      setReason('');
      setTimeout(() => setSaved(false), 3000);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="font-display font-semibold text-[15px]">Weather surcharge</p>
          <p className="text-[11px] text-mid mt-0.5">
            Applies when Open-Meteo reports rain, storm, or extreme heat. Takes effect on new quotes within ~1 minute.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setEnabled((v) => !v)}
          className="w-11 h-6 flex-none rounded-full relative transition-colors"
          style={{ background: enabled ? '#0A3D2C' : '#D5D2C8' }}
          aria-label={enabled ? 'Disable weather surcharge' : 'Enable weather surcharge'}
        >
          <span
            className="absolute top-[3px] w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-all"
            style={{ left: enabled ? 23 : 3 }}
          />
        </button>
      </div>

      <div className="flex items-center gap-3">
        <div className="flex flex-col gap-1 flex-1">
          <label className="text-[11px] font-semibold text-mid uppercase tracking-wide">Amount (₦)</label>
          <input
            type="number"
            min="0"
            value={amountNaira}
            onChange={(e) => setAmountNaira(e.target.value)}
            className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="Amount (₦)"/>
        </div>
        <div className="flex flex-col gap-1 flex-[2]">
          <label className="text-[11px] font-semibold text-mid uppercase tracking-wide">Reason (required)</label>
          <input
            value={reason}
            placeholder="e.g. heavy rain season"
            onChange={(e) => setReason(e.target.value)}
            className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="Reason (required)"/>
        </div>
      </div>

      {error && <p className="text-xs text-red-600">{error}</p>}

      <div className="flex items-center gap-3">
        <button
          disabled={saving || !reason.trim()}
          onClick={handleSave}
          className="self-start font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] px-5 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
        >
          Save weather settings
        </button>
        {saved && <span className="text-[12px] text-emerald font-semibold">Saved ✓</span>}
      </div>
    </div>
  );
}
