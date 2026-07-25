'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { ordersApi } from '@speedplus/api-client';
import { Skeleton } from '@speedplus/ui';
import type { Order } from '@speedplus/types';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

const STATUS_LABEL: Record<string, string> = {
  pending: 'Pending', confirmed: 'Confirmed', preparing: 'Preparing',
  ready_for_pickup: 'Ready', driver_assigned: 'Rider assigned',
  in_transit: 'On the way', delivered: 'Delivered',
  cancelled: 'Cancelled', refunded: 'Refunded',
};

const STATUS_STYLE: Record<string, string> = {
  delivered:      'bg-[#E9F3D8] text-[#0A3D2C]',
  in_transit:     'bg-[#E9F3D8] text-[#0A3D2C]',
  driver_assigned:'bg-[#E9F3D8] text-[#0A3D2C]',
  cancelled:      'bg-[#FEF2F2] text-[#DC2626]',
  refunded:       'bg-[#FEF2F2] text-[#DC2626]',
};

function VerticalIcon({ vertical }: { vertical: string }) {
  const icons: Record<string, React.ReactNode> = {
    package: <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" /><polyline points="3.27 6.96 12 12.01 20.73 6.96" /><line x1="12" y1="22.08" x2="12" y2="12" /></svg>,
    food:    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M18 8h1a4 4 0 010 8h-1" /><path d="M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" /><line x1="6" y1="1" x2="6" y2="4" /><line x1="10" y1="1" x2="10" y2="4" /><line x1="14" y1="1" x2="14" y2="4" /></svg>,
    gas:     <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" /><circle cx="12" cy="9" r="2.5" /></svg>,
    grocery: <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z" /><line x1="3" y1="6" x2="21" y2="6" /><path d="M16 10a4 4 0 01-8 0" /></svg>,
    pharmacy:<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /><line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" /></svg>,
  };
  return (
    <div className="w-10 h-10 rounded-xl bg-[#F7F5EF] flex items-center justify-center flex-shrink-0 text-[#63636E]">
      {icons[vertical] ?? icons.package}
    </div>
  );
}

const ACTIVE = new Set(['pending', 'confirmed', 'preparing', 'ready_for_pickup', 'driver_assigned', 'in_transit']);

export default function OrdersPage() {
  const router = useRouter();
  const { data, isLoading } = useQuery({ queryKey: ['orders-list'], queryFn: () => ordersApi.list({ page: 1 }), staleTime: 30_000 });
  const orders: Order[] = data?.orders ?? [];

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      {/* Header */}
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-5 flex items-center gap-3">
        <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
        </button>
        <h1 className="font-display font-semibold text-white text-[18px]">My orders</h1>
      </div>

      <div className="flex-1 px-5 py-5 flex flex-col gap-3 max-w-[600px] mx-auto w-full">
        {isLoading && (
          <>
            {[1, 2, 3].map((i) => (
              <div key={i} className="bg-white rounded-2xl border border-[#E4E0D6] p-4 flex items-center gap-3">
                <Skeleton className="w-10 h-10 rounded-xl" />
                <div className="flex-1 flex flex-col gap-2">
                  <Skeleton className="h-3.5 w-32" />
                  <Skeleton className="h-3 w-20" />
                </div>
                <Skeleton className="h-4 w-16" />
              </div>
            ))}
          </>
        )}

        {!isLoading && orders.length === 0 && (
          <div className="flex flex-col items-center gap-4 py-20 text-center">
            <div className="w-16 h-16 rounded-2xl bg-white border border-[#E4E0D6] flex items-center justify-center text-[#9A968D]">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
              </svg>
            </div>
            <div>
              <p className="font-display font-semibold text-[18px] text-[#121216]">No orders yet</p>
              <p className="text-[13px] text-[#63636E] mt-1">Your order history will appear here.</p>
            </div>
            <button
              onClick={() => router.push('/')}
              className="font-display text-[13px] font-semibold text-[#0A3D2C] bg-[#E9F3D8] rounded-xl px-6 py-2.5 hover:bg-[#D4EAC0] transition-colors"
            >
              Place your first order
            </button>
          </div>
        )}

        {orders.map((order, i) => {
          const isActive = ACTIVE.has(order.status);
          const statusStyle = STATUS_STYLE[order.status] ?? 'bg-[#F7F5EF] text-[#63636E]';
          return (
            <button
              key={order.id}
              onClick={() => { if (isActive && order.vertical === 'package') router.push('/package/tracking'); }}
              className={`w-full text-left bg-white rounded-2xl border px-4 py-4 flex items-center gap-3 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#C6F24E] active:scale-[0.99] ${isActive ? 'border-[#0A3D2C]/20 shadow-[0_0_0_1px_rgba(10,61,44,0.08)]' : 'border-[#E4E0D6] hover:border-[#0A3D2C]/20'}`}
              style={{ animation: `fadeUp 0.25s cubic-bezier(0.16,1,0.3,1) ${i * 40}ms both` }}
            >
              <VerticalIcon vertical={order.vertical} />
              <div className="flex-1 min-w-0">
                <p className="text-[13px] font-semibold text-[#121216] capitalize">{order.vertical} delivery</p>
                <p className="text-[11px] text-[#9A968D] mt-0.5">
                  {new Date(order.createdAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric' })}
                </p>
              </div>
              <div className="flex flex-col items-end gap-1.5 flex-shrink-0">
                <p className="font-display font-semibold text-[14px] text-[#121216]">{naira(order.total.amount)}</p>
                <span className={`text-[10px] font-bold rounded-full px-2 py-0.5 ${statusStyle}`}>
                  {STATUS_LABEL[order.status] ?? order.status}
                </span>
              </div>
              {isActive && (
                <svg className="flex-shrink-0 text-[#0A3D2C]" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
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
