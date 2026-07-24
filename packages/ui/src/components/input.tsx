import { forwardRef, type InputHTMLAttributes } from 'react';
import { cn } from '../lib/utils';

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, hint, id, ...props }, ref) => (
    <div className="flex flex-col gap-1.5">
      {label && <label htmlFor={id} className="text-sm font-medium text-[#121216]">{label}</label>}
      <input
        ref={ref}
        id={id}
        className={cn(
          'h-11 w-full rounded-[13px] border bg-white px-4 text-base text-[#121216] placeholder:text-[#63636E]',
          'focus:outline-none focus:ring-2 focus:ring-[#C6F24E] focus:border-transparent',
          'disabled:cursor-not-allowed disabled:opacity-50',
          error ? 'border-[#DC2626] focus:ring-[#DC2626]' : 'border-[#E4E0D6]',
          className,
        )}
        {...props}
      />
      {error && <p className="text-xs text-[#DC2626]">{error}</p>}
      {hint && !error && <p className="text-xs text-[#63636E]">{hint}</p>}
    </div>
  ),
);
Input.displayName = 'Input';
