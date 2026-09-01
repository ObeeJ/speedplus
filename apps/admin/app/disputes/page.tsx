'use client';

import { useState, useTransition } from 'react';
import { adminApi, proofApi, type OrderDetail, type ProofMediaView } from '@fourdat/api-client';
import { formatCurrency } from '@fourdat/utils';
import { Card, Button, Badge } from '@fourdat/ui';

type PendingAction =
  | { kind: 'freeze' }
  | { kind: 'release'; recipient: 'customer' | 'merchant' };

export default function DisputesPage() {
  const [orderId, setOrderId] = useState('');
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [media, setMedia] = useState<ProofMediaView[]>([]);
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
    setMedia([]);
    startLookup(async () => {
      try {
        const detail = await adminApi.getOrderDetail(orderId);
        setOrder(detail);
        // Proof-of-delivery chain of custody — best-effort, doesn't block lookup.
        try {
          setMedia(await proofApi.getMedia(orderId));
        } catch {
          setMedia([]);
        }
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
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6 max-w-xl">
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
            className="flex-1 border border-line rounded-xl px-4 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="Order UUID"/>
          <Button
            variant="outline"
            size="sm"
            disabled={isLookingUp || !orderId}
            onClick={lookupOrder}
          >
            {isLookingUp ? 'Looking up…' : 'Look up'}
          </Button>
        </div>
        {lookupError && <p className="text-sm text-red-600">{lookupError}</p>}
      </div>

      {order && (
        <Card className="flex flex-col gap-1 text-sm">
          <div className="flex justify-between">
            <span className="text-mid">Order</span>
            <span className="font-mono">{order.id}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-mid">Status</span>
            <Badge variant="default">{order.status}</Badge>
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
        </Card>
      )}

      {order && (
        <Card className="flex flex-col gap-3">
          <span className="text-[12px] font-semibold uppercase tracking-wide text-mid">
            Proof of delivery — chain of custody
          </span>
          {media.length === 0 ? (
            <span className="text-sm text-mid">No proof media recorded for this order.</span>
          ) : (
            <div className="grid grid-cols-2 gap-3 min-[700px]:grid-cols-3">
              {media.map((m) => (
                <a
                  key={m.id}
                  href={m.viewUrl}
                  target="_blank"
                  rel="noreferrer"
                  className="flex flex-col gap-1 rounded-lg border border-line p-2 hover:bg-sand transition-colors"
                >
                  <span className="text-[11px] font-semibold capitalize">
                    {m.kind.replace(/_/g, ' ')}
                  </span>
                  {m.kind.endsWith('_video') ? (
                    <video src={m.viewUrl} controls className="w-full rounded aspect-square object-cover bg-black" />
                  ) : (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={m.viewUrl} alt={m.kind} className="w-full rounded aspect-square object-cover bg-line" />
                  )}
                  <span className="text-[10px] text-mid">{new Date(m.capturedAt).toLocaleString()}</span>
                  {m.sealSerial && <span className="text-[10px] text-mid">Seal #{m.sealSerial}</span>}
                  <span className="text-[9px] font-mono text-mid break-all">sha256:{m.sha256.slice(0, 16)}…</span>
                  {m.capturedLat != null && m.capturedLng != null && (
                    <span className="text-[10px] text-mid">
                      GPS {m.capturedLat.toFixed(4)}, {m.capturedLng.toFixed(4)}
                    </span>
                  )}
                </a>
              ))}
            </div>
          )}
        </Card>
      )}

      {error && <p className="text-sm text-red-600">{error}</p>}
      {message && (
        <p className="text-sm font-semibold text-emerald bg-tile rounded-xl px-4 py-3">{message}</p>
      )}

      <div className="flex gap-3 flex-wrap">
        <Button
          variant="outline"
          size="sm"
          disabled={!order || isPending}
          onClick={() => openConfirm({ kind: 'freeze' })}
          className="border-amber text-amber hover:bg-amber/10"
        >
          Freeze Escrow
        </Button>
        <Button
          variant="primary"
          size="sm"
          disabled={!order || isPending}
          onClick={() => openConfirm({ kind: 'release', recipient: 'customer' })}
        >
          Release → Customer
        </Button>
        <Button
          variant="secondary"
          size="sm"
          disabled={!order || isPending}
          onClick={() => openConfirm({ kind: 'release', recipient: 'merchant' })}
        >
          Release → Merchant
        </Button>
      </div>

      {confirming && order && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
          <Card className="max-w-md w-full flex flex-col gap-4">
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
            aria-label="Why is this action being taken?" />

            <label className="text-sm font-semibold">
              Type the amount shown above (<span className="font-mono">{expectedConfirmText}</span>) to confirm
            </label>
            <input
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              className="border border-line rounded-xl px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald"
              placeholder={expectedConfirmText} aria-label="Type the amount shown above () to confirm"/>

            <div className="flex gap-3 justify-end pt-2">
              <Button variant="outline" size="sm" onClick={() => setConfirming(null)}>Cancel</Button>
              <Button
                variant="danger"
                size="sm"
                disabled={!canSubmit || isPending}
                isLoading={isPending}
                onClick={submitConfirmedAction}
              >
                Confirm & Execute
              </Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
