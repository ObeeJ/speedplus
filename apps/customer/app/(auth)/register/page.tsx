'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { AuthShell, Button, Input, PasswordInput, AlertCircleIcon, Badge } from '@speedplus/ui';
import { authApi } from '@speedplus/api-client';
import { useAuthStore } from '@/lib/store/auth.store';
import type { User } from '@speedplus/types';

export default function RegisterPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);

  const [form, setForm] = useState({ firstName: '', lastName: '', phone: '', password: '', referralCode: '' });
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
      router.replace('/home');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not create account. Try again.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={<>Join the<br /><span className="text-lime">movement.</span></>}
      subtext="Your wallet, your card, your deliveries — all in one place."
      formHeading="Create account"
      formSubheading="Takes less than a minute"
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-3">
          <Input id="firstName" label="First name" placeholder="Ada" value={form.firstName} onChange={update('firstName')} autoComplete="given-name" required />
          <Input id="lastName" label="Last name" placeholder="Obi" value={form.lastName} onChange={update('lastName')} autoComplete="family-name" required />
        </div>

        <Input id="phone" label="Phone number" type="tel" placeholder="08012345678" value={form.phone} onChange={update('phone')} autoComplete="tel" required />

        <PasswordInput
          id="password"
          label="Password"
          placeholder="Min. 8 characters"
          value={form.password}
          onChange={update('password')}
          autoComplete="new-password"
          required
        />

        <div className="relative">
          <Input
            id="referralCode"
            label="Referral code"
            placeholder="Optional — e.g. MUSA500"
            value={form.referralCode}
            onChange={update('referralCode')}
          />
          {form.referralCode && (
            <div className="absolute right-3 bottom-[11px]">
              <Badge variant="success" className="text-[10px] px-2 py-0.5">+₦500 bonus</Badge>
            </div>
          )}
        </div>

        {error && (
          <div className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-3.5 py-2.5" role="alert" aria-live="polite">
            <AlertCircleIcon color="#DC2626" />
            <p className="text-xs text-red-600">{error}</p>
          </div>
        )}

        <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full mt-1">
          Create account
        </Button>

        <p className="text-xs text-mid text-center leading-relaxed">
          By creating an account you agree to our{' '}
          <Link href="/terms" className="underline underline-offset-2 hover:text-ink">Terms of Service</Link>
          {' '}and{' '}
          <Link href="/privacy" className="underline underline-offset-2 hover:text-ink">Privacy Policy</Link>.
        </p>
      </form>

      <p className="text-center text-sm text-mid">
        Already have an account?{' '}
        <Link href="/login" className="font-semibold text-emerald hover:underline">Sign in</Link>
      </p>
    </AuthShell>
  );
}
