'use client';

import React, { useEffect } from 'react';
import { motion } from 'framer-motion';
import { Button } from './button';
import { AlertCircleIcon } from '../icons';
import { cn } from '../lib/utils';

export interface ErrorFallbackProps {
  /** The error caught by the nearest Next.js `error.tsx` boundary. */
  error: Error & { digest?: string };
  /** Provided by Next.js — re-renders the segment to attempt recovery. */
  reset: () => void;
  /** Segment name shown in the heading, e.g. "checkout" or "dashboard". */
  title?: string;
  /** Override the default supporting copy. */
  description?: string;
  /** Extra content rendered under the actions, e.g. a "Contact support" link. */
  children?: React.ReactNode;
  /** Render as a full-screen page (root/global boundaries) vs. an inline card
   *  that sits inside an existing app shell (dashboard/segment boundaries). */
  fullScreen?: boolean;
  className?: string;
}

/**
 * Shared, on-brand fallback UI for Next.js App Router `error.tsx` boundaries.
 *
 * Usage in an `app/**\/error.tsx`:
 *
 * ```tsx
 * 'use client';
 * import { ErrorFallback } from '@fourdat/ui';
 *
 * export default function Error({ error, reset }: { error: Error; reset: () => void }) {
 *   return <ErrorFallback error={error} reset={reset} title="orders" />;
 * }
 * ```
 */
export const ErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  reset,
  title,
  description,
  children,
  fullScreen = true,
  className,
}) => {
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.error('[error-boundary]', title ?? '', error);
  }, [error, title]);

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center text-center px-6 py-16',
        fullScreen && 'min-h-[60vh] w-full',
        className,
      )}
    >
      <motion.div
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25 }}
        className="flex flex-col items-center max-w-sm"
      >
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-[#0A3D2C] mb-5">
          <AlertCircleIcon color="#C6F24E" />
        </div>

        <h2 className="font-display font-bold text-[20px] text-[#1A1A2E] mb-2">
          {title ? `Something went wrong with ${title}` : 'Something went wrong'}
        </h2>

        <p className="text-[14px] text-mid mb-1">
          {description ?? "We hit a snag loading this page. It's on us — try again in a moment."}
        </p>

        {process.env.NODE_ENV === 'development' && error?.message && (
          <p className="mt-2 mb-1 text-[12px] font-mono text-red-600 bg-red-50 rounded-lg px-3 py-2 break-words max-w-full">
            {error.message}
            {error.digest && <span className="block text-red-400 mt-1">digest: {error.digest}</span>}
          </p>
        )}

        <div className="flex items-center gap-3 mt-6">
          <Button variant="primary" size="md" onClick={() => reset()}>
            Try again
          </Button>
          <Button variant="outline" size="md" onClick={() => { window.location.href = '/'; }}>
            Go home
          </Button>
        </div>

        {children && <div className="mt-4">{children}</div>}
      </motion.div>
    </div>
  );
};
