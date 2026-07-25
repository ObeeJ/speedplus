'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input } from '@speedplus/ui';
import { walletApi } from '@speedplus/api-client';
import { useAuthStore } from '@/lib/store/auth.store';

export default function WalletFundPage() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);

  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const PRESETS = [500, 1000, 2000, 5000];

  async function handleFund() {
    const amountNaira = parseFloat(amount);
    if (!amountNaira || amountNaira < 100) {
      setError('Minimum amount is ₦100');
      return;
    }
    if (!user?.email) {
      setError('Add an email address to your account before funding your wallet.');
      return;
    }
    setError('');
    setLoading(true);
    try {
      const key = `fund-${Date.now()}`;
      const callbackUrl = `${window.location.origin}/wallet?funded=1`;
      const result = await walletApi.fund(
        { amountKobo: Math.round(amountNaira * 100), email: user.email, callbackUrl },
        key,
      );
      // Redirect to Paystack/Flutterwave authorization page
      window.location.href = result.authorizationUrl;
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Could not initiate payment. Try again.');
      setLoading(false);
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
        <span className="font-display font-semibold text-sand text-lg">Add money</span>
      </div>

      <div className="flex-1 px-5 py-6 flex flex-col gap-5 min-[700px]:max-w-[480px] min-[700px]:mx-auto min-[700px]:w-full">

        {/* Preset amounts */}
        <div className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Quick amounts</span>
          <div className="grid grid-cols-4 gap-2">
            {PRESETS.map((p) => (
              <button
                key={p}
                type="button"
                onClick={() => setAmount(String(p))}
                className={`rounded-[13px] border py-3 text-[13px] font-semibold transition-all duration-150 ${
                  amount === String(p) ? 'border-emerald bg-tile text-emerald' : 'border-line bg-white text-ink hover:border-emerald/40'
                }`}
              >
                ₦{p.toLocaleString()}
              </button>
            ))}
          </div>
        </div>

        <Input
          id="amount"
          label="Or enter amount (₦)"
          type="number"
          placeholder="e.g. 3000"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          min="100"
        />

        {!user?.email && (
          <div className="bg-[#FFF7E6] border border-[#F0DFB4] rounded-xl px-4 py-3 text-[12px] text-[#8A6A1B]">
            You need an email address on your account to fund your wallet.{' '}
            <button onClick={() => router.push('/profile')} className="font-semibold underline">Add email</button>
          </div>
        )}

        {error && <p className="text-xs text-[#DC2626]" role="alert">{error}</p>}

        <Button
          variant="primary"
          size="lg"
          isLoading={loading}
          disabled={!amount || parseFloat(amount) < 100}
          onClick={handleFund}
          className="w-full"
        >
          Pay {amount ? `₦${parseFloat(amount).toLocaleString()}` : ''}
        </Button>

        <p className="text-[11px] text-mid text-center">
          Secured by Paystack. Your card details are never stored by SpeedPlus.
        </p>
      </div>
    </main>
  );
}
