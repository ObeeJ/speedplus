'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { SelectTile } from '../../components/select-tile';
import { useGasFlowStore, type CylinderSize, type GasMode } from '../../../lib/store/gas-flow.store';
import { gasApi } from '@speedplus/api-client';

const MODES: { id: GasMode; label: string; description: string }[] = [
  { id: 'refill', label: 'Refill mine', description: 'We take your cylinder, fill it, bring it back' },
  { id: 'swap', label: 'Swap it', description: 'We bring a full one, take your empty — faster (+₦500)' },
  { id: 'new_cylinder', label: 'New cylinder', description: 'Buy a brand-new cylinder, already filled' },
];

export default function GasCylinderPage() {
  const router = useRouter();
  const { cylinder, mode, setCylinder, setMode } = useGasFlowStore();
  const canContinue = Boolean(cylinder && mode);

  const { data: specs = [] } = useQuery({
    queryKey: ['gas-specs'],
    queryFn: () => gasApi.listSpecs(),
    staleTime: Infinity,
  });

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Which cylinder?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Cylinder size</span>
          <div className="grid grid-cols-1 min-[700px]:grid-cols-2 gap-2">
            {specs.map((s) => (
              <SelectTile
                key={s.id}
                label={s.label}
                description={`${s.sizeKg} kg · tare ${s.tareKg} kg`}
                selected={cylinder === String(s.sizeKg) as CylinderSize}
                onClick={() => setCylinder(String(s.sizeKg) as CylinderSize)}
              />
            ))}
          </div>
        </section>

        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">How should we get it?</span>
          <div className="flex flex-col gap-2">
            {MODES.map((m) => (
              <SelectTile key={m.id} label={m.label} description={m.description} selected={mode === m.id} onClick={() => setMode(m.id)} />
            ))}
          </div>
        </section>

        {canContinue && (
          <span className="text-[13px] text-mid">
            ✓ A <b className="text-emerald">{cylinder} kg</b> cylinder,{' '}
            <b className="text-emerald">{mode === 'swap' ? 'swapped for a full one' : mode === 'refill' ? 'refilled' : 'brand new'}</b>. Next: where do we come?
          </span>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!canContinue}
          onClick={() => router.push('/gas/deliver')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
