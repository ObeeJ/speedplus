'use client';

import React, { useState, useRef, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { cn } from '../lib/utils';
import { shakeVariants } from '../motion/input';

export interface OTPInputProps {
  length?: number;
  value?: string;
  onChange?: (code: string) => void;
  onComplete?: (code: string) => void;
  error?: string;
  disabled?: boolean;
  autoFocus?: boolean;
  resendSeconds?: number;
  onResend?: () => void;
  className?: string;
}

export const OTPInput: React.FC<OTPInputProps> = ({
  length = 6,
  value: externalValue,
  onChange,
  onComplete,
  error,
  disabled = false,
  autoFocus = true,
  resendSeconds = 60,
  onResend,
  className,
}) => {
  const [internalValue, setInternalValue] = useState<string[]>(Array(length).fill(''));
  const [focusedIndex, setFocusedIndex] = useState<number | null>(autoFocus ? 0 : null);
  const [countdown, setCountdown] = useState<number>(resendSeconds);
  const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

  // Sync external value
  useEffect(() => {
    if (externalValue !== undefined) {
      const arr = externalValue.split('').slice(0, length);
      while (arr.length < length) arr.push('');
      setInternalValue(arr);
    }
  }, [externalValue, length]);

  // Resend countdown timer
  useEffect(() => {
    if (countdown <= 0) return;
    const timer = setInterval(() => {
      setCountdown((prev) => prev - 1);
    }, 1000);
    return () => clearInterval(timer);
  }, [countdown]);

  const updateCode = useCallback(
    (newArr: string[]) => {
      setInternalValue(newArr);
      const codeStr = newArr.join('');
      onChange?.(codeStr);
      if (codeStr.length === length && newArr.every((char) => char !== '')) {
        onComplete?.(codeStr);
      }
    },
    [length, onChange, onComplete],
  );

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>, idx: number) => {
    const val = e.target.value;
    if (!val) return;

    const char = val.charAt(val.length - 1);
    if (!/^\d$/.test(char)) return; // Digits only

    const newArr = [...internalValue];
    newArr[idx] = char;
    updateCode(newArr);

    // Focus next box
    if (idx < length - 1) {
      inputRefs.current[idx + 1]?.focus();
      setFocusedIndex(idx + 1);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>, idx: number) => {
    if (e.key === 'Backspace') {
      e.preventDefault();
      const newArr = [...internalValue];

      if (newArr[idx]) {
        newArr[idx] = '';
        updateCode(newArr);
      } else if (idx > 0) {
        newArr[idx - 1] = '';
        updateCode(newArr);
        inputRefs.current[idx - 1]?.focus();
        setFocusedIndex(idx - 1);
      }
    } else if (e.key === 'ArrowLeft' && idx > 0) {
      inputRefs.current[idx - 1]?.focus();
      setFocusedIndex(idx - 1);
    } else if (e.key === 'ArrowRight' && idx < length - 1) {
      inputRefs.current[idx + 1]?.focus();
      setFocusedIndex(idx + 1);
    }
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLDivElement>) => {
    e.preventDefault();
    const pastedData = e.clipboardData.getData('text').trim();
    const digits = pastedData.replace(/\D/g, '').split('').slice(0, length);
    if (digits.length === 0) return;

    const newArr = [...internalValue];
    digits.forEach((digit, i) => {
      newArr[i] = digit;
    });
    updateCode(newArr);

    const nextFocus = Math.min(digits.length, length - 1);
    inputRefs.current[nextFocus]?.focus();
    setFocusedIndex(nextFocus);
  };

  const handleResendClick = () => {
    if (countdown > 0 || !onResend) return;
    setCountdown(resendSeconds);
    onResend();
  };

  return (
    <div className={cn('flex flex-col gap-3 items-center', className)}>
      <motion.div
        className="flex gap-2 sm:gap-3"
        onPaste={handlePaste}
        animate={error ? 'shake' : 'idle'}
        variants={shakeVariants}
      >
        {Array.from({ length }).map((_, idx) => {
          const char = internalValue[idx] || '';
          const isFocused = focusedIndex === idx;

          return (
            <motion.div
              key={idx}
              className="relative w-11 h-13 sm:w-12 sm:h-14"
              whileHover={{ scale: disabled ? 1 : 1.02 }}
              whileTap={{ scale: disabled ? 1 : 0.98 }}
            >
              <input
                ref={(el) => {
                  inputRefs.current[idx] = el;
                }}
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                autoComplete="one-time-code"
                maxLength={1}
                value={char}
                disabled={disabled}
                onFocus={() => setFocusedIndex(idx)}
                onBlur={() => setFocusedIndex(null)}
                onChange={(e) => handleInputChange(e, idx)}
                onKeyDown={(e) => handleKeyDown(e, idx)}
                className={cn(
                  'w-full h-full text-center text-xl font-bold font-mono outline-none rounded-xl',
                  'transition-all duration-150 select-none',
                  'bg-white border text-ink',
                  error
                    ? 'border-red-400 shadow-[0_0_0_2px_rgba(239,68,68,0.15)]'
                    : isFocused
                    ? 'border-emerald shadow-[0_0_0_3px_rgba(10,61,44,0.12)]'
                    : 'border-line hover:border-mid/40',
                  disabled && 'opacity-40 cursor-not-allowed',
                )}
                aria-label={`Digit ${idx + 1}`}
              />
            </motion.div>
          );
        })}
      </motion.div>

      {/* Validation error */}
      <AnimatePresence mode="wait">
        {error && (
          <motion.p
            key="error"
            className="text-xs font-medium text-red-600"
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
          >
            {error}
          </motion.p>
        )}
      </AnimatePresence>

      {/* Resend Countdown */}
      {onResend && (
        <div className="text-xs text-mid mt-1 flex items-center gap-1.5">
          <span>Didn't receive code?</span>
          {countdown > 0 ? (
            <span className="font-mono text-emerald font-medium">Resend in {countdown}s</span>
          ) : (
            <motion.button
              type="button"
              onClick={handleResendClick}
              className="text-emerald hover:text-emerald-600 font-semibold underline underline-offset-2 cursor-pointer"
              whileHover={{ scale: 1.04 }}
              whileTap={{ scale: 0.96 }}
            >
              Resend Code
            </motion.button>
          )}
        </div>
      )}
    </div>
  );
};
