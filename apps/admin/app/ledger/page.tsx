'use client';

import { useState, useTransition } from 'react';
import { adminApi, type LedgerEntry } from '@speedplus/api-client';
import { Button, ListCard } from '@speedplus/ui';

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
    <div className="px-4 sm:px-8 py-6 sm:py-7 flex flex-col gap-4">
      <h1 className="font-display font-semibold text-[26px] tracking-tight">Ledger Viewer</h1>
      <div className="flex gap-3">
        <input
          value={userId}
          onChange={(e) => setUserId(e.target.value)}
          placeholder="User UUID"
          className="flex-1 border border-line rounded-xl px-4 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="User UUID"/>
        <Button
          variant="primary"
          size="sm"
          disabled={isPending || !userId}
          isLoading={isPending}
          onClick={() => { setCursor(undefined); load(); }}
        >
          Load
        </Button>
      </div>
      {error && <p className="text-sm text-red-600">{error}</p>}
      {entries.length > 0 && (
        <ListCard noPadding>
          <div className="grid px-5 py-3 border-b border-line text-[10.5px] font-semibold text-mid tracking-[.5px]"
            style={{ gridTemplateColumns: '1fr 2fr 100px 140px' }}>
            <span>TYPE</span><span>DESCRIPTION</span><span>AMOUNT</span><span>DATE</span>
          </div>
          {entries.map((e) => (
            <div key={e.id} className="grid px-5 py-3 text-[12.5px] items-center border-b border-line last:border-0"
              style={{ gridTemplateColumns: '1fr 2fr 100px 140px' }}>
              <span className="text-mid">{e.refType || '—'}</span>
              <span>{e.description}</span>
              <span className={`font-semibold ${e.amountKobo >= 0 ? 'text-emerald' : 'text-red-600'}`}>
                {formatKobo(e.amountKobo)}
              </span>
              <span className="text-mid">{new Date(e.createdAt).toLocaleString()}</span>
            </div>
          ))}
        </ListCard>
      )}
      {hasMore && (
        <Button
          variant="outline"
          size="sm"
          disabled={isPending}
          isLoading={isPending}
          onClick={() => load(cursor)}
          className="self-start"
        >
          Load more
        </Button>
      )}
    </div>
  );
}
