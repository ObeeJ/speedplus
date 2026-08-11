'use client';

import { useState, useEffect, useRef } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { motion, AnimatePresence, useReducedMotion } from 'framer-motion';
import { SpeedPlusLogo, Badge, iconColors } from '@speedplus/ui';
import { staggerContainer, staggerItem, spring } from '@speedplus/ui';
import { usePackageFlowStore } from '../lib/store/package-flow.store';
import { ordersApi } from '@speedplus/api-client';
import { FEATURES } from '../lib/features';

// ── Intent routing ────────────────────────────────────────────────────────────

const INTENTS = [
  { patterns: [/send|package|parcel|deliver|courier/i],      route: '/package/where', label: 'Send a package' },
  ...(FEATURES.food ? [{ patterns: [/food|eat|meal|restaurant|hungry/i], route: '/food', label: 'Order food' }] : []),
  { patterns: [/gas|cylinder|cooking gas|lpg/i],             route: '/gas',           label: 'Get cooking gas' },
  { patterns: [/pharmacy|medicine|drug|prescription/i],      route: '/pharmacy',      label: 'Order from pharmacy' },
  { patterns: [/wallet|balance|fund|top.?up|money/i],        route: '/wallet',        label: 'Go to wallet' },
  { patterns: [/order|history|past|receipt/i],               route: '/orders',        label: 'View orders' },
  { patterns: [/track|where.*order|status/i],                route: '/orders',        label: 'Track an order' },
];

function resolveIntent(q: string) {
  if (!q.trim()) return null;
  for (const intent of INTENTS) {
    if (intent.patterns.some((p) => p.test(q))) return intent;
  }
  return null;
}

const SUGGESTIONS = [
  'Send a package across Lagos',
  'Get cooking gas delivered',
  'Order from pharmacy',
  'Check my wallet balance',
];

const ACTIVE_STATUSES = new Set(['pending', 'confirmed', 'preparing', 'ready_for_pickup', 'driver_assigned', 'in_transit']);

// ── Verticals ─────────────────────────────────────────────────────────────────

const verticals = [
  {
    label: 'Cooking Gas',
    description: 'Cylinder refills & swaps',
    href: '/gas',
    accent: iconColors.lime,
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" />
        <circle cx="12" cy="9" r="2.5" />
      </svg>
    ),
  },
  ...(FEATURES.food ? [{
    label: 'Food',
    description: 'Hot meals delivered fast',
    href: '/food',
    accent: iconColors.lime,
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M18 8h1a4 4 0 010 8h-1" /><path d="M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" />
        <line x1="6" y1="1" x2="6" y2="4" /><line x1="10" y1="1" x2="10" y2="4" /><line x1="14" y1="1" x2="14" y2="4" />
      </svg>
    ),
  }] : []),
  {
    label: 'Pharmacy',
    description: 'Meds & prescriptions',
    href: '/pharmacy',
    accent: iconColors.lime,
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <line x1="12" y1="8" x2="12" y2="16" /><line x1="8" y1="12" x2="16" y2="12" />
      </svg>
    ),
  },
  {
    label: 'Send Package',
    description: 'Anything, anywhere in the city',
    href: '/package/where',
    accent: iconColors.lime,
    featured: true,
    icon: (
      <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
        <polyline points="3.27 6.96 12 12.01 20.73 6.96" /><line x1="12" y1="22.08" x2="12" y2="12" />
      </svg>
    ),
  },
];

const NAV_LINKS = [
  {
    href: '/wallet', label: 'Wallet',
    icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" /></svg>,
  },
  {
    href: '/orders', label: 'Orders',
    icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><path d="M9 11l3 3L22 4" /><path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" /></svg>,
  },
  ...(FEATURES.loyalty ? [{
    href: '/loyalty', label: 'Points',
    icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2" /></svg>,
  }] : []),
  {
    href: '/profile', label: 'Profile',
    icon: <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="8" r="4" /><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7" /></svg>,
  },
];

// ── Component ─────────────────────────────────────────────────────────────────

export default function HomePage() {
  const router = useRouter();
  const reduced = useReducedMotion();
  const { orderId: activePackageOrderId } = usePackageFlowStore();

  const [query, setQuery] = useState('');
  const [searchFocused, setSearchFocused] = useState(false);
  const [activeOrder, setActiveOrder] = useState<{ id: string; vertical: string; status: string } | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    ordersApi.list({}).then((data) => {
      const active = data.orders.find((o) => ACTIVE_STATUSES.has(o.status));
      if (active) setActiveOrder({ id: active.id, vertical: active.vertical, status: active.status });
    }).catch(() => {});
  }, []);

  const intent = resolveIntent(query);
  const showDropdown = searchFocused && (intent || query.length === 0);

  function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (intent) router.push(intent.route);
  }

  function handleSuggestion(s: string) {
    const resolved = resolveIntent(s);
    if (resolved) router.push(resolved.route);
  }

  return (
    <main className="min-h-screen flex flex-col" style={{ background: '#080F0A' }}>

      {/* ── Hero ─────────────────────────────────────────────────────────── */}
      <div className="relative flex-shrink-0">
        <div className="absolute inset-0" style={{ background: '#080F0A' }} />
        <video
          src="/hero.mp4"
          poster="/hero-poster.jpg"
          autoPlay muted loop playsInline preload="metadata"
          className="absolute inset-0 w-full h-full object-cover"
          style={{ opacity: 0.25, mixBlendMode: 'luminosity' }}
        />
        {/* Gradient — stronger fade to ensure text legibility */}
        <div className="absolute inset-0" style={{
          background: 'linear-gradient(180deg, rgba(8,15,10,0.5) 0%, rgba(8,15,10,0.2) 40%, rgba(8,15,10,0.95) 100%)',
        }} />

        <div className="relative z-10 px-5 pt-14 pb-10">
          {/* Top bar */}
          <motion.div
            className="flex items-center justify-between mb-10"
            initial={reduced ? false : { opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ ...spring.smooth }}
          >
            <SpeedPlusLogo variant="full" theme="dark" size="md" />
            <Link
              href="/wallet"
              className="flex items-center gap-1.5 text-[12px] font-semibold text-white/70 hover:text-white transition-colors bg-white/[0.08] hover:bg-white/[0.12] border border-white/[0.08] rounded-full px-3.5 py-1.5"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" /></svg>
              Wallet
            </Link>
          </motion.div>

          {/* Hero copy */}
          <motion.div
            initial={reduced ? false : { opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1, ...spring.smooth }}
          >
            <h1 className="font-display font-bold text-white leading-[1.05] tracking-tight" style={{ fontSize: 'clamp(32px, 8vw, 42px)' }}>
              What do you<br />need today?
            </h1>
            <p className="text-white/40 mt-2 text-[13px] font-medium tracking-wide">Delivered fast across Lagos</p>
          </motion.div>
        </div>
      </div>

      {/* ── Search ───────────────────────────────────────────────────────── */}
      <motion.div
        className="px-5 pb-5 relative z-10 -mt-2"
        initial={reduced ? false : { opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.18, ...spring.smooth }}
      >
        <form onSubmit={handleSearch}>
          <motion.div
            className="flex items-center gap-3 rounded-2xl px-4 py-3.5 border"
            animate={
              searchFocused
                ? { borderColor: 'rgba(198,242,78,0.4)', backgroundColor: 'rgba(255,255,255,0.10)' }
                : { borderColor: 'rgba(255,255,255,0.08)', backgroundColor: 'rgba(255,255,255,0.06)' }
            }
            transition={{ duration: 0.15 }}
          >
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.4)" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onFocus={() => setSearchFocused(true)}
              onBlur={() => setTimeout(() => setSearchFocused(false), 150)}
              placeholder="Search — gas, food, pharmacy, packages…"
              className="flex-1 bg-transparent text-white placeholder-white/30 text-[13px] focus:outline-none" aria-label="Search — gas, food, pharmacy, packages"/>
            <AnimatePresence>
              {query && (
                <motion.button
                  type="button"
                  onClick={() => setQuery('')}
                  className="text-white/30 hover:text-white/60 transition-colors"
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.8 }}
                  transition={{ duration: 0.12 }}
                >
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
                </motion.button>
              )}
            </AnimatePresence>
          </motion.div>
        </form>

        <AnimatePresence>
          {showDropdown && (
            <motion.div
              className="absolute left-5 right-5 top-full mt-1.5 rounded-2xl overflow-hidden z-30 border border-white/[0.08]"
              style={{ background: '#0D1F14', boxShadow: '0 20px 60px rgba(0,0,0,0.5)' }}
              initial={{ opacity: 0, y: -8, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -6, scale: 0.98 }}
              transition={{ ...spring.snappy }}
            >
              {intent ? (
                <button
                  onClick={() => router.push(intent.route)}
                  className="w-full flex items-center gap-3 px-4 py-3.5 hover:bg-white/[0.05] transition-colors text-left"
                >
                  <span className="w-8 h-8 rounded-xl bg-lime/10 flex items-center justify-center flex-shrink-0">
                    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke={iconColors.lime} strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M5 12h14M12 5l7 7-7 7" /></svg>
                  </span>
                  <div>
                    <p className="text-[13px] font-semibold text-white">{intent.label}</p>
                    <p className="text-[11px] text-white/35 mt-0.5">Tap to continue</p>
                  </div>
                </button>
              ) : (
                <>
                  <p className="px-4 pt-3 pb-2 text-[10px] font-bold text-white/30 tracking-[0.08em] uppercase">Quick actions</p>
                  {SUGGESTIONS.map((s, i) => (
                    <motion.button
                      key={s}
                      onClick={() => handleSuggestion(s)}
                      className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-white/[0.05] transition-colors text-left"
                      initial={{ opacity: 0, x: -8 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: i * 0.04, ...spring.snappy }}
                    >
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.25)" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
                      <span className="text-[13px] text-white/60">{s}</span>
                    </motion.button>
                  ))}
                </>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>

      {/* ── Active order banner ───────────────────────────────────────────── */}
      <AnimatePresence>
        {activeOrder && (
          <motion.div
            className="px-5 pb-4"
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ ...spring.smooth }}
          >
            <button
              onClick={() => router.push(activeOrder.vertical === 'package' ? '/package/tracking' : '/orders')}
              className="w-full flex items-center gap-3 rounded-2xl px-4 py-3.5 border border-lime/20 hover:border-lime/40 transition-colors text-left"
              style={{ background: 'rgba(198,242,78,0.06)' }}
            >
              <span className="relative flex-shrink-0">
                <span className="w-2 h-2 rounded-full bg-lime block" />
                <span className="absolute inset-0 rounded-full bg-lime animate-ping opacity-60" />
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-[13px] font-semibold text-lime capitalize truncate">
                  {activeOrder.vertical} delivery in progress
                </p>
                <p className="text-[11px] text-white/40 mt-0.5 capitalize">
                  {activeOrder.status.replace(/_/g, ' ')} · tap to track
                </p>
              </div>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="rgba(198,242,78,0.6)" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><path d="M9 18l6-6-6-6" /></svg>
            </button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── Verticals ─────────────────────────────────────────────────────── */}
      <motion.div
        className="flex-1 px-5 pb-4"
        variants={reduced ? {} : staggerContainer}
        initial="hidden"
        animate="visible"
      >
        {/* Section label */}
        <motion.p
          className="text-[10px] font-bold text-white/25 tracking-[0.1em] uppercase mb-3"
          variants={reduced ? {} : staggerItem}
        >
          Services
        </motion.p>

        <div className="grid grid-cols-2 gap-2.5">
          {verticals.map((v) => (
            <motion.div key={v.href} variants={reduced ? {} : staggerItem}>
              <Link
                href={v.href}
                className={[
                  'group flex flex-col gap-3.5 rounded-2xl p-4 border transition-colors duration-200',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-lime',
                  v.featured
                    ? 'col-span-2 border-lime/15 hover:border-lime/30'
                    : 'border-white/[0.07] hover:border-white/[0.14]',
                ].join(' ')}
                style={{
                  background: v.featured
                    ? 'linear-gradient(135deg, rgba(10,61,44,0.9) 0%, rgba(6,30,20,0.95) 100%)'
                    : 'rgba(255,255,255,0.04)',
                  gridColumn: v.featured ? 'span 2' : undefined,
                }}
              >
                <div className="flex items-start justify-between">
                  <div
                    className="w-9 h-9 rounded-xl flex items-center justify-center text-lime/80 group-hover:text-lime transition-colors"
                    style={{ background: 'rgba(198,242,78,0.08)' }}
                  >
                    {v.icon}
                  </div>
                  {v.featured && (
                    <Badge className="bg-lime text-emerald text-[9px] px-2 py-0.5 tracking-wide">
                      POPULAR
                    </Badge>
                  )}
                </div>
                <div>
                  <p className="font-display font-semibold text-[14px] text-white/90 group-hover:text-white transition-colors leading-tight">
                    {v.label}
                  </p>
                  <p className="text-white/35 text-[11px] mt-0.5 leading-snug">{v.description}</p>
                </div>
              </Link>
            </motion.div>
          ))}
        </div>
      </motion.div>

      {/* ── Bottom nav ────────────────────────────────────────────────────── */}
      <div
        className="sticky bottom-0 border-t border-white/[0.06] px-2 py-2"
        style={{ background: 'rgba(8,15,10,0.96)', backdropFilter: 'blur(20px)' }}
      >
        <div className="flex justify-around max-w-sm mx-auto">
          {NAV_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="flex flex-col items-center gap-1 text-white/35 hover:text-white/80 transition-colors py-1.5 px-4 rounded-xl hover:bg-white/[0.05]"
            >
              {link.icon}
              <span className="text-[10px] font-semibold tracking-wide">{link.label}</span>
            </Link>
          ))}
        </div>
      </div>
    </main>
  );
}
