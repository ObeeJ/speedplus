'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { useAdminAuthStore } from '@/lib/store/auth.store';

const NAV: { href: string; label: string }[] = [
  { href: '/kyc',                         label: 'KYC Queue' },
  { href: '/merchants',                   label: 'Merchants' },
  { href: '/drivers',                     label: 'Drivers' },
  { href: '/orders',                      label: 'All Orders' },
  { href: '/orders/package',              label: '📦 Package Orders' },
  { href: '/disputes',                    label: 'Disputes' },
  { href: '/settings/cancellation-rules', label: 'Cancel Rules' },
  { href: '/settings/fees',               label: 'Fees' },
  { href: '/ledger',                      label: 'Ledger' },
];

export function AdminNav() {
  const path = usePathname();
  const router = useRouter();
  const clearAuth = useAdminAuthStore((s) => s.clearAuth);
  const user = useAdminAuthStore((s) => s.user);

  function handleSignOut() {
    clearAuth();
    router.replace('/login');
  }

  // Hide nav on login page
  if (path.startsWith('/login')) return null;

  return (
    <aside className="w-[220px] flex-none min-h-screen flex flex-col gap-6 p-6" style={{ background: '#08301F' }}>
      <span className="font-display font-bold text-xl text-sand tracking-tight">
        speed<span className="text-lime">+</span>{' '}
        <span className="font-medium text-[11px] text-sand/55">OPS</span>
      </span>

      <nav className="flex flex-col gap-1 flex-1">
        {NAV.map((item) => {
          const active = item.href === '/orders'
            ? path === '/orders'
            : path.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`rounded-xl px-4 py-2.5 text-[13.5px] font-medium transition-colors ${
                active
                  ? 'bg-lime/[.14] text-lime font-semibold'
                  : 'text-sand/70 hover:bg-sand/[.08] hover:text-sand'
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>

      {user && (
        <div className="flex flex-col gap-2 border-t border-sand/10 pt-4">
          <span className="text-[11px] text-sand/50 truncate">{user.firstName} {user.lastName}</span>
          <button
            onClick={handleSignOut}
            className="text-[12px] font-semibold text-sand/60 hover:text-sand text-left transition-colors"
          >
            Sign out
          </button>
        </div>
      )}
    </aside>
  );
}
