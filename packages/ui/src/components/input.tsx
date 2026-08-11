'use client';

import {
  forwardRef,
  type InputHTMLAttributes,
  type ReactNode,
  useState,
  useId,
} from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn } from '../lib/utils';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  success?: boolean;
  suffix?: ReactNode;
}

// ── Focus ring states ─────────────────────────────────────────────────────────

const ring = {
  idle:    '0 0 0 1.5px rgba(0,0,0,0.08), 0 1px 4px rgba(0,0,0,0.06)',
  focus:   '0 0 0 2px rgba(10,61,44,0.55), 0 0 0 5px rgba(10,61,44,0.07), 0 1px 4px rgba(0,0,0,0.08)',
  success: '0 0 0 2px rgba(22,163,74,0.50), 0 0 0 5px rgba(22,163,74,0.07)',
  error:   '0 0 0 2px rgba(220,38,38,0.45), 0 0 0 5px rgba(220,38,38,0.07)',
} as const;

// ── Component ─────────────────────────────────────────────────────────────────

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      className,
      label,
      error,
      hint,
      success,
      suffix,
      id: externalId,
      placeholder,
      onFocus,
      onBlur,
      value,
      defaultValue,
      ...props
    },
    ref,
  ) => {
    const autoId  = useId();
    const id      = externalId ?? autoId;
    const [focused, setFocused] = useState(false);

    // Floating label rises when focused OR has a value
    const hasValue  = value !== undefined ? String(value).length > 0 : false;
    const isFloated = focused || hasValue;
    const ringState = error ? 'error' : success ? 'success' : focused ? 'focus' : 'idle';

    return (
      <motion.div
        className="flex flex-col gap-1.5"
        animate={error ? 'shake' : 'idle'}
        variants={{
          idle:  { x: 0 },
          shake: { x: [0, -5, 5, -3, 3, -1, 1, 0], transition: { duration: 0.36 } },
        }}
      >
        <div className={cn('relative', label ? 'h-[58px]' : 'h-[52px]')}>
          {/* Input surface — the card */}
          <motion.div
            className="absolute inset-0 rounded-[14px] pointer-events-none"
            style={{ background: error ? 'rgba(254,242,242,0.7)' : 'rgba(255,255,255,0.92)' }}
            initial={{ boxShadow: ring.idle }}
            animate={{ boxShadow: ring[ringState] }}
            transition={{ type: 'spring', stiffness: 380, damping: 30 }}
          />

          {/* Floating label */}
          {label && (
            <motion.label
              htmlFor={id}
              className="absolute left-4 pointer-events-none select-none font-semibold"
              animate={{
                top:      isFloated ? '8px'    : '50%',
                y:        isFloated ? '0%'     : '-50%',
                fontSize: isFloated ? '10px'   : '13.5px',
                color:    error
                  ? '#DC2626'
                  : focused
                  ? '#0A3D2C'
                  : isFloated
                  ? '#63636E'
                  : 'rgba(99,99,110,0.55)',
                letterSpacing: isFloated ? '0.04em' : '0',
              }}
              transition={{ type: 'spring', stiffness: 420, damping: 32 }}
            >
              {label}
            </motion.label>
          )}

          <input
            ref={ref}
            id={id}
            value={value}
            defaultValue={defaultValue}
            placeholder={label ? (focused ? placeholder : undefined) : placeholder}
            className={cn(
              'relative w-full bg-transparent outline-none text-ink',
              'rounded-[14px] text-[14px]',
              'disabled:cursor-not-allowed disabled:opacity-40',
              // When label is present, shift text down to make room for floated label
              label ? 'h-[58px] px-4 pt-5 pb-2' : 'h-[52px] px-4',
              suffix ? 'pr-12' : '',
              'placeholder:text-mid/35',
              className,
            )}
            aria-invalid={error ? 'true' : undefined}
            aria-describedby={
              error ? `${id}-error` : hint ? `${id}-hint` : undefined
            }
            onFocus={(e) => { setFocused(true);  onFocus?.(e); }}
            onBlur={(e)  => { setFocused(false); onBlur?.(e);  }}
            {...props}
          />

          {/* Suffix slot (eye icon, etc.) */}
          {suffix && (
            <div className="absolute inset-y-0 right-0 flex items-center pr-3.5">
              {suffix}
            </div>
          )}

          {/* Success checkmark */}
          <AnimatePresence>
            {success && !suffix && (
              <motion.div
                className="absolute inset-y-0 right-0 flex items-center pr-3.5 pointer-events-none"
                initial={{ opacity: 0, scale: 0.5 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.5 }}
                transition={{ type: 'spring', stiffness: 500, damping: 26 }}
              >
                <div className="w-5 h-5 rounded-full flex items-center justify-center" style={{ background: 'rgba(22,163,74,0.12)' }}>
                  <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
                    <path d="M2 5l2 2 4-4" stroke="#16A34A" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Error / hint messages */}
        <AnimatePresence mode="wait">
          {error && (
            <motion.p
              key="error"
              id={`${id}-error`}
              className="text-[11px] font-medium text-red-600 pl-1"
              role="alert"
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -4 }}
              transition={{ duration: 0.14 }}
            >
              {error}
            </motion.p>
          )}
          {hint && !error && (
            <motion.p
              key="hint"
              id={`${id}-hint`}
              className="text-[11px] text-mid pl-1"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
            >
              {hint}
            </motion.p>
          )}
        </AnimatePresence>
      </motion.div>
    );
  },
);
Input.displayName = 'Input';
