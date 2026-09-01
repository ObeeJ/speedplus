'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input, Skeleton, SelectionCard, ListCard } from '@fourdat/ui';
import { FlowHeader } from '../../components/flow-header';
import { usePackageFlowStore, type AddressOption, type StopInput } from '../../../lib/store/package-flow.store';
import { usersApi } from '@fourdat/api-client';

function MapPinIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z" />
      <circle cx="12" cy="10" r="3" />
    </svg>
  );
}

function CrosshairIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" /><line x1="22" y1="12" x2="18" y2="12" /><line x1="6" y1="12" x2="2" y2="12" /><line x1="12" y1="6" x2="12" y2="2" /><line x1="12" y1="22" x2="12" y2="18" />
    </svg>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">{children}</p>;
}

function AddressCard({ address, selected, onClick }: { address: AddressOption; selected: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left rounded-2xl border-2 px-4 py-3.5 transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#C6F24E] active:scale-[0.99] ${
        selected
          ? 'border-[#0A3D2C] bg-[#E9F3D8] shadow-[0_0_0_1px_#0A3D2C]'
          : 'border-[#E4E0D6] bg-white hover:border-[#0A3D2C]/30 hover:bg-[#F7F5EF]'
      }`}
    >
      <div className="flex items-center gap-3">
        <span className={`w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0 ${selected ? 'bg-[#0A3D2C] text-[#C6F24E]' : 'bg-[#F7F5EF] text-[#63636E]'}`}>
          <MapPinIcon size={14} />
        </span>
        <div className="flex-1 min-w-0">
          <p className={`text-[13px] font-semibold truncate ${selected ? 'text-[#0A3D2C]' : 'text-[#121216]'}`}>
            {address.label || address.street}
          </p>
          <p className="text-[11px] text-[#63636E] truncate mt-0.5">{address.street}, {address.city}</p>
        </div>
        {selected && (
          <svg className="flex-shrink-0" width="18" height="18" viewBox="0 0 24 24" fill="none">
            <circle cx="12" cy="12" r="10" fill="#0A3D2C" />
            <path d="M8 12l3 3 5-5" stroke="#C6F24E" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        )}
      </div>
    </button>
  );
}

export default function PackageWherePage() {
  const router = useRouter();
  const { pickup, dropoff, recipientName, recipientPhone, isMultiDrop, stops, setPickup, setDropoff, setRecipientName, setRecipientPhone, setIsMultiDrop, addStop, removeStop } = usePackageFlowStore();

  const [savedAddresses, setSavedAddresses] = useState<AddressOption[]>([]);
  const [loadingAddresses, setLoadingAddresses] = useState(true);
  const [gpsLoading, setGpsLoading] = useState(false);
  const [gpsError, setGpsError] = useState('');
  const [editingStop, setEditingStop] = useState<Partial<StopInput> | null>(null);

  useEffect(() => {
    usersApi.listAddresses()
      .then((addresses) => {
        setSavedAddresses(addresses.map((a) => ({ id: a.id, label: a.label || a.street, street: a.street, city: a.city, lat: a.lat, lng: a.lng })));
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
            const res = await fetch(`https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lng}&format=json`, { headers: { 'Accept-Language': 'en' } });
            const geo = await res.json();
            const street = geo.address?.road || geo.display_name?.split(',')[0] || 'Current location';
            const city = geo.address?.city || geo.address?.town || 'Lagos';
            resolve({ id: `gps-${Date.now()}`, label: 'Current location', street, city, lat, lng });
          } catch {
            resolve({ id: `gps-${Date.now()}`, label: 'Current location', street: 'Current location', city: 'Lagos', lat, lng });
          }
          setGpsLoading(false);
        },
        () => { setGpsError('Could not get your location. Please select manually.'); setGpsLoading(false); resolve(null); },
        { timeout: 8000 },
      );
    });
  }

  function saveStop() {
    if (!editingStop?.address || !editingStop.recipientName?.trim() || !editingStop.recipientPhone?.trim()) return;
    addStop({ sequence: editingStop.sequence ?? stops.length + 1, address: editingStop.address, recipientName: editingStop.recipientName, recipientPhone: editingStop.recipientPhone, notes: editingStop.notes ?? '' });
    setEditingStop(null);
  }

  const singleOk = !isMultiDrop && Boolean(pickup && dropoff && recipientName.trim() && recipientPhone.trim());
  const multiOk = isMultiDrop && Boolean(pickup && stops.length >= 1);

  return (
    <main className="min-h-screen bg-[#F7F5EF] flex flex-col">
      <FlowHeader title="Where's it going?" step={1} backHref="/" />

      <div className="flex-1 px-5 py-6 flex flex-col gap-7 max-w-[600px] mx-auto w-full pb-32">

        {/* ── Pickup ── */}
        <section className="flex flex-col gap-3">
          <SectionLabel>Pickup address</SectionLabel>
          {loadingAddresses ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-[62px] rounded-2xl" />
              <Skeleton className="h-[62px] rounded-2xl" />
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {savedAddresses.map((a) => (
                <AddressCard key={a.id} address={a} selected={pickup?.id === a.id} onClick={() => setPickup(a)} />
              ))}
              <button
                type="button"
                onClick={async () => { const a = await getGPS(); if (a) setPickup(a); }}
                disabled={gpsLoading}
                className="flex items-center gap-3 rounded-2xl border-2 border-dashed border-[#E4E0D6] bg-white px-4 py-3.5 text-[13px] font-semibold text-[#0A3D2C] hover:border-[#0A3D2C]/40 hover:bg-[#F7F5EF] transition-all duration-150 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#C6F24E]"
              >
                <span className="w-8 h-8 rounded-xl bg-[#E9F3D8] flex items-center justify-center flex-shrink-0">
                  <CrosshairIcon />
                </span>
                {gpsLoading ? 'Getting your location…' : 'Use my current location'}
              </button>
            </div>
          )}
          {pickup && (
            <p className="text-[12px] text-[#0A3D2C] font-medium flex items-center gap-1.5">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
              {pickup.label}
            </p>
          )}
        </section>

        {/* ── Delivery type ── */}
        <section className="flex flex-col gap-3">
          <SectionLabel>Delivery type</SectionLabel>
          <div className="grid grid-cols-2 gap-2.5">
            <SelectionCard
              selected={!isMultiDrop}
              label="Single drop-off"
              description="One recipient, one address"
              icon={<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0118 0z" /><circle cx="12" cy="10" r="3" /></svg>}
              onClick={() => setIsMultiDrop(false)}
            />
            <SelectionCard
              selected={isMultiDrop}
              label="Multi-drop"
              description="Multiple stops, one trip"
              badge="Save time"
              icon={<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="5" r="2" /><circle cx="5" cy="19" r="2" /><circle cx="19" cy="19" r="2" /><path d="M12 7v4M8.5 17.5L12 11M15.5 17.5L12 11" /></svg>}
              onClick={() => setIsMultiDrop(true)}
            />
          </div>
        </section>

        {/* ── Single drop-off ── */}
        {!isMultiDrop && (
          <>
            <section className="flex flex-col gap-3">
              <SectionLabel>Drop-off address</SectionLabel>
              <div className="flex flex-col gap-2">
                {savedAddresses.map((a) => (
                  <AddressCard key={a.id} address={a} selected={dropoff?.id === a.id} onClick={() => setDropoff(a)} />
                ))}
                <button
                  type="button"
                  onClick={async () => { const a = await getGPS(); if (a) setDropoff(a); }}
                  disabled={gpsLoading}
                  className="flex items-center gap-3 rounded-2xl border-2 border-dashed border-[#E4E0D6] bg-white px-4 py-3.5 text-[13px] font-semibold text-[#0A3D2C] hover:border-[#0A3D2C]/40 hover:bg-[#F7F5EF] transition-all duration-150 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#C6F24E]"
                >
                  <span className="w-8 h-8 rounded-xl bg-[#E9F3D8] flex items-center justify-center flex-shrink-0">
                    <CrosshairIcon />
                  </span>
                  Use my current location
                </button>
              </div>
              {dropoff && (
                <p className="text-[12px] text-[#0A3D2C] font-medium flex items-center gap-1.5">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                  {dropoff.label}
                </p>
              )}
            </section>

            <section className="flex flex-col gap-3">
              <SectionLabel>Who is receiving it?</SectionLabel>
              <ListCard className="flex flex-col gap-3">
                <Input id="recipientName" label="Recipient name" placeholder="e.g. Amaka Obi" value={recipientName} onChange={(e) => setRecipientName(e.target.value)} />
                <Input id="recipientPhone" label="Recipient phone" type="tel" placeholder="08012345678" value={recipientPhone} onChange={(e) => setRecipientPhone(e.target.value)} />
              </ListCard>
            </section>
          </>
        )}

        {/* ── Multi-drop ── */}
        {isMultiDrop && (
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between">
              <SectionLabel>Drop-off stops ({stops.length})</SectionLabel>
              <button
                type="button"
                onClick={() => setEditingStop({ sequence: stops.length + 1 })}
                className="flex items-center gap-1.5 text-[12px] font-semibold text-[#0A3D2C] hover:text-[#0D4E38] transition-colors"
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" />
                </svg>
                Add stop
              </button>
            </div>

            {stops.length === 0 && (
              <div className="rounded-2xl border-2 border-dashed border-[#E4E0D6] bg-white px-5 py-8 flex flex-col items-center gap-2 text-center">
                <span className="w-10 h-10 rounded-xl bg-[#F7F5EF] flex items-center justify-center text-[#9A968D]">
                  <MapPinIcon size={18} />
                </span>
                <p className="text-[13px] font-semibold text-[#121216]">No stops yet</p>
                <p className="text-[12px] text-[#63636E]">Add at least one drop-off address.</p>
              </div>
            )}

            <div className="flex flex-col gap-2">
              {stops.map((stop, i) => (
                <ListCard
                  key={stop.sequence}
                  className="px-4 py-3.5 flex items-center gap-3"
                  style={{ animation: `fadeUp 0.2s cubic-bezier(0.16,1,0.3,1) ${i * 40}ms both` }}
                >
                  <span className="w-8 h-8 rounded-full bg-[#0A3D2C] flex items-center justify-center font-display font-bold text-[#C6F24E] text-[13px] flex-shrink-0">
                    {stop.sequence}
                  </span>
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-semibold text-[#121216] truncate">{stop.recipientName}</p>
                    <p className="text-[11px] text-[#63636E] truncate">{stop.address.street} · {stop.recipientPhone}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeStop(stop.sequence)}
                    className="w-7 h-7 rounded-lg flex items-center justify-center text-[#9A968D] hover:text-[#DC2626] hover:bg-[#FEF2F2] transition-colors flex-shrink-0"
                    aria-label="Remove stop"
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                      <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </button>
                </ListCard>
              ))}
            </div>

            {/* Stop editor */}
            {editingStop !== null && (
              <div
                className="bg-white rounded-2xl border-2 border-[#0A3D2C] p-5 flex flex-col gap-4"
                style={{ animation: 'fadeUp 0.2s cubic-bezier(0.16,1,0.3,1) both' }}
              >
                <div className="flex items-center justify-between">
                  <p className="font-display font-semibold text-[15px] text-[#121216]">Stop {editingStop.sequence}</p>
                  <button type="button" onClick={() => setEditingStop(null)} className="text-[#9A968D] hover:text-[#121216] transition-colors">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
                      <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                    </svg>
                  </button>
                </div>

                <div className="flex flex-col gap-2">
                  <p className="text-[11px] font-semibold text-[#9A968D] tracking-[0.7px] uppercase">Drop-off address</p>
                  {savedAddresses.map((a) => (
                    <AddressCard key={a.id} address={a} selected={editingStop.address?.id === a.id} onClick={() => setEditingStop((s) => ({ ...s, address: a }))} />
                  ))}
                </div>

                <Input label="Recipient name" placeholder="e.g. Emeka Obi" value={editingStop.recipientName ?? ''} onChange={(e) => setEditingStop((s) => ({ ...s, recipientName: e.target.value }))} />
                <Input label="Recipient phone" type="tel" placeholder="08012345678" value={editingStop.recipientPhone ?? ''} onChange={(e) => setEditingStop((s) => ({ ...s, recipientPhone: e.target.value }))} />
                <Input label="Notes (optional)" placeholder="e.g. Leave at gate, call on arrival" value={editingStop.notes ?? ''} onChange={(e) => setEditingStop((s) => ({ ...s, notes: e.target.value }))} />

                <Button
                  variant="primary"
                  size="md"
                  disabled={!editingStop.address || !editingStop.recipientName?.trim() || !editingStop.recipientPhone?.trim()}
                  onClick={saveStop}
                  className="w-full"
                >
                  Save stop
                </Button>
              </div>
            )}
          </section>
        )}

        {gpsError && (
          <p className="text-[12px] text-[#DC2626] flex items-center gap-1.5" role="alert">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" /></svg>
            {gpsError}
          </p>
        )}
      </div>

      {/* Sticky CTA */}
      <div className="fixed bottom-0 left-0 right-0 bg-[#F7F5EF]/95 backdrop-blur-sm border-t border-[#E4E0D6] px-5 py-4 pb-safe-bottom">
        <div className="max-w-[600px] mx-auto flex items-center justify-between gap-4">
          <div className="flex-1 min-w-0">
            {(singleOk || multiOk) && (
              <p className="text-[12px] text-[#0A3D2C] font-medium truncate">
                {isMultiDrop ? `${stops.length} stop${stops.length !== 1 ? 's' : ''} added` : `${pickup?.label} → ${dropoff?.label}`}
              </p>
            )}
            <p className="text-[11px] text-[#9A968D]">Nothing is paid yet</p>
          </div>
          <Button
            variant="primary"
            size="lg"
            disabled={!singleOk && !multiOk}
            onClick={() => router.push('/package/what')}
            className="flex-shrink-0"
          >
            Continue
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2.5} strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </Button>
        </div>
      </div>

      <style>{`
        /* package/where uses inline stagger for dynamic stop list items — kept intentionally */
      `}</style>
    </main>
  );
}
