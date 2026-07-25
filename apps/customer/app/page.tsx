import Link from 'next/link';
import { SpeedPlusLogo } from '@speedplus/ui';

const verticals = [
  {
    label: 'Cooking Gas',
    description: 'Cylinder refills & swaps. Never run out.',
    href: '/gas',
    icon: (
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" />
        <circle cx="12" cy="9" r="2.5" />
      </svg>
    ),
  },
  {
    label: 'Grocery',
    description: 'Fresh produce, pantry staples, essentials.',
    href: '/grocery',
    icon: (
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z" />
        <line x1="3" y1="6" x2="21" y2="6" />
        <path d="M16 10a4 4 0 01-8 0" />
      </svg>
    ),
  },
  {
    label: 'Food',
    description: 'Hot meals from local restaurants.',
    href: '/food',
    icon: (
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M18 8h1a4 4 0 010 8h-1" />
        <path d="M2 8h16v9a4 4 0 01-4 4H6a4 4 0 01-4-4V8z" />
        <line x1="6" y1="1" x2="6" y2="4" />
        <line x1="10" y1="1" x2="10" y2="4" />
        <line x1="14" y1="1" x2="14" y2="4" />
      </svg>
    ),
  },
  {
    label: 'Pharmacy',
    description: 'OTC meds and prescription fulfillment.',
    href: '/pharmacy',
    icon: (
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <line x1="12" y1="8" x2="12" y2="16" />
        <line x1="8" y1="12" x2="16" y2="12" />
      </svg>
    ),
  },
  {
    label: 'Package',
    description: 'Send anything across the city. Tracked.',
    href: '/package/where',
    icon: (
      <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#C6F24E" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
        <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
        <line x1="12" y1="22.08" x2="12" y2="12" />
      </svg>
    ),
    featured: true,
  },
];

const quickLinks = [
  {
    href: '/wallet',
    label: 'Wallet',
    icon: (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <rect x="2" y="5" width="20" height="14" rx="3" />
        <path d="M2 10h20" />
      </svg>
    ),
  },
  {
    href: '/orders',
    label: 'Orders',
    icon: (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M9 11l3 3L22 4" />
        <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" />
      </svg>
    ),
  },
  {
    href: '/referral',
    label: 'Refer',
    icon: (
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2" />
        <circle cx="9" cy="7" r="4" />
        <path d="M23 21v-2a4 4 0 00-3-3.87" />
        <path d="M16 3.13a4 4 0 010 7.75" />
      </svg>
    ),
  },
];

export default function HomePage() {
  return (
    <main className="min-h-screen bg-[#0A1F15] text-white flex flex-col">
      {/* Top bar */}
      <div className="px-5 pt-12 pb-6 flex items-center justify-between">
        <SpeedPlusLogo variant="full" theme="dark" size="md" />
        <Link
          href="/wallet"
          className="flex items-center gap-1.5 bg-white/10 hover:bg-white/15 transition-colors rounded-full px-3.5 py-1.5 text-[12px] font-semibold text-white/80"
        >
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <rect x="2" y="5" width="20" height="14" rx="3" /><path d="M2 10h20" />
          </svg>
          Wallet
        </Link>
      </div>

      {/* Hero */}
      <div className="px-5 pb-8">
        <h1 className="font-display font-bold text-[36px] leading-[1.1] tracking-tight">
          What do you<br />need today?
        </h1>
        <p className="text-white/50 mt-2 text-[15px]">Delivered fast across Lagos.</p>
      </div>

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
                <p className={`font-display font-semibold text-[15px] ${v.featured ? 'text-white' : 'text-white'} group-hover:text-[#C6F24E] transition-colors`}>
                  {v.label}
                </p>
                <p className="text-white/50 text-[12px] mt-0.5 leading-snug">{v.description}</p>
              </div>
              {v.featured && (
                <span className="absolute top-3 right-3 text-[9px] font-bold text-[#0A3D2C] bg-[#C6F24E] rounded-full px-2 py-0.5 tracking-wide">
                  SEND NOW
                </span>
              )}
            </Link>
          ))}
        </div>
      </div>

      {/* Bottom quick-access bar */}
      <div className="sticky bottom-0 bg-[#0A1F15]/95 backdrop-blur-sm border-t border-white/[0.08] px-5 py-3 pb-safe-bottom">
        <div className="flex justify-around max-w-sm mx-auto">
          {quickLinks.map((ql) => (
            <Link
              key={ql.href}
              href={ql.href}
              className="flex flex-col items-center gap-1 text-white/50 hover:text-white transition-colors py-1 px-4"
            >
              {ql.icon}
              <span className="text-[10px] font-semibold">{ql.label}</span>
            </Link>
          ))}
        </div>
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(14px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
