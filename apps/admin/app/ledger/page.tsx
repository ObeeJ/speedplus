'use client';

import { useState, useTransition } from 'react';
import { adminApi, type LedgerEntry } from '@speedplus/api-client';

function formatKobo(k: number) {
  const sign = k >= 0 ? '+' : '';
  return `${sign}₦${(Math.abs(k) / 100).toLocaleString('en-NG', { minimumFractionDigits: 2 })}`;
}

export default function LedgerPage() {
  const [userId, setUserId] = useState('');
  const [entries, setEntries] = useState<LedgerEntry[]>([]);
  const [cursor, setCursor] = useState<string | undefined>();
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState('');
  const [isPending, startTransition] = useTransition();

  function load(nextCursor?: string) {
    startTransition(async () => {
      try {
        const d = await adminApi.getLedger(userId, nextCursor);
        if (nextCursor) {
          setEntries((prev) => [...prev, ...d.entries]);
        } else {
          setEntries(d.entries);
        }
        setHasMore(d.entries.length === 50);
        if (d.entries.length > 0) {
          setCursor(d.entries[d.entries.length - 1]!.id);
        }
        setError('');
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed');
      }
    });
  }

  return (
    <div className="px-8 py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Ledger Viewer</h1>
      <div className="flex gap-3">
        <input
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
          placeholder="User UUID"
          className="flex-1 border border-line rounded-xl px-4 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald"
        />
        <button
          disabled={isPending || !userId}
          onClick={() => { setCursor(undefined); load(); }}
          className="font-display text-sm font-semibold text-emerald bg-lime rounded-[10px] px-5 py-2 hover:bg-lime-600 transition-colors disabled:opacity-50"
        >
          Load
        </button>
      </div>
      {error && <p className="text-sm text-red-600">{error}</p>}
      {entries.length > 0 && (
        <div className="bg-white border border-line rounded-2xl overflow-hidden">
          <div className="grid px-5 py-3 border-b border-[#EFECE3] text-[10.5px] font-semibold text-mid tracking-[.5px]"
            style={{ gridTemplateColumns: '1fr 2fr 100px 140px' }}>
            <span>TYPE</span><span>DESCRIPTION</span><span>AMOUNT</span><span>DATE</span>
          </div>
          {entries.map((e) => (
            <div key={e.id} className="grid px-5 py-3 text-[12.5px] items-center border-b border-[#EFECE3] last:border-0"
              style={{ gridTemplateColumns: '1fr 2fr 100px 140px' }}>
              <span className="text-mid">{e.refType || '—'}</span>
              <span>{e.description}</span>
              <span className={`font-semibold ${e.amountKobo >= 0 ? 'text-emerald' : 'text-red-600'}`}>
                {formatKobo(e.amountKobo)}
              </span>
              <span className="text-mid">{new Date(e.createdAt).toLocaleString()}</span>
            </div>
          ))}
        </div>
      )}
      {hasMore && (
        <button
          disabled={isPending}
          onClick={() => load(cursor)}
          className="self-start font-display text-sm font-semibold text-mid border border-line rounded-[10px] px-5 py-2 hover:bg-sand transition-colors disabled:opacity-50"
        >
          Load more
        </button>
      )}
    </div>
  );
}
