'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useMutation } from '@tanstack/react-query';
import { Button, Input, SelectionCard, StatusSteps, iconColors } from '@fourdat/ui';
import { kycApi } from '@fourdat/api-client';
import { useAuthStore } from '@/lib/store/auth.store';

type DocType = 'bvn' | 'nin';

const STEPS = [{ label: 'Choose document' }, { label: 'Enter details' }, { label: 'Submitted' }];

export default function KYCPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const [docType, setDocType] = useState<DocType>('bvn');
  const [value, setValue] = useState('');
  const [step, setStep] = useState(0);

  const submit = useMutation({
    mutationFn: (v: string) => docType === 'bvn' ? kycApi.submitBVN(v) : kycApi.submitNIN(v),
    onSuccess: () => setStep(2),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const clean = value.replace(/\s/g, '');
    if (clean.length < 10) return;
    submit.mutate(clean);
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="bg-emerald px-5 pt-12 pb-6">
        <div className="flex items-center gap-3 mb-2">
          <button
            onClick={() => router.back()}
            className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors"
            aria-label="Back"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
              <path d="M15 18l-6-6 6-6" />
            </svg>
          </button>
          <h1 className="font-display font-semibold text-white text-[18px]">Identity verification</h1>
        </div>
        <p className="text-[12px] text-white/50 ml-12">Required to unlock wallet transfers and higher limits.</p>
      </div>

      <div className="flex-1 px-5 py-6 max-w-[480px] mx-auto w-full flex flex-col gap-5">
        <StatusSteps steps={STEPS} currentIndex={step} />

        {/* Status banner */}
        {user && (
          <div className={`rounded-2xl px-4 py-3 flex items-center gap-3 ${user.isVerified ? 'bg-tile border border-emerald/20' : 'bg-amber-50 border border-amber-200'}`}>
            <div className={`w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0 ${user.isVerified ? 'bg-emerald' : 'bg-amber-400'}`}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                {user.isVerified
                  ? <polyline points="20 6 9 17 4 12" />
                  : <><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" /></>}
              </svg>
            </div>
            <p className={`text-[13px] font-semibold ${user.isVerified ? 'text-emerald' : 'text-amber-800'}`}>
              {user.isVerified ? 'Identity verified' : 'Not yet verified'}
            </p>
          </div>
        )}

        {step === 2 ? (
          <div className="flex flex-col items-center gap-5 py-8">
            <div className="w-16 h-16 rounded-2xl bg-tile border border-emerald/20 flex items-center justify-center">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke={iconColors.emerald} strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </div>
            <div className="text-center">
              <p className="font-display font-bold text-[20px] text-ink">Verification submitted</p>
              <p className="text-[13px] text-mid mt-1">We&apos;ll review your details and update your status within 24 hours.</p>
            </div>
            <Button variant="primary" size="md" onClick={() => router.back()} className="w-full max-w-[320px]">Done</Button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {/* Step 0: choose doc type */}
            <div className="flex flex-col gap-2">
              <p className="text-[13px] font-semibold text-ink">Choose document type</p>
              <SelectionCard
                selected={docType === 'bvn'}
                label="BVN"
                description="Bank Verification Number — linked to your bank account"
                onClick={() => { setDocType('bvn'); setValue(''); setStep(0); }}
              />
              <SelectionCard
                selected={docType === 'nin'}
                label="NIN"
                description="National Identification Number — from your NIMC slip or card"
                onClick={() => { setDocType('nin'); setValue(''); setStep(0); }}
              />
            </div>

            {/* Step 1: enter number */}
            <form onSubmit={handleSubmit} className="flex flex-col gap-3">
              <Input
                label={docType === 'bvn' ? 'Bank Verification Number (BVN)' : 'National Identification Number (NIN)'}
                type="text"
                inputMode="numeric"
                placeholder="11-digit number"
                value={value}
                onChange={(e) => { setValue(e.target.value.replace(/\D/g, '')); setStep(1); }}
                maxLength={11}
              />
              <p className="text-[11px] text-mid px-1">
                Your {docType.toUpperCase()} is used only for identity verification and is never shared with third parties.
              </p>
              {submit.isError && (
                <p className="text-xs text-red-600" role="alert">{(submit.error as Error).message}</p>
              )}
              <Button
                type="submit"
                variant="primary"
                size="md"
                isLoading={submit.isPending}
                disabled={value.length < 10}
                className="w-full"
              >
                Submit for verification
              </Button>
            </form>
          </div>
        )}
      </div>
    </main>
  );
}
