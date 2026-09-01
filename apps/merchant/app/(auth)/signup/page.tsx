'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { motion, AnimatePresence } from 'framer-motion';
import { AuthShell, Button, Input, PasswordInput, AlertCircleIcon, type AuthShellChip } from '@fourdat/ui';
import { authApi, merchantApi } from '@fourdat/api-client';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

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

export default function MerchantSignupPage() {
  const router = useRouter();
  const { setAuth, setMerchant } = useMerchantAuthStore();

  const [form, setForm] = useState({ firstName: '', lastName: '', phone: '', password: '' });
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
        firstName: form.firstName,
        lastName: form.lastName,
        phone: form.phone,
        password: form.password,
        role: 'merchant',
      });
      if (result.user.role !== 'merchant') {
        setError('Registration failed. Please try again.');
        return;
      }
      setAuth(result.user, result.accessToken, result.refreshToken);
      // Merchant row is created synchronously on the backend during register,
      // so getProfile is immediately available after signup.
      const profile = await merchantApi.getProfile();
      setMerchant(profile);
      router.replace('/');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not create account. Try again.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={<>Grow your<br /><span className="text-lime">business.</span></>}
      subtext="Join Fourdat and reach thousands of customers. Set up your store in minutes."
      heroImage="https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?w=1200&q=80"
      portalLabel="Partner Portal"
      chips={chips}
      formHeading="Create partner account"
      formSubheading="Start selling on Fourdat today."
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
        <div className="grid grid-cols-2 gap-3">
          <Input id="firstName" label="First name" placeholder="Ada" value={form.firstName} onChange={update('firstName')} autoComplete="given-name" required />
          <Input id="lastName" label="Last name" placeholder="Obi" value={form.lastName} onChange={update('lastName')} autoComplete="family-name" required />
        </div>

        <Input id="phone" label="Phone number" type="tel" placeholder="0801 234 5678" value={form.phone} onChange={update('phone')} autoComplete="tel" required />

        <PasswordInput id="password" label="Password" placeholder="Min. 8 characters" value={form.password} onChange={update('password')} autoComplete="new-password" required />

        <AnimatePresence>
          {error && (
            <motion.div
              className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-3.5 py-2.5"
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

        <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full mt-1">
          Create partner account
        </Button>

        <p className="text-center text-sm text-mid">
          Already have an account?{' '}
          <Link href="/login" className="font-semibold text-emerald hover:underline">Sign in</Link>
        </p>
      </form>
    </AuthShell>
  );
}
