'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type KYCCheck } from '@fourdat/api-client';
import { Badge, Button, Modal, Skeleton, Input } from '@fourdat/ui';

export default function KYCPage() {
  const [checks, setChecks] = useState<KYCCheck[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [isPending, startTransition] = useTransition();

  // Reject modal state
  const [rejectTarget, setRejectTarget] = useState<string | null>(null);
  const [rejectNote, setRejectNote] = useState('');

  useEffect(() => {
    adminApi.getKYCQueue()
      .then((d) => setChecks(d.checks))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  function handleApprove(id: string) {
    startTransition(async () => {
      try {
        await adminApi.approveKYC(id);
        setChecks((prev) => prev.filter((c) => c.id !== id));
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  function handleRejectConfirm() {
    if (!rejectTarget || !rejectNote.trim()) return;
    const id = rejectTarget;
    const note = rejectNote.trim();
    setRejectTarget(null);
    setRejectNote('');
    startTransition(async () => {
      try {
        await adminApi.rejectKYC(id, note);
        setChecks((prev) => prev.filter((c) => c.id !== id));
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">KYC Queue</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}
      {loading && (
        <div className="flex flex-col gap-3">
          {[1, 2, 3].map((i) => <Skeleton key={i} className="h-[76px] rounded-2xl" />)}
        </div>
      )}
      {!loading && checks.length === 0 && !error && (
        <p className="text-sm text-mid">No pending KYC submissions.</p>
      )}
      {checks.map((c) => (
        <div key={c.id} className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4">
          <div className="flex-1 flex flex-col gap-0.5">
            <span className="text-[13.5px] font-semibold">{c.docType.toUpperCase()}</span>
            <span className="text-[11px] text-mid">User {c.userId}</span>
            <span className="text-[11px] text-mid">{new Date(c.createdAt).toLocaleString()}</span>
          </div>
          <Badge>{c.status}</Badge>
          <div className="flex gap-2">
            <Button variant="primary" size="sm" disabled={isPending} onClick={() => handleApprove(c.id)}>
              Approve
            </Button>
            <Button variant="danger" size="sm" disabled={isPending} onClick={() => { setRejectTarget(c.id); setRejectNote(''); }}>
              Reject
            </Button>
          </div>
        </div>
      ))}

      <Modal
        isOpen={!!rejectTarget}
        onClose={() => { setRejectTarget(null); setRejectNote(''); }}
        title="Reject KYC submission"
        description="Provide a reason — this will be visible to the user."
      >
        <div className="flex flex-col gap-4">
          <Input
            label="Rejection reason"
            placeholder="e.g. Document image is blurry"
            value={rejectNote}
            onChange={(e) => setRejectNote(e.target.value)}
            autoFocus
          />
          <div className="flex gap-2 justify-end">
            <Button variant="outline" size="sm" onClick={() => { setRejectTarget(null); setRejectNote(''); }}>
              Cancel
            </Button>
            <Button variant="danger" size="sm" disabled={!rejectNote.trim()} onClick={handleRejectConfirm}>
              Confirm rejection
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
