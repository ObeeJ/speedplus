'use client';

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { merchantApi, type MerchantPrescription } from '@fourdat/api-client';
import { Card, Button, Input } from '@fourdat/ui';

export default function PrescriptionsPage() {
  const qc = useQueryClient();
  const [rejectingId, setRejectingId] = useState<string | null>(null);
  const [rejectNote, setRejectNote] = useState('');

  const rxQuery = useQuery({
    queryKey: ['merchant-prescriptions'],
    queryFn: () => merchantApi.listPrescriptions('pending'),
    refetchInterval: 20_000,
  });

  const reviewMutation = useMutation({
    mutationFn: ({ id, approve, note }: { id: string; approve: boolean; note?: string }) =>
      merchantApi.reviewPrescription(id, approve, note),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['merchant-prescriptions'] });
      setRejectingId(null);
      setRejectNote('');
    },
  });

  const prescriptions = rxQuery.data?.prescriptions ?? [];

  return (
    <>
      <div>
        <p className="text-xs font-semibold text-mid tracking-widest uppercase">Partner Portal</p>
        <h1 className="font-display font-bold text-2xl text-ink tracking-tight mt-0.5">
          Prescription review
        </h1>
      </div>

      {rxQuery.isLoading && <p className="text-sm text-mid">Loading…</p>}

      {!rxQuery.isLoading && prescriptions.length === 0 && (
        <Card>
          <p className="text-sm text-mid text-center py-8">No prescriptions awaiting review.</p>
        </Card>
      )}

      <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))' }}>
        {prescriptions.map((rx: MerchantPrescription) => (
          <Card key={rx.id} className="overflow-hidden p-0 flex flex-col">
            <div className="h-52 bg-tile flex items-center justify-center overflow-hidden">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={rx.viewUrl} alt="Prescription" className="w-full h-full object-contain" />
            </div>
            <div className="p-4 flex flex-col gap-3">
              <p className="text-xs text-mid">
                Customer #{rx.customerId.slice(0, 8)} · {new Date(rx.createdAt).toLocaleString()}
              </p>
              {rejectingId === rx.id ? (
                <div className="flex flex-col gap-2">
                  <Input
                    id={`reject-${rx.id}`}
                    label="Reason for rejection"
                    placeholder="Shown to customer"
                    value={rejectNote}
                    onChange={(e) => setRejectNote(e.target.value)}
                  />
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="danger"
                      className="flex-1"
                      onClick={() => reviewMutation.mutate({ id: rx.id, approve: false, note: rejectNote || undefined })}
                      isLoading={reviewMutation.isPending}
                    >
                      Confirm reject
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="flex-1"
                      onClick={() => { setRejectingId(null); setRejectNote(''); }}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    className="flex-[2]"
                    onClick={() => reviewMutation.mutate({ id: rx.id, approve: true })}
                    isLoading={reviewMutation.isPending}
                  >
                    ✓ Approve
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    className="flex-1"
                    onClick={() => setRejectingId(rx.id)}
                  >
                    Reject
                  </Button>
                </div>
              )}
              {reviewMutation.isError && (
                <p className="text-xs text-red-600">{(reviewMutation.error as Error).message}</p>
              )}
            </div>
          </Card>
        ))}
      </div>
    </>
  );
}
