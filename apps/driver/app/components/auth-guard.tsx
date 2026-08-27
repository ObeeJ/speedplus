'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useDriverAuthStore } from '@/lib/store/auth.store';

const PUBLIC_PATHS = ['/login', '/forgot-password'];

function isPublicPath(pathname: string): boolean {
  return PUBLIC_PATHS.some((p) => pathname.startsWith(p));
}

export function DriverAuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const isAuthenticated = useDriverAuthStore((s) => s.isAuthenticated);
  const isPublic = isPublicPath(pathname);

  useEffect(() => {
    if (!isAuthenticated && !isPublic) {
      router.replace('/login');
    }
  }, [isAuthenticated, isPublic, router]);

  if (!isAuthenticated && !isPublic) return null;
  return <>{children}</>;
}
