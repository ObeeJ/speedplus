'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useDriverStore } from '../lib/store/driver.store';
import { dispatchApi, paycodesApi, walletApi } from '@speedplus/api-client';
import { useQuery } from '@tanstack/react-query';

const WS_URL = process.env['NEXT_PUBLIC_WS_URL'] ?? 'ws://localhost:8000/api/v1/ws';
const LOCATION_INTERVAL_MS = 10_000;

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

// ── Icons ─────────────────────────────────────────────────────────────────────
function HomeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 11l9-8 9 8v9a1 1 0 01-1 1h-5v-6h-6v6H4a1 1 0 01-1-1v-9z" />
    </svg>
  );
}
function JobIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 21s-7-5.5-7-11a7 7 0 0114 0c0 5.5-7 11-7 11z" />
      <circle cx="12" cy="10" r="2.5" />
    </svg>
  );
}
function EarnIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="6" width="18" height="13" rx="3" />
      <path d="M3 10h18" />
    </svg>
  );
}
function MeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21a8 8 0 0116 0" />
    </svg>
  );
}

const STAGE_LABELS = ['Accepted — ride to pickup', 'Arrived at pickup', 'Package picked up', 'Arrived at drop-off', 'Delivered ✓'];
const STAGE_CTAS = ['', "I've arrived at pickup", 'I have the package', "I've arrived at drop-off", 'Confirm delivery', ''];

export default function DriverAppPage() {
  const router = useRouter();
  const { tab, online, pendingOffer, activeJob, setTab, setOnline, setPendingOffer, setActiveJob, advanceJobStage, confirmStop, clearJob } = useDriverStore();

  const wsRef = useRef<WebSocket | null>(null);
  const locationRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [deliveryCode, setDeliveryCode] = useState('');
  const [confirmError, setConfirmError] = useState('');
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [cashoutLoading, setCashoutLoading] = useState(false);
  const [cashoutDone, setCashoutDone] = useState(false);

  // Fetch real earnings
  const { data: walletData } = useQuery({
    queryKey: ['driver-wallet'],
    queryFn: () => walletApi.getBalance(),
    enabled: tab === 'earn',
    staleTime: 30_000,
  });

  // Location updates when online
  const sendLocation = useCallback(() => {
    if (!online) return;
    navigator.geolocation?.getCurrentPosition((pos) => {
      dispatchApi.updateLocation(pos.coords.latitude, pos.coords.longitude).catch(() => {});
    });
  }, [online]);

  useEffect(() => {
    if (online) {
      sendLocation();
      locationRef.current = setInterval(sendLocation, LOCATION_INTERVAL_MS);
    } else {
      if (locationRef.current) clearInterval(locationRef.current);
    }
    return () => { if (locationRef.current) clearInterval(locationRef.current); };
  }, [online, sendLocation]);

  // WS connection for offer push
  useEffect(() => {
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;

    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data as string) as { event: string; data: Record<string, unknown> };
        if (msg.event === 'new_offer') {
          setPendingOffer({
            offerId: msg.data.offerId as string,
            orderId: msg.data.orderId as string,
            vertical: msg.data.vertical as string,
            totalKobo: msg.data.totalKobo as number,
            pickupAddress: (msg.data.pickupAddress as string) ?? 'Pickup address',
            dropoffAddress: (msg.data.dropoffAddress as string) ?? 'Drop-off address',
            distanceKm: (msg.data.distanceKm as number) ?? 0,
          });
        }
      } catch {
        // ignore malformed
      }
    };

    return () => ws.close();
  }, [setPendingOffer]);

  async function handleAcceptOffer() {
    if (!pendingOffer) return;
    try {
      await dispatchApi.acceptOffer(pendingOffer.offerId);
      setActiveJob({
        orderId: pendingOffer.orderId,
        stage: 1,
        customerName: 'Customer',
        customerPhone: '',
        pickupAddress: pendingOffer.pickupAddress,
        dropoffAddress: pendingOffer.dropoffAddress,
        totalKobo: pendingOffer.totalKobo,
        deliveryCode: '',
        paymentMethod: 'wallet',
        stops: [],
        currentStopIndex: 0,
      });
      setPendingOffer(null);
      setTab('job');
    } catch {
      // offer taken — clear it
      setPendingOffer(null);
    }
  }

  async function handleRejectOffer() {
    if (!pendingOffer) return;
    await dispatchApi.rejectOffer(pendingOffer.offerId).catch(() => {});
    setPendingOffer(null);
  }

  async function handleAdvanceStage() {
    if (!activeJob) return;
    // Stage 4 = POD — wait for code entry
    if (activeJob.stage === 4) return;
    advanceJobStage();
  }

  async function handleConfirmDelivery() {
    if (!activeJob || !deliveryCode.trim()) return;
    setConfirmLoading(true);
    setConfirmError('');
    try {
      const isMultiDrop = activeJob.stops.length > 0;
      if (isMultiDrop) {
        // Confirm the current stop
        const currentStop = activeJob.stops[activeJob.currentStopIndex];
        if (!currentStop) return;
        // Call the stops confirm endpoint
        const { data } = await (await import('@speedplus/api-client')).apiClient.post(
          `/orders/${activeJob.orderId}/stops/confirm`,
          { sequence: currentStop.sequence, code: deliveryCode.trim() },
        );
        if ((data as { success: boolean }).success) {
          confirmStop(currentStop.sequence);
          advanceJobStage();
          setDeliveryCode('');
        }
      } else {
        // Single drop-off — use existing confirmByCode
        await paycodesApi.confirmByCode(activeJob.orderId, deliveryCode.trim());
        advanceJobStage();
        setDeliveryCode('');
      }
    } catch (err) {
      setConfirmError(err instanceof Error ? err.message : 'Invalid code. Try again.');
    } finally {
      setConfirmLoading(false);
    }
  }

  async function handleToggleOnline() {
    const next = !online;
    setOnline(next);
  }

  async function handleCashout() {
    if (!walletData || cashoutDone) return;
    setCashoutLoading(true);
    try {
      // EWA cashout — full unpaid balance
      const key = `cashout-${Date.now()}`;
      const { data } = await (await import('@speedplus/api-client')).apiClient.post(
        '/earnings/cashout',
        { amountKobo: walletData.balanceKobo },
        { headers: { 'Idempotency-Key': key } },
      );
      if ((data as { success: boolean }).success) setCashoutDone(true);
    } catch {
      // show nothing — user can retry
    } finally {
      setCashoutLoading(false);
    }
  }

  const showOffer = online && pendingOffer && !activeJob;
  const isJob = tab === 'job' && activeJob;
  const showPod = activeJob?.stage === 4;

  return (
    <main className="min-h-screen flex justify-center p-3 min-[500px]:p-6" style={{ background: '#E9E6DD' }}>
      <div className="w-full max-w-[430px] bg-sand rounded-3xl overflow-hidden shadow-[0_24px_60px_rgba(10,61,44,.18)] flex flex-col min-h-[780px]">

        {/* Header */}
        <div className="bg-emerald px-5 py-4 flex items-center gap-3">
          <span className="font-display font-bold text-lg text-sand tracking-tight">
            speed<span className="text-lime">+</span>{' '}
            <span className="font-medium text-xs text-sand/60">RIDER</span>
          </span>
          <span className="flex-1" />
          <button
            onClick={handleToggleOnline}
            className={`flex items-center gap-2 rounded-full px-3.5 py-1.5 transition-colors duration-200 ${online ? 'bg-lime' : 'bg-sand/[.14]'}`}
          >
            <span className={`w-2 h-2 rounded-full ${online ? 'bg-emerald animate-pulse' : 'bg-[#9A968D]'}`} />
            <span className={`text-xs font-bold ${online ? 'text-emerald' : 'text-sand/70'}`}>
              {online ? 'Online' : 'Offline'}
            </span>
          </button>
        </div>

        {/* Home tab */}
        {tab === 'home' && (
          <div className="flex-1 px-5 py-4 flex flex-col gap-3.5 overflow-y-auto">
            {/* Offer card */}
            {showOffer && pendingOffer && (
              <div
                className="bg-emerald rounded-[18px] p-4 flex flex-col gap-3 shadow-[0_12px_30px_rgba(10,61,44,.3)]"
                style={{ animation: 'slideDown 0.3s cubic-bezier(0.16,1,0.3,1) both' }}
              >
                <div className="flex items-center gap-2.5">
                  <span className="w-2 h-2 rounded-full bg-lime animate-pulse" />
                  <span className="font-display text-[15px] font-semibold text-sand capitalize">
                    New delivery — {pendingOffer.vertical}
                  </span>
                  <span className="ml-auto font-display text-lg font-bold text-lime">
                    {naira(pendingOffer.totalKobo)}
                  </span>
                </div>
                <div className="flex flex-col gap-1.5 bg-sand/[.08] rounded-xl px-3.5 py-2.5">
                  <span className="flex gap-2.5 text-xs text-sand">
                    <b className="text-lime">A</b>
                    {pendingOffer.pickupAddress}
                  </span>
                  <span className="flex gap-2.5 text-xs text-sand">
                    <b className="text-amber">B</b>
                    {pendingOffer.dropoffAddress}
                    <span className="ml-auto text-sand/55">{pendingOffer.distanceKm.toFixed(1)} km</span>
                  </span>
                </div>
                <div className="flex gap-2.5">
                  <button
                    onClick={handleRejectOffer}
                    className="flex-1 text-center font-display text-[13px] font-semibold text-sand border-[1.5px] border-sand/30 rounded-xl py-3 hover:bg-sand/10 transition-colors"
                  >
                    Decline
                  </button>
                  <button
                    onClick={handleAcceptOffer}
                    className="flex-[2] text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-xl py-3 hover:bg-lime-600 transition-colors"
                  >
                    Accept — {naira(pendingOffer.totalKobo)}
                  </button>
                </div>
              </div>
            )}

            {/* Waiting state */}
            {!showOffer && (
              <div className="border-[1.5px] border-dashed border-line rounded-2xl p-5 flex flex-col items-center gap-1.5 text-center">
                <span className="text-[13px] font-semibold text-mid">
                  {online
                    ? activeJob
                      ? 'You have an active delivery'
                      : 'Waiting for jobs…'
                    : 'You are offline'}
                </span>
                <span className="text-[11px] text-[#9A968D]">
                  {online
                    ? activeJob
                      ? 'Open the Delivery tab to continue.'
                      : 'Stay near busy areas for faster offers.'
                    : 'Go online to receive delivery offers.'}
                </span>
              </div>
            )}
          </div>
        )}

        {/* Job tab */}
        {isJob && activeJob && (
          <div className="flex-1 px-5 py-4 flex flex-col gap-3 overflow-y-auto">
            {/* Customer card */}
            <div className="flex items-center gap-3 bg-white border border-line rounded-[14px] px-3.5 py-3">
              <span className="w-10 h-10 rounded-full bg-tile flex items-center justify-center font-display font-bold text-emerald">
                {activeJob.customerName[0]}
              </span>
              <span className="flex-1 flex flex-col">
                <span className="text-[13px] font-semibold">{activeJob.customerName}</span>
                <span className="text-[10.5px] text-mid">
                  {naira(activeJob.totalKobo)} · {activeJob.paymentMethod === 'wallet' ? 'Wallet' : 'Pay on arrival'}
                  {activeJob.stops.length > 0 && ` · ${activeJob.stops.length} stops`}
                </span>
              </span>
              {activeJob.customerPhone && (
                <a
                  href={`tel:${activeJob.customerPhone}`}
                  className="w-[38px] h-[38px] rounded-[11px] bg-tile flex items-center justify-center hover:bg-[#DCEDC2] transition-colors"
                  aria-label="Call customer"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
                    <path d="M4 6c0 8 6 14 14 14l2.5-3-4-2-2 2c-2.5-1.2-4.3-3-5.5-5.5l2-2-2-4L6 4" />
                  </svg>
                </a>
              )}
            </div>

            {/* Stage progress */}
            <div className="bg-white border border-line rounded-[14px] px-3.5 py-3.5 flex flex-col gap-2">
              {STAGE_LABELS.map((label, i) => {
                const done = activeJob.stage > i;
                const current = activeJob.stage === i + 1;
                return (
                  <span key={label} className="flex items-center gap-2.5">
                    <span
                      className="w-2.5 h-2.5 rounded-full border-[2.5px] transition-all duration-300"
                      style={{
                        background: done ? '#0A3D2C' : '#FFFFFF',
                        borderColor: done ? '#C6F24E' : '#E4E0D6',
                      }}
                    />
                    <span
                      className="text-[12.5px] font-semibold transition-colors duration-300"
                      style={{ color: current ? '#0A3D2C' : done ? '#121216' : '#9A968D' }}
                    >
                      {label}
                    </span>
                  </span>
                );
              })}
            </div>

            {/* POD — delivery code entry */}
            {showPod && (
              <div className="bg-[#FFF7E6] border border-[#F0DFB4] rounded-[14px] px-3.5 py-3 flex flex-col gap-2.5">
                {activeJob.stops.length > 0 ? (
                  <>
                    <span className="text-[12.5px] font-bold">
                      Stop {activeJob.currentStopIndex + 1} of {activeJob.stops.length}
                    </span>
                    {activeJob.stops[activeJob.currentStopIndex] && (
                      <div className="bg-white rounded-[10px] px-3 py-2 flex flex-col gap-0.5">
                        <span className="text-[12px] font-semibold">
                          {activeJob.stops[activeJob.currentStopIndex]!.recipientName ?? 'Recipient'}
                        </span>
                        {activeJob.stops[activeJob.currentStopIndex]!.recipientPhone && (
                          <a
                            href={`tel:${activeJob.stops[activeJob.currentStopIndex]!.recipientPhone}`}
                            className="text-[11px] text-emerald font-semibold"
                          >
                            {activeJob.stops[activeJob.currentStopIndex]!.recipientPhone}
                          </a>
                        )}
                        {activeJob.stops[activeJob.currentStopIndex]!.notes && (
                          <span className="text-[11px] text-mid italic">
                            {activeJob.stops[activeJob.currentStopIndex]!.notes}
                          </span>
                        )}
                      </div>
                    )}
                  </>
                ) : (
                  <span className="text-[12.5px] font-bold">Proof of delivery</span>
                )}
                <span className="text-[11px] text-mid">
                  Ask the recipient for their 6-digit delivery code.
                </span>
                <input
                  type="text"
                  inputMode="numeric"
                  maxLength={6}
                  placeholder="000000"
                  value={deliveryCode}
                  onChange={(e) => setDeliveryCode(e.target.value.replace(/\D/g, ''))}
                  className="h-11 w-full rounded-[11px] border border-line bg-white px-4 text-center font-display text-xl font-bold tracking-[8px] text-emerald focus:outline-none focus:ring-2 focus:ring-emerald"
                />
                {confirmError && (
                  <p className="text-xs text-[#DC2626]" role="alert">{confirmError}</p>
                )}
                <button
                  onClick={handleConfirmDelivery}
                  disabled={deliveryCode.length !== 6 || confirmLoading}
                  className="w-full text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-[13px] py-3 hover:bg-lime-600 transition-colors disabled:opacity-50"
                >
                  {confirmLoading ? 'Confirming…' : 'Confirm delivery'}
                </button>
              </div>
            )}

            {/* CTA */}
            {!showPod && activeJob.stage < 5 && (
              <div className="mt-auto">
                <button
                  onClick={handleAdvanceStage}
                  className="w-full text-center font-display text-sm font-semibold text-emerald bg-lime rounded-[13px] py-3.5 hover:bg-lime-600 transition-colors"
                >
                  {STAGE_CTAS[activeJob.stage] || 'Continue'}
                </button>
              </div>
            )}
          </div>
        )}

        {/* Earn tab */}
        {tab === 'earn' && (
          <div className="flex-1 px-5 py-4 flex flex-col gap-3.5 overflow-y-auto">
            <div className="bg-emerald rounded-[18px] p-4.5 flex flex-col gap-1">
              <span className="text-[11px] font-semibold text-sand/60 tracking-[.6px]">WALLET BALANCE</span>
              <span className="font-display text-[34px] font-bold text-lime tracking-tight">
                {walletData ? naira(walletData.balanceKobo) : '—'}
              </span>
              <button
                onClick={handleCashout}
                disabled={cashoutLoading || cashoutDone || !walletData?.balanceKobo}
                className="mt-2.5 text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-xl py-3 hover:bg-lime-600 transition-colors disabled:opacity-50"
              >
                {cashoutDone ? '✓ Sent to your bank' : cashoutLoading ? 'Processing…' : 'Cash out to bank'}
              </button>
            </div>
          </div>
        )}

        {/* Me tab */}
        {tab === 'me' && (
          <div className="flex-1 px-5 py-4 flex flex-col gap-3.5 overflow-y-auto">
            <div className="flex items-center gap-3.5 bg-white border border-line rounded-2xl p-4">
              <span className="w-[54px] h-[54px] rounded-full bg-emerald flex items-center justify-center text-lime font-display font-bold text-xl">
                R
              </span>
              <span className="flex flex-col gap-0.5">
                <span className="font-display text-base font-semibold">Your profile</span>
                <span className="text-[11.5px] text-mid">Rider · SpeedPlus</span>
              </span>
            </div>
            <button
              onClick={() => router.push('/login')}
              className="text-sm font-semibold text-[#DC2626] text-left"
            >
              Sign out
            </button>
          </div>
        )}

        {/* Bottom nav */}
        <div className="bg-white border-t border-line px-5 pt-2 pb-3 flex justify-around">
          {(
            [
              { id: 'home', label: 'Home', Icon: HomeIcon },
              { id: 'job', label: 'Delivery', Icon: JobIcon },
              { id: 'earn', label: 'Earnings', Icon: EarnIcon },
              { id: 'me', label: 'Me', Icon: MeIcon },
            ] as const
          ).map(({ id, label, Icon }) => {
            const active = tab === id;
            return (
              <button
                key={id}
                onClick={() => setTab(id)}
                className={`flex flex-col items-center gap-0.5 text-[9.5px] transition-colors duration-150 ${active ? 'font-bold text-emerald' : 'font-medium text-mid'}`}
              >
                <Icon />
                {label}
              </button>
            );
          })}
        </div>
      </div>

      <style>{`
        @keyframes slideDown {
          from { opacity: 0; transform: translateY(-12px); }
          to   { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </main>
  );
}
