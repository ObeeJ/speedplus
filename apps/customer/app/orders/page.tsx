'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ordersApi } from '@speedplus/api-client';
import type { Order } from '@speedplus/types';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

const STATUS_LABEL: Record<string, string> = {
  pending: 'Pending',
  confirmed: 'Confirmed',
  preparing: 'Preparing',
  ready_for_pickup: 'Ready',
  driver_assigned: 'Rider assigned',
  in_transit: 'On the way',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  refunded: 'Refunded',
};

const STATUS_COLOR: Record<string, string> = {
  delivered: 'text-emerald bg-tile',
  in_transit: 'text-emerald bg-tile',
  driver_assigned: 'text-emerald bg-tile',
  cancelled: 'text-[#DC2626] bg-[#FEF2F2]',
  refunded: 'text-[#DC2626] bg-[#FEF2F2]',
};

const VERTICAL_ICON: Record<string, string> = {
  package: '📦', food: '🍽️', gas: '🔥', grocery: '🛒', pharmacy: '💊',
};

const ACTIVE = new Set(['pending', 'confirmed', 'preparing', 'ready_for_pickup', 'driver_assigned', 'in_transit']);

export default function OrdersPage() {
  const router = useRouter();

  const { data, isLoading } = useQuery({
    queryKey: ['orders-list'],
    queryFn: () => ordersApi.list({ page: 1 }),
    staleTime: 30_000,
  });

  const orders: Order[] = data?.orders ?? [];

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <div className="bg-emerald px-5 py-5 flex items-center gap-3">
        <button onClick={() => router.back()} className="text-sand/70 hover:text-sand transition-colors" aria-label="Back">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <path d="M19 12H5M12 5l-7 7 7 7" />
          </svg>
        </button>
        <span className="font-display font-semibold text-sand text-lg">My orders</span>
      </div>

      <div className="flex-1 px-5 py-5 flex flex-col gap-3 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:w-full">
        {isLoading && (
          <div className="flex flex-col gap-3">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-20 bg-white border border-line rounded-2xl animate-pulse" />
            ))}
          </div>
        )}

        {!isLoading && orders.length === 0 && (
          <div className="flex flex-col items-center gap-3 py-16 text-center">
            <span className="text-4xl">📦</span>
            <span className="font-display font-semibold text-lg text-ink">No orders yet</span>
            <span className="text-sm text-mid">Your order history will appear here.</span>
            <button
              onClick={() => router.push('/')}
              className="mt-2 font-display text-sm font-semibold text-emerald bg-lime rounded-[13px] px-6 py-2.5 hover:bg-lime-600 transition-colors"
            >
              Place your first order
            </button>
          </div>
        )}

        {orders.map((order, i) => {
          const isActive = ACTIVE.has(order.status);
          return (
            <button
              key={order.id}
              onClick={() => {
                if (isActive && order.vertical === 'package') {
                  router.push('/package/tracking');
                }
              }}
              className="w-full text-left bg-white border border-line rounded-2xl px-4 py-4 flex items-center gap-3 hover:border-emerald/30 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald"
              style={{ animation: `fadeUp 0.25s cubic-bezier(0.16,1,0.3,1) ${i * 40}ms both` }}
            >
              <span className="text-2xl flex-shrink-0">{VERTICAL_ICON[order.vertical] ?? '📦'}</span>
              <div className="flex-1 flex flex-col gap-0.5 min-w-0">
                <span className="text-[13px] font-semibold text-ink capitalize">{order.vertical} delivery</span>
                <span className="text-[11px] text-mid">
                  {new Date(order.createdAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric' })}
                </span>
              </div>
              <div className="flex flex-col items-end gap-1 flex-shrink-0">
                <span className="font-display font-semibold text-[13px] text-ink">{naira(order.total.amount)}</span>
                <span className={`text-[10px] font-bold rounded-full px-2 py-0.5 ${STATUS_COLOR[order.status] ?? 'text-mid bg-line'}`}>
                  {STATUS_LABEL[order.status] ?? order.status}
                </span>
              </div>
              {isActive && (
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" className="flex-shrink-0">
                  <path d="M9 18l6-6-6-6" />
                </svg>
              )}
            </button>
          );
        })}
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(12px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
