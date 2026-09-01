'use client';

import { type ReactNode } from 'react';
import { FourdatLogo } from '../logo';

interface Chip {
  icon: ReactNode;
  label: string;
}

interface AuthShellProps {
  headline: ReactNode;
  subtext?: string;
  portalLabel?: string;
  heroImage?: string;
  formHeading: string;
  formSubheading?: string;
  chips?: [Chip, Chip];
  children: ReactNode;
}

function GlassChip({ icon, label }: Chip) {
  return (
    <div
      className="flex items-center gap-2 px-3 py-2 rounded-xl backdrop-blur-md"
      style={{
        background: 'rgba(0,0,0,0.28)',
        border: '1px solid rgba(255,255,255,0.12)',
      }}
    >
      {icon}
      <span className="text-[11px] font-medium text-white/70 tracking-wide whitespace-nowrap">{label}</span>
    </div>
  );
}

export function AuthShell({
  headline,
  subtext,
  portalLabel,
  heroImage,
  formHeading,
  formSubheading,
  chips,
  children,
}: AuthShellProps) {
  return (
    <div className="min-h-screen w-full flex">

      {/* ── Brand panel ──────────────────────────────────────────────────── */}
      <div className="hidden lg:flex lg:w-[500px] xl:w-[580px] 2xl:w-[660px] flex-shrink-0 flex-col justify-between p-12 relative overflow-hidden">

        {/* Photo background */}
        {heroImage ? (
          <div
            className="absolute inset-0 bg-cover bg-center"
            style={{ backgroundImage: `url(${heroImage})` }}
          />
        ) : (
          <div className="absolute inset-0" style={{ background: 'linear-gradient(155deg, #0C4433 0%, #071E14 55%, #030D08 100%)' }} />
        )}

        {/* Dark gradient overlay — ensures text is always readable */}
        <div
          className="absolute inset-0"
          style={{ background: 'linear-gradient(to top, rgba(3,13,8,0.96) 0%, rgba(3,13,8,0.55) 45%, rgba(3,13,8,0.25) 100%)' }}
        />

        {/* Logo */}
        <a href="/" aria-label="Fourdat home" className="relative z-10 w-fit">
          <FourdatLogo variant="full" theme="dark" size="lg" />
        </a>

        {/* Headline block */}
        <div className="flex flex-col gap-5 relative z-10">
          {portalLabel && (
            <p className="text-[10px] font-bold tracking-[0.18em] uppercase" style={{ color: 'rgba(198,242,78,0.7)' }}>
              {portalLabel}
            </p>
          )}
          <h2
            className="font-display font-bold leading-[1.04] text-white"
            style={{ fontSize: 'clamp(36px, 4vw, 52px)', letterSpacing: '-0.025em' }}
          >
            {headline}
          </h2>
          {subtext && (
            <p className="max-w-[300px]" style={{ color: 'rgba(255,255,255,0.5)', fontSize: '14px', lineHeight: 1.7 }}>
              {subtext}
            </p>
          )}

          {/* Chips row */}
          {chips && (
            <div className="flex items-center gap-2 mt-2 flex-wrap">
              <GlassChip {...chips[0]} />
              <GlassChip {...chips[1]} />
            </div>
          )}
        </div>

        {/* Footer */}
        <p className="relative z-10 text-[10px] tracking-widest uppercase" style={{ color: 'rgba(255,255,255,0.2)' }}>
          © {new Date().getFullYear()} Fourdat Technologies
        </p>
      </div>

      {/* ── Form panel ───────────────────────────────────────────────────── */}
      <div
        className="flex-1 min-w-0 self-stretch flex flex-col items-center justify-center overflow-y-auto relative"
        style={{ background: 'linear-gradient(160deg, #F8F6F1 0%, #F2EFE7 100%)' }}
      >
        <div
          className="absolute top-0 right-0 w-[480px] h-[480px] pointer-events-none"
          aria-hidden="true"
          style={{ background: 'radial-gradient(circle at 100% 0%, rgba(10,61,44,0.05) 0%, transparent 55%)' }}
        />
        <div
          className="absolute bottom-0 left-0 w-[320px] h-[320px] pointer-events-none"
          aria-hidden="true"
          style={{ background: 'radial-gradient(circle at 0% 100%, rgba(198,242,78,0.04) 0%, transparent 60%)' }}
        />

        {/* Mobile logo */}
        <a href="/" aria-label="Fourdat home" className="lg:hidden mb-10 mt-14">
          <FourdatLogo variant="full" theme="light" size="lg" />
        </a>

        {/* Form card */}
        <div className="relative w-full max-w-[500px] mx-auto px-8 py-12 lg:py-0">
          <div className="mb-9">
            {portalLabel && (
              <p className="text-[10px] font-bold tracking-[0.14em] uppercase mb-3" style={{ color: '#0A3D2C' }}>
                {portalLabel}
              </p>
            )}
            <h1
              className="font-display font-bold text-ink"
              style={{ fontSize: 'clamp(24px, 2.8vw, 30px)', lineHeight: 1.06, letterSpacing: '-0.025em' }}
            >
              {formHeading}
            </h1>
            {formSubheading && (
              <p className="text-mid text-[13.5px] mt-2 leading-snug">{formSubheading}</p>
            )}
          </div>

          {children}
        </div>

        <div className="h-14 lg:hidden" />
      </div>
    </div>
  );
}

export type { Chip as AuthShellChip };
