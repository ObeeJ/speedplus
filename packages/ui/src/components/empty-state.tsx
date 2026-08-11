'use client';

import React, { type ReactNode } from 'react';
import { motion } from 'framer-motion';
import { cn } from '../lib/utils';

export interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon,
  title,
  description,
  action,
  className,
}) => {
  return (
    <motion.div
      className={cn(
        'flex flex-col items-center justify-center text-center p-8 sm:p-12 rounded-2xl border border-dashed border-white/10 bg-white/[0.02]',
        className,
      )}
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.25 }}
    >
      {icon && (
        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-white/5 border border-white/10 text-muted-foreground mb-4">
          {icon}
        </div>
      )}
      <h3 className="text-lg font-semibold text-foreground mb-1">{title}</h3>
      {description && <p className="text-sm text-muted-foreground max-w-sm mb-6">{description}</p>}
      {action && <div>{action}</div>}
    </motion.div>
  );
};
