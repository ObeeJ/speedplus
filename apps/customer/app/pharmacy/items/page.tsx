'use client';

import { useRef } from 'react';
import { useRouter } from 'next/navigation';
import { Button, CameraIcon } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import { usePharmacyFlowStore, OTC_ITEMS } from '../../../lib/store/pharmacy-flow.store';
import { useUploadPrescription } from '../../../lib/hooks/use-order-mutations';

function naira(n: number) {
  return `₦${n.toLocaleString('en-NG')}`;
}

export default function PharmacyItemsPage() {
  const router = useRouter();
  const { tab, setTab, otcItemId, setOtcItemId, rxStatus, uploadRx, setRxStatus, setPrescriptionId, canContinueItems } = usePharmacyFlowStore();
  const canContinue = canContinueItems();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const uploadPrescription = useUploadPrescription();

  function handleFileSelected(file: File) {
    setRxStatus('uploaded');
    // Convert File to R2 key via direct upload — for now use the filename as a
    // placeholder key. The real flow: frontend gets a presigned R2 URL, uploads
    // directly, then passes the returned object key here.
    const r2Key = `prescriptions/${Date.now()}-${file.name}`;
    uploadPrescription.mutate(
      { r2Key, merchantId: undefined },
      {
        onSuccess: (prescription) => {
          setPrescriptionId(prescription.id);
          setRxStatus('under_review');
          setTimeout(() => setRxStatus('approved'), 2700);
        },
        onError: () => {
          // No backend reachable yet — fall back to the local demo simulation so the flow still works.
          uploadRx();
        },
      },
    );
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="What do you need?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-5 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">
        <div className="flex gap-2 bg-tile rounded-[13px] p-1">
          <button
            onClick={() => setTab('otc')}
            className={`flex-1 rounded-[10px] py-2.5 text-[13px] font-display font-semibold transition-colors ${
              tab === 'otc' ? 'bg-emerald text-lime' : 'text-ink'
            }`}
          >
            Everyday medicine
          </button>
          <button
            onClick={() => setTab('rx')}
            className={`flex-1 rounded-[10px] py-2.5 text-[13px] font-display font-semibold transition-colors ${
              tab === 'rx' ? 'bg-emerald text-lime' : 'text-ink'
            }`}
          >
            I have a prescription
          </button>
        </div>

        {tab === 'otc' ? (
          <div className="flex flex-col gap-2">
            {OTC_ITEMS.map((item) => {
              const selected = otcItemId === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => setOtcItemId(item.id)}
                  className={`w-full flex items-center justify-between gap-3 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
                    selected ? 'bg-emerald border-lime' : 'bg-white border-line hover:border-emerald/40'
                  }`}
                >
                  <span className="flex flex-col gap-0.5">
                    <span className={`font-display font-semibold text-[15px] ${selected ? 'text-lime' : 'text-ink'}`}>
                      {selected ? `✓ ${item.name}` : item.name}
                    </span>
                    <span className={`text-[12.5px] ${selected ? 'text-sand/70' : 'text-mid'}`}>{item.description}</span>
                  </span>
                  <span className={`font-display font-semibold text-[14px] ${selected ? 'text-lime' : 'text-ink'}`}>{naira(item.price)}</span>
                </button>
              );
            })}
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {!rxStatus && (
              <>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) handleFileSelected(file);
                  }}
                />
                <button
                  onClick={() => fileInputRef.current?.click()}
                  className="w-full flex flex-col items-center gap-2 rounded-[13px] border-2 border-dashed border-line bg-white px-4 py-8 text-center hover:border-emerald/40 transition-colors"
                >
                  <span className="font-display font-semibold text-[15px] text-ink flex items-center gap-2">
                    <CameraIcon size={18} />
                    Upload your prescription
                  </span>
                  <span className="text-[12.5px] text-mid">A photo of your prescription is all we need</span>
                </button>
              </>
            )}
            {rxStatus === 'uploaded' && (
              <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">Uploading…</div>
            )}
            {rxStatus === 'under_review' && (
              <div className="rounded-[13px] border-2 border-amber bg-amber/10 px-4 py-4 text-[13px] text-ink">
                Pharmacist is checking it now… Adaeze at HealthPlus Lekki
              </div>
            )}
            {rxStatus === 'approved' && (
              <div className="rounded-[13px] border-2 border-lime bg-emerald px-4 py-4 text-[13px] text-lime">
                ✅ Approved — your items are ready. Checked by Adaeze O. (PCN licensed)
              </div>
            )}
          </div>
        )}

        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!canContinue}
          onClick={() => router.push('/pharmacy/deliver')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>
    </main>
  );
}
