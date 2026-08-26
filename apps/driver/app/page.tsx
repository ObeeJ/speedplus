'use client';

import { useEffect, useRef, useState, useCallback, type ReactElement } from 'react';
import { useRouter } from 'next/navigation';
import { useDriverStore } from '../lib/store/driver.store';
import { useDriverAuthStore } from '../lib/store/auth.store';
import { dispatchApi, paycodesApi, walletApi, authApi, usersApi, ordersApi, earningsApi } from '@speedplus/api-client';
import {
  SparkIcon,
  BoxStackIcon,
  RocketIcon,
  TrophyIcon,
  StarIcon,
  ShieldCheckIcon,
  DashboardIcon,
  RunIcon,
  WalletIcon,
  UsersIcon,
  Badge,
  StatusSteps,
  iconColors,
  type DuotoneIconProps,
} from '@speedplus/ui';
import { ProofCapture } from './components/proof-capture';
import { buildWsUrl, buildWsProtocols } from '@speedplus/api-client';
import { useQuery } from '@tanstack/react-query';

const LOCATION_INTERVAL_MS = 10_000;

function naira(kobo: number) {
  return `₦${(kobo / 100).toLocaleString('en-NG', { minimumFractionDigits: 0 })}`;
}

// ── Badge metadata ────────────────────────────────────────────────────────────
// Keys mirror driver_badges.badge_type. Duotone icons, not emoji — the accent
// stroke carries the second color. Module-scope so it isn't rebuilt per render.
type BadgeMeta = {
  label: string;
  Icon: (props: DuotoneIconProps) => ReactElement;
  color: string;
  accent: string;
};

const BADGE_META: Record<string, BadgeMeta> = {
  first_delivery:   { label: 'First delivery',  Icon: SparkIcon,       color: iconColors.amberBg, accent: iconColors.amberAccent },
  '10_deliveries':  { label: '10 deliveries',   Icon: BoxStackIcon,    color: iconColors.tile,    accent: iconColors.accent },
  '50_deliveries':  { label: '50 deliveries',   Icon: RocketIcon,      color: iconColors.tile,    accent: iconColors.accent },
  '100_deliveries': { label: '100 deliveries',  Icon: TrophyIcon,      color: iconColors.amberBg, accent: iconColors.amberAccent },
  top_rated:        { label: 'Top rated',       Icon: StarIcon,        color: iconColors.amberBg, accent: iconColors.amberAccent },
  zero_complaints:  { label: 'Zero complaints', Icon: ShieldCheckIcon, color: iconColors.tile,    accent: iconColors.accent },
};

// Unknown badge_type from a newer API build still renders something sane.
const FALLBACK_BADGE = (badgeType: string): BadgeMeta => ({
  label: badgeType.replace(/_/g, ' '),
  Icon: SparkIcon,
  color: iconColors.sand,
  accent: iconColors.mutedAccent,
});

// ── Icons ─────────────────────────────────────────────────────────────────────
function HomeIcon({ active = false }: { active?: boolean }) {
  return <DashboardIcon size={18} active={active} color={active ? iconColors.lime : iconColors.stroke} accent={iconColors.accent} />;
}
function JobIcon({ active = false }: { active?: boolean }) {
  return <RunIcon size={18} active={active} color={active ? iconColors.lime : iconColors.stroke} accent={iconColors.accent} />;
}
function EarnIcon({ active = false }: { active?: boolean }) {
  return <WalletIcon size={18} active={active} color={active ? iconColors.lime : iconColors.stroke} accent={iconColors.accent} />;
}
function MeIcon({ active = false }: { active?: boolean }) {
  return <UsersIcon size={18} active={active} color={active ? iconColors.lime : iconColors.stroke} accent={iconColors.accent} />;
}

const STAGE_LABELS = ['Accepted — ride to pickup', 'Arrived at pickup', 'Package picked up', 'Arrived at drop-off', 'Delivered ✓'];
const STAGE_CTAS = ['', "I've arrived at pickup", 'I have the package', "I've arrived at drop-off", 'Confirm delivery', ''];

export default function DriverAppPage() {
  const router = useRouter();
  const { tab, online, pendingOffer, activeJob, setTab, setOnline, setPendingOffer, setActiveJob, advanceJobStage, confirmStop, clearJob } = useDriverStore();
  const clearAuth = useDriverAuthStore((s) => s.clearAuth);
  const [resolvePayload, setResolvePayload] = useState('');
  const [resolveResult, setResolveResult] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const locationRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [deliveryCode, setDeliveryCode] = useState('');
  const [confirmError, setConfirmError] = useState('');
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [cashoutLoading, setCashoutLoading] = useState(false);
  const [cashoutDone, setCashoutDone] = useState(false);
  const [emptyCollected, setEmptyCollected] = useState(false);
  const [emptyCylinderSerial, setEmptyCylinderSerial] = useState('');

  // Fetch real earnings
  const { data: walletData } = useQuery({
    queryKey: ['driver-wallet'],
    queryFn: () => walletApi.getBalance(),
    enabled: tab === 'earn',
    staleTime: 30_000,
  });

  const { data: bankAccount, refetch: refetchBank } = useQuery({
    queryKey: ['driver-bank-account'],
    queryFn: () => earningsApi.getBankAccount(),
    enabled: tab === 'earn',
    staleTime: Infinity,
  });

  const [showBankForm, setShowBankForm] = useState(false);
  const { data: bankList } = useQuery({
    queryKey: ['banks'],
    queryFn: () => earningsApi.listBanks(),
    enabled: showBankForm,
    staleTime: 24 * 60 * 60 * 1000,
  });
  const [bankDraft, setBankDraft] = useState({ bankCode: '', bankName: '', accountNumber: '' });
  const [resolvedName, setResolvedName] = useState<string | null>(null);
  const [bankError, setBankError] = useState('');

  const [bankLoading, setBankLoading] = useState(false);
  const { data: badgesData } = useQuery({
    queryKey: ['driver-badges'],
    queryFn: async () => {
      const profile = await usersApi.getDriverProfile();
      return usersApi.getDriverBadges(profile.userId);
    },
    enabled: tab === 'me',
    staleTime: 60_000,
  });

  // Subscribe to active order WS for cancellation / status updates.
  //
  // Keyed on the order id alone, not the activeJob object: the store hands back
  // a new object identity on every unrelated field change, which would tear
  // down and re-open the socket mid-delivery. Hoisting the primitive lets the
  // dependency array be exhaustive and honest rather than suppressed.
  const activeOrderId = activeJob?.orderId;
  useEffect(() => {
    if (!activeOrderId) return;
    const ws = new WebSocket(buildWsUrl(), buildWsProtocols());
    ws.onopen = () => ws.send(JSON.stringify({ action: 'subscribe', channel: `order:${activeOrderId}` }));
    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data as string) as { event: string };
        if (msg.event === 'order_cancelled') {
          clearJob();
          setTab('home');
        }
      } catch { /* ignore */ }
    };
    return () => ws.close();
  }, [activeOrderId, clearJob, setTab]);

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
    const ws = new WebSocket(buildWsUrl(), buildWsProtocols());
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

      // Fetch stops from API for multi-drop orders
      let stops: import('../lib/store/driver.store').JobStop[] = [];
      if ((pendingOffer.stopCount ?? 0) > 1) {
        try {
          const raw = await ordersApi.getStops(pendingOffer.orderId);
          stops = raw.map((s) => ({ ...s, status: s.status as 'pending' | 'confirmed' }));
        } catch { /* non-fatal — driver proceeds without stop details */ }
      }

      setActiveJob({
        orderId: pendingOffer.orderId,
        vertical: pendingOffer.vertical,
        stage: 1,
        customerName: 'Customer',
        customerPhone: '',
        pickupAddress: pendingOffer.pickupAddress,
        dropoffAddress: pendingOffer.dropoffAddress,
        totalKobo: pendingOffer.totalKobo,
        deliveryCode: '',
        paymentMethod: 'wallet',
        stops,
        currentStopIndex: 0,
      });
      setPendingOffer(null);
      setTab('job');
    } catch {
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
        const currentStop = activeJob.stops[activeJob.currentStopIndex];
        if (!currentStop) return;
        await ordersApi.confirmStop(activeJob.orderId, {
          sequence: currentStop.sequence,
          code: deliveryCode.trim(),
          ...(activeJob.vertical === 'gas' && {
            emptyCollected,
            ...(emptyCylinderSerial.trim() && { emptyCylinderSerial: emptyCylinderSerial.trim() }),
          }),
        });
        confirmStop(currentStop.sequence);
        advanceJobStage();
        setDeliveryCode('');
        setEmptyCollected(false);
        setEmptyCylinderSerial('');
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

  async function handleResolveAccount() {
    if (!bankDraft.bankCode || !bankDraft.accountNumber) return;
    setBankLoading(true);
    setBankError('');
    setResolvedName(null);
    try {
      const result = await earningsApi.resolveAccount(bankDraft.bankCode, bankDraft.accountNumber);
      setResolvedName(result.accountName);
    } catch (e) {
      setBankError(e instanceof Error ? e.message : 'Could not verify account');
    } finally {
      setBankLoading(false);
    }
  }

  async function handleSaveBankAccount() {
    if (!resolvedName) return;
    setBankLoading(true);
    setBankError('');
    try {
      await earningsApi.saveBankAccount(bankDraft);
      await refetchBank();
      setShowBankForm(false);
      setResolvedName(null);
      setBankDraft({ bankCode: '', bankName: '', accountNumber: '' });
    } catch (e) {
      setBankError(e instanceof Error ? e.message : 'Failed to save account');
    } finally {
      setBankLoading(false);
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
      await earningsApi.cashout(walletData.balanceKobo, key);
      setCashoutDone(true);
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
    <main className="min-h-screen flex justify-center p-3 min-[500px]:p-6 bg-sand">
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
            <span className={`w-2 h-2 rounded-full ${online ? 'bg-emerald animate-pulse' : 'bg-mid/60'}`} />
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
                className="animate-slide-down bg-emerald rounded-[18px] p-4 flex flex-col gap-3 shadow-[0_12px_30px_rgba(10,61,44,.3)]"
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
                <span className="text-[11px] text-mid">
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
                  className="w-[38px] h-[38px] rounded-[11px] bg-tile flex items-center justify-center hover:bg-tile/70 transition-colors"
                  aria-label="Call customer"
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke={iconColors.emerald} strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
                    <path d="M4 6c0 8 6 14 14 14l2.5-3-4-2-2 2c-2.5-1.2-4.3-3-5.5-5.5l2-2-2-4L6 4" />
                  </svg>
                </a>
              )}
            </div>

            {/* Stage progress */}
            <div className="bg-white border border-line rounded-[14px] px-3.5 py-3.5">
              <StatusSteps
                steps={STAGE_LABELS.map((label) => ({ label }))}
                currentIndex={activeJob.stage - 1}
              />
            </div>

            {/* POD — delivery code entry */}
            {showPod && (
              <div className="bg-amber/10 border border-amber/30 rounded-[14px] px-3.5 py-3 flex flex-col gap-2.5">
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
                {/* Gas orders: weight photo is required before the delivery code */}
                {activeJob.vertical === 'gas' ? (
                  <>
                    <ProofCapture
                      orderId={activeJob.orderId}
                      kind="weight_photo"
                      label="Weigh the cylinder at the door — photograph the scale reading"
                    />
                    <label className="flex items-center gap-2 bg-white rounded-[10px] px-3 py-2.5 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={emptyCollected}
                        onChange={(e) => setEmptyCollected(e.target.checked)}
                        className="w-4 h-4 accent-emerald"
                      />
                      <span className="text-[12.5px] font-semibold">I collected the empty cylinder</span>
                    </label>
                    {emptyCollected && (
                      <input
                        type="text"
                        value={emptyCylinderSerial}
                        onChange={(e) => setEmptyCylinderSerial(e.target.value)}
                        placeholder="Empty cylinder serial (optional)"
                        className="bg-white border border-line rounded-[10px] px-3 py-2.5 text-[12.5px] outline-none focus:border-emerald" aria-label="Empty cylinder serial (optional)"/>
                    )}
                  </>
                ) : (
                  <ProofCapture
                    orderId={activeJob.orderId}
                    kind="dropoff_photo"
                    label="Photograph the sealed package at drop-off"
                  />
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
                  className="h-11 w-full rounded-[11px] border border-line bg-white px-4 text-center font-display text-xl font-bold tracking-[8px] text-emerald focus:outline-none focus:ring-2 focus:ring-emerald" aria-label="000000"/>
                {confirmError && (
                  <p className="text-xs text-red-600" role="alert">{confirmError}</p>
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
                disabled={cashoutLoading || cashoutDone || !walletData?.balanceKobo || !bankAccount}
                className="mt-2.5 text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-xl py-3 hover:bg-lime-600 transition-colors disabled:opacity-50"
              >
                {cashoutDone ? '✓ Sent to your bank' : cashoutLoading ? 'Processing…' : !bankAccount ? 'Add bank account to cash out' : 'Cash out to bank'}
              </button>
            </div>

            {/* Bank account */}
            {bankAccount ? (
              <div className="bg-white border border-line rounded-2xl px-4 py-3 flex items-center gap-3">
                <span className="flex-1 flex flex-col">
                  <span className="text-[13px] font-semibold">{bankAccount.accountName}</span>
                  <span className="text-[11px] text-mid">{bankAccount.bankName} · ****{bankAccount.accountNumber.slice(-4)}</span>
                </span>
                <button onClick={() => setShowBankForm(true)} className="text-[11.5px] font-semibold text-emerald">Change</button>
              </div>
            ) : (
              <button
                onClick={() => setShowBankForm(true)}
                className="w-full border-[1.5px] border-dashed border-line rounded-2xl px-4 py-3.5 text-[13px] font-semibold text-mid hover:border-emerald hover:text-emerald transition-colors"
              >
                + Add bank account to receive payouts
              </button>
            )}

            {showBankForm && (
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">BANK ACCOUNT</span>
                <select
                  aria-label="Bank"
                  value={bankDraft.bankCode}
                  onChange={(e) => {
                    const selected = bankList?.find((b) => b.code === e.target.value);
                    setBankDraft((d) => ({ ...d, bankCode: e.target.value, bankName: selected?.name ?? '' }));
                    setResolvedName(null);
                  }}
                  className="border border-line rounded-xl px-3 py-2.5 text-[13px] bg-white focus:outline-none focus:border-emerald"
                >
                  <option value="">Select bank…</option>
                  {(bankList ?? []).map((b) => (
                    <option key={b.code} value={b.code}>{b.name}</option>
                  ))}
                </select>
                <input
                  placeholder="Account number"
                  inputMode="numeric"
                  maxLength={10}
                  value={bankDraft.accountNumber}
                  onChange={(e) => { setBankDraft((d) => ({ ...d, accountNumber: e.target.value.replace(/\D/g, '') })); setResolvedName(null); }}
                  className="border border-line rounded-xl px-3 py-2.5 text-[13px] focus:outline-none focus:border-emerald" aria-label="Account number"/>
                {resolvedName && (
                  <div className="bg-tile rounded-xl px-3 py-2.5">
                    <span className="text-[11px] font-semibold text-mid">Account name</span>
                    <p className="text-[13px] font-bold text-emerald">{resolvedName}</p>
                    <p className="text-[11px] text-mid mt-0.5">Confirm this is correct before saving.</p>
                  </div>
                )}
                {bankError && <p className="text-[12px] text-red-600" role="alert">{bankError}</p>}
                <div className="flex gap-2">
                  {!resolvedName ? (
                    <button
                      onClick={handleResolveAccount}
                      disabled={!bankDraft.bankCode || !bankDraft.accountNumber || bankLoading}
                      className="flex-1 bg-emerald text-lime font-display font-semibold text-[13px] rounded-xl py-2.5 disabled:opacity-50"
                    >
                      {bankLoading ? 'Verifying…' : 'Verify account'}
                    </button>
                  ) : (
                    <button
                      onClick={handleSaveBankAccount}
                      disabled={bankLoading}
                      className="flex-1 bg-lime text-emerald font-display font-semibold text-[13px] rounded-xl py-2.5 disabled:opacity-50"
                    >
                      {bankLoading ? 'Saving…' : 'Save account'}
                    </button>
                  )}
                  <button
                    onClick={() => { setShowBankForm(false); setResolvedName(null); setBankError(''); }}
                    className="px-4 text-[13px] font-semibold text-mid"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
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

            {/* Badges */}
            {badgesData && badgesData.length > 0 && (
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">YOUR BADGES</span>
                <div className="flex flex-wrap gap-2">
                  {badgesData.map((b) => {
                    const meta = BADGE_META[b.badgeType] ?? FALLBACK_BADGE(b.badgeType);
                    const { Icon, label, accent } = meta;
                    const isAmber = accent === iconColors.amberAccent;
                    return (
                      <Badge
                        key={b.badgeType}
                        variant={isAmber ? 'warning' : 'success'}
                        className="flex items-center gap-1.5"
                      >
                        <Icon size={15} accent={accent} />
                        <span>{label}</span>
                      </Badge>
                    );
                  })}
                </div>
              </div>
            )}

            {/* Paycode resolve */}
            <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
              <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">RESOLVE PAYCODE</span>
              <input
                value={resolvePayload}
                onChange={(e) => { setResolvePayload(e.target.value); setResolveResult(null); }}
                placeholder="Scan or paste QR payload"
                className="w-full border border-line rounded-xl px-3 py-2 text-[12px] font-mono text-ink placeholder-mid focus:outline-none focus:border-emerald" aria-label="Scan or paste QR payload"/>
              {resolveResult && (
                <p className="text-[12px] font-semibold text-emerald" role="status">✓ {resolveResult}</p>
              )}
              <button
                onClick={async () => {
                  if (!resolvePayload.trim()) return;
                  try {
                    await paycodesApi.resolve(resolvePayload.trim());
                    setResolveResult('Order found');
                    setResolvePayload('');
                  } catch (e) {
                    setResolveResult((e as Error).message);
                  }
                }}
                disabled={!resolvePayload.trim()}
                className="w-full bg-emerald text-lime font-display font-semibold text-[12px] rounded-xl py-2 hover:bg-emerald/90 transition-colors disabled:opacity-50"
              >
                Resolve
              </button>
            </div>

            <button
              onClick={async () => {
                await authApi.logout().catch(() => {});
                clearAuth();
                router.replace('/login');
              }}
              className="text-sm font-semibold text-red-600 text-left"
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
                <Icon active={active} />
                {label}
              </button>
            );
          })}
        </div>
      </div>


    </main>
  );
}
