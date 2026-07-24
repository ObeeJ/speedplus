'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

const NAV: { href: string; label: string }[] = [
  { href: '/kyc',                          label: 'KYC Queue' },
  { href: '/merchants',                    label: 'Merchants' },
  { href: '/drivers',                      label: 'Drivers' },
  { href: '/orders',                       label: 'Orders' },
  { href: '/disputes',                     label: 'Disputes' },
  { href: '/settings/cancellation-rules',  label: 'Cancel Rules' },
  { href: '/ledger',                       label: 'Ledger' },
];

export function AdminNav() {
  const path = usePathname();
  return (
    <aside className="w-[220px] flex-none min-h-screen flex flex-col gap-6 p-6" style={{ background: '#08301F' }}>
      <span className="font-display font-bold text-xl text-sand tracking-tight">
        speed<span className="text-lime">+</span>{' '}
        <span className="font-medium text-[11px] text-sand/55">OPS</span>
      </span>
      <nav className="flex flex-col gap-1">
        {NAV.map((item) => {
          const active = path.startsWith(item.href);
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
    </aside>
  );
}
