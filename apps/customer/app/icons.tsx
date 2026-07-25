export function SearchIcon({ size = 18, stroke = '#0A3D2C' }: { size?: number; stroke?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="7" />
      <path d="M21 21l-4.3-4.3" />
    </svg>
  );
}

export function PackageIcon({ size = 22, active = false }: { size?: number; active?: boolean }) {
  const stroke = active ? '#C6F24E' : '#1C3A2E';
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={active ? 1.8 : 1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M3.5 7.5L12 3l8.5 4.5v9L12 21l-8.5-4.5v-9z" />
      <path d="M3.5 7.5L12 12l8.5-4.5M12 12v9" />
    </svg>
  );
}

export function GasIcon({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="#1C3A2E" strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <rect x="8" y="7" width="8" height="14" rx="3" />
      <path d="M10 7V4.5M14 7V4.5M9 4.5h6" />
      <path d="M10.5 11.5h3" stroke="#7BA05B" />
    </svg>
  );
}

export function FoodIcon({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="#1C3A2E" strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 16a8 8 0 0116 0z" />
      <path d="M3 19h18" />
      <path d="M12 8V6.8" stroke="#7BA05B" />
    </svg>
  );
}

export function MedicineIcon({ size = 22 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="#1C3A2E" strokeWidth={1.7} strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="5.5" />
      <path d="M12 8.5v7M8.5 12h7" stroke="#7BA05B" />
    </svg>
  );
}

export function HomeIcon({ size = 17, stroke = '#0A3D2C', strokeWidth = 2 }: { size?: number; stroke?: string; strokeWidth?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={strokeWidth} strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 11l9-8 9 8v9a1 1 0 01-1 1h-5v-6h-6v6H4a1 1 0 01-1-1v-9z" />
    </svg>
  );
}

export function OrdersIcon({ size = 17, stroke = '#63636E' }: { size?: number; stroke?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 21s-7-5.5-7-11a7 7 0 0114 0c0 5.5-7 11-7 11z" />
      <circle cx="12" cy="10" r="2.5" />
    </svg>
  );
}

export function MoneyIcon({ size = 17, stroke = '#63636E' }: { size?: number; stroke?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="6" width="18" height="13" rx="3" />
      <path d="M3 10h18" />
    </svg>
  );
}

export function MeIcon({ size = 17, stroke = '#63636E' }: { size?: number; stroke?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={stroke} strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21a8 8 0 0116 0" />
    </svg>
  );
}
