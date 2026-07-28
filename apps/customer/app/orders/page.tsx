'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ordersApi } from '@speedplus/api-client';
import { apiClient } from '@speedplus/api-client';
import type { ApiResponse } from '@speedplus/types';
import { Skeleton } from '@speedplus/ui';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

function fmt(iso: string) {
  return new Date(iso).toLocaleDateString('en-NG', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' });
}

const VERTICALS = ['all', 'package', 'food', 'grocery', 'pharmacy', 'gas'];
const STATUSES = ['all', 'pending', 'in_transit', 'delivered', 'cancelled'];

const STATUS_LABEL: Record<string, string> = {
  pending: 'Pending', confirmed: 'Confirmed', preparing: 'Preparing',
  ready_for_pickup: 'Ready', driver_assigned: 'Rider assigned',
  in_transit: 'On the way', delivered: 'Delivered',
  cancelled: 'Cancelled', refunded: 'Refunded',
};

const STATUS_COLOR: Record<string, { bg: string; text: string }> = {
  delivered:       { bg: '#E9F3D8', text: '#0A3D2C' },
  in_transit:      { bg: '#E9F3D8', text: '#0A3D2C' },
  driver_assigned: { bg: '#E9F3D8', text: '#0A3D2C' },
  ready_for_pickup:{ bg: '#FFF7E6', text: '#8A6A1B' },
  preparing:       { bg: '#FFF7E6', text: '#8A6A1B' },
  confirmed:       { bg: '#FFF7E6', text: '#8A6A1B' },
  pending:         { bg: '#F7F5EF', text: '#63636E' },
  cancelled:       { bg: '#FEF2F2', text: '#DC2626' },
  refunded:        { bg: '#FEF2F2', text: '#DC2626' },
};

const VERTICAL_ICON: Record<string, React.ReactNode> = {
  package:  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" /><polyline points="3.27 6.96 12 12.01 20.73 6.96" /><line x1="12" y1="22.08" x2="12" y2="12" /></svg>,
  food:     <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M18 8h1a4 4 0 010 8h-1" /><path d="M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" /><line x1="6" y1="1" x2="6" y2="4" /><line x1="10" y1="1" x2="10" y2="4" /><line x1="14" y1="1" x2="14" y2="4" /></svg>,
  gas:      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" /><circle cx="12" cy="9" r="2.5" /></svg>,
  grocery:  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z" /><line x1="3" y1="6" x2="21" y2="6" /><path d="M16 10a4 4 0 01-8 0" /></svg>,
  pharmacy: <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /><line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" /></svg>,
};

const ACTIVE = new Set(['pending', 'confirmed', 'preparing', 'ready_for_pickup', 'driver_assigned', 'in_transit']);

interface Receipt {
  orderId: string; vertical: string; status: string; paymentMethod: string;
  items: { name: string; quantity: number; unitPriceKobo: number; totalKobo: number }[];
  subtotalKobo: number; deliveryKobo: number; serviceKobo: number; tipKobo: number; totalKobo: number;
  merchantName: string; driverName: string; createdAt: string; deliveredAt?: string;
  review?: { rating: number; comment?: string };
}

function StarRating({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <div className="flex gap-1">
      {[1, 2, 3, 4, 5].map((s) => (
        <button key={s} type="button" onClick={() => onChange(s)} className="focus-visible:outline-none">
          <svg width="28" height="28" viewBox="0 0 24 24" fill={s <= value ? '#E8B14E' : 'none'} stroke={s <= value ? '#E8B14E' : '#D5D2C8'} strokeWidth={1.5}>
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
          </svg>
        </button>
      ))}
    </div>
  );
}

export default function OrdersPage() {
  const router = useRouter();
  const qc = useQueryClient();

  const [vertical, setVertical] = useState('all');
  const [status, setStatus] = useState('all');
  const [receiptOrder, setReceiptOrder] = useState<Receipt | null>(null);
  const [reviewOrder, setReviewOrder] = useState<{ orderId: string; driverId?: string; merchantId: string } | null>(null);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState('');
  const [reviewTarget, setReviewTarget] = useState<'driver' | 'merchant'>('driver');

  const { data, isLoading } = useQuery({
    queryKey: ['orders-list', vertical, status],
    queryFn: () => ordersApi.list({
      ...(vertical !== 'all' && { vertical }),
      ...(status !== 'all' && { status }),
    }),
    staleTime: 30_000,
  });

  const orders = data?.orders ?? [];

  async function openReceipt(orderId: string) {
    const { data } = await apiClient.get<ApiResponse<Receipt>>(`/orders/${orderId}/receipt`);
    if (data.success) setReceiptOrder(data.data);
  }

  const reviewMutation = useMutation({
    mutationFn: ({ orderId, revieweeId, revieweeType, rating, comment }: {
      orderId: string; revieweeId: string; revieweeType: string; rating: number; comment?: string;
    }) => apiClient.post(`/orders/${orderId}/review`, { revieweeId, revieweeType, rating, comment }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['orders-list'] });
      setReviewOrder(null);
      setRating(5);
      setComment('');
    },
  });

  function submitReview() {
    if (!reviewOrder) return;
    const revieweeId = reviewTarget === 'driver' ? reviewOrder.driverId : reviewOrder.merchantId;
    if (!revieweeId) return;
    reviewMutation.mutate({ orderId: reviewOrder.orderId, revieweeId, revieweeType: reviewTarget, rating, comment: comment || undefined });
  }

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      {/* Header */}
      <div className="bg-[#0A3D2C] px-5 pt-12 pb-5 flex items-center gap-3">
        <button onClick={() => router.back()} className="w-9 h-9 rounded-full bg-white/10 flex items-center justify-center hover:bg-white/20 transition-colors" aria-label="Back">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
        </button>
        <h1 className="font-display font-semibold text-white text-[18px]">My orders</h1>
      </div>

      {/* Filters */}
      <div className="px-5 pt-4 pb-2 flex flex-col gap-2.5">
        <div className="flex gap-2 overflow-x-auto pb-1 scrollbar-none">
          {VERTICALS.map((v) => (
            <button key={v} onClick={() => setVertical(v)}
              className={`flex-shrink-0 rounded-full px-3.5 py-1.5 text-[12px] font-semibold capitalize transition-colors ${vertical === v ? 'bg-[#0A3D2C] text-white' : 'bg-white border border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30'}`}>
              {v === 'all' ? 'All types' : v}
            </button>
          ))}
        </div>
        <div className="flex gap-2 overflow-x-auto pb-1 scrollbar-none">
          {STATUSES.map((s) => (
            <button key={s} onClick={() => setStatus(s)}
              className={`flex-shrink-0 rounded-full px-3.5 py-1.5 text-[12px] font-semibold capitalize transition-colors ${status === s ? 'bg-[#0A3D2C] text-white' : 'bg-white border border-[#E4E0D6] text-[#63636E] hover:border-[#0A3D2C]/30'}`}>
              {s === 'all' ? 'All statuses' : STATUS_LABEL[s] ?? s}
            </button>
          ))}
        </div>
      </div>

      {/* List */}
      <div className="flex-1 px-5 py-3 flex flex-col gap-2.5 max-w-[600px] mx-auto w-full pb-8">
        {isLoading && [1, 2, 3].map((i) => (
          <div key={i} className="bg-white rounded-2xl border border-[#E4E0D6] p-4 flex items-center gap-3">
            <Skeleton className="w-10 h-10 rounded-xl" />
            <div className="flex-1 flex flex-col gap-2"><Skeleton className="h-3.5 w-32" /><Skeleton className="h-3 w-20" /></div>
            <Skeleton className="h-4 w-16" />
          </div>
        ))}

        {!isLoading && orders.length === 0 && (
          <div className="flex flex-col items-center gap-4 py-20 text-center">
            <div className="w-16 h-16 rounded-2xl bg-white border border-[#E4E0D6] flex items-center justify-center text-[#9A968D]">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" /></svg>
            </div>
            <div>
              <p className="font-display font-semibold text-[18px] text-[#121216]">No orders found</p>
              <p className="text-[13px] text-[#63636E] mt-1">Try changing the filters above.</p>
            </div>
          </div>
        )}

        {orders.map((order, i) => {
          const isActive = ACTIVE.has(order.status);
          const sc = STATUS_COLOR[order.status] ?? { bg: '#F7F5EF', text: '#63636E' };
          const isDelivered = order.status === 'delivered';
          return (
            <div
              key={order.id}
              className="bg-white rounded-2xl border border-[#E4E0D6] px-4 py-3.5 flex flex-col gap-2.5"
              style={{ animation: `fadeUp 0.2s cubic-bezier(0.16,1,0.3,1) ${i * 30}ms both` }}
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-[#F7F5EF] flex items-center justify-center flex-shrink-0 text-[#63636E]">
                  {VERTICAL_ICON[order.vertical] ?? VERTICAL_ICON.package}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-[13px] font-semibold text-[#121216] capitalize">{order.vertical} delivery</p>
                  <p className="text-[11px] text-[#9A968D] mt-0.5">{fmt(order.createdAt)}</p>
                </div>
                <div className="flex flex-col items-end gap-1.5 flex-shrink-0">
                  <p className="font-display font-semibold text-[14px] text-[#121216]">{naira(order.total.amount)}</p>
                  <span className="text-[10px] font-bold rounded-full px-2 py-0.5" style={{ background: sc.bg, color: sc.text }}>
                    {STATUS_LABEL[order.status] ?? order.status}
                  </span>
                </div>
              </div>

              <div className="flex gap-2">
                {isActive && order.vertical === 'package' && (
                  <button
                    onClick={() => router.push('/package/tracking')}
                    className="flex-1 text-center text-[12px] font-semibold text-[#0A3D2C] bg-[#E9F3D8] rounded-xl py-2 hover:bg-[#D4EAC0] transition-colors"
                  >
                    Track order
                  </button>
                )}
                <button
                  onClick={() => openReceipt(order.id)}
                  className="flex-1 text-center text-[12px] font-semibold text-[#63636E] bg-[#F7F5EF] rounded-xl py-2 hover:bg-[#EFECE3] transition-colors"
                >
                  View receipt
                </button>
                {isDelivered && (
                  <button
                    onClick={() => setReviewOrder({ orderId: order.id, driverId: order.driverId ?? undefined, merchantId: order.merchantId })}
                    className="flex-1 text-center text-[12px] font-semibold text-[#0A3D2C] border border-[#0A3D2C]/20 rounded-xl py-2 hover:bg-[#E9F3D8] transition-colors"
                  >
                    Rate & review
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Receipt modal */}
      {receiptOrder && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/40 px-0 sm:px-4" onClick={() => setReceiptOrder(null)}>
          <div className="bg-white w-full sm:max-w-[480px] rounded-t-3xl sm:rounded-2xl p-6 flex flex-col gap-4 max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <div>
                <p className="font-display font-bold text-[18px] text-[#121216]">Receipt</p>
                <p className="text-[11px] text-[#9A968D] mt-0.5">#{receiptOrder.orderId.slice(0, 8).toUpperCase()}</p>
              </div>
              <button onClick={() => setReceiptOrder(null)} className="w-8 h-8 rounded-full bg-[#F7F5EF] flex items-center justify-center hover:bg-[#EFECE3] transition-colors">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#63636E" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
              </button>
            </div>

            {/* Meta */}
            <div className="bg-[#F7F5EF] rounded-xl px-4 py-3 flex flex-col gap-1.5 text-[12px]">
              <div className="flex justify-between"><span className="text-[#63636E]">From</span><span className="font-semibold text-[#121216]">{receiptOrder.merchantName}</span></div>
              {receiptOrder.driverName && <div className="flex justify-between"><span className="text-[#63636E]">Rider</span><span className="font-semibold text-[#121216]">{receiptOrder.driverName}</span></div>}
              <div className="flex justify-between"><span className="text-[#63636E]">Payment</span><span className="font-semibold capitalize text-[#121216]">{receiptOrder.paymentMethod.replace('_', ' ')}</span></div>
              <div className="flex justify-between"><span className="text-[#63636E]">Date</span><span className="font-semibold text-[#121216]">{fmt(receiptOrder.createdAt)}</span></div>
              {receiptOrder.deliveredAt && <div className="flex justify-between"><span className="text-[#63636E]">Delivered</span><span className="font-semibold text-[#121216]">{fmt(receiptOrder.deliveredAt)}</span></div>}
            </div>

            {/* Items */}
            <div className="flex flex-col gap-1">
              <p className="text-[10.5px] font-semibold text-[#9A968D] tracking-[0.6px]">ITEMS</p>
              {receiptOrder.items.map((item, i) => (
                <div key={i} className="flex justify-between text-[13px]">
                  <span className="text-[#121216]">{item.name} <span className="text-[#9A968D]">×{item.quantity}</span></span>
                  <span className="font-medium text-[#121216]">{naira(item.totalKobo)}</span>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="border-t border-[#E4E0D6] pt-3 flex flex-col gap-1.5">
              <div className="flex justify-between text-[12px] text-[#63636E]"><span>Subtotal</span><span>{naira(receiptOrder.subtotalKobo)}</span></div>
              <div className="flex justify-between text-[12px] text-[#63636E]"><span>Delivery</span><span>{naira(receiptOrder.deliveryKobo)}</span></div>
              {receiptOrder.serviceKobo > 0 && <div className="flex justify-between text-[12px] text-[#63636E]"><span>Service fee</span><span>{naira(receiptOrder.serviceKobo)}</span></div>}
              {receiptOrder.tipKobo > 0 && <div className="flex justify-between text-[12px] text-[#63636E]"><span>Tip</span><span>{naira(receiptOrder.tipKobo)}</span></div>}
              <div className="flex justify-between text-[15px] font-bold text-[#0A3D2C] mt-1"><span>Total</span><span>{naira(receiptOrder.totalKobo)}</span></div>
            </div>

            {/* Existing review */}
            {receiptOrder.review && (
              <div className="bg-[#E9F3D8] rounded-xl px-4 py-3">
                <p className="text-[11px] font-semibold text-[#0A3D2C] mb-1">Your review</p>
                <div className="flex gap-0.5 mb-1">
                  {[1,2,3,4,5].map((s) => (
                    <svg key={s} width="14" height="14" viewBox="0 0 24 24" fill={s <= receiptOrder.review!.rating ? '#E8B14E' : 'none'} stroke={s <= receiptOrder.review!.rating ? '#E8B14E' : '#D5D2C8'} strokeWidth={1.5}>
                      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" />
                    </svg>
                  ))}
                </div>
                {receiptOrder.review.comment && <p className="text-[12px] text-[#0A3D2C]/80 italic">"{receiptOrder.review.comment}"</p>}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Review modal */}
      {reviewOrder && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/40 px-0 sm:px-4" onClick={() => setReviewOrder(null)}>
          <div className="bg-white w-full sm:max-w-[420px] rounded-t-3xl sm:rounded-2xl p-6 flex flex-col gap-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <p className="font-display font-bold text-[18px]">Rate your delivery</p>
              <button onClick={() => setReviewOrder(null)} className="w-8 h-8 rounded-full bg-[#F7F5EF] flex items-center justify-center">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#63636E" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
              </button>
            </div>

            {/* Who to review */}
            <div className="flex gap-2">
              {reviewOrder.driverId && (
                <button onClick={() => setReviewTarget('driver')}
                  className={`flex-1 rounded-xl py-2 text-[12px] font-semibold transition-colors ${reviewTarget === 'driver' ? 'bg-[#0A3D2C] text-white' : 'bg-[#F7F5EF] text-[#63636E]'}`}>
                  Rate rider
                </button>
              )}
              <button onClick={() => setReviewTarget('merchant')}
                className={`flex-1 rounded-xl py-2 text-[12px] font-semibold transition-colors ${reviewTarget === 'merchant' ? 'bg-[#0A3D2C] text-white' : 'bg-[#F7F5EF] text-[#63636E]'}`}>
                Rate merchant
              </button>
            </div>

            <div className="flex flex-col items-center gap-2">
              <p className="text-[13px] text-[#63636E]">How was your experience?</p>
              <StarRating value={rating} onChange={setRating} />
            </div>

            <textarea
              placeholder="Leave a comment (optional)"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              rows={3}
              className="border border-[#E4E0D6] rounded-xl px-3 py-2.5 text-[13px] resize-none focus:outline-none focus:ring-2 focus:ring-[#0A3D2C]"
            />

            {reviewMutation.isError && (
              <p className="text-[12px] text-[#DC2626]">{(reviewMutation.error as Error).message}</p>
            )}

            <button
              onClick={submitReview}
              disabled={reviewMutation.isPending}
              className="w-full font-display text-sm font-semibold text-[#0A3D2C] bg-[#C6F24E] rounded-xl py-3 hover:bg-[#B8E040] transition-colors disabled:opacity-50"
            >
              {reviewMutation.isPending ? 'Submitting…' : 'Submit review'}
            </button>
          </div>
        </div>
      )}

      <style>{`
        @keyframes fadeUp { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
        .scrollbar-none::-webkit-scrollbar { display: none; }
      `}</style>
    </main>
  );
}
