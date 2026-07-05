import type { SVGProps } from 'react';

interface SpeedPlusLogoProps {
  variant?: 'full' | 'mark' | 'wordmark';
  theme?: 'dark' | 'light';
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

const sizes = {
  sm: { markSize: 24, fontSize: 16, gap: 8 },
  md: { markSize: 32, fontSize: 22, gap: 10 },
  lg: { markSize: 44, fontSize: 30, gap: 14 },
  xl: { markSize: 64, fontSize: 44, gap: 20 },
};

const SPEARMINT = '#00C48C';

function PlusMark({ size, color, ...props }: { size: number; color: string } & SVGProps<SVGSVGElement>) {
  const arm = Math.round(size * 0.2);
  const len = Math.round(size * 0.62);
  const r = Math.round(arm / 2);
  const cx = size / 2;
  const cy = size / 2;

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      {...props}
    >
      <rect x={cx - arm / 2} y={cy - len / 2} width={arm} height={len} rx={r} fill={color} />
      <rect x={cx - len / 2} y={cy - arm / 2} width={len} height={arm} rx={r} fill={color} />
    </svg>
  );
}

export function SpeedPlusLogo({ variant = 'full', theme = 'dark', size = 'md', className }: SpeedPlusLogoProps) {
  const { markSize, fontSize, gap } = sizes[size];
  const textColor = theme === 'dark' ? '#FFFFFF' : '#1A1A2E';
  const markColor = variant === 'mark' && theme === 'light' ? '#1A1A2E' : SPEARMINT;

  if (variant === 'mark') {
    return <PlusMark size={markSize} color={markColor} className={className} role="img" aria-label="SpeedPlus" />;
  }

  if (variant === 'wordmark') {
    const w = Math.round(fontSize * 4.8);
    const h = Math.round(fontSize * 1.2);
    return (
      <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} fill="none" xmlns="http://www.w3.org/2000/svg" className={className} role="img" aria-label="SpeedPlus">
        <text y={fontSize} fontFamily="'Plus Jakarta Sans', Inter, sans-serif" fontWeight="800" fontSize={fontSize} fill={textColor}>
          Speed<tspan fill={SPEARMINT}>+</tspan>
        </text>
      </svg>
    );
  }

  const totalWidth = markSize + gap + Math.round(fontSize * 4.8);
  const totalHeight = Math.max(markSize, Math.round(fontSize * 1.2));

  return (
    <svg width={totalWidth} height={totalHeight} viewBox={`0 0 ${totalWidth} ${totalHeight}`} fill="none" xmlns="http://www.w3.org/2000/svg" className={className} role="img" aria-label="SpeedPlus">
      <g transform={`translate(0, ${(totalHeight - markSize) / 2})`}>
        <PlusMark size={markSize} color={SPEARMINT} />
      </g>
      <text
        x={markSize + gap}
        y={(totalHeight + fontSize * 0.75) / 2}
        fontFamily="'Plus Jakarta Sans', Inter, sans-serif"
        fontWeight="800"
        fontSize={fontSize}
        fill={textColor}
      >
        Speed<tspan fill={SPEARMINT}>+</tspan>
      </text>
    </svg>
  );
}
