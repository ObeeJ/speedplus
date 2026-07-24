'use client';

import { useRouter } from 'next/navigation';
import { Button } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { SelectTile } from '../../components/select-tile';
import { useGasFlowStore, type CylinderSize, type GasMode } from '../../../lib/store/gas-flow.store';

const CYLINDERS: { id: CylinderSize; label: string; description: string }[] = [
  { id: '3', label: '3 kg', description: 'Small — a few weeks of cooking' },
  { id: '6', label: '6 kg', description: 'Standard household size' },
  { id: '12.5', label: '12.5 kg', description: 'Most popular — a family for a month' },
  { id: '25', label: '25 kg', description: 'Large — heavy use or business' },
];

const MODES: { id: GasMode; label: string; description: string }[] = [
  { id: 'refill', label: 'Refill mine', description: 'We take your cylinder, fill it, bring it back' },
  { id: 'swap', label: 'Swap it', description: 'We bring a full one, take your empty — faster (+₦500)' },
];

export default function GasCylinderPage() {
  const router = useRouter();
  const { cylinder, mode, setCylinder, setMode } = useGasFlowStore();
  const canContinue = Boolean(cylinder && mode);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Which cylinder?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Cylinder size</span>
          <div className="grid grid-cols-1 min-[700px]:grid-cols-2 gap-2">
            {CYLINDERS.map((c) => (
              <SelectTile key={c.id} label={c.label} description={c.description} selected={cylinder === c.id} onClick={() => setCylinder(c.id)} />
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
            ✓ A <b className="text-emerald">{CYLINDERS.find((c) => c.id === cylinder)!.label}</b> cylinder,{' '}
            <b className="text-emerald">{mode === 'swap' ? 'swapped for a full one' : 'refilled'}</b>. Next: where do we come?
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
