import { forwardRef } from 'react';
import { cn } from '../lib/utils';
import type { HTMLAttributes } from 'react';

export interface ListCardProps extends HTMLAttributes<HTMLDivElement> {
  /** Remove default padding — useful when the card contains a flush table/list */
  noPadding?: boolean;
}

export const ListCard = forwardRef<HTMLDivElement, ListCardProps>(
  ({ className, noPadding, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'bg-white rounded-2xl border border-line',
        !noPadding && 'p-5',
        className,
      )}
      {...props}
    />
  ),
);
ListCard.displayName = 'ListCard';
