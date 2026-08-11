'use client';

import { useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Button, CameraIcon, Skeleton } from '@speedplus/ui';
import { isValidPrescriptionFile } from '@speedplus/utils';
import { catalogApi } from '@speedplus/api-client';
import { FlowHeader } from '../../components/flow-header';
import { usePharmacyFlowStore } from '../../../lib/store/pharmacy-flow.store';
import { useUploadPrescription, usePrescriptionStatus } from '../../../lib/hooks/use-order-mutations';

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG')}`;
}

export default function PharmacyItemsPage() {
  const router = useRouter();
  const { tab, setTab, otcItemId, setOtcItem, merchantId, rxStatus, setRxStatus, prescriptionId, setPrescriptionId, canContinueItems } =
    usePharmacyFlowStore();
  const canContinue = canContinueItems();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const uploadPrescription = useUploadPrescription();
  const [uploadError, setUploadError] = useState<string | null>(null);

  // A pharmacy must be chosen before a prescription can be reviewed.
  if (tab === 'rx' && !merchantId) {
    router.replace('/pharmacy');
  }

  // Load OTC products from the catalog — no hard-coded items.
  const productsQuery = useQuery({
    queryKey: ['pharmacy-products', merchantId],
    queryFn: () => catalogApi.listProducts(merchantId!, 'otc'),
    enabled: Boolean(merchantId) && tab === 'otc',
  });

  // Poll real server-side review status once we have a prescription id.
  const statusQuery = usePrescriptionStatus(prescriptionId);
  const serverStatus = statusQuery.data?.status;
  if (serverStatus && serverStatus !== rxStatus) {
    setRxStatus(serverStatus === 'consumed' ? 'approved' : serverStatus);
  }

  function handleFileSelected(file: File) {
    if (!merchantId) {
      router.replace('/pharmacy');
      return;
    }
    const validation = isValidPrescriptionFile(file);
    if (!validation.valid) {
      setUploadError(validation.error ?? 'That file cannot be used as a prescription.');
      return;
    }
    setUploadError(null);
    setRxStatus('uploading');
    uploadPrescription.mutate(
      { file, merchantId },
      {
        onSuccess: (prescription) => {
          setPrescriptionId(prescription.id);
          setRxStatus(prescription.status as typeof rxStatus);
        },
        onError: (err) => {
          setRxStatus(null);
          setUploadError(err instanceof Error ? err.message : 'Upload failed. Please try again.');
        },
      },
    );
  }

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="What do you need?" step={1} totalSteps={4} backHref="/pharmacy" />

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
            {productsQuery.isLoading && (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-[62px] rounded-[13px]" />
                <Skeleton className="h-[62px] rounded-[13px]" />
                <Skeleton className="h-[62px] rounded-[13px]" />
              </div>
            )}
            {productsQuery.isError && (
              <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">
                Couldn&apos;t load products. Please try again.
              </div>
            )}
            {productsQuery.data?.products.filter((p) => p.isAvailable).map((product) => {
              const selected = otcItemId === product.id;
              return (
                <button
                  key={product.id}
                  onClick={() => setOtcItem(product.id, product.priceKobo)}
                  className={`w-full flex items-center justify-between gap-3 rounded-[13px] border-2 px-4 py-3.5 text-left transition-all ${
                    selected ? 'bg-emerald border-lime' : 'bg-white border-line hover:border-emerald/40'
                  }`}
                >
                  <span className="flex flex-col gap-0.5">
                    <span className={`font-display font-semibold text-[15px] ${selected ? 'text-lime' : 'text-ink'}`}>
                      {selected ? `✓ ${product.name}` : product.name}
                    </span>
                    {product.description && (
                      <span className={`text-[12.5px] ${selected ? 'text-sand/70' : 'text-mid'}`}>{product.description}</span>
                    )}
                  </span>
                  <span className={`font-display font-semibold text-[14px] ${selected ? 'text-lime' : 'text-ink'}`}>
                    {naira(product.priceKobo)}
                  </span>
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
            {rxStatus === 'uploading' && (
              <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-mid">Uploading…</div>
            )}
            {rxStatus === 'pending' && (
              <div className="rounded-[13px] border-2 border-amber bg-amber/10 px-4 py-4 text-[13px] text-ink">
                Waiting for the pharmacist to review your prescription…
              </div>
            )}
            {rxStatus === 'approved' && (
              <div className="rounded-[13px] border-2 border-lime bg-emerald px-4 py-4 text-[13px] text-lime">
                ✅ Approved by the pharmacy — your items are ready.
              </div>
            )}
            {rxStatus === 'rejected' && (
              <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-ink">
                <p className="font-semibold mb-1">This prescription wasn&apos;t approved.</p>
                {statusQuery.data?.reviewNote && (
                  <p className="text-mid">{statusQuery.data.reviewNote}</p>
                )}
                <button
                  onClick={() => { setPrescriptionId(null); setRxStatus(null); }}
                  className="mt-2 text-emerald underline text-[12.5px]"
                >
                  Upload a different prescription
                </button>
              </div>
            )}
            {rxStatus === 'expired' && (
              <div className="rounded-[13px] border-2 border-line bg-white px-4 py-4 text-[13px] text-ink">
                <p className="mb-2">This approval has expired.</p>
                <button
                  onClick={() => { setPrescriptionId(null); setRxStatus(null); }}
                  className="text-emerald underline text-[12.5px]"
                >
                  Upload again
                </button>
              </div>
            )}
            {uploadError && (
              <div className="rounded-[13px] border-2 border-red-300 bg-red-50 px-4 py-3 text-[12.5px] text-red-700">
                {uploadError}
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
