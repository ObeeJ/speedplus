import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import { type ButtonHTMLAttributes, forwardRef } from 'react';

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-xl font-semibold transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 active:scale-[0.98]',
  {
    variants: {
      variant: {
        primary:   'bg-[#00C48C] text-white hover:bg-[#008F66] focus-visible:ring-[#00C48C]',
        secondary: 'bg-[#1A1A2E] text-white hover:bg-[#1A1A2E]/90 focus-visible:ring-[#1A1A2E]',
        outline:   'border-2 border-[#00C48C] text-[#00C48C] hover:bg-[#E0FBF4] focus-visible:ring-[#00C48C]',
        ghost:     'text-[#1A1A2E] hover:bg-[#E0FBF4] focus-visible:ring-[#00C48C]',
        danger:    'bg-[#DC2626] text-white hover:bg-[#DC2626]/90 focus-visible:ring-[#DC2626]',
      },
      size: {
        sm:   'h-9 px-4 text-sm',
        md:   'h-11 px-6 text-base',
        lg:   'h-13 px-8 text-lg',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
);

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  isLoading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, isLoading, children, disabled, ...props }, ref) => (
    <button ref={ref} className={cn(buttonVariants({ variant, size }), className)} disabled={disabled ?? isLoading} {...props}>
      {isLoading && (
        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      )}
      {children}
    </button>
  ),
);
Button.displayName = 'Button';

export { buttonVariants };
