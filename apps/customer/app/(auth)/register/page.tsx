'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Button, Input } from '@speedplus/ui';
import { SpeedPlusLogo } from '@speedplus/ui';
import { authApi } from '@speedplus/api-client';
import { useAuthStore } from '@/lib/store/auth.store';
import type { User } from '@speedplus/types';

export default function RegisterPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);

  const [form, setForm] = useState({ firstName: '', lastName: '', phone: '', password: '', referralCode: '' });
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const update = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((p) => ({ ...p, [k]: e.target.value }));

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    if (form.password.length < 8) { setError('Password must be at least 8 characters'); return; }
    setLoading(true);
    try {
      const result = await authApi.register({
        firstName: form.firstName, lastName: form.lastName,
        phone: form.phone, password: form.password,
        referralCode: form.referralCode || undefined,
      });
      setAuth(result.user as User & { referralCode?: string }, result.accessToken, result.refreshToken);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not create account. Try again.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex">
      {/* Brand panel */}
      <div className="hidden lg:flex lg:w-[480px] xl:w-[560px] flex-shrink-0 bg-[#0A3D2C] flex-col justify-between p-12 relative overflow-hidden">
        <div className="absolute inset-0 opacity-[0.04]" style={{
          backgroundImage: 'radial-gradient(circle at 20% 80%, #C6F24E 0%, transparent 50%), radial-gradient(circle at 80% 20%, #C6F24E 0%, transparent 50%)',
        }} />
        <SpeedPlusLogo variant="full" theme="dark" size="lg" />
        <div className="flex flex-col gap-6 relative z-10">
          <h2 className="font-display font-bold text-[42px] leading-[1.1] text-white">
            Join the<br />
            <span className="text-[#C6F24E]">movement.</span>
          </h2>
          <p className="text-white/60 text-lg leading-relaxed max-w-xs">
            Your wallet, your card, your deliveries — all in one place.
          </p>
        </div>
        <p className="text-white/30 text-xs relative z-10">&copy; {new Date().getFullYear()} SpeedPlus Technologies</p>
      </div>

      {/* Form panel */}
      <div className="flex-1 flex flex-col items-center justify-center bg-[#F7F5EF] px-5 py-12 lg:px-16 overflow-y-auto">
        <div className="lg:hidden mb-10">
          <SpeedPlusLogo variant="full" theme="light" size="lg" />
        </div>

        <div
          className="w-full max-w-[400px] flex flex-col gap-7"
          style={{ animation: 'fadeUp 0.4s cubic-bezier(0.16,1,0.3,1) both' }}
        >
          <div>
            <h1 className="font-display font-bold text-[28px] text-[#121216] tracking-tight">Create account</h1>
            <p className="text-[#63636E] mt-1">Takes less than a minute</p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="grid grid-cols-2 gap-3">
              <Input id="firstName" label="First name" placeholder="Ada" value={form.firstName} onChange={update('firstName')} autoComplete="given-name" required />
              <Input id="lastName" label="Last name" placeholder="Obi" value={form.lastName} onChange={update('lastName')} autoComplete="family-name" required />
            </div>

            <Input id="phone" label="Phone number" type="tel" placeholder="08012345678" value={form.phone} onChange={update('phone')} autoComplete="tel" required />

            <div className="relative">
              <Input
                id="password" label="Password" type={showPw ? 'text' : 'password'}
                placeholder="Min. 8 characters" value={form.password} onChange={update('password')}
                autoComplete="new-password" required
              />
              <button type="button" onClick={() => setShowPw((v) => !v)}
                className="absolute right-3 bottom-[11px] text-[#63636E] hover:text-[#121216] transition-colors"
                aria-label={showPw ? 'Hide' : 'Show'}>
                {showPw
                  ? <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19m-6.72-1.07a3 3 0 11-4.24-4.24" /><line x1="1" y1="1" x2="23" y2="23" /></svg>
                  : <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" /><circle cx="12" cy="12" r="3" /></svg>
                }
              </button>
            </div>

            <div className="relative">
              <Input id="referralCode" label="Referral code" placeholder="Optional — e.g. MUSA500" value={form.referralCode} onChange={update('referralCode')} />
              {form.referralCode && (
                <span className="absolute right-3 bottom-[11px] text-[10px] font-bold text-[#0A3D2C] bg-[#E9F3D8] rounded-full px-2 py-0.5">
                  +₦500 bonus
                </span>
              )}
            </div>

            {error && (
              <div className="flex items-start gap-2.5 bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-3.5 py-2.5" role="alert">
                <svg className="flex-shrink-0 mt-0.5" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#DC2626" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" />
                </svg>
                <p className="text-xs text-[#DC2626]">{error}</p>
              </div>
            )}

            <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full mt-2">
              Create account
            </Button>

            <p className="text-[11px] text-[#9A968D] text-center leading-relaxed">
              By creating an account you agree to our Terms of Service and Privacy Policy.
            </p>
          </form>

          <p className="text-center text-sm text-[#63636E]">
            Already have an account?{' '}
            <Link href="/login" className="font-semibold text-[#0A3D2C] hover:underline">Sign in</Link>
          </p>
        </div>
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(16px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}
