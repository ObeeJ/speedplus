'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { kycApi } from '@fourdat/api-client';

type DocType = 'bvn' | 'nin';

export default function DriverKYCPage() {
  const router = useRouter();
  const [docType, setDocType] = useState<DocType>('bvn');
  const [value, setValue] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [submitted, setSubmitted] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    if (!value.trim()) { setError('Please enter your ' + docType.toUpperCase()); return; }
    setLoading(true);
    try {
      if (docType === 'bvn') {
        await kycApi.submitBVN(value.trim());
      } else {
        await kycApi.submitNIN(value.trim());
      }
      setSubmitted(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Submission failed. Try again.');
    } finally {
      setLoading(false);
    }
  }

  if (submitted) {
    return (
      <main className="min-h-screen bg-sand flex flex-col items-center justify-center px-5 gap-5">
        <div className="w-16 h-16 rounded-full bg-lime flex items-center justify-center">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={3} strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </div>
        <div className="text-center flex flex-col gap-1.5">
          <p className="font-display font-bold text-lg text-ink">Verification submitted</p>
          <p className="text-[13px] text-mid">Our team will review your details. You&apos;ll be notified once approved.</p>
        </div>
        <button
          onClick={() => router.replace('/')}
          className="text-[13px] font-semibold text-emerald hover:underline"
        >
          Back to home
        </button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      {/* Header */}
      <div className="bg-emerald px-5 py-4 flex items-center gap-3">
        <button
          onClick={() => router.back()}
          className="w-8 h-8 rounded-full bg-sand/[.14] flex items-center justify-center"
          aria-label="Go back"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#F5F0E8" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <path d="M19 12H5M12 5l-7 7 7 7" />
          </svg>
        </button>
        <span className="font-display font-bold text-base text-sand">Identity verification</span>
      </div>

      <div className="flex-1 px-5 py-6 flex flex-col gap-5 max-w-[430px] mx-auto w-full">
        <p className="text-[13px] text-mid leading-relaxed">
          To start receiving deliveries, we need to verify your identity. This is a one-time step required by Nigerian regulations.
        </p>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {/* Doc type selector */}
          <div className="flex gap-2">
            {(['bvn', 'nin'] as DocType[]).map((t) => (
              <button
                key={t}
                type="button"
                onClick={() => { setDocType(t); setValue(''); setError(''); }}
                className={`flex-1 rounded-xl py-2.5 text-[13px] font-semibold border transition-colors ${
                  docType === t
                    ? 'bg-emerald text-lime border-emerald'
                    : 'bg-white text-mid border-line hover:border-emerald/40'
                }`}
              >
                {t.toUpperCase()}
              </button>
            ))}
          </div>

          <div className="flex flex-col gap-1.5">
            <label htmlFor="doc-value" className="text-[12px] font-semibold text-mid">
              {docType === 'bvn' ? 'Bank Verification Number (BVN)' : 'National Identification Number (NIN)'}
            </label>
            <input
              id="doc-value"
              type="text"
              inputMode="numeric"
              maxLength={11}
              placeholder={docType === 'bvn' ? '12345678901' : '12345678901'}
              value={value}
              onChange={(e) => setValue(e.target.value.replace(/\D/g, ''))}
              className="h-11 w-full rounded-[11px] border border-line bg-white px-4 text-[14px] font-mono text-ink focus:outline-none focus:ring-2 focus:ring-emerald"
            />
            <p className="text-[11px] text-mid">
              {docType === 'bvn'
                ? 'Your 11-digit BVN. Dial *565*0# on any network to retrieve it.'
                : 'Your 11-digit NIN. Check your National ID card or NIMC slip.'}
            </p>
          </div>

          {error && (
            <p className="text-xs text-red-600" role="alert">{error}</p>
          )}

          <button
            type="submit"
            disabled={loading || value.length !== 11}
            className="w-full text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-[13px] py-3.5 hover:bg-lime-600 transition-colors disabled:opacity-50"
          >
            {loading ? 'Submitting…' : 'Submit for verification'}
          </button>
        </form>

        <p className="text-[11px] text-mid text-center leading-relaxed">
          Your information is encrypted and only used for identity verification. We never share it with third parties.
        </p>
      </div>
    </main>
  );
}
