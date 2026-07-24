'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Button, Input } from '@speedplus/ui';
import { SpeedPlusLogo } from '@speedplus/ui';
import { authApi } from '@speedplus/api-client';
import { useAuthStore } from '@/lib/store/auth.store';
import type { User } from '@speedplus/types';

export default function LoginPage() {
  const router = useRouter();
  const setAuth = useAuthStore((s) => s.setAuth);

  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const result = await authApi.login({ phone, password });
      setAuth(result.user as User & { referralCode?: string }, result.accessToken, result.refreshToken);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid phone or password');
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
          <p className="text-sm text-mid text-center">Sign in to your account</p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <Input
            id="phone"
            label="Phone number"
            type="tel"
            placeholder="08012345678"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            autoComplete="tel"
            required
          />
          <Input
            id="password"
            label="Password"
            type="password"
            placeholder="••••••••"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
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
            Sign in
          </Button>
        </form>

        <p className="text-center text-sm text-mid">
          No account?{' '}
          <Link href="/register" className="font-semibold text-emerald hover:underline">
            Create one
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
