'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth.store';

export default function ReferralPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);
  const referralCode = (user as (typeof user & { referralCode?: string }) | null)?.referralCode ?? '';

  const [copied, setCopied] = useState(false);

  function handleCopy() {
    if (!referralCode) return;
    navigator.clipboard?.writeText(referralCode).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  function handleShare() {
    const text = `Use my SpeedPlus code ${referralCode} when you sign up and we both get ₦500! https://speedplus.app`;
    if (navigator.share) {
      navigator.share({ text }).catch(() => {});
    } else {
      navigator.clipboard?.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="bg-emerald px-5 py-5 flex items-center gap-3">
        <button onClick={() => router.back()} className="text-sand/70 hover:text-sand transition-colors" aria-label="Back">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <path d="M19 12H5M12 5l-7 7 7 7" />
          </svg>
        </button>
        <span className="font-display font-semibold text-sand text-lg">Refer & earn</span>
      </div>

      <div className="flex-1 px-5 py-8 flex flex-col gap-6 items-center min-[700px]:max-w-[480px] min-[700px]:mx-auto min-[700px]:w-full">

        {/* Hero */}
        <div
          className="w-full bg-emerald rounded-2xl px-6 py-8 flex flex-col items-center gap-3 text-center"
          style={{ animation: 'fadeUp 0.35s cubic-bezier(0.16,1,0.3,1) both' }}
        >
          <span className="text-5xl">🎁</span>
          <span className="font-display font-bold text-2xl text-sand">Give ₦500, get ₦500</span>
          <span className="text-sm text-sand/70 max-w-xs">
            Share your code. When a friend signs up and places their first order over ₦2,000, you both get ₦500 in your wallets.
          </span>
        </div>

        {/* Code display */}
        {referralCode ? (
          <div className="w-full flex flex-col gap-3">
            <span className="text-[11px] font-semibold text-mid tracking-widest uppercase text-center">Your referral code</span>
            <div className="bg-white border-2 border-emerald rounded-2xl px-6 py-5 flex items-center justify-between">
              <span className="font-display font-bold text-3xl text-emerald tracking-[8px]">{referralCode}</span>
              <button
                onClick={handleCopy}
                className="text-[12px] font-semibold text-emerald border border-emerald/30 rounded-[10px] px-3 py-1.5 hover:bg-tile transition-colors"
              >
                {copied ? '✓ Copied' : 'Copy'}
              </button>
            </div>

            <button
              onClick={handleShare}
              className="w-full font-display text-sm font-semibold text-emerald bg-lime rounded-[13px] py-3.5 hover:bg-lime-600 transition-colors"
            >
              Share with friends
            </button>
          </div>
        ) : (
          <div className="h-20 w-full bg-line rounded-2xl animate-pulse" />
        )}

        {/* How it works */}
        <div className="w-full flex flex-col gap-3">
          <span className="text-[11px] font-semibold text-mid tracking-widest uppercase">How it works</span>
          {[
            { step: '1', text: 'Share your code with a friend' },
            { step: '2', text: 'They sign up using your code' },
            { step: '3', text: 'They place their first order over ₦2,000' },
            { step: '4', text: 'You both get ₦500 added to your wallets' },
          ].map((item) => (
            <div key={item.step} className="flex items-center gap-3 bg-white border border-line rounded-xl px-4 py-3">
              <span className="w-7 h-7 rounded-full bg-tile flex items-center justify-center font-display font-bold text-emerald text-sm flex-shrink-0">
                {item.step}
              </span>
              <span className="text-[13px] text-ink">{item.text}</span>
            </div>
          ))}
        </div>
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(16px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
