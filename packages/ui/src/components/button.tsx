import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../lib/utils';
import { type ButtonHTMLAttributes, forwardRef } from 'react';

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-[13px] font-display font-semibold transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 active:scale-[0.98]',
  {
    variants: {
      variant: {
        primary:   'bg-[#C6F24E] text-[#0A3D2C] hover:bg-[#AEE032] active:bg-[#98C92B] focus-visible:ring-[#C6F24E]',
        secondary: 'bg-[#0A3D2C] text-[#F7F5EF] hover:bg-[#0D4E38] active:bg-[#072D20] focus-visible:ring-[#0A3D2C]',
        outline:   'border-2 border-[#0A3D2C] text-[#0A3D2C] hover:bg-[#E9F3D8] focus-visible:ring-[#0A3D2C]',
        ghost:     'text-[#0A3D2C] hover:bg-[rgba(10,61,44,0.07)] focus-visible:ring-[#0A3D2C]',
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
