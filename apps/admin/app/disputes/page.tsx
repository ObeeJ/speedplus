'use client';

import { useState, useTransition } from 'react';
import { adminApi, type OrderDetail } from '@speedplus/api-client';
import { formatCurrency } from '@speedplus/utils';

type PendingAction =
  | { kind: 'freeze' }
  | { kind: 'release'; recipient: 'customer' | 'merchant' };

export default function DisputesPage() {
  const [orderId, setOrderId] = useState('');
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [lookupError, setLookupError] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();
  const [isLookingUp, startLookup] = useTransition();

  const [confirming, setConfirming] = useState<PendingAction | null>(null);
  const [reason, setReason] = useState('');
  const [confirmText, setConfirmText] = useState('');

  function lookupOrder() {
    setLookupError('');
    setOrder(null);
    startLookup(async () => {
      try {
        const detail = await adminApi.getOrderDetail(orderId);
        setOrder(detail);
      } catch (e: unknown) {
        setLookupError(e instanceof Error ? e.message : 'Order not found');
      }
    });
  }

  function openConfirm(action: PendingAction) {
    setReason('');
    setConfirmText('');
    setConfirming(action);
  }

  // Type-to-confirm gate: the admin must retype the order's total amount
  // exactly. This is the industry-standard pattern for irreversible
  // high-value actions (AWS "delete this bucket" flows use the same idea) —
  // it forces the admin to actually read the amount on screen rather than
  // reflexively clicking through a native window.prompt() with no context.
  const expectedConfirmText = order ? formatCurrency({ amount: order.totalKobo, currency: 'NGN' }) : '';
  const canSubmit = confirmText.trim() === expectedConfirmText && reason.trim().length > 0;

  function submitConfirmedAction() {
    if (!confirming || !order || !canSubmit) return;
    const action = confirming;
    startTransition(async () => {
      try {
        const r =
          action.kind === 'freeze'
            ? await adminApi.freezeEscrow(order.id, reason.trim())
            : await adminApi.releaseEscrow(order.id, action.recipient, reason.trim());
        setMessage(r.message);
        setError('');
        setConfirming(null);
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
        the second executes the ledger transfer. Look up the order first — freeze/release are
        disabled until the order details are confirmed on screen.
      </p>

      <div className="flex flex-col gap-3">
        <label className="text-sm font-semibold">Order UUID</label>
        <div className="flex gap-2">
          <input
            value={orderId}
            onChange={(e) => {
              setOrderId(e.target.value);
              setOrder(null);
            }}
            placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            className="flex-1 border border-line rounded-xl px-4 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald"
          />
          <button
            disabled={isLookingUp || !orderId}
            onClick={lookupOrder}
            className="font-display text-sm font-semibold rounded-[10px] px-4 py-2.5 border-[1.5px] border-line disabled:opacity-50"
          >
            {isLookingUp ? 'Looking up…' : 'Look up'}
          </button>
        </div>
        {lookupError && <p className="text-sm text-red-600">{lookupError}</p>}
      </div>

      {order && (
        <div className="border border-line rounded-xl px-4 py-3 flex flex-col gap-1 text-sm">
          <div className="flex justify-between">
            <span className="text-mid">Order</span>
            <span className="font-mono">{order.id}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-mid">Status</span>
            <span className="font-semibold">{order.status}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-mid">Amount</span>
            <span className="font-display font-semibold text-[18px]">
              {formatCurrency({ amount: order.totalKobo, currency: 'NGN' })}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-mid">Customer</span>
            <span className="font-mono">{order.customerId}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-mid">Merchant</span>
            <span className="font-mono">{order.merchantId}</span>
          </div>
        </div>
      )}

      {error && <p className="text-sm text-red-600">{error}</p>}
      {message && (
        <p className="text-sm font-semibold text-emerald bg-[#E9F3D8] rounded-xl px-4 py-3">{message}</p>
      )}

      <div className="flex gap-3 flex-wrap">
        <button
          disabled={!order || isPending}
          onClick={() => openConfirm({ kind: 'freeze' })}
          className="font-display text-sm font-semibold rounded-[10px] px-5 py-2.5 border-[1.5px] transition-colors disabled:opacity-50"
          style={{ color: '#8A6A1B', borderColor: '#E8B14E' }}
        >
          Freeze Escrow
        </button>
        <button
          disabled={!order || isPending}
          onClick={() => openConfirm({ kind: 'release', recipient: 'customer' })}
          className="font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] px-5 py-2.5 hover:bg-lime-600 transition-colors disabled:opacity-50"
        >
          Release → Customer
        </button>
        <button
          disabled={!order || isPending}
          onClick={() => openConfirm({ kind: 'release', recipient: 'merchant' })}
          className="font-display text-sm font-semibold bg-[#0A3D2C] text-sand rounded-[10px] px-5 py-2.5 hover:bg-[#0D4E38] transition-colors disabled:opacity-50"
        >
          Release → Merchant
        </button>
      </div>

      {confirming && order && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
          <div className="bg-white rounded-2xl p-6 max-w-md w-full flex flex-col gap-4">
            <h2 className="font-display font-semibold text-[20px]">
              {confirming.kind === 'freeze'
                ? 'Confirm: Freeze Escrow'
                : `Confirm: Release ${formatCurrency({ amount: order.totalKobo, currency: 'NGN' })} → ${confirming.recipient}`}
            </h2>
            <div className="border border-line rounded-xl px-4 py-3 flex flex-col gap-1 text-sm bg-sand/40">
              <div className="flex justify-between">
                <span className="text-mid">Order</span>
                <span className="font-mono">{order.id}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-mid">Amount</span>
                <span className="font-semibold">{formatCurrency({ amount: order.totalKobo, currency: 'NGN' })}</span>
              </div>
              {confirming.kind === 'release' && (
                <div className="flex justify-between">
                  <span className="text-mid">Recipient</span>
                  <span className="font-semibold capitalize">
                    {confirming.recipient} ({confirming.recipient === 'customer' ? order.customerId : order.merchantId})
                  </span>
                </div>
              )}
            </div>

            <label className="text-sm font-semibold">Reason (required)</label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
              placeholder="Why is this action being taken?"
            />

            <label className="text-sm font-semibold">
              Type the amount shown above (<span className="font-mono">{expectedConfirmText}</span>) to confirm
            </label>
            <input
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              className="border border-line rounded-xl px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald"
              placeholder={expectedConfirmText}
            />

            <div className="flex gap-3 justify-end pt-2">
              <button
                onClick={() => setConfirming(null)}
                className="font-display text-sm font-semibold rounded-[10px] px-4 py-2.5 border-[1.5px] border-line"
              >
                Cancel
              </button>
              <button
                disabled={!canSubmit || isPending}
                onClick={submitConfirmedAction}
                className="font-display text-sm font-semibold text-white bg-red-600 rounded-[10px] px-5 py-2.5 hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                {isPending ? 'Processing…' : 'Confirm & Execute'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
