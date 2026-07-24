'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi, type KYCCheck } from '@speedplus/api-client';
import { Badge } from '@speedplus/ui';

export default function KYCPage() {
  const [checks, setChecks] = useState<KYCCheck[]>([]);
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    adminApi.getKYCQueue()
      .then((d) => setChecks(d.checks))
      .catch((e: Error) => setError(e.message));
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

  function handleReject(id: string) {
    const note = window.prompt('Rejection reason (required):');
    if (!note) return;
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
    <div className="px-8 py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">KYC Queue</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}
      {checks.length === 0 && !error && (
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
            <button
              disabled={isPending}
              onClick={() => handleApprove(c.id)}
              className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
            >
              Approve
            </button>
            <button
              disabled={isPending}
              onClick={() => handleReject(c.id)}
              className="font-display text-xs font-semibold rounded-[10px] px-4 py-2 border-[1.5px] transition-colors disabled:opacity-50"
              style={{ color: '#B4231F', borderColor: '#E5B5B3' }}
            >
              Reject
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
