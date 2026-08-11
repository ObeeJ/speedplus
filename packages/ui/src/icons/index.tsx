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

/** Camera — photo capture / upload affordances. */
export function CameraIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M3.5 8.5h3l1.5-2.5h8l1.5 2.5h3v10h-17z" />
      <circle cx="12" cy="13" r="3.4" stroke={accent} />
    </svg>
  );
}

// ── Driver milestone badges ───────────────────────────────────────────────────
// One per badge_type in driver_badges (zero_complaints reuses ShieldCheckIcon).

/** first_delivery — a single spark/burst. */
export function SparkIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3.5l2.2 5.6 5.8 1.9-4.6 3.6.6 6-4-3.1-4 3.1.6-6-4.6-3.6 5.8-1.9z" />
      <path d="M12 8.5v4" stroke={accent} />
    </svg>
  );
}

/** 10_deliveries — stacked parcels. */
export function BoxStackIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 13l8-3.5 8 3.5v5.5L12 21.5 4 18.5z" />
      <path d="M4 13l8 3.5 8-3.5" />
      <path d="M7.5 6.5L12 4.5l4.5 2" stroke={accent} />
    </svg>
  );
}

/** 50_deliveries — a rocket, for sustained volume. */
export function RocketIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2.5c3 2.5 4.5 6 4.5 9.5L12 16l-4.5-4c0-3.5 1.5-7 4.5-9.5z" />
      <path d="M7.5 12L5 14.5l1.5 4 3-2M16.5 12L19 14.5l-1.5 4-3-2" stroke={accent} />
      <circle cx="12" cy="9" r="1.6" />
    </svg>
  );
}

/** 100_deliveries — trophy. */
export function TrophyIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M7.5 4h9v5a4.5 4.5 0 01-9 0z" />
      <path d="M7.5 5.5H5a2.5 2.5 0 002.5 2.5M16.5 5.5H19a2.5 2.5 0 01-2.5 2.5" stroke={accent} />
      <path d="M12 13.5v3.5M8.5 20.5h7" />
    </svg>
  );
}

/** top_rated — star. */
export function StarIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3.5l2.6 5.5 6 .8-4.3 4.3 1 6-5.3-2.9-5.3 2.9 1-6L3.4 9.8l6-.8z" />
      <path d="M10.5 13.5l1.5 1.5 2.5-3" stroke={accent} />
    </svg>
  );
}

export function EyeIcon({ size = 16, color = 'currentColor' }: { size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

export function EyeOffIcon({ size = 16, color = 'currentColor' }: { size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19m-6.72-1.07a3 3 0 11-4.24-4.24" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </svg>
  );
}

export function AlertCircleIcon({ size = 14, color = 'currentColor' }: { size?: number; color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="12" cy="12" r="10" />
      <line x1="12" y1="8" x2="12" y2="12" />
      <line x1="12" y1="16" x2="12.01" y2="16" />
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

// ── Admin / ops nav icons ─────────────────────────────────────────────────────

/** Metrics / analytics — bar chart with trend line. */
export function MetricsIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="13" width="3.5" height="7" rx="1" stroke={stroke} strokeWidth={1.7} />
      <rect x="10.25" y="8" width="3.5" height="12" rx="1" stroke={accent} strokeWidth={1.7} />
      <rect x="16.5" y="4" width="3.5" height="16" rx="1" stroke={stroke} strokeWidth={1.7} />
      <path d="M4 10l4.5-3.5 4 2.5 5-5" stroke={accent} strokeWidth={1.5} strokeDasharray="1.5 1.5" />
    </svg>
  );
}

/** Ledger / double-entry — two columns with a divider. */
export function LedgerIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3.5" width="18" height="17" rx="2.5" stroke={stroke} strokeWidth={1.7} />
      <path d="M12 3.5v17" stroke={accent} strokeWidth={1.5} />
      <path d="M6 8h3.5M6 11.5h3.5M6 15h2" stroke={stroke} strokeWidth={1.5} />
      <path d="M14.5 8H18M14.5 11.5H18M14.5 15h2" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Delivery run — route with multiple stops. */
export function RunIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="5" cy="12" r="2" stroke={stroke} strokeWidth={1.7} />
      <circle cx="19" cy="12" r="2" stroke={stroke} strokeWidth={1.7} />
      <circle cx="12" cy="7" r="2" stroke={accent} strokeWidth={1.7} />
      <path d="M7 12h5M14 12h3" stroke={stroke} strokeWidth={1.5} />
      <path d="M6.5 10.5L10.5 8.5M13.5 8.5L17.5 10.5" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** KYC / identity check — person with a tick. */
export function KYCIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="10" cy="7.5" r="3.5" stroke={stroke} strokeWidth={1.7} />
      <path d="M3.5 20c0-3.5 2.9-6 6.5-6" stroke={stroke} strokeWidth={1.7} />
      <path d="M14.5 15l2 2 4-4" stroke={accent} strokeWidth={1.9} />
    </svg>
  );
}

/** Merchant / storefront — shop front with awning. */
export function StoreIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 10.5V20h16v-9.5" stroke={stroke} strokeWidth={1.7} />
      <path d="M2.5 7l1.5-3.5h16L21.5 7" stroke={stroke} strokeWidth={1.7} />
      <path d="M2.5 7c0 1.7 1.3 3 3 3s3-1.3 3-3c0 1.7 1.3 3 3 3s3-1.3 3-3c0 1.7 1.3 3 3 3s3-1.3 3-3" stroke={accent} strokeWidth={1.5} />
      <rect x="9" y="14" width="6" height="6" rx="1" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Driver / rider — motorcycle silhouette. */
export function DriverIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="6" cy="16" r="3" stroke={stroke} strokeWidth={1.7} />
      <circle cx="18" cy="16" r="3" stroke={stroke} strokeWidth={1.7} />
      <path d="M9 16h6M12 16l-2-5h4l2 5" stroke={stroke} strokeWidth={1.5} />
      <path d="M10 11l2-4h3l2 2" stroke={accent} strokeWidth={1.7} />
      <circle cx="14.5" cy="6.5" r="1.5" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Users / people — two overlapping silhouettes. */
export function UsersIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="9" cy="7.5" r="3" stroke={stroke} strokeWidth={1.7} />
      <path d="M3 20c0-3.3 2.7-5.5 6-5.5s6 2.2 6 5.5" stroke={stroke} strokeWidth={1.7} />
      <circle cx="17" cy="7.5" r="2.5" stroke={accent} strokeWidth={1.5} />
      <path d="M20 20c0-2.8-1.8-4.8-4-5.3" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Dispute / scales of justice. */
export function DisputeIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 4v16M8 20h8" stroke={stroke} strokeWidth={1.7} />
      <path d="M12 4l-6 2M12 4l6 2" stroke={stroke} strokeWidth={1.5} />
      <path d="M6 6l-3 5h6l-3-5z" stroke={accent} strokeWidth={1.5} />
      <path d="M18 6l-3 5h6l-3-5z" stroke={stroke} strokeWidth={1.5} />
    </svg>
  );
}

/** Subscription / recurring — circular arrows with calendar. */
export function SubscriptionIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3.5" y="5.5" width="17" height="15" rx="2.5" stroke={stroke} strokeWidth={1.7} />
      <path d="M3.5 10h17" stroke={stroke} strokeWidth={1.5} />
      <path d="M8 3.5v4M16 3.5v4" stroke={accent} strokeWidth={1.7} />
      <path d="M9 15.5c0-1.7 1.3-3 3-3s3 1.3 3 3" stroke={accent} strokeWidth={1.5} />
      <path d="M14.5 13.5l.5 2h-2" stroke={accent} strokeWidth={1.3} />
    </svg>
  );
}

/** Prescription / Rx — pill bottle with Rx label. */
export function PrescriptionIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <rect x="8" y="3.5" width="8" height="17" rx="3" stroke={stroke} strokeWidth={1.7} />
      <path d="M8 9h8" stroke={stroke} strokeWidth={1.5} />
      <path d="M11 6h2" stroke={accent} strokeWidth={1.7} />
      <path d="M10.5 13h1.5v1.5M10.5 13v3M12 14.5l2 2" stroke={accent} strokeWidth={1.4} />
    </svg>
  );
}

/** Gas / flame — LPG cylinder with flame. */
export function GasIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <rect x="7" y="7" width="10" height="13" rx="3" stroke={stroke} strokeWidth={1.7} />
      <path d="M9.5 7V5.5h5V7" stroke={stroke} strokeWidth={1.5} />
      <path d="M12 5.5V4" stroke={accent} strokeWidth={1.5} />
      <path d="M12 11c0 0-2 1.5-2 3a2 2 0 004 0c0-1.5-2-3-2-3z" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Zone / map area — polygon with location pin. */
export function ZoneIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 6l5-2.5 6 2.5 5-2.5v13L15 19.5l-6-2.5-5 2.5z" stroke={stroke} strokeWidth={1.7} />
      <path d="M9 3.5v13M15 6v13" stroke={accent} strokeWidth={1.3} strokeDasharray="2 2" />
      <circle cx="15" cy="10" r="2" stroke={accent} strokeWidth={1.5} />
      <path d="M15 12v2" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** LPG price / fuel pump. */
export function FuelIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 21V5.5A2.5 2.5 0 017.5 3h5A2.5 2.5 0 0115 5.5V21" stroke={stroke} strokeWidth={1.7} />
      <path d="M3.5 21h13" stroke={stroke} strokeWidth={1.7} />
      <path d="M15 8.5h2a2 2 0 012 2v4a1.5 1.5 0 003 0V9l-2-2.5" stroke={accent} strokeWidth={1.5} />
      <rect x="7" y="6" width="6" height="4" rx="1" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Fee / pricing — tag with currency symbol. */
export function FeeIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3.5 12.5L12 4l8 8-8.5 8.5-8-8z" stroke={stroke} strokeWidth={1.7} />
      <circle cx="15.5" cy="8.5" r="1.5" stroke={accent} strokeWidth={1.5} />
      <path d="M9.5 13.5l1.5 1.5" stroke={accent} strokeWidth={1.7} />
      <path d="M11.5 11.5l1.5 1.5" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}

/** Cancel / rules — document with X. */
export function RulesIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M6 3.5h12v17H6z" rx="1.5" stroke={stroke} strokeWidth={1.7} />
      <path d="M9 8h6M9 11.5h4" stroke={stroke} strokeWidth={1.5} />
      <path d="M9.5 16.5l3-3M12.5 16.5l-3-3" stroke={accent} strokeWidth={1.7} />
    </svg>
  );
}

/** Package / parcel — box with tape stripe. */
export function PackageIcon({ size = 20, active = false, color = DEFAULT_COLOR, accent = DEFAULT_ACCENT }: DuotoneIconProps) {
  const stroke = active ? ACTIVE_COLOR : color;
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3.5 7.5L12 3l8.5 4.5v9L12 21l-8.5-4.5v-9z" stroke={stroke} strokeWidth={1.7} />
      <path d="M3.5 7.5L12 12l8.5-4.5" stroke={stroke} strokeWidth={1.5} />
      <path d="M12 12v9" stroke={stroke} strokeWidth={1.5} />
      <path d="M8 5.5l8 4.5" stroke={accent} strokeWidth={1.5} />
    </svg>
  );
}
