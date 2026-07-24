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

  const [form, setForm] = useState({
    firstName: '',
    lastName: '',
    phone: '',
    password: '',
    referralCode: '',
  });
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  function update(field: keyof typeof form) {
    return (e: React.ChangeEvent<HTMLInputElement>) =>
      setForm((prev) => ({ ...prev, [field]: e.target.value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    if (form.password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }
    setLoading(true);
    try {
      const result = await authApi.register({
        firstName: form.firstName,
        lastName: form.lastName,
        phone: form.phone,
        password: form.password,
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
    <main className="min-h-screen bg-emerald flex flex-col items-center justify-center px-5 py-12">
      <div
        className="w-full max-w-[400px] bg-sand rounded-2xl p-8 flex flex-col gap-6 shadow-[0_24px_60px_rgba(0,0,0,0.25)]"
        style={{ animation: 'slideUp 0.35s cubic-bezier(0.16,1,0.3,1) both' }}
      >
        <div className="flex flex-col items-center gap-3">
          <SpeedPlusLogo variant="full" theme="light" size="lg" />
          <p className="text-sm text-mid text-center">Create your account</p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="grid grid-cols-2 gap-3">
            <Input
              id="firstName"
              label="First name"
              placeholder="Ada"
              value={form.firstName}
              onChange={update('firstName')}
              autoComplete="given-name"
              required
            />
            <Input
              id="lastName"
              label="Last name"
              placeholder="Obi"
              value={form.lastName}
              onChange={update('lastName')}
              autoComplete="family-name"
              required
            />
          </div>
          <Input
            id="phone"
            label="Phone number"
            type="tel"
            placeholder="08012345678"
            value={form.phone}
            onChange={update('phone')}
            autoComplete="tel"
            required
          />
          <Input
            id="password"
            label="Password"
            type="password"
            placeholder="Min. 8 characters"
            value={form.password}
            onChange={update('password')}
            autoComplete="new-password"
            required
          />
          <Input
            id="referralCode"
            label="Referral code (optional)"
            placeholder="e.g. MUSA500"
            value={form.referralCode}
            onChange={update('referralCode')}
          />

          {error && (
            <p
              className="text-xs text-[#DC2626] bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-3 py-2"
              role="alert"
            >
              {error}
            </p>
          )}

          <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full mt-1">
            Create account
          </Button>
        </form>

        <p className="text-center text-sm text-mid">
          Already have an account?{' '}
          <Link href="/login" className="font-semibold text-emerald hover:underline">
            Sign in
          </Link>
        </p>
      </div>

      <style>{`
        @keyframes slideUp {
          from { opacity: 0; transform: translateY(24px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
