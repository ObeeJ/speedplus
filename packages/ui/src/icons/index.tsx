// Shared duotone icon set — hand-drawn inline SVGs, not a generic icon
// library. Each icon uses two stroke colors: a primary outline plus one
// accent-colored detail stroke (matching the pattern already established in
// apps/customer/app/icons.tsx: PackageIcon/GasIcon/FoodIcon/MedicineIcon).
// `active` swaps the primary stroke to the brand lime for the selected state.

export interface DuotoneIconProps {
  size?: number;
  active?: boolean;
  /** Primary outline color when not active. Defaults to the ink/emerald tone. */
  color?: string;
  /** Accent detail-stroke color. Defaults to a muted sage. */
  accent?: string;
}

const DEFAULT_COLOR = '#1C3A2E';
const ACTIVE_COLOR = '#C6F24E';
const DEFAULT_ACCENT = '#7BA05B';

export function DashboardIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <rect x="3.5" y="3.5" width="7.5" height="7.5" rx="1.6" />
      <rect x="13" y="3.5" width="7.5" height="4.5" rx="1.6" stroke={accent} />
      <rect x="13" y="10" width="7.5" height="10.5" rx="1.6" />
      <rect x="3.5" y="13" width="7.5" height="7.5" rx="1.6" stroke={accent} />
    </svg>
  );
}

export function ReceiptIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M6 3.5h12v17l-2.5-1.5L13 20.5 10.5 19 8 20.5 5.5 19V3.5z" />
      <path d="M8.5 8h7M8.5 11.5h7" />
      <path d="M8.5 15h4" stroke={accent} />
    </svg>
  );
}

export function PillIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <g transform="translate(0 -4)">
        <path d="M7 21L21 7a5 5 0 10-7-7L0 14a5 5 0 007 7z" transform="translate(3 3)" stroke={stroke} />
        <path d="M9.5 14.5l5-5" transform="translate(3 3)" stroke={accent} />
      </g>
    </svg>
  );
}

export function BoxIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M3.5 7.5L12 3l8.5 4.5v9L12 21l-8.5-4.5v-9z" />
      <path d="M3.5 7.5L12 12l8.5-4.5" />
      <path d="M12 12v9" stroke={accent} />
    </svg>
  );
}

export function WalletIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="6" width="18" height="13" rx="3" />
      <path d="M3 10h18" />
      <circle cx="16.5" cy="14" r="1.4" fill={accent} stroke="none" />
    </svg>
  );
}

export function ShieldCheckIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l7 3v6c0 4.5-3 8-7 9-4-1-7-4.5-7-9V6z" />
      <path d="M9 12l2 2 4-4.5" stroke={accent} />
    </svg>
  );
}

export function PowerIcon({ size = 16, color = '#9A968D' }: { size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3v7" />
      <path d="M6.5 6.5a8 8 0 1011 0" />
    </svg>
  );
}
