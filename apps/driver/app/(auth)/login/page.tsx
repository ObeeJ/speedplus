'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { motion, AnimatePresence } from 'framer-motion';
import { ArrowRight } from 'lucide-react';
import { authApi } from '@fourdat/api-client';
import { useDriverAuthStore } from '@/lib/store/auth.store';
import { AuthShell, Button, Input, PasswordInput, AlertCircleIcon, type AuthShellChip } from '@fourdat/ui';

const chips: [AuthShellChip, AuthShellChip] = [
  {
    icon: <span className="relative flex h-2 w-2 flex-shrink-0"><span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-lime opacity-60" /><span className="relative inline-flex rounded-full h-2 w-2 bg-lime" /></span>,
    label: '2,400+ deliveries today',
  },
  {
    icon: <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true"><path d="M6 1l1.3 2.6L10 4.1 8 6l.5 2.9L6 7.5 3.5 8.9 4 6 2 4.1l2.7-.5L6 1z" fill="#C6F24E" /></svg>,
    label: '4.9 avg rider rating',
  },
];

export default function DriverLoginPage() {
  const router  = useRouter();
  const setAuth = useDriverAuthStore((s) => s.setAuth);

  const [phone,    setPhone]    = useState('');
  const [password, setPassword] = useState('');
  const [error,    setError]    = useState('');
  const [loading,  setLoading]  = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const result = await authApi.login({ phone, password });
      if (result.user.role !== 'driver') {
        setError('This app is for riders only. Use the customer app instead.');
        return;
      }
      setAuth(result.user, result.accessToken, result.refreshToken);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid phone or password');
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={<>Deliver.<br />Earn.<br /><span className="text-lime">Grow.</span></>}
      subtext="Your earnings, your schedule, your city. Start riding with Fourdat."
      heroImage="https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=1200&q=80"
      portalLabel="Rider Portal"
      chips={chips}
      formHeading="Welcome back"
      formSubheading="Sign in to your rider account"
    >
      <form onSubmit={handleSubmit} noValidate>
        <div className="flex flex-col gap-4">

          {/* Phone */}
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

          {/* Password */}
          <PasswordInput
            id="password"
            label="Password"
            placeholder="Enter your password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />

          {/* Forgot password */}
          <div className="flex justify-end -mt-1">
            <Link
              href="/forgot-password"
              className="text-[12px] font-medium text-mid hover:text-ink transition-colors underline-offset-2 hover:underline"
            >
              Forgot password?
            </Link>
          </div>

          {/* Error banner */}
          <AnimatePresence>
            {error && (
              <motion.div
                className="flex items-start gap-3 rounded-2xl px-4 py-3.5"
                style={{
                  background: 'rgba(254,242,242,0.8)',
                  border: '1px solid rgba(220,38,38,0.15)',
                  boxShadow: '0 1px 4px rgba(220,38,38,0.08)',
                }}
                role="alert"
                aria-live="polite"
                initial={{ opacity: 0, y: -8, scale: 0.97 }}
                animate={{ opacity: 1, y: 0,  scale: 1    }}
                exit={{    opacity: 0, y: -6, scale: 0.97 }}
                transition={{ type: 'spring', stiffness: 400, damping: 30 }}
              >
                <AlertCircleIcon color="#DC2626" aria-hidden="true" />
                <p className="text-[12.5px] text-red-700 leading-snug font-medium">{error}</p>
              </motion.div>
            )}
          </AnimatePresence>

          {/* Submit */}
          <Button
            type="submit"
            variant="primary"
            size="lg"
            isLoading={loading}
            className="w-full mt-1 group"
          >
            {!loading && (
              <>
                Sign in
                <motion.span
                  className="inline-flex"
                  initial={{ x: 0 }}
                  whileHover={{ x: 3 }}
                  transition={{ type: 'spring', stiffness: 500, damping: 28 }}
                >
                  <ArrowRight size={15} />
                </motion.span>
              </>
            )}
            {loading && 'Signing in…'}
          </Button>

        </div>
      </form>
    </AuthShell>
  );
}
