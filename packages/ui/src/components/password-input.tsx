'use client';

import React, { useState, forwardRef } from 'react';
import { Input, type InputProps } from './input';
import { Eye, EyeOff } from 'lucide-react';
import { motion } from 'framer-motion';

export interface PasswordInputProps extends Omit<InputProps, 'type' | 'suffix'> {
  showStrength?: boolean;
}

export const PasswordInput = forwardRef<HTMLInputElement, PasswordInputProps>(
  ({ showStrength = false, value, onChange, ...props }, ref) => {
    const [showPassword, setShowPassword] = useState(false);

    const togglePasswordVisibility = () => {
      setShowPassword((prev) => !prev);
    };

    return (
      <Input
        ref={ref}
        type={showPassword ? 'text' : 'password'}
        value={value}
        onChange={onChange}
        suffix={
          <motion.button
            type="button"
            onClick={togglePasswordVisibility}
            className="text-mid hover:text-ink transition-colors p-1 cursor-pointer"
            whileHover={{ scale: 1.1 }}
            whileTap={{ scale: 0.9 }}
            aria-label={showPassword ? 'Hide password' : 'Show password'}
          >
            {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </motion.button>
        }
        {...props}
      />
    );
  },
);
PasswordInput.displayName = 'PasswordInput';
