'use client';

import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { useState } from 'react';
import { QueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';

// Auth-guarding lives once at layout.tsx (MerchantAuthGuard wraps Providers'
// children there) — do not also guard here, or every route redirect check
// runs twice.
export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: { queries: { staleTime: 60_000, retry: (n, e: any) => e?.code === 'UNAUTHORIZED' ? false : n < 2 } },
  }));
  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
