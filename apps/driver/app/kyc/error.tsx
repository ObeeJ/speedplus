'use client';

import { ErrorFallback } from '@fourdat/ui';

export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <ErrorFallback
      error={error}
      reset={reset}
      title="verification"
      description="We couldn't process your documents. Your progress is saved — try again."
    />
  );
}
