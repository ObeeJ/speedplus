'use client';

import { ErrorFallback } from '@fourdat/ui';

export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <ErrorFallback
      error={error}
      reset={reset}
      title="payment"
      description="Your payment didn't go through cleanly. No charge was made — try again, or check your wallet before retrying."
    />
  );
}
