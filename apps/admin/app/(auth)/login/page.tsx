'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { AuthShell, Button, Input, PasswordInput, AlertCircleIcon, type AuthShellChip } from '@fourdat/ui';

const chips: [AuthShellChip, AuthShellChip] = [
  {
    icon: <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true"><rect x="1" y="1" width="4" height="4" rx="1" fill="#C6F24E" /><rect x="7" y="1" width="4" height="4" rx="1" fill="rgba(198,242,78,0.4)" /><rect x="1" y="7" width="4" height="4" rx="1" fill="rgba(198,242,78,0.4)" /><rect x="7" y="7" width="4" height="4" rx="1" fill="rgba(198,242,78,0.4)" /></svg>,
    label: 'Ops dashboard live',
  },
  {
    icon: <span className="relative flex h-2 w-2 flex-shrink-0"><span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-lime opacity-60" /><span className="relative inline-flex rounded-full h-2 w-2 bg-lime" /></span>,
    label: 'All systems operational',
  },
];
import { authApi } from '@fourdat/api-client';
import { useAdminAuthStore } from '@/lib/store/auth.store';

export default function AdminLoginPage() {
  const router = useRouter();
  const setAuth = useAdminAuthStore((s) => s.setAuth);

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
      if (result.user.role !== 'admin') {
        setError('Access denied. This portal is for Fourdat operations staff only.');
        return;
      }
      setAuth(result.user, result.accessToken, result.refreshToken);
      router.replace('/kyc');
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Invalid credentials');
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthShell
      headline={<>Operations<br /><span className="text-lime">Command.</span></>}
      subtext="Internal access only. Fourdat operations staff."
      heroImage="https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&q=80"
      portalLabel="Operations"
      chips={chips}
      formHeading="Sign in"
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
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

        <PasswordInput
          id="password"
          label="Password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          required
        />

        {error && (
          <div className="flex items-start gap-2.5 bg-red-50 border border-red-200 rounded-xl px-3.5 py-2.5" role="alert" aria-live="polite">
            <AlertCircleIcon color="#DC2626" />
            <p className="text-xs text-red-600">{error}</p>
          </div>
        )}

        <Button type="submit" variant="primary" size="lg" isLoading={loading} className="w-full mt-1">
          Sign in to Ops
        </Button>
      </form>
    </AuthShell>
  );
}
