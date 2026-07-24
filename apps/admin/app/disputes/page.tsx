'use client';

import { useState, useTransition } from 'react';
import { adminApi } from '@speedplus/api-client';

export default function DisputesPage() {
  const [orderId, setOrderId] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  function handleFreeze() {
    const reason = window.prompt('Freeze reason (required):');
    if (!reason) return;
    startTransition(async () => {
      try {
        const r = await adminApi.freezeEscrow(orderId, reason);
        setMessage(r.message);
        setError('');
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  function handleRelease(recipient: 'customer' | 'merchant') {
    const reason = window.prompt(`Release to ${recipient} — reason (required):`);
    if (!reason) return;
    startTransition(async () => {
      try {
        const r = await adminApi.releaseEscrow(orderId, recipient, reason);
        setMessage(r.message);
        setError('');
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-8 py-7 flex flex-col gap-6 max-w-xl">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Disputes</h1>
      <p className="text-sm text-mid">
        Escrow release requires two different admins to approve. The first approval records intent;
        the second executes the ledger transfer.
      </p>
      <div className="flex flex-col gap-3">
        <label className="text-sm font-semibold">Order UUID</label>
        <input
          value={orderId}
          onChange={(e) => setOrderId(e.target.value)}
          placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
          className="border border-line rounded-xl px-4 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald"
        />
      </div>
      {error && <p className="text-sm text-red-600">{error}</p>}
      {message && (
        <p className="text-sm font-semibold text-emerald bg-[#E9F3D8] rounded-xl px-4 py-3">{message}</p>
      )}
      <div className="flex gap-3 flex-wrap">
        <button
          disabled={isPending || !orderId}
          onClick={handleFreeze}
          className="font-display text-sm font-semibold rounded-[10px] px-5 py-2.5 border-[1.5px] transition-colors disabled:opacity-50"
          style={{ color: '#8A6A1B', borderColor: '#E8B14E' }}
        >
          Freeze Escrow
        </button>
        <button
          disabled={isPending || !orderId}
          onClick={() => handleRelease('customer')}
          className="font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] px-5 py-2.5 hover:bg-lime-600 transition-colors disabled:opacity-50"
        >
          Release → Customer
        </button>
        <button
          disabled={isPending || !orderId}
          onClick={() => handleRelease('merchant')}
          className="font-display text-sm font-semibold bg-[#0A3D2C] text-sand rounded-[10px] px-5 py-2.5 hover:bg-[#0D4E38] transition-colors disabled:opacity-50"
        >
          Release → Merchant
        </button>
      </div>
    </div>
  );
}
