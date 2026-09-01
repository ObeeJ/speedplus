'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { merchantApi, authApi } from '@fourdat/api-client';
import {
  FourdatLogo, Avatar,
  DashboardIcon, ReceiptIcon, PillIcon, BoxIcon,
  WalletIcon, ShieldCheckIcon, FeeIcon, GasIcon,
  PowerIcon, type DuotoneIconProps,
} from '@fourdat/ui';
import { useMerchantAuthStore } from '@/lib/store/auth.store';
import { cn } from '@fourdat/ui';

type NavItem = {
  href: string;
  label: string;
  Icon: (p: DuotoneIconProps) => React.JSX.Element;
  badge?: 'orders' | 'rx';
};

const NAV: NavItem[] = [
  { href: '/',               label: 'Dashboard',     Icon: DashboardIcon },
  { href: '/orders',         label: 'Orders',        Icon: ReceiptIcon,    badge: 'orders' },
  { href: '/prescriptions',  label: 'Prescriptions', Icon: PillIcon,       badge: 'rx' },
  { href: '/products',       label: 'Products',      Icon: BoxIcon },
  { href: '/earnings',       label: 'Earnings',      Icon: WalletIcon },
  { href: '/verification',   label: 'Verification',  Icon: ShieldCheckIcon },
  { href: '/payments',       label: 'Payments',      Icon: FeeIcon as unknown as (p: DuotoneIconProps) => React.JSX.Element },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { user, merchant, clearAuth } = useMerchantAuthStore();

  const profileQuery = useQuery({
    queryKey: ['merchant-profile'],
    queryFn: () => merchantApi.getProfile(),
    initialData: merchant ?? undefined,
  });

  const ordersQuery = useQuery({
    queryKey: ['merchant-orders'],
    queryFn: () => merchantApi.listOrders(),
    refetchInterval: 15_000,
  });

  const rxQuery = useQuery({
    queryKey: ['merchant-prescriptions'],
    queryFn: () => merchantApi.listPrescriptions('pending'),
    refetchInterval: 20_000,
  });

  const newOrderCount = (ordersQuery.data?.orders ?? []).filter(
    (o) => o.status === 'pending' || o.status === 'confirmed',
  ).length;
  const pendingRxCount = rxQuery.data?.prescriptions.length ?? 0;

  const badgeCount = (key: string) => {
    if (key === 'orders') return newOrderCount;
    if (key === 'rx') return pendingRxCount;
    return 0;
  };

  const initials = `${user?.firstName?.charAt(0) ?? 'M'}${user?.lastName?.charAt(0) ?? ''}`;

  return (
    <div className="min-h-screen flex bg-sand">
      {/* ── Sidebar ─────────────────────────────────────────────────────── */}
      <aside className="hidden lg:flex w-[240px] flex-none flex-col bg-emerald min-h-screen">
        {/* Logo */}
        <div className="px-6 pt-6 pb-4">
          <FourdatLogo variant="full" theme="dark" size="md" />
          <p className="mt-1.5 text-[11px] text-sand/50 font-medium truncate">
            {profileQuery.data?.businessName ?? '…'}
          </p>
        </div>

        {/* Open/closed toggle */}
        {profileQuery.data && (
          <div className="px-4 pb-3">
            <div
              className="flex items-center gap-2.5 rounded-xl px-3.5 py-2.5"
              style={{ background: profileQuery.data.isOpen ? 'rgba(198,242,78,.14)' : 'rgba(245,245,240,.08)' }}
            >
              <span className={cn('w-2 h-2 rounded-full flex-none', profileQuery.data.isOpen ? 'bg-lime animate-pulse' : 'bg-sand/40')} />
              <span className={cn('text-xs font-semibold', profileQuery.data.isOpen ? 'text-lime' : 'text-sand/60')}>
                {profileQuery.data.isOpen ? 'Open for orders' : 'Closed'}
              </span>
            </div>
          </div>
        )}

        {/* Nav */}
        <nav className="flex-1 flex flex-col gap-0.5 px-3 pb-4">
          {NAV.map(({ href, label, Icon, badge }) => {
            const active = pathname === href || (href !== '/' && pathname.startsWith(href));
            const count = badge ? badgeCount(badge) : 0;
            return (
              <Link
                key={href}
                href={href}
                className={cn(
                  'flex items-center gap-2.5 rounded-xl px-3.5 py-2.5 text-[13px] font-medium transition-colors',
                  active
                    ? 'bg-lime/[.14] text-lime font-semibold'
                    : 'text-sand/65 hover:bg-sand/[.08] hover:text-sand',
                )}
              >
                <Icon
                  active={active}
                  color={active ? '#C6F24E' : 'rgba(247,245,239,0.5)'}
                  accent={active ? '#AEE032' : 'rgba(247,245,239,0.28)'}
                />
                <span className="flex-1">{label}</span>
                {count > 0 && (
                  <span className="text-[10px] font-bold rounded-full px-2 py-0.5 bg-lime text-emerald">
                    {count}
                  </span>
                )}
              </Link>
            );
          })}
          {profileQuery.data?.vertical === 'gas' && (
            <Link
              href="/gas"
              className={cn(
                'flex items-center gap-2.5 rounded-xl px-3.5 py-2.5 text-[13px] font-medium transition-colors',
                pathname === '/gas'
                  ? 'bg-lime/[.14] text-lime font-semibold'
                  : 'text-sand/65 hover:bg-sand/[.08] hover:text-sand',
              )}
            >
              <GasIcon
                active={pathname === '/gas'}
                color={pathname === '/gas' ? '#C6F24E' : 'rgba(247,245,239,0.5)'}
                accent="rgba(247,245,239,0.28)"
              />
              Gas ops
            </Link>
          )}
        </nav>

        {/* User footer */}
        <div className="px-4 pb-5">
          <div className="flex items-center gap-2.5 bg-sand/[.06] rounded-[13px] p-2.5">
            <Avatar initials={initials} size="sm" variant="emerald" />
            <div className="flex flex-col flex-1 min-w-0">
              <span className="text-[12px] font-semibold text-sand truncate">
                {user?.firstName ?? 'Merchant'}
              </span>
              <span className="text-[10px] text-sand/50 capitalize">
                {profileQuery.data?.kycStatus === 'approved'
                  ? 'Verified ✓'
                  : (profileQuery.data?.kycStatus ?? '—').replace('_', ' ')}
              </span>
            </div>
            <button
              onClick={async () => { await authApi.logout().catch(() => {}); clearAuth(); }}
              className="text-sand/40 hover:text-sand/70 transition-colors"
              aria-label="Sign out"
            >
              <PowerIcon color="currentColor" />
            </button>
          </div>
        </div>
      </aside>

      {/* ── Main ────────────────────────────────────────────────────────── */}
      <div className="flex-1 min-w-0 flex flex-col">
        {/* Top bar — mobile only */}
        <header className="lg:hidden flex h-14 items-center justify-between border-b border-line bg-white px-4">
          <FourdatLogo variant="full" theme="light" size="md" />
          <Avatar initials={initials} size="sm" />
        </header>

        {/* Page content */}
        <main className="flex-1 px-6 py-7 lg:px-10 lg:py-8 flex flex-col gap-6">
          {children}
        </main>
      </div>
    </div>
  );
}
