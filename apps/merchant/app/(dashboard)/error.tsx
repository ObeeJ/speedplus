'use client';

import { ErrorFallback } from '@fourdat/ui';

export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  // fullScreen=false: renders inside the persisting dashboard shell (sidebar/nav
  // stay mounted since this boundary only replaces DashboardLayout's `children`).
  return <ErrorFallback error={error} reset={reset} title="the dashboard" fullScreen={false} />;
}
