'use client';

import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/lib/store/auth.store';

// '/' must be an EXACT match, not a prefix — startsWith('/') matches every
// route in the app and would disable the guard entirely.
const PUBLIC_EXACT = ['/'];
const PUBLIC_PREFIX = ['/login', '/register', '/forgot-password', '/pay', '/merchants', '/products'];

function isPublicPath(pathname: string): boolean {
  return PUBLIC_EXACT.includes(pathname) || PUBLIC_PREFIX.some((p) => pathname.startsWith(p));
}

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isPublic = isPublicPath(pathname);

  useEffect(() => {
    if (!isAuthenticated && !isPublic) {
      router.replace(`/login?next=${encodeURIComponent(pathname)}`);
    }
  }, [isAuthenticated, isPublic, pathname, router]);

  // Render public paths immediately; protected paths render only when authed.
  if (!isAuthenticated && !isPublic) {
    return null;
  }

  return <>{children}</>;
}
