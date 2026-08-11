'use client';

import React, { type ReactNode } from 'react';
import { motion } from 'framer-motion';
import { cn } from '../lib/utils';
import { cardHover } from '../motion/card';

export interface HeroCardProps {
  title: string;
  description?: string;
  badge?: string;
  action?: ReactNode;
  icon?: ReactNode;
  className?: string;
  children?: ReactNode;
}

export const HeroCard: React.FC<HeroCardProps> = ({
  title,
  description,
  badge,
  action,
  icon,
  className,
  children,
}) => {
  return (
    <motion.div
      className={cn(
        'relative overflow-hidden rounded-2xl border border-white/10 bg-gradient-to-b from-white/10 to-white/5 p-6 sm:p-8 backdrop-blur-md',
        className,
      )}
      variants={cardHover}
      initial="rest"
      whileHover="hover"
    >
      {/* Background ambient glow */}
      <div className="absolute -right-12 -top-12 h-40 w-40 rounded-full bg-blue-600/20 blur-3xl pointer-events-none" />

      <div className="relative z-10 flex flex-col sm:flex-row sm:items-center justify-between gap-6">
        <div className="flex items-start gap-4">
          {icon && (
            <div className="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-xl bg-blue-500/10 border border-blue-500/20 text-blue-400">
              {icon}
            </div>
          )}
          <div className="space-y-1">
            {badge && (
              <span className="inline-block rounded-full bg-blue-500/10 border border-blue-500/20 px-2.5 py-0.5 text-xs font-semibold text-blue-400">
                {badge}
              </span>
            )}
            <h2 className="text-xl sm:text-2xl font-bold tracking-tight text-white">{title}</h2>
            {description && <p className="text-sm text-muted-foreground">{description}</p>}
          </div>
        </div>

        {action && <div className="flex-shrink-0">{action}</div>}
      </div>

      {children && <div className="relative z-10 mt-6 pt-6 border-t border-white/5">{children}</div>}
    </motion.div>
  );
};
