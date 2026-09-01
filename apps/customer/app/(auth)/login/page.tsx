'use client';

import { Suspense, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { motion, AnimatePresence } from 'framer-motion';
import { AuthShell, Button, Input, PasswordInput, AlertCircleIcon, type AuthShellChip } from '@fourdat/ui';

const chips: [AuthShellChip, AuthShellChip] = [
  {
    icon: <span className="relative flex h-2 w-2 flex-shrink-0"><span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-lime opacity-60" /><span className="relative inline-flex rounded-full h-2 w-2 bg-lime" /></span>,
    label: '50,000+ happy customers',
  },
  {
    icon: <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true"><path d="M6 1l1.3 2.6L10 4.1 8 6l.5 2.9L6 7.5 3.5 8.9 4 6 2 4.1l2.7-.5L6 1z" fill="#C6F24E" /></svg>,
    label: '30 min avg delivery',
  },
];
import { authApi } from '@fourdat/api-client';
import { useAuthStore } from '@/lib/store/auth.store';
import type { User } from '@fourdat/types';

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginForm />
    </Suspense>
  );
}

function LoginForm() {
  const router = useRouter();
  const params = useSearchParams();
  const next = params.get('next') ?? '/home';
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
      router.replace(next);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid phone or password');
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={<>Faster.<br />Cheaper.<br /><span className="text-lime">Better.</span></>}
      subtext="Essential delivery for everyday Nigeria — gas, food, pharmacy, packages."
      heroImage="https://images.unsplash.com/photo-1601628828688-632f38a5a7d0?w=1200&q=80"
      chips={chips}
      formHeading="Welcome back"
      formSubheading="Sign in to your account"
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
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

        <div className="flex flex-col gap-1.5">
          <PasswordInput
            id="password"
            label="Password"
            placeholder="Enter your password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
          <div className="flex justify-end">
            <Link
              href="/forgot-password"
              className="text-[12px] text-mid hover:text-ink transition-colors underline-offset-2 hover:underline"
            >
              Forgot password?
            </Link>
          </div>
        </div>

        <AnimatePresence mode="wait">
          {error && (
            <motion.div
              key="error"
              className="flex items-start gap-2.5 bg-red-50 border border-red-200/80 rounded-[var(--radius-md)] px-3.5 py-3"
              role="alert"
              aria-live="polite"
              initial={{ opacity: 0, y: -6, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -4 }}
              transition={{ duration: 0.2 }}
            >
              <AlertCircleIcon color="#DC2626" />
              <p className="text-[12px] text-red-700 leading-snug">{error}</p>
            </motion.div>
          )}
        </AnimatePresence>

        <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full">
          Sign in
        </Button>
      </form>

      {/* Divider */}
      <div className="flex items-center gap-3 my-1">
        <div className="flex-1 h-px bg-line" />
        <span className="text-[11px] text-mid/60 font-medium">or</span>
        <div className="flex-1 h-px bg-line" />
      </div>

      <p className="text-center text-[13px] text-mid">
        New to Fourdat?{' '}
        <Link href="/register" className="font-semibold text-emerald hover:text-emerald-600 transition-colors">
          Create an account
        </Link>
      </p>
    </AuthShell>
  );
}
