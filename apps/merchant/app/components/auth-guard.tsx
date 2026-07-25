'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

const PUBLIC_PATHS = ['/login'];

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const isAuthenticated = useMerchantAuthStore((s) => s.isAuthenticated);

  useEffect(() => {
    if (!isAuthenticated && !PUBLIC_PATHS.some((p) => pathname.startsWith(p))) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
    }
  }, [isAuthenticated, pathname, router]);

  if (!isAuthenticated && !PUBLIC_PATHS.some((p) => pathname.startsWith(p))) {
    return null;
  }

  return <>{children}</>;
}
