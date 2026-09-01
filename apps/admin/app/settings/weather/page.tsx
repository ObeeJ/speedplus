'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi } from '@fourdat/api-client';
import { FuelIcon } from '@fourdat/ui';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 2 })}`;
}

export default function WeatherSurchargePage() {
  const [enabled, setEnabled] = useState(false);
  const [amountKobo, setAmountKobo] = useState(0);
  const [draft, setDraft] = useState('');
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    adminApi.getWeatherSurcharge()
      .then((d) => { setEnabled(d.enabled); setAmountKobo(d.amountKobo); setDraft(String(d.amountKobo / 100)); })
      .catch((e: Error) => setError(e.message));
  }, []);

  function handleSave() {
    const kobo = Math.round(Number(draft) * 100);
    if (isNaN(kobo) || kobo < 0) { setError('Enter a valid amount.'); return; }
    setError('');
    startTransition(async () => {
      try {
        await adminApi.setWeatherSurcharge({ enabled, amountKobo: kobo, reason: 'Admin update' });
        setAmountKobo(kobo);
        setSaved(true);
        setTimeout(() => setSaved(false), 2500);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to save');
      }
    });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6 max-w-lg">
      {/* Header */}
      <div className="flex items-center gap-3">
        <div className="w-9 h-9 rounded-[11px] flex items-center justify-center" style={{ background: '#FFF7E6' }}>
          <FuelIcon size={18} color="#8A6A1B" accent="#E8B14E" />
        </div>
        <div>
          <h1 className="font-display font-semibold text-[22px] tracking-tight text-ink">Weather Surcharge</h1>
          <p className="text-[12px] text-mid">Applied to all orders when enabled</p>
        </div>
      </div>

      {error && (
        <div className="bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-4 py-3 text-sm text-[#DC2626]">{error}</div>
      )}
      {saved && (
        <div className="bg-[#E9F3D8] border border-[#C6F24E]/40 rounded-xl px-4 py-3 text-sm text-[#0A3D2C] font-semibold">Saved.</div>
      )}

      <div className="bg-white border border-line rounded-2xl p-6 flex flex-col gap-5">
        {/* Toggle */}
        <div className="flex items-center justify-between">
          <div>
            <p className="font-display font-semibold text-[14px] text-ink">Surcharge active</p>
            <p className="text-[12px] text-mid mt-0.5">
              {enabled ? `Currently adding ${naira(amountKobo)} to every order` : 'No surcharge applied'}
            </p>
          </div>
          <button
            onClick={() => setEnabled((v) => !v)}
            className={`relative w-11 h-6 rounded-full transition-colors ${enabled ? 'bg-[#C6F24E]' : 'bg-[#E4E0D6]'}`}
          >
            <span
              className={`absolute top-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform ${enabled ? 'translate-x-5' : 'translate-x-0.5'}`}
            />
          </button>
        </div>

        {/* Amount */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[11.5px] font-semibold text-mid uppercase tracking-[0.5px]">Surcharge amount (₦)</label>
          <div className="relative">
            <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-[14px] font-semibold text-mid">₦</span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              className="w-full h-11 rounded-[11px] border border-line bg-[#F7F5EF] pl-8 pr-4 text-[14px] text-ink focus:outline-none focus:border-[#0A3D2C]/40" aria-label="Surcharge amount (₦)"/>
          </div>
        </div>

        <button
          disabled={isPending}
          onClick={handleSave}
          className="font-display text-[13px] font-semibold text-[#0A3D2C] bg-[#C6F24E] rounded-[11px] h-11 hover:bg-[#AEE032] transition-colors disabled:opacity-50"
        >
          {isPending ? 'Saving…' : 'Save changes'}
        </button>
      </div>
    </div>
  );
}
