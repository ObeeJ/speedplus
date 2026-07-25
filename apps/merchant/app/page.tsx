'use client';

import { useState } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { merchantApi, type MerchantOrder, type MerchantProduct, type ProductInput } from '@speedplus/api-client';
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

const NAV_ITEMS: { id: MerchantTab; label: string; icon: string }[] = [
  { id: 'dash', label: 'Dashboard', icon: '▦' },
  { id: 'orders', label: 'Orders', icon: '🛒' },
  { id: 'rx', label: 'Prescriptions', icon: '💊' },
  { id: 'prod', label: 'Products', icon: '📋' },
  { id: 'earn', label: 'Earnings', icon: '₦' },
  { id: 'set', label: 'Verification', icon: '⚙' },
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
  const createProductMutation = useMutation({
    mutationFn: (input: ProductInput) => merchantApi.createProduct(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-products'] });
      setShowAddProduct(false);
      setNewProduct(EMPTY_PRODUCT);
    },
  });

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
            const count = item.id === 'orders' ? newCount : null;
            return (
              <button
                key={item.id}
                onClick={() => setTab(item.id)}
                className={`flex items-center gap-2.5 rounded-xl px-4 py-2.5 text-[13.5px] font-medium transition-colors ${
                  active ? 'bg-lime/[.14] text-lime font-semibold' : 'text-sand/70 hover:bg-sand/[.08] hover:text-sand'
                }`}
              >
                {item.icon} {item.label}
                {count !== null && count > 0 && (
                  <span className="ml-auto text-[10.5px] font-bold rounded-full px-2 py-0.5" style={{ color: '#0A3D2C', background: '#C6F24E' }}>
                    {count}
                  </span>
                )}
              </button>
            );
          })}
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
          <button onClick={clearAuth} className="text-[10px] text-sand/40 hover:text-sand/70" aria-label="Sign out">
            ⏻
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
            <div className="bg-white border border-line rounded-2xl p-6 flex flex-col gap-2 items-start">
              <span className="text-[13px] font-semibold">Coming soon</span>
              <span className="text-[12.5px] text-mid max-w-[48ch]">
                Prescription approval isn&apos;t wired to the live backend yet — orders requiring a prescription
                are still gated server-side (an order can&apos;t be placed without an approved prescription), but
                the merchant-side review screen is not yet built.
              </span>
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
              <div className="bg-emerald rounded-2xl p-4.5 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-sand/60 tracking-[.5px]">WALLET BALANCE</span>
                <span className="font-display text-[28px] font-bold text-lime">
                  {walletQuery.isLoading ? '…' : `₦${naira(walletQuery.data?.balanceKobo ?? 0)}`}
                </span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">PLATFORM COMMISSION</span>
                <span className="font-display text-[28px] font-bold text-ink">8%</span>
                <span className="text-[11px] text-mid">flat, no hidden charges</span>
              </div>
            </div>
            <div className="bg-white border border-line rounded-2xl overflow-hidden">
              <div className="flex justify-between px-4.5 py-3.25 border-b border-[#EFECE3] text-[11px] font-semibold text-mid tracking-[.5px]">
                <span>DESCRIPTION</span>
                <span>AMOUNT</span>
              </div>
              {(transactionsQuery.data?.transactions as { id: string; description: string; amountKobo: number; createdAt: string }[] | undefined ?? []).map((tx, i, arr) => (
                <div key={tx.id} className={`flex justify-between px-4.5 py-3.25 text-[13px] ${i < arr.length - 1 ? 'border-b border-[#EFECE3]' : ''}`}>
                  <span>{tx.description}</span>
                  <b className={tx.amountKobo >= 0 ? 'text-emerald' : 'text-[#B4231F]'}>
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
              <span className="text-[17px]">🏥</span>
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
      </div>
    </main>
  );
}
