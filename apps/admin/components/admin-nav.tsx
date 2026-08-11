'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useAdminAuthStore } from '@/lib/store/auth.store';
import { authApi } from '@speedplus/api-client';
import {
  MetricsIcon,
  LedgerIcon,
  RunIcon,
  KYCIcon,
  StoreIcon,
  DriverIcon,
  UsersIcon,
  ReceiptIcon,
  PackageIcon,
  DisputeIcon,
  SubscriptionIcon,
  PrescriptionIcon,
  GasIcon,
  ZoneIcon,
  FuelIcon,
  FeeIcon,
  RulesIcon,
  PowerIcon,
  type DuotoneIconProps,
} from '@speedplus/ui';

type NavItem = {
  href: string;
  label: string;
  Icon: (props: DuotoneIconProps) => React.JSX.Element;
  /** exact match only (avoids /orders matching /orders/package) */
  exact?: boolean;
};

type NavSection = { section: string; items: NavItem[] };

const NAV: NavSection[] = [
  {
    section: 'Overview',
    items: [
      { href: '/metrics', label: 'Dashboard',    Icon: MetricsIcon },
      { href: '/ledger',  label: 'Ledger',        Icon: LedgerIcon },
      { href: '/runs',    label: 'Delivery Runs', Icon: RunIcon },
    ],
  },
  {
    section: 'People',
    items: [
      { href: '/kyc',       label: 'KYC Queue',  Icon: KYCIcon },
      { href: '/merchants', label: 'Merchants',  Icon: StoreIcon },
      { href: '/drivers',   label: 'Drivers',    Icon: DriverIcon },
      { href: '/users',     label: 'Users',      Icon: UsersIcon },
    ],
  },
  {
    section: 'Orders',
    items: [
      { href: '/orders',        label: 'All Orders',     Icon: ReceiptIcon, exact: true },
      { href: '/orders/package',label: 'Package',        Icon: PackageIcon },
      { href: '/disputes',      label: 'Disputes',       Icon: DisputeIcon },
      { href: '/subscriptions', label: 'Subscriptions',  Icon: SubscriptionIcon },
      { href: '/pharmacy',      label: 'Prescriptions',  Icon: PrescriptionIcon },
    ],
  },
  {
    section: 'Gas',
    items: [
      { href: '/gas/merchants',   label: 'Fill Accuracy', Icon: GasIcon },
      { href: '/gas/zones',       label: 'Zones',         Icon: ZoneIcon },
      { href: '/gas/price-index', label: 'LPG Price',     Icon: FuelIcon },
    ],
  },
  {
    section: 'Settings',
    items: [
      { href: '/settings/fees',               label: 'Fee Configs',      Icon: FeeIcon },
      { href: '/settings/cancellation-rules', label: 'Cancel Rules',     Icon: RulesIcon },
      { href: '/settings/weather',            label: 'Weather Surcharge',Icon: FuelIcon },
    ],
  },
];

const ACTIVE_COLOR  = '#C6F24E';
const ACTIVE_ACCENT = '#A8D43A';
const IDLE_COLOR    = 'rgba(247,245,239,0.55)';
const IDLE_ACCENT   = 'rgba(247,245,239,0.30)';

export function AdminNav() {
  const path = usePathname();
  const router = useRouter();
  const clearAuth = useAdminAuthStore((s) => s.clearAuth);
  const user = useAdminAuthStore((s) => s.user);

  async function handleSignOut() {
    await authApi.logout().catch(() => {});
    clearAuth();
    router.replace('/login');
  }

  if (path.startsWith('/login')) return null;

  return (
    <aside
      className="w-full lg:w-[220px] lg:min-h-screen flex-none flex flex-col gap-0 lg:overflow-y-auto"
      style={{ background: '#07291A' }}
    >
      {/* Logo */}
      <div className="px-5 pt-5 pb-4 lg:pt-6 lg:pb-5 flex items-center justify-between">
        <span className="font-display font-bold text-[19px] text-[#F7F5EF] tracking-tight select-none">
          speed<span style={{ color: '#C6F24E' }}>+</span>{' '}
          <span className="font-medium text-[10px] text-[#F7F5EF]/40 tracking-widest uppercase">OPS</span>
        </span>
      </div>

      {/* Nav sections — horizontal scroll on mobile, vertical on desktop */}
      <nav className="flex lg:flex-col gap-5 flex-1 px-3 pb-3 lg:pb-4 overflow-x-auto lg:overflow-x-visible">
        {NAV.map(({ section, items }) => (
          <div key={section} className="flex lg:flex-col gap-0.5 shrink-0">
            <span className="hidden lg:block px-3 mb-1 text-[9.5px] font-semibold tracking-[0.9px] uppercase text-[#F7F5EF]/30">
              {section}
            </span>
            {items.map(({ href, label, Icon, exact }) => {
              const active = exact ? path === href : path.startsWith(href);
              return (
                <Link
                  key={href}
                  href={href}
                  className={`flex items-center gap-2 rounded-[11px] px-3 py-2 text-[12.5px] font-medium transition-all whitespace-nowrap ${
                    active
                      ? 'bg-[#C6F24E]/[.12] text-[#C6F24E] font-semibold'
                      : 'text-[#F7F5EF]/55 hover:bg-[#F7F5EF]/[.06] hover:text-[#F7F5EF]/85'
                  }`}
                >
                  <Icon
                    size={16}
                    active={active}
                    color={active ? ACTIVE_COLOR : IDLE_COLOR}
                    accent={active ? ACTIVE_ACCENT : IDLE_ACCENT}
                  />
                  {label}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      {/* User footer — desktop only */}
      {user && (
        <div className="hidden lg:flex px-4 py-4 border-t border-[#F7F5EF]/[.07] items-center justify-between gap-2">
          <div className="flex flex-col min-w-0">
            <span className="text-[11px] font-semibold text-[#F7F5EF]/70 truncate">
              {user.firstName} {user.lastName}
            </span>
            <span className="text-[10px] text-[#F7F5EF]/35 truncate">Admin</span>
          </div>
          <button
            onClick={handleSignOut}
            title="Sign out"
            className="shrink-0 p-1.5 rounded-lg hover:bg-[#F7F5EF]/[.08] transition-colors"
          >
            <PowerIcon size={15} color="rgba(247,245,239,0.4)" />
          </button>
        </div>
      )}
    </aside>
  );
}
