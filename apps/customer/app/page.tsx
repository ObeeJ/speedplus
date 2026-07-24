import Link from 'next/link';
import { SpeedPlusLogo } from '@speedplus/ui';

const verticals = [
  {
    label: 'Cooking Gas',
    description: 'Cylinder refills & swaps delivered to your door.',
    href: '/gas',
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" />
        <circle cx="12" cy="9" r="2.5" />
      </svg>
    ),
  },
  {
    label: 'Grocery',
    description: 'Fresh produce, pantry staples, household essentials.',
    href: '/grocery',
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M6 2L3 6v14a2 2 0 002 2h14a2 2 0 002-2V6l-3-4z" />
        <line x1="3" y1="6" x2="21" y2="6" />
        <path d="M16 10a4 4 0 01-8 0" />
      </svg>
    ),
  },
  {
    label: 'Food',
    description: 'Hot meals from local restaurants, delivered fast.',
    href: '/food',
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
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
    description: 'OTC meds and prescription fulfillment. Upload your Rx.',
    href: '/pharmacy',
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <line x1="12" y1="8" x2="12" y2="16" />
        <line x1="8" y1="12" x2="16" y2="12" />
      </svg>
    ),
  },
  {
    label: 'Package',
    description: 'Send anything across the city. Fast, tracked, insured.',
    href: '/package/where',
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
        <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
        <line x1="12" y1="22.08" x2="12" y2="12" />
      </svg>
    ),
  },
];

export default function HomePage() {
  return (
    <main className="min-h-screen bg-midnight text-white">
      <div className="mx-auto max-w-5xl px-6 py-16 flex flex-col items-center gap-12">

        <header
          className="flex flex-col items-center gap-4 text-center"
          style={{ animation: 'fadeDown 0.4s cubic-bezier(0.16,1,0.3,1) both' }}
        >
          <SpeedPlusLogo variant="full" theme="dark" size="xl" />
          <p className="text-mid text-lg max-w-md">
            Faster. Cheaper. Better. Essential delivery for everyday Nigeria.
          </p>
        </header>

        <section className="w-full">
          <h2 className="text-center text-sm font-semibold uppercase tracking-widest text-mid mb-6">
            What do you need?
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {verticals.map((v, i) => (
              <Link
                key={v.href}
                href={v.href}
                className="group rounded-2xl border border-white/10 bg-white/5 p-6 hover:bg-white/10 hover:border-lime/40 transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-lime"
                style={{ animation: `fadeUp 0.4s cubic-bezier(0.16,1,0.3,1) ${i * 60}ms both` }}
              >
                <div className="w-12 h-12 rounded-xl bg-tile flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-200">
                  {v.icon}
                </div>
                <h3 className="font-display font-bold text-xl text-white mb-1 group-hover:text-lime transition-colors">
                  {v.label}
                </h3>
                <p className="text-mid text-sm leading-relaxed">{v.description}</p>
              </Link>
            ))}
          </div>
        </section>

        <footer className="text-center text-xs text-mid/60 mt-8">
          Built for the rest of us. &copy; {new Date().getFullYear()} SpeedPlus
        </footer>
      </div>

      <style>{`
        @keyframes fadeDown {
          from { opacity: 0; transform: translateY(-16px); }
          to   { opacity: 1; transform: translateY(0); }
        }
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(20px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
