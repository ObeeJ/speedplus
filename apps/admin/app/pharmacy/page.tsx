'use client';

import { useEffect, useState, useTransition } from 'react';
import { adminApi } from '@speedplus/api-client';
import { PrescriptionIcon, KYCIcon } from '@speedplus/ui';

type PrescriptionRow = Awaited<ReturnType<typeof adminApi.listPrescriptions>>['prescriptions'][number];
type RxStatus = 'pending' | 'approved' | 'rejected' | 'consumed' | 'expired';

const STATUS_META: Record<RxStatus, { label: string; bg: string; text: string; dot: string }> = {
  pending:  { label: 'Pending',  bg: '#FFF7E6',   text: '#8A6A1B', dot: '#E8B14E' },
  approved: { label: 'Approved', bg: '#E9F3D8',   text: '#0A3D2C', dot: '#7BA05B' },
  rejected: { label: 'Rejected', bg: '#FEF2F2',   text: '#B4231F', dot: '#DC2626' },
  consumed: { label: 'Consumed', bg: '#EFECE3',   text: '#63636E', dot: '#BDBAB2' },
  expired:  { label: 'Expired',  bg: '#F3F4F6',   text: '#6B7280', dot: '#D1D5DB' },
};

function getRxMeta(status: string) {
  return STATUS_META[status as RxStatus] ?? STATUS_META.pending;
}

const FILTERS: { value: string; label: string }[] = [
  { value: '',         label: 'All' },
  { value: 'pending',  label: 'Pending' },
  { value: 'approved', label: 'Approved' },
  { value: 'rejected', label: 'Rejected' },
];

export default function PharmacyPage() {
  const [rxList, setRxList] = useState<PrescriptionRow[]>([]);
  const [filter, setFilter] = useState('pending');
  const [selected, setSelected] = useState<PrescriptionRow | null>(null);
  const [note, setNote] = useState('');
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  function load() {
    startTransition(async () => {
      try {
        const d = await adminApi.listPrescriptions(filter || undefined);
        setError('');
        setRxList(d.prescriptions ?? []);
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load prescriptions');
      }
    });
  }

  useEffect(() => { load(); }, [filter]); // eslint-disable-line react-hooks/exhaustive-deps

  function handleApprove(id: string) {
    startTransition(async () => {
      try {
        await adminApi.approveKYC(id, note || undefined); // reuses KYC approve pattern
        setSelected(null);
        setNote('');
        load();
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  function handleReject(id: string) {
    if (!note.trim()) { setError('Rejection note is required.'); return; }
    startTransition(async () => {
      try {
        await adminApi.rejectKYC(id, note);
        setSelected(null);
        setNote('');
        load();
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  const visible = filter ? rxList.filter((r) => r.status === filter) : rxList;

  return (
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-[11px] flex items-center justify-center" style={{ background: '#EEF2FF' }}>
            <PrescriptionIcon size={18} color="#3730A3" accent="#6366F1" />
          </div>
          <div>
            <h1 className="font-display font-semibold text-[22px] tracking-tight text-ink">Prescriptions</h1>
            <p className="text-[12px] text-mid">Pharmacy Rx review queue</p>
          </div>
        </div>
        <span className="font-display text-[13px] font-semibold text-mid">
          {visible.length} item{visible.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Filters */}
      <div className="flex gap-2">
        {FILTERS.map(({ value, label }) => (
          <button
            key={value}
            onClick={() => setFilter(value)}
            className={`text-[11.5px] font-semibold rounded-full px-3.5 py-1.5 transition-colors border ${
              filter === value
                ? 'bg-[#0A3D2C] text-[#F7F5EF] border-[#0A3D2C]'
                : 'bg-white text-mid border-line hover:border-[#0A3D2C]/30 hover:text-ink'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {error && (
        <div className="bg-[#FEF2F2] border border-[#FECACA] rounded-xl px-4 py-3 text-sm text-[#DC2626]">
          {error}
        </div>
      )}

      {/* Detail panel */}
      {selected && (
        <div className="bg-white border border-line rounded-2xl p-5 flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <KYCIcon size={16} color="#0A3D2C" accent="#7BA05B" />
              <span className="font-display font-semibold text-[15px]">
                Rx {selected.id.slice(0, 8).toUpperCase()}
              </span>
            </div>
            <button
              onClick={() => { setSelected(null); setNote(''); setError(''); }}
              className="text-[12px] text-mid hover:text-ink transition-colors"
            >
              Close
            </button>
          </div>

          <div className="grid grid-cols-2 gap-2 text-[12.5px]">
            <div><span className="text-mid">Customer</span><br /><span className="font-mono">{selected.customerId.slice(0, 12)}</span></div>
            <div><span className="text-mid">Pharmacy</span><br /><span>{selected.merchantName ?? selected.merchantId.slice(0, 12)}</span></div>
            <div><span className="text-mid">Submitted</span><br /><span>{new Date(selected.createdAt).toLocaleDateString('en-NG')}</span></div>
            {selected.expiresAt && (
              <div><span className="text-mid">Expires</span><br /><span>{new Date(selected.expiresAt).toLocaleDateString('en-NG')}</span></div>
            )}
          </div>

          {selected.status === 'pending' && (
            <>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                placeholder="Review note (required for rejection)…"
                rows={3}
                className="w-full rounded-[11px] border border-line bg-[#F7F5EF] px-3.5 py-2.5 text-[13px] text-ink placeholder:text-mid focus:outline-none focus:border-[#0A3D2C]/40 resize-none"
              aria-label="Review note (required for rejection)" />
              <div className="flex gap-3">
                <button
                  disabled={isPending}
                  onClick={() => handleApprove(selected.id)}
                  className="font-display text-[13px] font-semibold text-[#0A3D2C] bg-[#C6F24E] rounded-[11px] px-5 py-2.5 hover:bg-[#AEE032] transition-colors disabled:opacity-50"
                >
                  Approve
                </button>
                <button
                  disabled={isPending}
                  onClick={() => handleReject(selected.id)}
                  className="font-display text-[13px] font-semibold rounded-[11px] px-5 py-2.5 border-[1.5px] transition-colors disabled:opacity-50"
                  style={{ color: '#B4231F', borderColor: '#E5B5B3' }}
                >
                  Reject
                </button>
              </div>
            </>
          )}

          {selected.reviewNote && (
            <div className="bg-[#F7F5EF] rounded-[11px] px-3.5 py-2.5 text-[12.5px] text-mid">
              <span className="font-semibold text-ink">Note: </span>{selected.reviewNote}
            </div>
          )}
        </div>
      )}

      {/* List */}
      {visible.length === 0 && !error ? (
        <div className="flex flex-col items-center justify-center py-20 gap-4">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center" style={{ background: '#EEF2FF' }}>
            <PrescriptionIcon size={26} color="#3730A3" accent="#6366F1" />
          </div>
          <p className="text-[13px] text-mid">No prescriptions in this queue.</p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {visible.map((rx) => {
            const meta = getRxMeta(rx.status);
            return (
              <button
                key={rx.id}
                onClick={() => { setSelected(rx); setNote(''); setError(''); }}
                className="bg-white border border-line rounded-2xl px-5 py-4 flex items-center gap-4 text-left hover:border-[#0A3D2C]/30 hover:shadow-sm transition-all"
              >
                <div className="w-2 h-2 rounded-full shrink-0" style={{ background: meta.dot }} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-0.5">
                    <span className="font-display font-semibold text-[13.5px] text-ink">
                      Rx {rx.id.slice(0, 8).toUpperCase()}
                    </span>
                    <span
                      className="text-[10.5px] font-bold rounded-full px-2.5 py-0.5"
                      style={{ background: meta.bg, color: meta.text }}
                    >
                      {meta.label}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-[11.5px] text-mid">
                    <span>{rx.merchantName ?? rx.merchantId.slice(0, 8)}</span>
                    <span>{new Date(rx.createdAt).toLocaleDateString('en-NG', { day: 'numeric', month: 'short' })}</span>
                  </div>
                </div>
                {rx.status === 'pending' && (
                  <span className="text-[11px] font-semibold text-[#E8B14E] shrink-0">Review</span>
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
