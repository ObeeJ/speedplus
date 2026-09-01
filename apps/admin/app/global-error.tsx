'use client';

import { ErrorFallback } from '@fourdat/ui';
import './globals.css';

// Next.js renders this in place of the ROOT layout when an error escapes
// even app/error.tsx (e.g. a crash in layout.tsx itself). It must supply
// its own <html>/<body> since the real root layout is not mounted.
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <html lang="en">
      <body>
        <ErrorFallback error={error} reset={reset} title="the app" />
      </body>
    </html>
  );
}
