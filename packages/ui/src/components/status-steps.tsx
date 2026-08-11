import { cn } from '../lib/utils';

interface Step {
  label: string;
  sublabel?: string;
}

interface StatusStepsProps {
  steps: Step[];
  currentIndex: number;
  className?: string;
}

export function StatusSteps({ steps, currentIndex, className }: StatusStepsProps) {
  return (
    <div className={cn('flex flex-col', className)}>
      {steps.map((step, i) => {
        const done = i < currentIndex;
        const active = i === currentIndex;
        const upcoming = i > currentIndex;
        return (
          <div key={step.label} className="flex gap-3.5">
            {/* Track */}
            <div className="flex flex-col items-center">
              <span className={cn(
                'w-3 h-3 rounded-full flex-shrink-0 transition-all duration-300 mt-0.5',
                done    ? 'bg-emerald' :
                active  ? 'bg-lime ring-4 ring-lime/20' :
                          'bg-line',
              )} />
              {i < steps.length - 1 && (
                <span className={cn(
                  'w-0.5 flex-1 min-h-[32px] transition-colors duration-500',
                  done ? 'bg-emerald' : 'bg-line',
                )} />
              )}
            </div>
            {/* Label */}
            <div className={cn('pb-8', i === steps.length - 1 && 'pb-0')}>
              <span className={cn(
                'block text-[14px] font-medium transition-colors duration-300',
                done    ? 'text-ink' :
                active  ? 'text-emerald font-semibold' :
                          'text-mid/60',
              )}>
                {step.label}
              </span>
              {step.sublabel && active && (
                <span className="block text-[11px] text-mid mt-0.5">{step.sublabel}</span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
