interface SpeedPlusLogoProps {
  variant?: 'full' | 'mark' | 'wordmark';
  theme?: 'dark' | 'light';
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

const LIME    = '#C6F24E';
const EMERALD = '#0A3D2C';

const sizes = {
  sm: { mark: 16,  text: 14 },
  md: { mark: 22,  text: 19 },
  lg: { mark: 28,  text: 26 },
  xl: { mark: 42,  text: 38 },
};

/**
 * Logomark: teardrop location pin in a square frame.
 * Pin is centered vertically within the square so it aligns with the text cap-height.
 */
function SpeedMark({ px, theme }: { px: number; theme: 'dark' | 'light' }) {
  const bg  = theme === 'dark' ? EMERALD : LIME;
  const fg  = theme === 'dark' ? LIME    : EMERALD;
  const fg2 = theme === 'dark' ? 'rgba(198,242,78,0.5)' : 'rgba(10,61,44,0.4)';

  // Square viewBox — pin sits centred inside
  const vw = 48;
  const vh = 48;
  const cx = vw / 2;        // 24
  const cr = vw * 0.36;     // circle radius — leaves room for tail below
  const cy = vw * 0.38;     // circle centre — shifted up so tail fits in frame

  const bx1 = cx - cr * 0.22;
  const by1 = cy + cr * 0.52;
  const bx2 = cx + cr * 0.22;
  const by2 = cy - cr * 0.52;
  const bsw = cr * 0.30;

  const bmx = (bx1 + bx2) / 2;
  const bmy = (by1 + by2) / 2;
  const cbw = cr * 0.38;
  const csw = bsw * 0.75;

  return (
    <svg
      width={px}
      height={px}
      viewBox={`0 0 ${vw} ${vh}`}
      fill="none"
      aria-hidden="true"
      style={{ flexShrink: 0 }}
    >
      <circle cx={cx} cy={cy} r={cr} fill={bg} />
      <path
        d={`M ${cx - cr * 0.55} ${cy + cr * 0.75} L ${cx} ${vh - 2} L ${cx + cr * 0.55} ${cy + cr * 0.75} Z`}
        fill={bg}
      />
      <line x1={bx1} y1={by1} x2={bx2} y2={by2}
        stroke={fg} strokeWidth={bsw} strokeLinecap="round" />
      <line x1={bmx - cbw} y1={bmy} x2={bmx + cbw} y2={bmy}
        stroke={fg2} strokeWidth={csw} strokeLinecap="round" />
    </svg>
  );
}

export function SpeedPlusLogo({ variant = 'full', theme = 'dark', size = 'md', className }: SpeedPlusLogoProps) {
  const { mark, text } = sizes[size];
  const textColor = theme === 'dark' ? '#FFFFFF' : '#121216';

  if (variant === 'mark') {
    return (
      <span role="img" aria-label="SpeedPlus" className={className} style={{ display: 'inline-flex' }}>
        <SpeedMark px={mark} theme={theme} />
      </span>
    );
  }

  const wordmark = (
    <span
      style={{
        fontFamily: 'var(--font-display, "Space Grotesk", sans-serif)',
        fontWeight: 800,
        fontSize: text,
        lineHeight: 1,
        color: textColor,
        letterSpacing: '-0.02em',
      }}
    >
      speed
    </span>
  );

  if (variant === 'wordmark') {
    return (
      <span role="img" aria-label="SpeedPlus" className={className} style={{ display: 'inline-flex', alignItems: 'center' }}>
        {wordmark}
      </span>
    );
  }

  // full: "speed" then pin mark as suffix
  return (
    <span
      role="img"
      aria-label="SpeedPlus"
      className={className}
      style={{ display: 'inline-flex', alignItems: 'center', gap: Math.round(mark * 0.18) - 2 }}
    >
      {wordmark}
      <SpeedMark px={mark} theme={theme} />
    </span>
  );
}
