'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input } from '@speedplus/ui';
import { FlowHeader } from '../../components/flow-header';
import {
  usePackageFlowStore,
  type AddressOption,
  type StopInput,
} from '../../../lib/store/package-flow.store';
import { apiClient } from '@speedplus/api-client';
import type { ApiResponse } from '@speedplus/types';

interface SavedAddress {
  id: string;
  label?: string;
  street: string;
  city: string;
  lat: number;
  lng: number;
  isDefault: boolean;
}

function AddressTile({
  address,
  selected,
  onClick,
}: {
  address: AddressOption;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left rounded-[14px] border px-4 py-3 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald ${
        selected ? 'border-emerald bg-tile' : 'border-line bg-white hover:border-emerald/40'
      }`}
    >
      <span className="block text-[13px] font-semibold text-ink">{address.label || address.street}</span>
      <span className="block text-[11px] text-mid mt-0.5">{address.street}, {address.city}</span>
    </button>
  );
}

function GPSButton({ onSelect, loading }: { onSelect: (a: AddressOption, target: 'pickup' | 'dropoff') => void; loading: boolean }) {
  return null; // rendered inline below
}

export default function PackageWherePage() {
  const router = useRouter();
  const {
    pickup, dropoff, recipientName, recipientPhone,
    isMultiDrop, stops,
    setPickup, setDropoff, setRecipientName, setRecipientPhone,
    setIsMultiDrop, addStop, removeStop,
  } = usePackageFlowStore();

  const [savedAddresses, setSavedAddresses] = useState<AddressOption[]>([]);
  const [loadingAddresses, setLoadingAddresses] = useState(true);
  const [gpsLoading, setGpsLoading] = useState(false);
  const [gpsError, setGpsError] = useState('');

  // Stop editor state
  const [editingStop, setEditingStop] = useState<Partial<StopInput> | null>(null);

  useEffect(() => {
    apiClient
      .get<ApiResponse<{ addresses: SavedAddress[] }>>('/users/me/addresses')
      .then(({ data }) => {
        if (data.success) {
          setSavedAddresses(
            data.data.addresses.map((a) => ({
              id: a.id,
              label: a.label || a.street,
              street: a.street,
              city: a.city,
              lat: a.lat,
              lng: a.lng,
            })),
          );
        }
      })
      .catch(() => {})
      .finally(() => setLoadingAddresses(false));
  }, []);

  async function getGPS(): Promise<AddressOption | null> {
    return new Promise((resolve) => {
      setGpsLoading(true);
      setGpsError('');
      navigator.geolocation.getCurrentPosition(
        async (pos) => {
          const { latitude: lat, longitude: lng } = pos.coords;
          try {
            const res = await fetch(
              `https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lng}&format=json`,
              { headers: { 'Accept-Language': 'en' } },
            );
            const geo = await res.json();
            const street = geo.address?.road || geo.display_name?.split(',')[0] || 'Current location';
            const city = geo.address?.city || geo.address?.town || 'Lagos';
            resolve({ id: `gps-${Date.now()}`, label: 'Current location', street, city, lat, lng });
          } catch {
            resolve({ id: `gps-${Date.now()}`, label: 'Current location', street: 'Current location', city: 'Lagos', lat, lng });
          }
          setGpsLoading(false);
        },
        () => {
          setGpsError('Could not get your location.');
          setGpsLoading(false);
          resolve(null);
        },
        { timeout: 8000 },
      );
    });
  }

  function saveStop() {
    if (!editingStop?.address || !editingStop.recipientName?.trim() || !editingStop.recipientPhone?.trim()) return;
    const seq = editingStop.sequence ?? stops.length + 1;
    addStop({
      sequence: seq,
      address: editingStop.address,
      recipientName: editingStop.recipientName,
      recipientPhone: editingStop.recipientPhone,
      notes: editingStop.notes ?? '',
    });
    setEditingStop(null);
  }

  const singleCanContinue = !isMultiDrop && Boolean(pickup && dropoff && recipientName.trim() && recipientPhone.trim());
  const multiCanContinue = isMultiDrop && Boolean(pickup && stops.length >= 1);

  return (
    <main className="min-h-screen bg-sand flex flex-col">
      <FlowHeader title="Where's it going?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-5 flex flex-col gap-6 min-[700px]:max-w-[860px] min-[700px]:mx-auto min-[700px]:px-8 min-[700px]:py-10 min-[700px]:w-full">

        {/* Pickup */}
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Pickup address</span>
          {loadingAddresses ? (
            <div className="h-11 rounded-[14px] bg-line animate-pulse" />
          ) : (
            <div className="flex flex-col gap-2">
              {savedAddresses.map((a) => (
                <AddressTile key={a.id} address={a} selected={pickup?.id === a.id} onClick={() => setPickup(a)} />
              ))}
              <button
                type="button"
                onClick={async () => { const a = await getGPS(); if (a) setPickup(a); }}
                disabled={gpsLoading}
                className="flex items-center gap-2.5 rounded-[14px] border border-line bg-white px-4 py-3 text-[13px] font-semibold text-emerald hover:border-emerald/40 transition-all duration-150 disabled:opacity-50"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="3" /><path d="M12 1v4M12 19v4M1 12h4M19 12h4" />
                </svg>
                {gpsLoading ? 'Getting location…' : 'Use my current location'}
              </button>
            </div>
          )}
          {pickup && <span className="text-[12px] text-emerald font-medium">✓ {pickup.label} — {pickup.street}</span>}
        </section>

        {/* Delivery mode toggle */}
        <section className="flex flex-col gap-2.5">
          <span className="text-[13px] font-semibold text-mid">Delivery type</span>
          <div className="flex gap-2">
            {[
              { id: false, label: 'Single drop-off', desc: 'One recipient' },
              { id: true, label: 'Multi-drop', desc: 'Multiple recipients, one trip' },
            ].map((opt) => (
              <button
                key={String(opt.id)}
                type="button"
                onClick={() => setIsMultiDrop(opt.id)}
                className={`flex-1 text-left rounded-[14px] border px-4 py-3 transition-all duration-150 ${
                  isMultiDrop === opt.id ? 'border-emerald bg-tile' : 'border-line bg-white hover:border-emerald/40'
                }`}
              >
                <span className="block text-[13px] font-semibold text-ink">{opt.label}</span>
                <span className="block text-[11px] text-mid mt-0.5">{opt.desc}</span>
              </button>
            ))}
          </div>
        </section>

        {/* Single drop-off */}
        {!isMultiDrop && (
          <>
            <section className="flex flex-col gap-2.5">
              <span className="text-[13px] font-semibold text-mid">Drop-off address</span>
              <div className="flex flex-col gap-2">
                {savedAddresses.map((a) => (
                  <AddressTile key={a.id} address={a} selected={dropoff?.id === a.id} onClick={() => setDropoff(a)} />
                ))}
                <button
                  type="button"
                  onClick={async () => { const a = await getGPS(); if (a) setDropoff(a); }}
                  disabled={gpsLoading}
                  className="flex items-center gap-2.5 rounded-[14px] border border-line bg-white px-4 py-3 text-[13px] font-semibold text-emerald hover:border-emerald/40 transition-all duration-150 disabled:opacity-50"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                    <circle cx="12" cy="12" r="3" /><path d="M12 1v4M12 19v4M1 12h4M19 12h4" />
                  </svg>
                  Use my current location
                </button>
              </div>
              {dropoff && <span className="text-[12px] text-emerald font-medium">✓ {dropoff.label} — {dropoff.street}</span>}
            </section>

            <section className="flex flex-col gap-3">
              <span className="text-[13px] font-semibold text-mid">Who is receiving it?</span>
              <Input id="recipientName" label="Recipient name" placeholder="e.g. Amaka Obi" value={recipientName} onChange={(e) => setRecipientName(e.target.value)} />
              <Input id="recipientPhone" label="Recipient phone" type="tel" placeholder="08012345678" value={recipientPhone} onChange={(e) => setRecipientPhone(e.target.value)} />
            </section>
          </>
        )}

        {/* Multi-drop stops */}
        {isMultiDrop && (
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <span className="text-[13px] font-semibold text-mid">Drop-off stops ({stops.length})</span>
              <button
                type="button"
                onClick={() => setEditingStop({ sequence: stops.length + 1 })}
                className="text-[12px] font-semibold text-emerald hover:underline"
              >
                + Add stop
              </button>
            </div>

            {stops.length === 0 && (
              <div className="border-[1.5px] border-dashed border-line rounded-[14px] px-4 py-5 text-center text-[12px] text-mid">
                No stops yet. Add at least one drop-off.
              </div>
            )}

            {stops.map((stop, i) => (
              <div
                key={stop.sequence}
                className="bg-white border border-line rounded-[14px] px-4 py-3 flex items-center gap-3"
                style={{ animation: `fadeUp 0.25s cubic-bezier(0.16,1,0.3,1) ${i * 50}ms both` }}
              >
                <span className="w-7 h-7 rounded-full bg-tile flex items-center justify-center font-display font-bold text-emerald text-sm flex-shrink-0">
                  {stop.sequence}
                </span>
                <div className="flex-1 flex flex-col gap-0.5">
                  <span className="text-[13px] font-semibold">{stop.recipientName}</span>
                  <span className="text-[11px] text-mid">{stop.address.street}, {stop.address.city}</span>
                  <span className="text-[11px] text-mid">{stop.recipientPhone}</span>
                </div>
                <button
                  type="button"
                  onClick={() => removeStop(stop.sequence)}
                  className="text-[11px] text-[#DC2626] hover:underline"
                >
                  Remove
                </button>
              </div>
            ))}

            {/* Stop editor */}
            {editingStop !== null && (
              <div className="bg-white border-2 border-emerald rounded-[14px] px-4 py-4 flex flex-col gap-3">
                <span className="text-[13px] font-semibold text-ink">Stop {editingStop.sequence}</span>
                <div className="flex flex-col gap-2">
                  <span className="text-[11px] font-semibold text-mid">Drop-off address</span>
                  {savedAddresses.map((a) => (
                    <AddressTile
                      key={a.id}
                      address={a}
                      selected={editingStop.address?.id === a.id}
                      onClick={() => setEditingStop((s) => ({ ...s, address: a }))}
                    />
                  ))}
                </div>
                <Input
                  label="Recipient name"
                  placeholder="e.g. Emeka Obi"
                  value={editingStop.recipientName ?? ''}
                  onChange={(e) => setEditingStop((s) => ({ ...s, recipientName: e.target.value }))}
                />
                <Input
                  label="Recipient phone"
                  type="tel"
                  placeholder="08012345678"
                  value={editingStop.recipientPhone ?? ''}
                  onChange={(e) => setEditingStop((s) => ({ ...s, recipientPhone: e.target.value }))}
                />
                <Input
                  label="Notes (optional)"
                  placeholder="e.g. Leave at gate"
                  value={editingStop.notes ?? ''}
                  onChange={(e) => setEditingStop((s) => ({ ...s, notes: e.target.value }))}
                />
                <div className="flex gap-2">
                  <Button
                    variant="primary"
                    size="sm"
                    disabled={!editingStop.address || !editingStop.recipientName?.trim() || !editingStop.recipientPhone?.trim()}
                    onClick={saveStop}
                  >
                    Save stop
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setEditingStop(null)}>Cancel</Button>
                </div>
              </div>
            )}
          </section>
        )}

        {gpsError && <p className="text-xs text-[#DC2626]" role="alert">{gpsError}</p>}
        <span className="text-xs text-mid">Nothing is paid yet.</span>

        <Button
          variant="primary"
          size="lg"
          disabled={!singleCanContinue && !multiCanContinue}
          onClick={() => router.push('/package/what')}
          className="w-full min-[700px]:max-w-[380px]"
        >
          Continue
        </Button>
      </div>

      <style>{`
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(10px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
