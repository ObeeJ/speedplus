'use client';

import { useState, useTransition } from 'react';
import { useQuery } from '@tanstack/react-query';
import { adminApi, type OrderSummary, type OrderDetail } from '@speedplus/api-client';
import { Badge, Skeleton } from '@speedplus/ui';

function formatKobo(k: number) {
  return `₦${(k / 100).toLocaleString('en-NG', { minimumFractionDigits: 2 })}`;
}

export default function OrdersPage() {
  const [detail, setDetail] = useState<OrderDetail | null>(null);
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('');
  const [actionError, setActionError] = useState('');
  const [isPending, startTransition] = useTransition();

  // useQuery rather than useEffect + setState — see drivers/page.tsx. This page
  // matters most: the search box drives the query, so every keystroke fired an
  // uncancelled request and responses could arrive out of order, painting an
  // earlier query's orders under the current search text.
  const { data, isPending: isLoading, error } = useQuery({
    queryKey: ['admin-orders', q, status],
    queryFn: () => adminApi.searchOrders(q || undefined, status || undefined),
  });
  const orders: OrderSummary[] = data?.orders ?? [];
  const errorMessage = actionError || (error instanceof Error ? error.message : '');

  function openDetail(id: string) {
    adminApi.getOrderDetail(id)
      .then(setDetail)
      .catch((e: Error) => setActionError(e.message));
  }

  function handleAssign(orderId: string) {
    const driverId = window.prompt('Driver UUID:');
    if (!driverId) return;
    startTransition(async () => {
      try {
        await adminApi.assignDriver(orderId, driverId);
        setDetail(null);
      } catch (e: unknown) {
        setActionError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  if (detail) {
    return (
      <div className="px-8 py-7 flex flex-col gap-4">
        <button onClick={() => setDetail(null)} className="text-sm text-mid hover:text-emerald transition-colors">← Back</button>
        <h1 className="font-display font-semibold text-[22px]">Order {detail.id.slice(0, 8)}…</h1>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <span><b>Status:</b> {detail.status}</span>
          <span><b>Total:</b> {formatKobo(detail.totalKobo)}</span>
          <span><b>Vertical:</b> {detail.vertical}</span>
          <span><b>Driver:</b> {detail.driverId ?? 'Unassigned'}</span>
        </div>
        {!detail.driverId && (
          <button
            disabled={isPending}
            onClick={() => handleAssign(detail.id)}
            className="self-start font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
          >
            Assign Driver
          </button>
        )}
        <h2 className="font-display font-semibold text-[16px] mt-2">Timeline</h2>
        <ol className="flex flex-col gap-2">
          {detail.events.map((ev) => (
            <li key={ev.id} className="flex gap-3 text-sm">
              <span className="text-mid w-36 shrink-0">{new Date(ev.createdAt).toLocaleTimeString()}</span>
              <span>
                <b>{ev.actorRole}</b> · {ev.fromStatus || '—'} → <b>{ev.toStatus}</b>
                {ev.note && <span className="text-mid"> ({ev.note})</span>}
              </span>
            </li>
          ))}
        </ol>
      </div>
    );
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[22px] sm:text-[26px] tracking-tight">Orders</h1>
      <div className="flex gap-3 flex-wrap">
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search by order ID or customer ID…"
          className="flex-1 min-w-[200px] border border-line rounded-xl px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="Search by order ID or customer ID"/>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none"
        >
          <option value="">All statuses</option>
          {['pending','confirmed','preparing','ready_for_pickup','driver_assigned','in_transit','delivered','cancelled','refunded'].map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </div>
      {errorMessage && <p className="text-sm text-red-600">{errorMessage}</p>}
      <div className="bg-white border border-line rounded-2xl overflow-x-auto">
        <div className="min-w-[480px]">
        <div className="grid px-5 py-3 border-b border-[#EFECE3] text-[10.5px] font-semibold text-mid tracking-[.5px]"
          style={{ gridTemplateColumns: '1fr 1fr 120px 100px' }}>
          <span>ORDER ID</span><span>VERTICAL</span><span>STATUS</span><span>TOTAL</span>
        </div>
        {isLoading && (
          <div className="p-4 flex flex-col gap-3">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="grid gap-4 items-center" style={{ gridTemplateColumns: '1fr 1fr 120px 100px' }}>
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-5 w-20 rounded-full" />
                <Skeleton className="h-4 w-14" />
              </div>
            ))}
          </div>
        )}
        {!isLoading && orders.map((o) => (
          <button
            key={o.id}
            onClick={() => openDetail(o.id)}
            className="w-full grid px-5 py-3 text-[12.5px] items-center border-b border-[#EFECE3] last:border-0 hover:bg-sand/50 transition-colors text-left"
            style={{ gridTemplateColumns: '1fr 1fr 120px 100px' }}
          >
            <span className="font-mono text-xs">{o.id.slice(0, 8)}…</span>
            <span>{o.vertical}</span>
            <Badge>{o.status}</Badge>
            <span className="font-semibold">{formatKobo(o.totalKobo)}</span>
          </button>
        ))}
        {orders.length === 0 && (
          <p className="px-5 py-4 text-sm text-mid">No orders found.</p>
        )}
        </div>
      </div>
    </div>
  );
}
