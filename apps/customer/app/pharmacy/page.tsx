'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { catalogApi } from '@speedplus/api-client';
import { FlowHeader } from '../components/flow-header';
import { usePharmacyFlowStore } from '../../lib/store/pharmacy-flow.store';

// The pharmacy picker is step 0 of the flow — a prescription must be
// submitted to a specific licensed pharmacy (merchantId is required by the
// backend), so choosing the pharmacy has to happen before upload, not after.
// This also fixes the dead /pharmacy link from the home page: there was
// previously no page.tsx at this route at all.
export default function PharmacyPickerPage() {
  const router = useRouter();
  const setMerchant = usePharmacyFlowStore((s) => s.setMerchant);

  const { data, isLoading, isError } = useQuery({
    queryKey: ['pharmacy-merchants'],
    queryFn: () => catalogApi.listMerchants('pharmacy'),
  });

  function choose(merchantId: string, lat: number, lng: number) {
    setMerchant(merchantId, lat, lng);
    router.push('/pharmacy/items');
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Choose a pharmacy" step={1} totalSteps={4} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-3 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        {isLoading && <div className="text-[13px] text-mid px-1">Finding nearby pharmacies…</div>}

        {isError && (
          <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
            Couldn't load pharmacies right now. Please try again.
          </div>
        )}

        {data?.merchants.length === 0 && (
          <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
            No pharmacies are available in your area yet.
          </div>
        )}

        {data?.merchants.map((m) => (
          <button
            key={m.id}
            onClick={() => choose(m.id, m.lat, m.lng)}
            disabled={!m.isOpen}
            className={`w-full flex items-center justify-between gap-3 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
              m.isOpen ? 'bg-white border-line hover:border-emerald/40' : 'bg-white/50 border-line opacity-50'
            }`}
          >
            <span className="flex flex-col gap-0.5">
              <span className="font-display font-semibold text-[15px] text-ink">{m.businessName}</span>
              <span className="text-[12.5px] text-mid">
                {m.isOpen ? `★ ${m.rating.toFixed(1)}` : 'Closed'}
              </span>
            </span>
          </button>
        ))}
      </div>
    </main>
  );
}
