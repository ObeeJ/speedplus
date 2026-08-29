'use client';

import Link from 'next/link';
import { useEffect, useRef, useState } from 'react';
import { SpeedPlusLogo } from '@speedplus/ui';

// ── Area ticker ───────────────────────────────────────────────────────────────
const AREAS = ['in your hands.', 'at your door.', 'confirmed by you.', 'with you.'];

// ── Area ticker component ─────────────────────────────────────────────────────
function AreaTicker() {
  const [index, setIndex]     = useState(0);
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const id = setInterval(() => {
      setVisible(false);
      setTimeout(() => {
        setIndex((i) => (i + 1) % AREAS.length);
        setVisible(true);
      }, 280);
    }, 2600);
    return () => clearInterval(id);
  }, []);

  return (
    <span
      style={{
        display:    'inline-block',
        color:      '#C6F24E',
        opacity:    visible ? 1 : 0,
        transform:  visible ? 'translateY(0)' : 'translateY(8px)',
        transition: 'opacity 0.26s ease, transform 0.26s ease',
      }}
    >
      {AREAS[index]}
    </span>
  );
}

// ── Stats ─────────────────────────────────────────────────────────────────────
const STATS = [
  { value: 'You stay in control', label: 'of your money' },
  { value: 'Pay when it arrives', label: 'not before' },
  { value: 'Gas, meds, packages', label: 'all in one place' },
];

// ── Services ──────────────────────────────────────────────────────────────────
const SERVICES = [
  {
    label: 'Cooking Gas',
    sub:   'Cylinder refills & swaps',
    href:  '/gas',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 2C8 2 5 5 5 9c0 5 7 13 7 13s7-8 7-13c0-4-3-7-7-7z" />
        <circle cx="12" cy="9" r="2.5" />
      </svg>
    ),
  },
  {
    label: 'Pharmacy',
    sub:   'Meds & prescriptions',
    href:  '/pharmacy',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <line x1="12" y1="8" x2="12" y2="16" />
        <line x1="8" y1="12" x2="16" y2="12" />
      </svg>
    ),
  },
  {
    label: 'Send Package',
    sub:   'Anything, anywhere in the city',
    href:  '/package/where',
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
        <path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z" />
        <polyline points="3.27 6.96 12 12.01 20.73 6.96" />
        <line x1="12" y1="22.08" x2="12" y2="12" />
      </svg>
    ),
  },
];

// ── Page ──────────────────────────────────────────────────────────────────────
export default function LandingPage() {
  const videoRef = useRef<HTMLVideoElement>(null);

  return (
    <main className="min-h-screen flex flex-col" style={{ background: '#080F0A' }}>

      {/* ── Nav ────────────────────────────────────────────────────────── */}
      <header className="relative z-20 flex items-center justify-between px-6 pt-6 lg:pt-8 max-w-6xl mx-auto w-full">
        <SpeedPlusLogo variant="full" theme="dark" size="md" />

        <nav className="hidden md:flex items-center gap-0.5">
          {['For Riders', 'For Merchants', 'About'].map((item) => (
            <Link
              key={item}
              href="#"
              className="px-4 py-2 text-[13px] font-medium text-white/45 hover:text-white/80 transition-colors rounded-lg hover:bg-white/[0.05]"
            >
              {item}
            </Link>
          ))}
        </nav>

        <div className="flex items-center gap-2">
          <Link
            href="/login"
            className="px-4 py-2 text-[13px] font-semibold text-white/60 hover:text-white transition-colors"
          >
            Sign in
          </Link>
          <Link
            href="/register"
            className="px-4 py-2 text-[13px] font-semibold text-emerald bg-lime rounded-lg hover:bg-lime-600 transition-colors"
          >
            Get started
          </Link>
        </div>
      </header>

      {/* ── Hero ───────────────────────────────────────────────────────── */}
      <section className="relative flex-1 flex flex-col">

        {/* Video layer */}
        <div className="absolute inset-0 overflow-hidden">
          <video
            ref={videoRef}
            src="/hero.mp4"
            poster="/hero-poster.jpg"
            autoPlay
            muted
            loop
            playsInline
            preload="metadata"
            className="w-full h-full object-cover"
            style={{ opacity: 0.07, mixBlendMode: 'luminosity' }}
          />
          {/* Solid dark wash — no multi-stop gradient */}
          <div className="absolute inset-0" style={{ background: 'rgba(8,15,10,0.52)' }} />
          {/* Bottom fade to match page bg */}
          <div
            className="absolute bottom-0 left-0 right-0 h-56"
            style={{ background: 'linear-gradient(to bottom, transparent, #080F0A)' }}
          />
        </div>

        {/* Content */}
        <div className="relative z-10 flex flex-col flex-1 items-center text-center px-6 pt-12 pb-16 lg:pt-24 w-full max-w-4xl mx-auto">

          {/* Headline */}
          <h1
            className="font-display font-bold text-white leading-[1.03] tracking-tight animate-fade-up-1 text-center"
            style={{ fontSize: 'clamp(38px, 7.5vw, 76px)' }}
          >
            You pay only
            <br />
            <span style={{ color: 'rgba(255,255,255,0.28)' }}>when it is </span>
            <AreaTicker />
          </h1>

          {/* Body */}
          <p className="mt-5 text-[15px] leading-relaxed max-w-sm mx-auto animate-fade-up-2" style={{ color: 'rgba(255,255,255,0.4)', fontFamily: 'var(--font-body)' }}>
            Gas, medicine and packages delivered across Lagos. You only pay when your order is in your hands.
          </p>

          {/* CTAs */}
          <div className="flex flex-wrap items-center justify-center gap-3 mt-8 animate-fade-up-3">
            <Link
              href="/register"
              className="inline-flex items-center gap-2 px-6 py-3.5 bg-lime text-emerald text-[14px] font-bold rounded-lg hover:bg-lime-600 transition-colors"
            >
              Order now
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </Link>
            <Link
              href="#"
              className="inline-flex items-center gap-2 px-6 py-3.5 text-[14px] font-semibold rounded-lg border border-white/[0.12] hover:border-white/25 transition-colors"
              style={{ color: 'rgba(255,255,255,0.55)' }}
            >
              Become a rider
            </Link>
          </div>

          {/* Proof points */}
          <div className="flex flex-wrap justify-center gap-x-8 gap-y-3 mt-12 animate-fade-up-4">
            {STATS.map((s, i) => (
              <div key={s.label} className="flex items-center gap-2">
                {i > 0 && <span className="hidden sm:block w-px h-3" style={{ background: 'rgba(255,255,255,0.12)' }} />}
                <span className="text-[13px] font-semibold" style={{ color: 'rgba(255,255,255,0.85)' }}>{s.value}</span>
                <span className="text-[13px]" style={{ color: 'rgba(255,255,255,0.3)' }}>{s.label}</span>
              </div>
            ))}
          </div>
        </div>

        {/* ── Service strip ──────────────────────────────────────────── */}
        <div className="relative z-10 px-6 pb-14 w-full max-w-4xl mx-auto animate-fade-up-5">
          <div
            className="flex flex-col sm:flex-row overflow-hidden rounded-xl border border-white/[0.07]"
            style={{ background: 'rgba(255,255,255,0.03)' }}
          >
            {SERVICES.map((svc, i) => (
              <Link
                key={svc.href}
                href={svc.href}
                className={[
                  'group flex-1 flex items-center gap-3.5 px-5 py-4 transition-colors hover:bg-white/[0.05]',
                  i < SERVICES.length - 1
                    ? 'border-b sm:border-b-0 sm:border-r border-white/[0.06]'
                    : '',
                ].join(' ')}
              >
                <span
                  className="w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0 transition-colors"
                  style={{
                    background: 'rgba(198,242,78,0.07)',
                    color: 'rgba(198,242,78,0.65)',
                  }}
                >
                  {svc.icon}
                </span>
                <span className="flex flex-col gap-0.5 min-w-0">
                  <span
                    className="text-[13px] font-semibold leading-tight transition-colors group-hover:text-white"
                    style={{ color: 'rgba(255,255,255,0.75)' }}
                  >
                    {svc.label}
                  </span>
                  <span className="text-[11px] truncate" style={{ color: 'rgba(255,255,255,0.28)' }}>
                    {svc.sub}
                  </span>
                </span>
                <svg
                  className="ml-auto flex-shrink-0 transition-colors group-hover:opacity-60"
                  style={{ opacity: 0.2 }}
                  width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"
                >
                  <path d="M9 18l6-6-6-6" />
                </svg>
              </Link>
            ))}
          </div>
        </div>

      </section>
    </main>
  );
}
