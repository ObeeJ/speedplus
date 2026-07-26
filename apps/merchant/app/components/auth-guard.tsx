'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useMerchantAuthStore } from '@/lib/store/auth.store';

export function MerchantAuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const isAuthenticated = useMerchantAuthStore((s) => s.isAuthenticated);

  useEffect(() => {
    if (!isAuthenticated && !pathname.startsWith('/login')) {
      router.replace('/login');
    }
  }, [isAuthenticated, pathname, router]);

  if (!isAuthenticated && !pathname.startsWith('/login')) return null;
  return <>{children}</>;
}
