'use client';

import React, { type ReactNode } from 'react';
import { motion } from 'framer-motion';
import { cn } from '../lib/utils';
import { AlertCircle, CheckCircle2, Info, AlertTriangle } from 'lucide-react';

export interface AlertProps {
  variant?: 'info' | 'success' | 'warning' | 'error';
  title?: string;
  children: ReactNode;
  icon?: ReactNode;
  className?: string;
}

const variantStyles = {
  info: 'bg-blue-500/10 border-blue-500/20 text-blue-400',
  success: 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400',
  warning: 'bg-amber-500/10 border-amber-500/20 text-amber-400',
  error: 'bg-red-500/10 border-red-500/20 text-red-400',
};

const defaultIcons = {
  info: <Info className="w-5 h-5 flex-shrink-0" />,
  success: <CheckCircle2 className="w-5 h-5 flex-shrink-0" />,
  warning: <AlertTriangle className="w-5 h-5 flex-shrink-0" />,
  error: <AlertCircle className="w-5 h-5 flex-shrink-0" />,
};

export const Alert: React.FC<AlertProps> = ({
  variant = 'info',
  title,
  children,
  icon,
  className,
}) => {
  return (
    <motion.div
      className={cn(
        'flex items-start gap-3 rounded-xl border p-4 text-sm backdrop-blur-sm',
        variantStyles[variant],
        className,
      )}
      initial={{ opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -6 }}
    >
      {icon ?? defaultIcons[variant]}
      <div className="flex-1 space-y-0.5">
        {title && <h4 className="font-semibold text-foreground">{title}</h4>}
        <div className="text-muted-foreground">{children}</div>
      </div>
    </motion.div>
  );
};
