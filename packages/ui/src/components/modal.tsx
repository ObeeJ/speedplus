'use client';

import React, { type ReactNode, useEffect, useId, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn } from '../lib/utils';
import { backdropVariants, dialogContentVariants } from '../motion/dialog';
import { X } from 'lucide-react';

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: ReactNode;
  className?: string;
}

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  description,
  children,
  className,
}) => {
  const titleId = useId();
  const descId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);

  // Escape dismissal + focus management.
  //
  // role="dialog" and aria-modal tell a screen reader this content is modal,
  // but they do not MOVE focus. Without the rest of this effect a keyboard user
  // stays on the trigger behind the overlay and Tab walks them through the page
  // underneath — the dialog is announced but unreachable. So on open we move
  // focus inside, cycle Tab within the dialog, and put focus back where it came
  // from on close (WCAG 2.4.3). Escape and the close button both still exit, so
  // this is a focus loop, never a keyboard trap (2.1.2).
  useEffect(() => {
    if (!isOpen) return;

    const previouslyFocused = document.activeElement as HTMLElement | null;

    const focusable = () =>
      Array.from(
        dialogRef.current?.querySelectorAll<HTMLElement>(
          'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
        ) ?? [],
      ).filter((el) => el.offsetParent !== null);

    // Focus the first control, falling back to the dialog itself so focus is
    // never left behind the overlay even when the body has no focusable child.
    const initial = focusable()[0] ?? dialogRef.current;
    initial?.focus();

    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab') return;

      const items = focusable();
      const first = items[0];
      const last = items[items.length - 1];
      if (!first || !last) {
        // Nothing focusable inside: swallow Tab so focus cannot escape to the
        // page behind the overlay.
        e.preventDefault();
        return;
      }
      const active = document.activeElement;

      if (e.shiftKey && (active === first || active === dialogRef.current)) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handler);

    // Lock background scroll so the page behind cannot be moved under the overlay.
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handler);
      document.body.style.overflow = previousOverflow;
      previouslyFocused?.focus();
    };
  }, [isOpen, onClose]);

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          {/* Backdrop */}
          <motion.div
            className="fixed inset-0 bg-black/70 backdrop-blur-sm"
            variants={backdropVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
            onClick={onClose}
            aria-hidden="true"
          />

          {/* Dialog */}
          <motion.div
            ref={dialogRef}
            role="dialog"
            aria-modal="true"
            // tabIndex -1 makes the dialog itself programmatically focusable as
            // the fallback target when it contains no focusable control.
            tabIndex={-1}
            aria-labelledby={title ? titleId : undefined}
            aria-describedby={description ? descId : undefined}
            className={cn(
              'relative z-10 w-full max-w-lg overflow-hidden rounded-2xl border border-line bg-white p-6 shadow-xl text-ink',
              className,
            )}
            variants={dialogContentVariants}
            initial="hidden"
            animate="visible"
            exit="exit"
          >
            {/* Header */}
            <div className="flex items-start justify-between gap-4 mb-4">
              <div>
                {title && <h3 id={titleId} className="text-lg font-bold text-ink">{title}</h3>}
                {description && <p id={descId} className="text-sm text-mid">{description}</p>}
              </div>
              <button
                onClick={onClose}
                className="rounded-lg p-1.5 text-mid hover:bg-sand hover:text-ink transition-colors cursor-pointer"
                aria-label="Close modal"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Content */}
            <div>{children}</div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
};
