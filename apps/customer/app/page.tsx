'use client';

import { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { SpeedPlusLogo } from '@speedplus/ui';
import { usePackageFlowStore } from '../lib/store/package-flow.store';
import { ordersApi } from '@speedplus/api-client';

// ── Search intent routing ─────────────────────────────────────────────────────
// Maps natural language phrases to app routes. No AI — pure keyword matching.
const INTENTS: { patterns: RegExp[]; route: string; label: string }[] = [
  { patterns: [/send|package|parcel|deliver|courier/i],          route: '/package/where', label: 'Send a package' },
  { patterns: [/food|eat|meal|restaurant|hungry|lunch|dinner/i], route: '/food',          label: 'Order food' },
  { patterns: [/gas|cylinder|cooking gas|lpg/i],                 route: '/gas',           label: 'Get cooking gas' },
  { patterns: [/grocery|groceries|market|vegetables|fruit/i],    route: '/grocery',       label: 'Order groceries' },
  { patterns: [/pharmacy|medicine|drug|prescription|meds/i],     route: '/pharmacy',      label: 'Order from pharmacy' },
  { patterns: [/wallet|balance|fund|top.?up|money/i],            route: '/wallet',        label: 'Go to wallet' },
  { patterns: [/order|history|past|receipt|invoice/i],           route: '/orders',        label: 'View order history' },
  { patterns: [/refer|referral|invite|friend/i],                 route: '/referral',      label: 'Refer a friend' },
  { patterns: [/track|where.*order|status/i],                    route: '/orders',        label: 'Track an order' },
];

function resolveIntent(query: string): { route: string; label: string } | null {
  const q = query.trim();
  if (!q) return null;
  for (const intent of INTENTS) {
    if (intent.patterns.some((p) => p.test(q))) return intent;
  }
  return null;
}

const SUGGESTIONS = [
  'Send a package across Lagos',
  'Order food near me',
  'Get cooking gas delivered',
  'Check my wallet balance',
  'View my order history',
];

const ACTIVE_STATUSES = new Set(['pending', 'confirmed', 'preparing', 'ready_for_pickup', 'driver_assigned', 'in_transit']);

const verticals = [
  { label: 'Cooking Gas', description: 'Cylinder refills & swaps.', href: '/gas', icon: <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" /><circle cx="12" cy="9" r="2.5" /></svg> },
  { label: 'Grocery', description: 'Fresh produce, essentials.', href: '/grocery', icon: <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z" /><line x1="3" y1="6" x2="21" y2="6" /><path d="M16 10a4 4 0 01-8 0" /></svg> },
  { label: 'Food', description: 'Hot meals from local restaurants.', href: '/food', icon: <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M18 8h1a4 4 0 010 8h-1" /><path d="M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" /><line x1="6" y1="1" x2="6" y2="4" /><line x1="10" y1="1" x2="10" y2="4" /><line x1="14" y1="1" x2="14" y2="4" /></svg> },
  { label: 'Pharmacy', description: 'OTC meds and prescriptions.', href: '/pharmacy', icon: <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /><line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" /></svg> },
  { label: 'Package', description: 'Send anything across the city.', href: '/package/where', featured: true, icon: <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" /><polyline points="3.27 6.96 12 12.01 20.73 6.96" /><line x1="12" y1="22.08" x2="12" y2="12" /></svg> },
];

const quickLinks = [
  { href: '/wallet', label: 'Wallet', icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" /></svg> },
  { href: '/orders', label: 'Orders', icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M9 11l3 3L22 4" /><path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" /></svg> },
  { href: '/loyalty', label: 'Points', icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" /></svg> },
  { href: '/profile', label: 'Profile', icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="8" r="4" /><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7" /></svg> },
];

export default function HomePage() {
  const router = useRouter();
  const { orderId: activePackageOrderId } = usePackageFlowStore();

  const [query, setQuery] = useState('');
  const [searchFocused, setSearchFocused] = useState(false);
  const [activeOrder, setActiveOrder] = useState<{ id: string; vertical: string; status: string } | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Detect active order on mount — resume flow
  useEffect(() => {
    ordersApi.list({}).then((data) => {
      const active = data.orders.find((o) => ACTIVE_STATUSES.has(o.status));
      if (active) setActiveOrder({ id: active.id, vertical: active.vertical, status: active.status });
    }).catch(() => {});
  }, []);

  const intent = resolveIntent(query);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (intent) {
      router.push(intent.route);
    }
  }

  function handleSuggestion(s: string) {
    setQuery(s);
    const resolved = resolveIntent(s);
    if (resolved) router.push(resolved.route);
  }

  const showDropdown = searchFocused && (intent || query.length === 0);

  return (
    <main className="min-h-screen bg-[#0A1F15] text-white flex flex-col">
      {/* Top bar */}
      <div className="px-5 pt-12 pb-4 flex items-center justify-between">
        <SpeedPlusLogo variant="full" theme="dark" size="md" />
        <Link href="/wallet" className="flex items-center gap-1.5 bg-white/10 hover:bg-white/15 transition-colors rounded-full px-3.5 py-1.5 text-[12px] font-semibold text-white/80">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" /></svg>
          Wallet
        </Link>
      </div>

      {/* Hero */}
      <div className="px-5 pb-5">
        <h1 className="font-display font-bold text-[34px] leading-[1.1] tracking-tight">
          What do you<br />need today?
        </h1>
        <p className="text-white/50 mt-1.5 text-[14px]">Delivered fast across Lagos.</p>
      </div>

      {/* Universal search */}
      <div className="px-5 pb-4 relative">
        <form onSubmit={handleSearch}>
          <div className={`flex items-center gap-3 bg-white/10 border rounded-2xl px-4 py-3 transition-all duration-200 ${searchFocused ? 'border-[#C6F24E]/60 bg-white/[.13]' : 'border-white/10'}`}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.5)" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onFocus={() => setSearchFocused(true)}
              onBlur={() => setTimeout(() => setSearchFocused(false), 150)}
              placeholder="What do you need? e.g. send a package"
              className="flex-1 bg-transparent text-white placeholder-white/40 text-[14px] focus:outline-none"
            />
            {query && (
              <button type="button" onClick={() => setQuery('')} className="text-white/40 hover:text-white/70 transition-colors">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
              </button>
            )}
          </div>
        </form>

        {/* Dropdown */}
        {showDropdown && (
          <div className="absolute left-5 right-5 top-full mt-1 bg-[#0D2B1C] border border-white/10 rounded-2xl overflow-hidden z-20 shadow-xl">
            {intent ? (
              <button
                onClick={() => router.push(intent.route)}
                className="w-full flex items-center gap-3 px-4 py-3.5 hover:bg-white/[.07] transition-colors text-left"
              >
                <span className="w-8 h-8 rounded-xl bg-[#C6F24E]/10 flex items-center justify-center flex-shrink-0">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14M12 5l7 7-7 7" /></svg>
                </span>
                <div>
                  <p className="text-[13px] font-semibold text-white">{intent.label}</p>
                  <p className="text-[11px] text-white/40">Tap to go there</p>
                </div>
              </button>
            ) : (
              <>
                <p className="px-4 pt-3 pb-1.5 text-[10.5px] font-semibold text-white/40 tracking-[0.6px]">SUGGESTIONS</p>
                {SUGGESTIONS.map((s) => (
                  <button
                    key={s}
                    onClick={() => handleSuggestion(s)}
                    className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-white/[.07] transition-colors text-left"
                  >
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.3)" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
                    <span className="text-[13px] text-white/70">{s}</span>
                  </button>
                ))}
              </>
            )}
          </div>
        )}
      </div>

      {/* Active order resume banner */}
      {activeOrder && (
        <div className="px-5 pb-4">
          <button
            onClick={() => {
              if (activeOrder.vertical === 'package') router.push('/package/tracking');
              else router.push('/orders');
            }}
            className="w-full flex items-center gap-3 bg-[#C6F24E]/10 border border-[#C6F24E]/30 rounded-2xl px-4 py-3 hover:bg-[#C6F24E]/15 transition-colors"
            style={{ animation: 'fadeUp 0.3s cubic-bezier(0.16,1,0.3,1) both' }}
          >
            <span className="w-2 h-2 rounded-full bg-[#C6F24E] animate-pulse flex-shrink-0" />
            <div className="flex-1 text-left">
              <p className="text-[13px] font-semibold text-[#C6F24E] capitalize">{activeOrder.vertical} delivery in progress</p>
              <p className="text-[11px] text-white/50 capitalize">{activeOrder.status.replace(/_/g, ' ')} — tap to track</p>
            </div>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
          </button>
        </div>
      )}

      {/* Verticals grid */}
      <div className="flex-1 px-5 pb-6">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          {verticals.map((v, i) => (
            <Link
              key={v.href}
              href={v.href}
              className={`group relative rounded-2xl p-5 flex flex-col gap-3 transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#C6F24E] active:scale-[0.98] ${
                v.featured
                  ? 'bg-[#0A3D2C] border border-[#C6F24E]/20 hover:border-[#C6F24E]/50 col-span-2 sm:col-span-1'
                  : 'bg-white/[0.06] border border-white/[0.08] hover:bg-white/[0.10] hover:border-white/20'
              }`}
              style={{ animation: `fadeUp 0.4s cubic-bezier(0.16,1,0.3,1) ${i * 55}ms both` }}
            >
              <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${v.featured ? 'bg-[#C6F24E]/10' : 'bg-white/[0.08]'}`}>
                {v.icon}
              </div>
              <div>
                <p className="font-display font-semibold text-[15px] text-white group-hover:text-[#C6F24E] transition-colors">{v.label}</p>
                <p className="text-white/50 text-[12px] mt-0.5 leading-snug">{v.description}</p>
              </div>
              {v.featured && (
                <span className="absolute top-3 right-3 text-[9px] font-bold text-[#0A3D2C] bg-[#C6F24E] rounded-full px-2 py-0.5 tracking-wide">SEND NOW</span>
              )}
            </Link>
          ))}
        </div>
      </div>

      {/* Bottom quick-access bar */}
      <div className="sticky bottom-0 bg-[#0A1F15]/95 backdrop-blur-sm border-t border-white/[0.08] px-5 py-3 pb-safe-bottom">
        <div className="flex justify-around max-w-sm mx-auto">
          {quickLinks.map((ql) => (
            <Link key={ql.href} href={ql.href} className="flex flex-col items-center gap-1 text-white/50 hover:text-white transition-colors py-1 px-4">
              {ql.icon}
              <span className="text-[10px] font-semibold">{ql.label}</span>
            </Link>
          ))}
        </div>
      </div>

      <style>{`
        @keyframes fadeUp { from { opacity: 0; transform: translateY(14px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </main>
  );
}
