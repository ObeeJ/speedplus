'use client';

import { useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { paymentLinksApi, walletApi } from '@speedplus/api-client';
import { Button, SpeedPlusLogo } from '@speedplus/ui';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 2 })}`;
}

export default function PayLinkPage() {
  const { slug } = useParams<{ slug: string }>();
  const router = useRouter();

  const [link, setLink] = useState<{ amountKobo: number; note?: string; expiresAt: string } | null>(null);
  const [balance, setBalance] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [paying, setPaying] = useState(false);
  const [paid, setPaid] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([
      paymentLinksApi.resolve(slug),
      walletApi.getBalance().catch(() => null),
    ]).then(([l, w]) => {
      setLink(l);
      if (w) setBalance(w.balanceKobo);
    }).catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [slug]);

  async function handlePay() {
    setPaying(true);
    setError('');
    try {
      const key = `pay-${slug}-${Date.now()}`;
      await paymentLinksApi.pay(slug, key);
      setPaid(true);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Payment failed');
    } finally {
      setPaying(false);
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center" style={{ background: '#F7F5EF' }}>
        <div className="w-10 h-10 rounded-full border-2 border-[#C6F24E] border-t-transparent animate-spin" />
      </div>
    );
  }

  if (paid) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center gap-6 px-6" style={{ background: '#F7F5EF' }}>
        <div className="w-16 h-16 rounded-full flex items-center justify-center" style={{ background: '#C6F24E' }}>
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
            <path d="M5 13l4 4L19 7" />
          </svg>
        </div>
        <div className="text-center">
          <p className="font-display font-bold text-[22px] text-ink">Payment sent</p>
          <p className="text-[13px] text-mid mt-1">{link ? naira(link.amountKobo) : ''} paid successfully</p>
        </div>
        <Button variant="secondary" onClick={() => router.push('/')}>Back to home</Button>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-5 py-10" style={{ background: '#F7F5EF' }}>
      <div className="w-full max-w-sm flex flex-col gap-5">
        {/* Logo */}
        <div className="flex justify-center mb-2">
          <SpeedPlusLogo size="md" />
        </div>

        {error && (
          <div className="bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-4 py-3 text-sm text-[#DC2626]">{error}</div>
        )}

        {link ? (
          <div className="bg-white border border-line rounded-2xl p-6 flex flex-col gap-5">
            {/* Amount */}
            <div className="text-center">
              <p className="font-display font-bold text-[36px] text-ink leading-none">{naira(link.amountKobo)}</p>
              {link.note && <p className="text-[13px] text-mid mt-2">{link.note}</p>}
            </div>

            <div className="h-px bg-line" />

            {/* Wallet balance */}
            {balance !== null && (
              <div className="flex items-center justify-between text-[13px]">
                <span className="text-mid">Wallet balance</span>
                <span className={`font-semibold ${balance >= link.amountKobo ? 'text-[#0A3D2C]' : 'text-[#DC2626]'}`}>
                  {naira(balance)}
                </span>
              </div>
            )}

            {/* Expiry */}
            <p className="text-[11.5px] text-mid text-center">
              Expires {new Date(link.expiresAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric' })}
            </p>

            <Button
              onClick={handlePay}
              isLoading={paying}
              disabled={paying || (balance !== null && balance < link.amountKobo)}
            >
              {balance !== null && balance < link.amountKobo ? 'Insufficient balance' : `Pay ${naira(link.amountKobo)}`}
            </Button>

            {balance !== null && balance < link.amountKobo && (
              <Button variant="outline" onClick={() => router.push('/wallet/fund')}>
                Top up wallet
              </Button>
            )}
          </div>
        ) : (
          <div className="bg-white border border-line rounded-2xl p-6 text-center">
            <p className="text-[14px] text-mid">This payment link is invalid or has expired.</p>
          </div>
        )}
      </div>
    </div>
  );
}
