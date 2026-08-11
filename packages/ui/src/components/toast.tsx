'use client';

import React, { useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn } from '../lib/utils';
import { toastVariants } from '../motion/toast';
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react';

export interface ToastProps {
  id: string;
  type?: 'success' | 'error' | 'info';
  title: string;
  description?: string;
  durationMs?: number;
  onDismiss: (id: string) => void;
}

export const Toast: React.FC<ToastProps> = ({
  id,
  type = 'info',
  title,
  description,
  durationMs = 4000,
  onDismiss,
}) => {
  useEffect(() => {
    const timer = setTimeout(() => {
      onDismiss(id);
    }, durationMs);
    return () => clearTimeout(timer);
  }, [id, durationMs, onDismiss]);

  const icons = {
    success: <CheckCircle2 className="w-5 h-5 text-emerald-400" />,
    error: <AlertCircle className="w-5 h-5 text-red-400" />,
    info: <Info className="w-5 h-5 text-blue-400" />,
  };

  return (
    <motion.div
      className={cn(
        'relative overflow-hidden flex items-start gap-3 w-80 max-w-full p-4 rounded-xl border border-white/10 bg-slate-900/90 backdrop-blur-md shadow-2xl text-foreground',
      )}
      variants={toastVariants}
      initial="initial"
      animate="animate"
      exit="exit"
    >
      {icons[type]}
      <div className="flex-1 space-y-0.5">
        <h4 className="text-sm font-semibold">{title}</h4>
        {description && <p className="text-xs text-muted-foreground">{description}</p>}
      </div>
      <button
        onClick={() => onDismiss(id)}
        className="text-muted-foreground hover:text-foreground transition-colors p-1 cursor-pointer"
        aria-label="Dismiss"
      >
        <X className="w-4 h-4" />
      </button>

      {/* Timer line */}
      <motion.div
        className="absolute bottom-0 left-0 h-0.5 bg-blue-500"
        initial={{ width: '100%' }}
        animate={{ width: '0%' }}
        transition={{ duration: durationMs / 1000, ease: 'linear' }}
      />
    </motion.div>
  );
};
