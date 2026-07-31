'use client';

import { useState } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { merchantApi, authApi, paycodesApi, type MerchantOrder, type MerchantProduct, type ProductInput, type MerchantPrescription } from '@speedplus/api-client';
import { DashboardIcon, ReceiptIcon, PillIcon, BoxIcon, WalletIcon, ShieldCheckIcon, PowerIcon, type DuotoneIconProps } from '@speedplus/ui';
import { useMerchantStore, type MerchantTab } from '../lib/store/merchant.store';
import { useMerchantAuthStore } from '../lib/store/auth.store';

function naira(kobo: number) {
  return (kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 });
}

const ORDER_STAGE_META: Record<
  string,
  { status: string; chipC: string; chipB: string; dot: string; actLabel: string | null; next: string | null; doneLabel: string | null }
> = {
  pending: { status: 'NEW', chipC: '#0A3D2C', chipB: '#C6F24E', dot: '#C6F24E', actLabel: 'Confirm & prepare', next: 'confirmed', doneLabel: null },
  confirmed: { status: 'CONFIRMED', chipC: '#0A3D2C', chipB: '#C6F24E', dot: '#C6F24E', actLabel: 'Start preparing', next: 'preparing', doneLabel: null },
  preparing: { status: 'PREPARING', chipC: '#8A6A1B', chipB: '#FFF3D6', dot: '#E8B14E', actLabel: 'Mark ready for rider', next: 'ready_for_pickup', doneLabel: null },
  ready_for_pickup: { status: 'AWAITING RIDER', chipC: '#0A3D2C', chipB: '#E9F3D8', dot: '#0A3D2C', actLabel: null, next: null, doneLabel: 'Rider on the way' },
  driver_assigned: { status: 'RIDER ASSIGNED', chipC: '#0A3D2C', chipB: '#E9F3D8', dot: '#0A3D2C', actLabel: null, next: null, doneLabel: 'Rider on the way' },
  in_transit: { status: 'IN TRANSIT', chipC: '#0A3D2C', chipB: '#E9F3D8', dot: '#0A3D2C', actLabel: null, next: null, doneLabel: 'Out for delivery' },
  delivered: { status: 'DELIVERED', chipC: '#63636E', chipB: '#EFECE3', dot: '#BDBAB2', actLabel: null, next: null, doneLabel: 'Completed' },
  cancelled: { status: 'CANCELLED', chipC: '#B4231F', chipB: '#FEF2F2', dot: '#DC2626', actLabel: null, next: null, doneLabel: 'Cancelled' },
};

function FlameIcon({ active }: { active?: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round" opacity={active ? 1 : 0.7}>
      <path d="M12 2c0 0-4 4-4 8a4 4 0 008 0c0-1.5-.5-3-1.5-4.5C14 7 14 9 12 10c0 0 1-2 0-4-1 2-3 3-3 5a3 3 0 006 0c0-3-3-9-3-9z" />
    </svg>
  );
}

function PayIcon({ active }: { active?: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round" opacity={active ? 1 : 0.7}>
      <rect x="2" y="5" width="20" height="14" rx="2" /><path d="M2 10h20" />
    </svg>
  );
}

const NAV_ITEMS: { id: MerchantTab; label: string; Icon: (props: DuotoneIconProps) => React.JSX.Element; gasOnly?: boolean }[] = [
  { id: 'dash',   label: 'Dashboard',    Icon: DashboardIcon },
  { id: 'orders', label: 'Orders',       Icon: ReceiptIcon },
  { id: 'rx',     label: 'Prescriptions',Icon: PillIcon },
  { id: 'prod',   label: 'Products',     Icon: BoxIcon },
  { id: 'earn',   label: 'Earnings',     Icon: WalletIcon },
  { id: 'set',    label: 'Verification', Icon: ShieldCheckIcon },
  { id: 'pay',    label: 'Payments',     Icon: PayIcon as unknown as (props: DuotoneIconProps) => React.JSX.Element },
];

const EMPTY_PRODUCT: ProductInput = { name: '', description: undefined, priceKobo: 0, category: '', isAvailable: true };

export default function MerchantPortalPage() {
  const { tab, setTab } = useMerchantStore();
  const { user, merchant, setMerchant, clearAuth } = useMerchantAuthStore();
  const qc = useQueryClient();

  const profileQuery = useQuery({
    queryKey: ['merchant-profile'],
    queryFn: async () => {
      const p = await merchantApi.getProfile();
      setMerchant(p);
      return p;
    },
    initialData: merchant ?? undefined,
  });

  const ordersQuery = useQuery({
    queryKey: ['merchant-orders'],
    queryFn: () => merchantApi.listOrders(),
    refetchInterval: 15_000,
  });

  const productsQuery = useQuery({
    queryKey: ['merchant-products'],
    queryFn: () => merchantApi.listProducts(),
  });

  const walletQuery = useQuery({
    queryKey: ['merchant-wallet'],
    queryFn: () => merchantApi.getWallet(),
  });

  const transactionsQuery = useQuery({
    queryKey: ['merchant-transactions'],
    queryFn: () => merchantApi.getTransactions(),
    enabled: tab === 'earn',
  });

  const toggleOpenMutation = useMutation({
    mutationFn: (isOpen: boolean) => merchantApi.setOpen(isOpen),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-profile'] }),
  });

  const transitionMutation = useMutation({
    mutationFn: ({ id, to }: { id: string; to: string }) => merchantApi.transitionOrder(id, to),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-orders'] }),
  });

  const toggleProductMutation = useMutation({
    mutationFn: ({ id, available }: { id: string; available: boolean }) => merchantApi.setProductAvailability(id, available),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['merchant-products'] }),
  });

  const [showAddProduct, setShowAddProduct] = useState(false);
  const [newProduct, setNewProduct] = useState<ProductInput>(EMPTY_PRODUCT);

  // Earnings — bank account + withdrawal
  const bankAccountQuery = useQuery({
    queryKey: ['merchant-bank-account'],
    queryFn: () => merchantApi.getBankAccount(),
    enabled: tab === 'earn',
  });
  const [showBankForm, setShowBankForm] = useState(false);
  const [bankDraft, setBankDraft] = useState({ bankCode: '', bankName: '', accountNumber: '', accountName: '' });
  const saveBankMutation = useMutation({
    mutationFn: () => merchantApi.saveBankAccount(bankDraft),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['merchant-bank-account'] }); setShowBankForm(false); },
  });
  const [showWithdraw, setShowWithdraw] = useState(false);
  const [withdrawAmount, setWithdrawAmount] = useState('');
  const [withdrawPin, setWithdrawPin] = useState('');
  const [withdrawType, setWithdrawType] = useState<'standard' | 'instant'>('standard');
  const withdrawMutation = useMutation({
    mutationFn: () => merchantApi.withdraw(
      Math.round(Number(withdrawAmount) * 100),
      withdrawPin,
      crypto.randomUUID(),
      withdrawType,
    ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-wallet'] });
      qc.invalidateQueries({ queryKey: ['merchant-transactions'] });
      setShowWithdraw(false);
      setWithdrawAmount('');
      setWithdrawPin('');
    },
  });
  const createProductMutation = useMutation({
    mutationFn: (input: ProductInput) => merchantApi.createProduct(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-products'] });
      setShowAddProduct(false);
      setNewProduct(EMPTY_PRODUCT);
    },
  });

  // Prescriptions — always polled (not gated on tab) so the sidebar badge
  // shows the pending count regardless of which tab is currently open.
  const prescriptionsQuery = useQuery({
    queryKey: ['merchant-prescriptions'],
    queryFn: () => merchantApi.listPrescriptions('pending'),
    refetchInterval: 20_000,
  });
  const [rejectingId, setRejectingId] = useState<string | null>(null);
  const [rejectNote, setRejectNote] = useState('');

  // Paycode / payment tab state
  const [paycodeOrderId, setPaycodeOrderId] = useState('');
  const [generatedPaycode, setGeneratedPaycode] = useState<{ id: string; payload: string } | null>(null);
  const [scanCardPayload, setScanCardPayload] = useState('');
  const [scanCardPin, setScanCardPin] = useState('');
  const [scanResult, setScanResult] = useState<string | null>(null);

  const generatePaycode = useMutation({
    mutationFn: (orderId: string) => paycodesApi.generate(orderId),
    onSuccess: (data) => setGeneratedPaycode({ id: data.id, payload: data.payload }),
  });

  const scanCard = useMutation({
    mutationFn: () => paycodesApi.scanCard(scanCardPayload.trim(), scanCardPin),
    onSuccess: () => { setScanResult('Payment confirmed'); setScanCardPayload(''); setScanCardPin(''); },
    onError: (e: Error) => setScanResult(e.message),
  });
  const reviewMutation = useMutation({
    mutationFn: ({ id, approve, note }: { id: string; approve: boolean; note?: string }) =>
      merchantApi.reviewPrescription(id, approve, note),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-prescriptions'] });
      setRejectingId(null);
      setRejectNote('');
    },
  });
  const pendingRxCount = prescriptionsQuery.data?.prescriptions.length ?? 0;

  const orders: MerchantOrder[] = ordersQuery.data?.orders ?? [];
  const products: MerchantProduct[] = productsQuery.data?.products ?? [];
  const newCount = orders.filter((o) => o.status === 'pending' || o.status === 'confirmed').length;
  const todaySalesKobo = orders
    .filter((o) => o.status === 'delivered')
    .reduce((sum, o) => sum + o.total.amount, 0);

  return (
    <main className="min-h-screen flex bg-sand">
      <aside className="w-[240px] flex-none bg-emerald p-6 flex flex-col gap-6 min-h-screen">
        <div className="px-2 flex flex-col gap-1">
          <span className="font-display font-bold text-xl text-sand tracking-tight">
            speed<span className="text-lime">+</span> <span className="font-medium text-[11px] text-sand/55">PARTNER</span>
          </span>
          <span className="text-[11px] text-sand/55">
            {profileQuery.data?.businessName ?? 'Loading…'}
          </span>
        </div>

        {profileQuery.data && (
          <button
            onClick={() => toggleOpenMutation.mutate(!profileQuery.data!.isOpen)}
            disabled={toggleOpenMutation.isPending}
            className="flex items-center gap-2.5 rounded-xl px-3.5 py-2.5 transition-colors"
            style={{ background: profileQuery.data.isOpen ? 'rgba(198,242,78,.14)' : 'rgba(245,245,240,.08)' }}
          >
            <span className={`w-2 h-2 rounded-full ${profileQuery.data.isOpen ? 'bg-lime animate-pulse' : 'bg-sand/40'}`} />
            <span className={`text-xs font-bold ${profileQuery.data.isOpen ? 'text-lime' : 'text-sand/70'}`}>
              {profileQuery.data.isOpen ? 'Open for orders' : 'Closed — tap to open'}
            </span>
          </button>
        )}

        <nav className="flex flex-col gap-1">
          {NAV_ITEMS.map((item) => {
            const active = tab === item.id;
            const count = item.id === 'orders' ? newCount : item.id === 'rx' ? pendingRxCount : null;
            return (
              <button
                key={item.id}
                onClick={() => setTab(item.id)}
                className={`flex items-center gap-2.5 rounded-xl px-4 py-2.5 text-[13.5px] font-medium transition-colors ${
                  active ? 'bg-lime/[.14] text-lime font-semibold' : 'text-sand/70 hover:bg-sand/[.08] hover:text-sand'
                }`}
              >
                <item.Icon active={active} color="#C4C0B4" accent="#7BA05B" />
                {item.label}
                {count !== null && count > 0 && (
                  <span className="ml-auto text-[10.5px] font-bold rounded-full px-2 py-0.5" style={{ color: '#0A3D2C', background: '#C6F24E' }}>
                    {count}
                  </span>
                )}
              </button>
            );
          })}
          {/* Gas operations tab — only shown for gas merchants */}
          {profileQuery.data?.vertical === 'gas' && (
            <button
              onClick={() => setTab('gas')}
              className={`flex items-center gap-2.5 rounded-xl px-4 py-2.5 text-[13.5px] font-medium transition-colors ${
                tab === 'gas' ? 'bg-lime/[.14] text-lime font-semibold' : 'text-sand/70 hover:bg-sand/[.08] hover:text-sand'
              }`}
            >
              <FlameIcon active={tab === 'gas'} />
              Gas ops
            </button>
          )}
          <button
            onClick={() => setTab('pay')}
            className={`flex items-center gap-2.5 rounded-xl px-4 py-2.5 text-[13.5px] font-medium transition-colors ${
              tab === 'pay' ? 'bg-lime/[.14] text-lime font-semibold' : 'text-sand/70 hover:bg-sand/[.08] hover:text-sand'
            }`}
          >
            <PayIcon active={tab === 'pay'} />
            Payments
          </button>
        </nav>

        <div className="mt-auto flex items-center gap-2.5 bg-sand/[.06] rounded-[13px] p-2.75">
          <span className="w-9 h-9 rounded-full bg-lime flex items-center justify-center text-emerald font-display font-bold text-[13px]">
            {user?.firstName?.charAt(0) ?? 'M'}
          </span>
          <span className="flex flex-col flex-1 min-w-0">
            <span className="text-[12.5px] font-semibold text-sand truncate">{user?.firstName ?? 'Merchant'}</span>
            <span className="text-[10px] text-sand/55 capitalize">
              {profileQuery.data?.kycStatus === 'approved' ? 'Verified ✓' : profileQuery.data?.kycStatus.replace('_', ' ') ?? '—'}
            </span>
          </span>
          <button onClick={async () => { await authApi.logout().catch(() => {}); clearAuth(); }} className="text-sand/40 hover:text-sand/70" aria-label="Sign out">
            <PowerIcon color="currentColor" />
          </button>
        </div>
      </aside>

      <div className="flex-1 min-w-0 px-8.5 py-7.5 flex flex-col gap-4.5">
        {tab === 'dash' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">
              Good day, {profileQuery.data?.businessName ?? '…'}
            </h1>
            <div className="grid grid-cols-4 gap-3.5">
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">DELIVERED TODAY</span>
                <span className="font-display text-2xl font-bold text-emerald">₦{naira(todaySalesKobo)}</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">ORDERS</span>
                <span className="font-display text-2xl font-bold text-ink">{orders.length}</span>
                <span className="text-[11px] text-mid">{newCount} need action</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">PRODUCTS</span>
                <span className="font-display text-2xl font-bold text-ink">{products.length}</span>
                <span className="text-[11px] text-mid">{products.filter((p) => p.isAvailable).length} listed</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">RATING</span>
                <span className="font-display text-2xl font-bold text-ink">★ {profileQuery.data?.rating.toFixed(1) ?? '—'}</span>
              </div>
            </div>

            {/* Fill accuracy — gas merchants only */}
            {profileQuery.data?.vertical === 'gas' && (
              <div className="grid grid-cols-2 gap-3.5">
                <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                  <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">FILL ACCURACY</span>
                  {profileQuery.data.fillAccuracyPct != null ? (
                    <>
                      <span
                        className="font-display text-2xl font-bold"
                        style={{ color: profileQuery.data.fillAccuracyPct >= 0.98 ? '#0A3D2C' : profileQuery.data.fillAccuracyPct >= 0.95 ? '#8A6A1B' : '#B4231F' }}
                      >
                        {(profileQuery.data.fillAccuracyPct * 100).toFixed(1)}%
                      </span>
                      <span className="text-[11px] text-mid">{profileQuery.data.fillSampleCount ?? 0} verified fills</span>
                    </>
                  ) : (
                    <>
                      <span className="font-display text-2xl font-bold text-mid">—</span>
                      <span className="text-[11px] text-mid">No verified fills yet</span>
                    </>
                  )}
                </div>
                <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                  <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">GAS ORDERS TODAY</span>
                  <span className="font-display text-2xl font-bold text-ink">
                    {orders.filter((o) => o.vertical === 'gas').length}
                  </span>
                  <span className="text-[11px] text-mid">
                    {orders.filter((o) => o.vertical === 'gas' && o.status === 'delivered').length} delivered
                  </span>
                </div>
              </div>
            )}

            {newCount > 0 && (
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-3">
                <span className="text-[11px] font-semibold text-mid tracking-[.6px]">NEEDS YOUR ACTION NOW</span>
                <div className="flex items-center gap-2.5 px-3.25 py-2.75 rounded-xl bg-tile">
                  <span className="w-[7px] h-[7px] rounded-full bg-emerald" />
                  <span className="flex-1 text-[12.5px] font-semibold">{newCount} order{newCount > 1 ? 's' : ''} to confirm</span>
                  <button
                    onClick={() => setTab('orders')}
                    className="font-display text-xs font-semibold text-emerald border-[1.5px] border-emerald rounded-[10px] px-3.5 py-1.75 hover:bg-emerald/[.07] transition-colors"
                  >
                    Open
                  </button>
                </div>
              </div>
            )}
          </>
        )}

        {tab === 'orders' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Orders</h1>
            {ordersQuery.isLoading && <p className="text-sm text-mid">Loading orders…</p>}
            {!ordersQuery.isLoading && orders.length === 0 && <p className="text-sm text-mid">No orders yet.</p>}
            {orders.map((o) => {
              const meta = ORDER_STAGE_META[o.status] ?? ORDER_STAGE_META['pending']!;
              const itemSummary = o.items.map((i) => `${i.name} ×${i.quantity}`).join(', ');
              return (
                <div key={o.id} className="bg-white border border-line rounded-2xl px-4.5 py-3.75 flex items-center gap-3.5">
                  <span className="w-2 h-2 flex-none rounded-full" style={{ background: meta.dot }} />
                  <span className="flex-1 flex flex-col min-w-0">
                    <span className="text-[13.5px] font-semibold truncate">{itemSummary || 'Order'}</span>
                    <span className="text-[11px] text-mid">#{o.id.slice(0, 8)} · {o.paymentMethod}</span>
                  </span>
                  <span className="text-[11px] font-bold rounded-full px-2.75 py-1" style={{ color: meta.chipC, background: meta.chipB }}>
                    {meta.status}
                  </span>
                  <b className="font-display text-[15px] text-emerald w-[90px] text-right">₦{naira(o.total.amount)}</b>
                  {meta.actLabel && meta.next ? (
                    <button
                      onClick={() => transitionMutation.mutate({ id: o.id, to: meta.next! })}
                      disabled={transitionMutation.isPending}
                      className="w-[160px] font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
                    >
                      {meta.actLabel}
                    </button>
                  ) : (
                    <span className="w-[160px] text-center text-[11.5px]" style={{ color: '#9A968D' }}>
                      {meta.doneLabel}
                    </span>
                  )}
                </div>
              );
            })}
          </>
        )}

        {tab === 'rx' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Prescription review</h1>
            {prescriptionsQuery.isLoading && <p className="text-sm text-mid">Loading…</p>}
            {!prescriptionsQuery.isLoading && (prescriptionsQuery.data?.prescriptions.length ?? 0) === 0 && (
              <p className="text-sm text-mid">No prescriptions awaiting review.</p>
            )}
            <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))' }}>
              {(prescriptionsQuery.data?.prescriptions ?? []).map((rx: MerchantPrescription) => (
                <div key={rx.id} className="bg-white border border-line rounded-2xl overflow-hidden flex flex-col">
                  <div className="h-[220px] bg-tile flex items-center justify-center overflow-hidden">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={rx.viewUrl} alt="Prescription" className="w-full h-full object-contain" />
                  </div>
                  <div className="p-4 flex flex-col gap-3">
                    <span className="text-[11px] text-mid">Customer #{rx.customerId.slice(0, 8)} · {new Date(rx.createdAt).toLocaleString()}</span>

                    {rejectingId === rx.id ? (
                      <div className="flex flex-col gap-2">
                        <input
                          placeholder="Reason for rejection (shown to customer)"
                          value={rejectNote}
                          onChange={(e) => setRejectNote(e.target.value)}
                          className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                        />
                        <div className="flex gap-2">
                          <button
                            onClick={() => reviewMutation.mutate({ id: rx.id, approve: false, note: rejectNote || undefined })}
                            disabled={reviewMutation.isPending}
                            className="flex-1 font-display text-xs font-semibold rounded-[10px] py-2.5 disabled:opacity-50"
                            style={{ color: '#B4231F', border: '1.5px solid #E5B5B3' }}
                          >
                            Confirm reject
                          </button>
                          <button
                            onClick={() => { setRejectingId(null); setRejectNote(''); }}
                            className="flex-1 font-display text-xs font-semibold text-mid border border-line rounded-[10px] py-2.5"
                          >
                            Cancel
                          </button>
                        </div>
                      </div>
                    ) : (
                      <div className="flex gap-2.5">
                        <button
                          onClick={() => reviewMutation.mutate({ id: rx.id, approve: true })}
                          disabled={reviewMutation.isPending}
                          className="flex-[2] font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] py-2.5 hover:bg-lime-600 transition-colors disabled:opacity-50"
                        >
                          ✓ Approve
                        </button>
                        <button
                          onClick={() => setRejectingId(rx.id)}
                          disabled={reviewMutation.isPending}
                          className="flex-1 font-display text-xs font-semibold rounded-[10px] py-2.5 border-[1.5px] transition-colors disabled:opacity-50"
                          style={{ color: '#B4231F', borderColor: '#E5B5B3' }}
                        >
                          Reject
                        </button>
                      </div>
                    )}

                    {reviewMutation.isError && (
                      <span className="text-[11px] text-red-600">{(reviewMutation.error as Error).message}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </>
        )}

        {tab === 'prod' && (
          <>
            <div className="flex items-center justify-between">
              <h1 className="font-display font-semibold text-[26px] tracking-tight">Products</h1>
              <button
                onClick={() => setShowAddProduct((v) => !v)}
                className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2.25 hover:bg-lime-600 transition-colors"
              >
                + Add product
              </button>
            </div>

            {showAddProduct && (
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-3">
                <div className="grid grid-cols-2 gap-3">
                  <input
                    placeholder="Product name"
                    value={newProduct.name}
                    onChange={(e) => setNewProduct((p) => ({ ...p, name: e.target.value }))}
                    className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                  />
                  <input
                    placeholder="Category"
                    value={newProduct.category}
                    onChange={(e) => setNewProduct((p) => ({ ...p, category: e.target.value }))}
                    className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                  />
                  <input
                    type="number"
                    placeholder="Price (₦)"
                    value={newProduct.priceKobo ? newProduct.priceKobo / 100 : ''}
                    onChange={(e) => setNewProduct((p) => ({ ...p, priceKobo: Math.round(Number(e.target.value) * 100) }))}
                    className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                  />
                </div>
                <div className="flex gap-2.5">
                  <button
                    onClick={() => createProductMutation.mutate(newProduct)}
                    disabled={!newProduct.name || !newProduct.priceKobo || createProductMutation.isPending}
                    className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2.25 hover:bg-lime-600 transition-colors disabled:opacity-50"
                  >
                    {createProductMutation.isPending ? 'Saving…' : 'Save product'}
                  </button>
                  <button
                    onClick={() => setShowAddProduct(false)}
                    className="font-display text-xs font-semibold text-mid px-4 py-2.25"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}

            {productsQuery.isLoading && <p className="text-sm text-mid">Loading products…</p>}
            {!productsQuery.isLoading && products.length === 0 && <p className="text-sm text-mid">No products yet — add your first one.</p>}
            {products.map((p) => (
              <div key={p.id} className="bg-white border border-line rounded-2xl px-4.5 py-3.25 flex items-center gap-3.5">
                <span className="flex-1 flex flex-col">
                  <span className="text-[13px] font-semibold">{p.name}</span>
                  <span className="text-[11px] text-mid">{p.category || 'Uncategorized'}</span>
                </span>
                <b className="font-display text-sm w-20 text-right">₦{naira(p.priceKobo)}</b>
                <button
                  onClick={() => toggleProductMutation.mutate({ id: p.id, available: !p.isAvailable })}
                  disabled={toggleProductMutation.isPending}
                  className="w-11 h-6 flex-none rounded-full relative transition-colors"
                  style={{ background: p.isAvailable ? '#0A3D2C' : '#D5D2C8' }}
                  aria-label={p.isAvailable ? 'Disable product' : 'Enable product'}
                >
                  <span
                    className="absolute top-[3px] w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-all"
                    style={{ left: p.isAvailable ? 23 : 3 }}
                  />
                </button>
              </div>
            ))}
          </>
        )}

        {tab === 'earn' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Earnings</h1>
            <div className="grid grid-cols-2 gap-3.5">
              <div className="bg-emerald rounded-2xl p-4.5 flex flex-col gap-2">
                <span className="text-[10.5px] font-semibold text-sand/60 tracking-[.5px]">WALLET BALANCE</span>
                <span className="font-display text-[28px] font-bold text-lime">
                  {walletQuery.isLoading ? '…' : `₦${naira(walletQuery.data?.balanceKobo ?? 0)}`}
                </span>
                <button
                  onClick={() => setShowWithdraw(true)}
                  className="mt-1 self-start font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-3.5 py-1.75 hover:bg-lime-600 transition-colors"
                >
                  Withdraw
                </button>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">PLATFORM COMMISSION</span>
                <span className="font-display text-[28px] font-bold text-ink">8%</span>
                <span className="text-[11px] text-mid">flat, no hidden charges</span>
              </div>
            </div>

            {/* Bank account */}
            <div className="bg-white border border-line rounded-2xl px-4.5 py-3.75 flex items-center gap-3.5">
              {bankAccountQuery.data ? (
                <>
                  <span className="flex-1 flex flex-col">
                    <span className="text-[13px] font-semibold">{bankAccountQuery.data.accountName}</span>
                    <span className="text-[11px] text-mid">{bankAccountQuery.data.bankName} · {bankAccountQuery.data.accountNumber}</span>
                  </span>
                  <button onClick={() => { setBankDraft(bankAccountQuery.data!); setShowBankForm(true); }} className="text-[11.5px] font-semibold text-emerald">Edit</button>
                </>
              ) : (
                <>
                  <span className="flex-1 text-[13px] text-mid">No bank account linked — add one to withdraw</span>
                  <button onClick={() => setShowBankForm(true)} className="font-display text-xs font-semibold text-emerald border-[1.5px] border-emerald rounded-[10px] px-3.5 py-1.75 hover:bg-emerald/[.07] transition-colors">Add account</button>
                </>
              )}
            </div>

            {showBankForm && (
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-3">
                <span className="text-[11px] font-semibold text-mid tracking-[.5px]">BANK ACCOUNT DETAILS</span>
                <div className="grid grid-cols-2 gap-3">
                  {(['bankCode', 'bankName', 'accountNumber', 'accountName'] as const).map((f) => (
                    <input
                      key={f}
                      placeholder={{ bankCode: 'Bank code', bankName: 'Bank name', accountNumber: 'Account number', accountName: 'Account name' }[f]}
                      value={bankDraft[f]}
                      onChange={(e) => setBankDraft((d) => ({ ...d, [f]: e.target.value }))}
                      className="border border-line rounded-xl px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                    />
                  ))}
                </div>
                <div className="flex gap-2.5">
                  <button
                    onClick={() => saveBankMutation.mutate()}
                    disabled={!bankDraft.bankCode || !bankDraft.accountNumber || saveBankMutation.isPending}
                    className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2.25 hover:bg-lime-600 transition-colors disabled:opacity-50"
                  >
                    {saveBankMutation.isPending ? 'Saving…' : 'Save'}
                  </button>
                  <button onClick={() => setShowBankForm(false)} className="font-display text-xs font-semibold text-mid px-4 py-2.25">Cancel</button>
                </div>
              </div>
            )}

            {/* Withdraw modal */}
            {showWithdraw && (
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
                <div className="bg-white rounded-2xl p-6 w-[360px] flex flex-col gap-4 shadow-xl">
                  <span className="font-display font-semibold text-[18px]">Withdraw funds</span>
                  {!bankAccountQuery.data ? (
                    <p className="text-[13px] text-mid">Add a bank account first before withdrawing.</p>
                  ) : (
                    <>
                      <div className="flex flex-col gap-1">
                        <span className="text-[11px] font-semibold text-mid tracking-[.5px]">TO</span>
                        <span className="text-[13px] font-semibold">{bankAccountQuery.data.accountName}</span>
                        <span className="text-[11px] text-mid">{bankAccountQuery.data.bankName} · {bankAccountQuery.data.accountNumber}</span>
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        {(['standard', 'instant'] as const).map((type) => (
                          <button
                            key={type}
                            onClick={() => setWithdrawType(type)}
                            className={`rounded-xl border-[1.5px] px-3 py-2.5 text-left transition-colors ${
                              withdrawType === type ? 'border-emerald bg-emerald/[.06]' : 'border-line'
                            }`}
                          >
                            <span className="block text-[12.5px] font-semibold capitalize">{type}</span>
                            <span className="block text-[11px] text-mid">
                              {type === 'standard' ? 'Free · next business day' : '1% fee · within minutes'}
                            </span>
                          </button>
                        ))}
                      </div>
                      <input
                        type="number"
                        placeholder="Amount (₦)"
                        value={withdrawAmount}
                        onChange={(e) => setWithdrawAmount(e.target.value)}
                        className="border border-line rounded-xl px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                      />
                      {withdrawType === 'instant' && Number(withdrawAmount) > 0 && (() => {
                        const amtKobo = Math.round(Number(withdrawAmount) * 100);
                        const fee = Math.min(Math.max(Math.round(amtKobo * 0.01), 1000), 50000);
                        return (
                          <span className="text-[11px] text-mid">
                            Fee: ₦{(fee / 100).toLocaleString('en-NG')} · You receive: ₦{((amtKobo - fee) / 100).toLocaleString('en-NG')}
                          </span>
                        );
                      })()}
                      <input
                        type="password"
                        placeholder="Wallet PIN"
                        maxLength={6}
                        value={withdrawPin}
                        onChange={(e) => setWithdrawPin(e.target.value)}
                        className="border border-line rounded-xl px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-emerald"
                      />
                      {withdrawMutation.isError && (
                        <span className="text-[12px] text-red-600">{(withdrawMutation.error as Error).message}</span>
                      )}
                    </>
                  )}
                  <div className="flex gap-2.5">
                    {bankAccountQuery.data && (
                      <button
                        onClick={() => withdrawMutation.mutate()}
                        disabled={!withdrawAmount || !withdrawPin || withdrawMutation.isPending}
                        className="flex-1 font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] py-2.5 hover:bg-lime-600 transition-colors disabled:opacity-50"
                      >
                        {withdrawMutation.isPending ? 'Processing…' : 'Confirm'}
                      </button>
                    )}
                    <button onClick={() => setShowWithdraw(false)} className="flex-1 font-display text-sm font-semibold text-mid border border-line rounded-[10px] py-2.5">Cancel</button>
                  </div>
                </div>
              </div>
            )}

            <div className="bg-white border border-line rounded-2xl overflow-hidden">
              <div className="flex justify-between px-4.5 py-3.25 border-b border-[#EFECE3] text-[11px] font-semibold text-mid tracking-[.5px]">
                <span>DESCRIPTION</span>
                <span>AMOUNT</span>
              </div>
              {(transactionsQuery.data?.transactions as { id: string; description: string; amountKobo: number; createdAt: string }[] | undefined ?? []).map((tx, i, arr) => (
                <div key={tx.id} className={`flex justify-between px-4.5 py-3.25 text-[13px] ${i < arr.length - 1 ? 'border-b border-[#EFECE3]' : ''}`}>
                  <span>{tx.description}</span>
                  <b className={tx.amountKobo >= 0 ? 'text-emerald' : 'text-red-600'}>
                    {tx.amountKobo >= 0 ? '+' : ''}₦{naira(tx.amountKobo)}
                  </b>
                </div>
              ))}
              {transactionsQuery.data && (transactionsQuery.data.transactions as unknown[]).length === 0 && (
                <p className="px-4.5 py-4 text-sm text-mid">No transactions yet.</p>
              )}
            </div>
          </>
        )}

        {tab === 'set' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Verification &amp; onboarding</h1>
            <span className="text-[13px] text-mid" style={{ maxWidth: '56ch' }}>
              Your customers see this status — trust is the product.
            </span>
            <div className="bg-white border border-line rounded-2xl px-4.5 py-3.75 flex items-center gap-3.25" style={{ maxWidth: 760 }}>
              <ShieldCheckIcon size={22} />
              <span className="flex-1 flex flex-col">
                <span className="text-[13.5px] font-semibold">{profileQuery.data?.businessName}</span>
                <span className="text-[11px] text-mid capitalize">Vertical: {profileQuery.data?.vertical}</span>
              </span>
              <span
                className="text-[11px] font-bold rounded-full px-2.75 py-1 capitalize"
                style={
                  profileQuery.data?.kycStatus === 'approved'
                    ? { color: '#0A3D2C', background: '#E9F3D8' }
                    : { color: '#8A6A1B', background: '#FFF3D6' }
                }
              >
                {profileQuery.data?.kycStatus === 'approved' ? '✓ Verified' : profileQuery.data?.kycStatus.replace('_', ' ')}
              </span>
            </div>
          </>
        )}

        {/* ── Gas operations tab ─────────────────────────────────────────── */}
        {tab === 'gas' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Gas operations</h1>

            {/* Fill accuracy score */}
            <section className="flex flex-col gap-2">
              <span className="text-[11px] font-semibold text-mid tracking-[.6px]">FILL ACCURACY SCORE</span>
              <div className="bg-white border border-line rounded-2xl p-5 flex items-start gap-6">
                <div className="flex flex-col gap-0.5">
                  {profileQuery.data?.fillAccuracyPct != null ? (
                    <>
                      <span
                        className="font-display text-[42px] font-bold leading-none"
                        style={{ color: profileQuery.data.fillAccuracyPct >= 0.98 ? '#0A3D2C' : profileQuery.data.fillAccuracyPct >= 0.95 ? '#8A6A1B' : '#B4231F' }}
                      >
                        {(profileQuery.data.fillAccuracyPct * 100).toFixed(1)}%
                      </span>
                      <span className="text-[12px] text-mid mt-1">{profileQuery.data.fillSampleCount ?? 0} verified fills</span>
                    </>
                  ) : (
                    <>
                      <span className="font-display text-[42px] font-bold leading-none text-mid">—</span>
                      <span className="text-[12px] text-mid mt-1">No verified fills yet</span>
                    </>
                  )}
                </div>
                <div className="flex-1 flex flex-col gap-1.5 text-[12px] text-mid" style={{ maxWidth: 340 }}>
                  <span className="font-semibold text-ink text-[13px]">How this works</span>
                  <span>Every gas delivery is weighed at the customer's door. The scale photo is recorded and ordered vs measured weight is compared automatically.</span>
                  <span>Short by more than 2%? The difference is refunded from your settlement before the rider leaves.</span>
                  <span
                    className="font-semibold"
                    style={{ color: profileQuery.data?.fillAccuracyPct != null && profileQuery.data.fillAccuracyPct < 0.95 ? '#B4231F' : '#0A3D2C' }}
                  >
                    {profileQuery.data?.fillAccuracyPct != null && profileQuery.data.fillAccuracyPct < 0.95
                      ? '⚠️ Below 95% — risk of delisting. Improve fill accuracy to stay on the platform.'
                      : '✓ Good standing'}
                  </span>
                </div>
              </div>
            </section>

            {/* Cylinder float stock */}
            <section className="flex flex-col gap-2">
              <span className="text-[11px] font-semibold text-mid tracking-[.6px]">CYLINDER FLOAT STOCK</span>
              <div className="bg-white border border-line rounded-2xl overflow-hidden">
                {(() => {
                  const cylinderProducts = products.filter((p) => p.name.toLowerCase().includes('kg'));
                  if (cylinderProducts.length === 0) {
                    return <p className="p-5 text-[13px] text-mid">No cylinder products found. Add them in the Products tab.</p>;
                  }
                  return (
                    <div className="grid divide-x divide-line" style={{ gridTemplateColumns: `repeat(${cylinderProducts.length}, 1fr)` }}>
                      {cylinderProducts.map((p) => (
                        <div key={p.id} className="p-4 flex flex-col gap-2">
                          <span className="text-[11px] font-semibold text-mid tracking-[.5px]">
                            {p.name.replace(' LPG cylinder', '').replace(' cylinder', '').toUpperCase()}
                          </span>
                          <div className="flex items-center gap-2">
                            <span className="w-2.5 h-2.5 rounded-full" style={{ background: p.isAvailable ? '#C6F24E' : '#D5D2C8' }} />
                            <span className="text-[13px] font-semibold">{p.isAvailable ? 'In stock' : 'Out of stock'}</span>
                          </div>
                          <span className="text-[11px] text-mid">₦{naira(p.priceKobo)}</span>
                          <button
                            onClick={() => toggleProductMutation.mutate({ id: p.id, available: !p.isAvailable })}
                            disabled={toggleProductMutation.isPending}
                            className="text-[11px] font-semibold rounded-lg px-2.5 py-1.5 transition-colors disabled:opacity-50"
                            style={p.isAvailable
                              ? { color: '#B4231F', border: '1.5px solid #E5B5B3' }
                              : { color: '#0A3D2C', border: '1.5px solid #C6F24E', background: '#E9F3D8' }}
                          >
                            {p.isAvailable ? 'Mark out of stock' : 'Mark in stock'}
                          </button>
                        </div>
                      ))}
                    </div>
                  );
                })()}
              </div>
            </section>

            {/* Gas order queue */}
            <section className="flex flex-col gap-2">
              <span className="text-[11px] font-semibold text-mid tracking-[.6px]">GAS ORDER QUEUE</span>
              {ordersQuery.isLoading && <p className="text-sm text-mid">Loading…</p>}
              {(() => {
                const gasOrders = orders.filter((o) => o.vertical === 'gas');
                if (!ordersQuery.isLoading && gasOrders.length === 0) {
                  return <p className="text-[13px] text-mid">No gas orders yet.</p>;
                }
                const GAS_LABEL: Record<string, { label: string; chipC: string; chipB: string }> = {
                  pending:          { label: 'NEW',             chipC: '#0A3D2C', chipB: '#C6F24E' },
                  confirmed:        { label: 'CONFIRMED',       chipC: '#0A3D2C', chipB: '#C6F24E' },
                  preparing:        { label: 'PREPARING',       chipC: '#8A6A1B', chipB: '#FFF3D6' },
                  ready_for_pickup: { label: 'AWAITING RIDER',  chipC: '#0A3D2C', chipB: '#E9F3D8' },
                  driver_assigned:  { label: 'RIDER ASSIGNED',  chipC: '#0A3D2C', chipB: '#E9F3D8' },
                  in_transit:       { label: 'OUT FOR DELIVERY', chipC: '#0A3D2C', chipB: '#E9F3D8' },
                  delivered:        { label: 'DELIVERED',        chipC: '#63636E', chipB: '#EFECE3' },
                  cancelled:        { label: 'CANCELLED',        chipC: '#B4231F', chipB: '#FEF2F2' },
                };
                return (
                  <div className="flex flex-col gap-2">
                    {gasOrders.map((o) => {
                      const meta = GAS_LABEL[o.status] ?? GAS_LABEL['pending']!;
                      const cylinderName = o.items[0]?.name ?? 'Gas cylinder';
                      return (
                        <div key={o.id} className="bg-white border border-line rounded-2xl px-4.5 py-3.5 flex items-center gap-3.5">
                          <FlameIcon />
                          <span className="flex-1 flex flex-col min-w-0">
                            <span className="text-[13.5px] font-semibold truncate">{cylinderName}</span>
                            <span className="text-[11px] text-mid">#{o.id.slice(0, 8)} · {o.paymentMethod}</span>
                          </span>
                          <span className="text-[11px] font-bold rounded-full px-2.75 py-1" style={{ color: meta.chipC, background: meta.chipB }}>
                            {meta.label}
                          </span>
                          <b className="font-display text-[15px] text-emerald w-[90px] text-right">₦{naira(o.total.amount)}</b>
                          {o.status === 'pending' && (
                            <button
                              onClick={() => transitionMutation.mutate({ id: o.id, to: 'confirmed' })}
                              disabled={transitionMutation.isPending}
                              className="w-[140px] font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
                            >
                              Confirm order
                            </button>
                          )}
                          {o.status === 'confirmed' && (
                            <button
                              onClick={() => transitionMutation.mutate({ id: o.id, to: 'ready_for_pickup' })}
                              disabled={transitionMutation.isPending}
                              className="w-[140px] font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
                            >
                              Ready for rider
                            </button>
                          )}
                          {!['pending', 'confirmed'].includes(o.status) && (
                            <span className="w-[140px] text-center text-[11.5px] text-mid">{meta.label.toLowerCase()}</span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                );
              })()}
            </section>
          </>
        )}
        {tab === 'pay' && (
          <div className="flex-1 px-4 py-4 flex flex-col gap-4 overflow-y-auto">
            <p className="text-[11px] font-semibold text-mid tracking-[.5px] uppercase">Generate paycode</p>
            <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-3">
              <p className="text-[13px] text-ink">Enter an order ID to generate a 6-digit delivery code for the rider.</p>
              <input
                value={paycodeOrderId}
                onChange={(e) => { setPaycodeOrderId(e.target.value); setGeneratedPaycode(null); }}
                placeholder="Order UUID"
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] font-mono text-ink placeholder-mid focus:outline-none focus:border-emerald"
              />
              {generatePaycode.isError && <p className="text-xs text-red-500">{(generatePaycode.error as Error).message}</p>}
              {generatedPaycode && (
                <div className="bg-[#E9F3D8] rounded-xl px-4 py-3 flex flex-col items-center gap-1">
                  <p className="text-[10px] font-semibold text-mid uppercase tracking-[.5px]">Paycode payload</p>
                  <p className="font-mono text-[13px] font-bold text-emerald break-all text-center">{generatedPaycode.payload}</p>
                </div>
              )}
              <button
                onClick={() => { if (paycodeOrderId.trim()) generatePaycode.mutate(paycodeOrderId.trim()); }}
                disabled={!paycodeOrderId.trim() || generatePaycode.isPending}
                className="w-full bg-emerald text-lime font-display font-semibold text-[13px] rounded-xl py-2.5 hover:bg-emerald/90 transition-colors disabled:opacity-50"
              >
                {generatePaycode.isPending ? 'Generating…' : 'Generate'}
              </button>
            </div>

            <p className="text-[11px] font-semibold text-mid tracking-[.5px] uppercase">Scan SpeedPlus card</p>
            <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-3">
              <p className="text-[13px] text-ink">Scan a customer's SpeedPlus card QR payload to confirm payment.</p>
              <input
                value={scanCardPayload}
                onChange={(e) => { setScanCardPayload(e.target.value); setScanResult(null); }}
                placeholder="Card QR payload"
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[13px] font-mono text-ink placeholder-mid focus:outline-none focus:border-emerald"
              />
              <input
                type="password"
                inputMode="numeric"
                maxLength={6}
                value={scanCardPin}
                onChange={(e) => setScanCardPin(e.target.value.replace(/\D/g, ''))}
                placeholder="Customer PIN"
                className="w-full border border-line rounded-xl px-3 py-2.5 text-[20px] text-ink placeholder-mid tracking-widest focus:outline-none focus:border-emerald"
              />
              {scanResult && (
                <p className={`text-[13px] font-semibold ${scanResult === 'Payment confirmed' ? 'text-emerald' : 'text-red-500'}`} role="status">
                  {scanResult === 'Payment confirmed' ? '✓ ' : '✗ '}{scanResult}
                </p>
              )}
              <button
                onClick={() => scanCard.mutate()}
                disabled={!scanCardPayload.trim() || scanCardPin.length < 4 || scanCard.isPending}
                className="w-full bg-emerald text-lime font-display font-semibold text-[13px] rounded-xl py-2.5 hover:bg-emerald/90 transition-colors disabled:opacity-50"
              >
                {scanCard.isPending ? 'Confirming…' : 'Confirm payment'}
              </button>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
