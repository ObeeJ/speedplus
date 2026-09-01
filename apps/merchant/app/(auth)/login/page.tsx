'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { motion, AnimatePresence } from 'framer-motion';
import { AuthShell, Button, Input, PasswordInput, AlertCircleIcon, type AuthShellChip } from '@fourdat/ui';

const chips: [AuthShellChip, AuthShellChip] = [
  {
    icon: <span className="relative flex h-2 w-2 flex-shrink-0"><span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-lime opacity-60" /><span className="relative inline-flex rounded-full h-2 w-2 bg-lime" /></span>,
    label: '1,200+ active merchants',
  },
  {
    icon: <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true"><path d="M6 1l1.3 2.6L10 4.1 8 6l.5 2.9L6 7.5 3.5 8.9 4 6 2 4.1l2.7-.5L6 1z" fill="#C6F24E" /></svg>,
    label: '₦2.4M avg monthly earnings',
  },
];
import { authApi, merchantApi } from '@fourdat/api-client';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

export default function MerchantLoginPage() {
  const router = useRouter();
  const { setAuth, setMerchant } = useMerchantAuthStore();

  const [phone, setPhone]       = useState('');
  const [password, setPassword] = useState('');
  const [error, setError]       = useState('');
  const [loading, setLoading]   = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const result = await authApi.login({ phone, password });
      if (result.user.role !== 'merchant') {
        setError('This portal is for Fourdat merchant partners only.');
        return;
      }
      setAuth(result.user, result.accessToken, result.refreshToken);
      const profile = await merchantApi.getProfile();
      setMerchant(profile);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid credentials');
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={<>Your business,<br /><span className="text-lime">amplified.</span></>}
      subtext="Manage orders, products, prescriptions, and earnings — all in one place."
      heroImage="https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?w=1200&q=80"
      portalLabel="Partner Portal"
      chips={chips}
      formHeading="Sign in"
      formSubheading="Welcome back. Continue managing your business."
    >
      <form onSubmit={handleSubmit} className="flex flex-col" noValidate>

        {/* Phone */}
        <Input
          id="phone"
          label="Phone number"
          type="tel"
          placeholder="0801 234 5678"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          autoComplete="tel"
          required
        />

        {/* Password — 20px below phone */}
        <div className="mt-5">
          <PasswordInput
            id="password"
            label="Password"
            placeholder="Enter your password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
        </div>

        {/* Forgot password — 10px below password, right-aligned */}
        <div className="flex justify-end mt-2.5">
          <a
            href="/forgot-password"
            className="text-[12px] text-mid hover:text-ink transition-colors underline-offset-2 hover:underline"
          >
            Forgot password?
          </a>
        </div>

        {/* Error — only when present */}
        <AnimatePresence>
          {error && (
            <motion.div
              className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-3.5 py-2.5 mt-5"
              role="alert"
              aria-live="polite"
              initial={{ opacity: 0, y: -6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.18 }}
            >
              <AlertCircleIcon color="#DC2626" />
              <p className="text-xs text-red-600">{error}</p>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Button — 28px below the last element, dominates */}
        <Button
          type="submit"
          variant="primary"
          size="lg"
          isLoading={loading}
          className="w-full mt-7"
        >
          Sign in to Partner Portal
        </Button>

        <p className="text-center text-sm text-mid mt-4">
          New merchant?{' '}
          <a href="/signup" className="font-semibold text-emerald hover:underline">Create an account</a>
        </p>
      </form>
    </AuthShell>
  );
}
