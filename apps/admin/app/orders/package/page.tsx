'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type OrderDetail } from '@speedplus/api-client';
import { apiClient } from '@speedplus/api-client';
import type { ApiResponse } from '@speedplus/types';
import { Badge, BoxIcon } from '@speedplus/ui';

function naira(k: number) {
  return `₦${(k / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

interface PackageOrderSummary {
  id: string;
  customerId: string;
  driverId?: string;
  status: string;
  totalKobo: number;
  recipientName?: string;
  recipientPhone?: string;
  paymentMethod: string;
  createdAt: string;
}

interface OrderStop {
  id: string;
  sequence: number;
  addressId: string;
  recipientName?: string;
  recipientPhone?: string;
  notes?: string;
  status: string;
  confirmedAt?: string;
}

const STATUS_COLORS: Record<string, string> = {
  pending: 'bg-amber/20 text-amber',
  confirmed: 'bg-blue-100 text-blue-700',
  driver_assigned: 'bg-purple-100 text-purple-700',
  in_transit: 'bg-emerald/20 text-emerald',
  delivered: 'bg-lime/30 text-emerald',
  cancelled: 'bg-red-100 text-red-600',
};

export default function PackageOrdersPage() {
  const [orders, setOrders] = useState<PackageOrderSummary[]>([]);
  const [detail, setDetail] = useState<OrderDetail | null>(null);
  const [stops, setStops] = useState<OrderStop[]>([]);
  const [statusFilter, setStatusFilter] = useState('');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    setError('');
    adminApi.searchOrders(undefined, statusFilter || undefined)
      .then((d) => {
        setOrders(d.orders as unknown as PackageOrderSummary[]);
      })
      .catch((e: Error) => setError(e.message));
  }, [statusFilter]);

  async function openDetail(id: string) {
    try {
      const [det, stopsRes] = await Promise.all([
        adminApi.getOrderDetail(id),
        apiClient.get<ApiResponse<{ stops: OrderStop[] }>>(`/orders/${id}/stops`)
          .then((r) => r.data.success ? r.data.data.stops : [])
          .catch(() => []),
      ]);
      setDetail(det);
      setStops(stopsRes);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load order');
    }
  }

  function handleAssign(orderId: string) {
    const driverId = window.prompt('Enter driver UUID to assign:');
    if (!driverId?.trim()) return;
    startTransition(async () => {
      try {
        await adminApi.assignDriver(orderId, driverId.trim());
        await openDetail(orderId);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Assignment failed');
      }
    });
  }

  if (detail) {
    return (
      <div className="px-8 py-7 flex flex-col gap-5 max-w-3xl">
        <button
          onClick={() => { setDetail(null); setStops([]); }}
          className="text-sm text-mid hover:text-emerald transition-colors self-start"
        >
          ← Back to package orders
        </button>

        <div className="flex items-center gap-3">
          <h1 className="font-display font-semibold text-[22px]">
            Order <span className="font-mono text-[18px]">{detail.id.slice(0, 8)}…</span>
          </h1>
          <span className={`text-[11px] font-bold rounded-full px-3 py-1 ${STATUS_COLORS[detail.status] ?? 'bg-line text-mid'}`}>
            {detail.status.replace(/_/g, ' ')}
          </span>
        </div>

        {/* Order summary */}
        <div className="bg-white border border-line rounded-2xl p-5 grid grid-cols-2 gap-3 text-sm">
          <div><span className="text-mid">Total</span><p className="font-display font-bold text-lg text-emerald">{naira(detail.totalKobo)}</p></div>
          <div><span className="text-mid">Payment</span><p className="font-semibold capitalize">{(detail as unknown as { paymentMethod?: string }).paymentMethod ?? 'wallet'}</p></div>
          <div><span className="text-mid">Customer</span><p className="font-mono text-xs">{detail.customerId}</p></div>
          <div><span className="text-mid">Driver</span><p className="font-mono text-xs">{detail.driverId ?? '—'}</p></div>
          {(detail as unknown as { recipientName?: string }).recipientName && (
            <div><span className="text-mid">Recipient</span><p className="font-semibold">{(detail as unknown as { recipientName?: string }).recipientName}</p></div>
          )}
          {(detail as unknown as { recipientPhone?: string }).recipientPhone && (
            <div><span className="text-mid">Recipient phone</span><p className="font-semibold">{(detail as unknown as { recipientPhone?: string }).recipientPhone}</p></div>
          )}
        </div>

        {/* Multi-drop stops */}
        {stops.length > 0 && (
          <div className="flex flex-col gap-2">
            <h2 className="font-display font-semibold text-[16px]">Delivery Stops ({stops.length})</h2>
            <div className="bg-white border border-line rounded-2xl overflow-hidden">
              {stops.map((stop, i) => (
                <div
                  key={stop.id}
                  className={`flex items-center gap-4 px-5 py-3.5 ${i < stops.length - 1 ? 'border-b border-[#EFECE3]' : ''}`}
                >
                  <span className="w-7 h-7 rounded-full bg-tile flex items-center justify-center font-display font-bold text-emerald text-sm flex-shrink-0">
                    {stop.sequence}
                  </span>
                  <div className="flex-1 flex flex-col gap-0.5">
                    <span className="text-[13px] font-semibold">{stop.recipientName ?? 'Recipient'}</span>
                    {stop.recipientPhone && <span className="text-[11px] text-mid">{stop.recipientPhone}</span>}
                    {stop.notes && <span className="text-[11px] text-mid italic">{stop.notes}</span>}
                  </div>
                  <span className={`text-[11px] font-bold rounded-full px-2.5 py-1 ${
                    stop.status === 'confirmed' ? 'bg-lime/30 text-emerald' : 'bg-line text-mid'
                  }`}>
                    {stop.status}
                  </span>
                  {stop.confirmedAt && (
                    <span className="text-[10px] text-mid">{new Date(stop.confirmedAt).toLocaleTimeString()}</span>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Manual dispatch */}
        {!detail.driverId && (
          <div className="flex items-center gap-3 bg-[#FFF7E6] border border-[#F0DFB4] rounded-xl px-4 py-3">
            <span className="text-[12.5px] text-[#8A6A1B] flex-1">No driver assigned. You can manually assign one.</span>
            <button
              disabled={isPending}
              onClick={() => handleAssign(detail.id)}
              className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
            >
              Assign Driver
            </button>
          </div>
        )}

        {/* Order timeline */}
        <div className="flex flex-col gap-2">
          <h2 className="font-display font-semibold text-[16px]">Timeline</h2>
          <div className="bg-white border border-line rounded-2xl overflow-hidden">
            {detail.events.map((ev, i) => (
              <div
                key={ev.id}
                className={`flex gap-4 px-5 py-3 text-sm ${i < detail.events.length - 1 ? 'border-b border-[#EFECE3]' : ''}`}
              >
                <span className="text-mid w-28 shrink-0 text-[11px]">
                  {new Date(ev.createdAt).toLocaleTimeString()}
                </span>
                <span className="flex-1">
                  <span className="font-semibold capitalize">{ev.actorRole}</span>
                  {' · '}
                  <span className="text-mid">{ev.fromStatus || '—'}</span>
                  {' → '}
                  <span className="font-semibold">{ev.toStatus}</span>
                  {ev.note && <span className="text-mid"> ({ev.note})</span>}
                </span>
              </div>
            ))}
          </div>
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}
      </div>
    );
  }

  return (
    <div className="px-8 py-7 flex flex-col gap-4">
      <div className="flex items-baseline justify-between">
        <h1 className="font-display font-semibold text-[26px] tracking-tight flex items-center gap-2">
          <BoxIcon size={24} />
          Package Orders
        </h1>
        <span className="text-sm text-mid">{orders.length} orders</span>
      </div>

      {/* Status filter */}
      <div className="flex gap-2 flex-wrap">
        {['', 'pending', 'driver_assigned', 'in_transit', 'delivered', 'cancelled'].map((s) => (
          <button
            key={s}
            onClick={() => setStatusFilter(s)}
            className={`text-[11px] font-bold rounded-full px-3 py-1 transition-colors ${
              statusFilter === s
                ? 'bg-[#0A3D2C] text-sand'
                : 'bg-white border border-line text-mid hover:bg-sand'
            }`}
          >
            {s || 'All'}
          </button>
        ))}
      </div>

      {error && <p className="text-sm text-red-600">{error}</p>}

      <div className="bg-white border border-line rounded-2xl overflow-hidden">
        <div
          className="grid px-5 py-3 border-b border-[#EFECE3] text-[10.5px] font-semibold text-mid tracking-[.5px]"
          style={{ gridTemplateColumns: '1fr 1fr 1fr 120px 100px 80px' }}
        >
          <span>ORDER ID</span>
          <span>RECIPIENT</span>
          <span>DRIVER</span>
          <span>STATUS</span>
          <span>TOTAL</span>
          <span>PAYMENT</span>
        </div>

        {orders.length === 0 && (
          <p className="px-5 py-4 text-sm text-mid">No package orders found.</p>
        )}

        {orders.map((o) => (
          <button
            key={o.id}
            onClick={() => openDetail(o.id)}
            className="w-full grid px-5 py-3 text-[12.5px] items-center border-b border-[#EFECE3] last:border-0 hover:bg-sand/50 transition-colors text-left"
            style={{ gridTemplateColumns: '1fr 1fr 1fr 120px 100px 80px' }}
          >
            <span className="font-mono text-xs">{o.id.slice(0, 8)}…</span>
            <span className="flex flex-col">
              <span className="font-semibold">{o.recipientName ?? '—'}</span>
              {o.recipientPhone && <span className="text-[10px] text-mid">{o.recipientPhone}</span>}
            </span>
            <span className="font-mono text-xs text-mid">{o.driverId ? o.driverId.slice(0, 8) + '…' : 'Unassigned'}</span>
            <span>
              <span className={`text-[10px] font-bold rounded-full px-2 py-0.5 ${STATUS_COLORS[o.status] ?? 'bg-line text-mid'}`}>
                {o.status.replace(/_/g, ' ')}
              </span>
            </span>
            <span className="font-semibold">{naira(o.totalKobo)}</span>
            <span className="text-[10px] text-mid capitalize">{o.paymentMethod ?? 'wallet'}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
