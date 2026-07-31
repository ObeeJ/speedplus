'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { Button } from '@speedplus/ui';
import { kycApi } from '@speedplus/api-client';
import { useAuthStore } from '@/lib/store/auth.store';

type DocType = 'bvn' | 'nin';

export default function KYCPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [docType, setDocType] = useState<DocType>('bvn');
  const [value, setValue] = useState('');
  const [done, setDone] = useState(false);

  const submit = useMutation({
    mutationFn: (v: string) => docType === 'bvn' ? kycApi.submitBVN(v) : kycApi.submitNIN(v),
    onSuccess: () => setDone(true),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const clean = value.replace(/\s/g, '');
    if (clean.length < 10) return;
    submit.mutate(clean);
  }

  if (done) {
    return (
      <main className="min-h-screen bg-[#F7F5EF] flex flex-col items-center justify-center px-5 gap-5">
        <div className="w-16 h-16 rounded-2xl bg-[#E9F3D8] flex items-center justify-center">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
        </div>
        <div className="text-center">
          <p className="font-display font-bold text-[20px] text-[#121216]">Verification submitted</p>
          <p className="text-[13px] text-[#63636E] mt-1">We'll review your details and update your status within 24 hours.</p>
        </div>
        <Button variant="primary" size="md" onClick={() => router.back()} className="w-full max-w-[320px]">Done</Button>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-6">
        <div className="flex items-center gap-3 mb-2">
          <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Identity verification</h1>
        </div>
        <p className="text-[12px] text-white/50 ml-12">Required to unlock wallet transfers and higher limits.</p>
      </div>

      <div className="flex-1 px-5 py-6 max-w-[480px] mx-auto w-full flex flex-col gap-5">
        {/* Status banner */}
        {user && (
          <div className={`rounded-2xl px-4 py-3 flex items-center gap-3 ${user.isVerified ? 'bg-[#E9F3D8] border border-[#0A3D2C]/20' : 'bg-amber-50 border border-amber-200'}`}>
            <div className={`w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0 ${user.isVerified ? 'bg-[#0A3D2C]' : 'bg-amber-400'}`}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                {user.isVerified
                  ? <polyline points="20 6 9 17 4 12" />
                  : <><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" /></>}
              </svg>
            </div>
            <p className={`text-[13px] font-semibold ${user.isVerified ? 'text-[#0A3D2C]' : 'text-amber-800'}`}>
              {user.isVerified ? 'Identity verified' : 'Not yet verified'}
            </p>
          </div>
        )}

        <div className="bg-white rounded-2xl border border-[#E4E0D6] p-5 flex flex-col gap-4">
          <p className="text-[14px] font-semibold text-[#121216]">Choose document type</p>
          <div className="flex gap-2">
            {(['bvn', 'nin'] as const).map((t) => (
              <button
                key={t}
                onClick={() => { setDocType(t); setValue(''); }}
                className={`flex-1 rounded-xl border-2 py-2.5 text-[13px] font-semibold transition-all ${docType === t ? 'border-[#0A3D2C] bg-[#E9F3D8] text-[#0A3D2C]' : 'border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30'}`}
              >
                {t.toUpperCase()}
              </button>
            ))}
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold text-[#63636E]">
                {docType === 'bvn' ? 'Bank Verification Number (BVN)' : 'National Identification Number (NIN)'}
              </label>
              <input
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                maxLength={11}
                value={value}
                onChange={(e) => setValue(e.target.value.replace(/\D/g, ''))}
                placeholder={docType === 'bvn' ? '12345678901' : '12345678901'}
                className="w-full border border-[#E4E0D6] rounded-xl px-4 py-3 text-[15px] font-mono text-[#121216] placeholder-[#C5C2BB] tracking-widest focus:outline-none focus:border-[#0A3D2C] transition-colors"
                aria-label={docType.toUpperCase()}
              />
            </div>

            <p className="text-[11px] text-[#9A968D]">
              Your {docType.toUpperCase()} is used only for identity verification and is never shared with third parties.
            </p>

            {submit.isError && <p className="text-xs text-red-600" role="alert">{(submit.error as Error).message}</p>}

            <Button type="submit" variant="primary" size="md" isLoading={submit.isPending} disabled={value.length < 10} className="w-full">
              Submit for verification
            </Button>
          </form>
        </div>
      </div>
    </main>
  );
}
