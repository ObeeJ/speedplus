'use client';

import { useDriverStore } from '../lib/store/driver.store';

const STAGE_LABELS = ['Accepted — ride to pickup', 'Arrived at pickup', 'Package picked up', 'Arrived at drop-off', 'Delivered ✓'];
const CTAS = ['', "I've arrived at pickup", 'I have the package', "I've arrived at drop-off", 'Confirm delivery — code entered', 'Done — back to home'];
const HINTS = [
  '',
  'Tap when you reach 12 Admiralty Way',
  'Check it matches: medium box, under 10 kg',
  'Tap when you reach Herbert Macaulay Way',
  'Collect ₦840 cash, then confirm',
  '₦840 added to today’s earnings 🎉',
];
const DAYS = [38, 52, 44, 70, 58, 90, 64].map((h, i) => ({ h, label: ['M', 'T', 'W', 'T', 'F', 'S', 'S'][i], highlight: i === 5 }));

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

export default function DriverAppPage() {
  const { tab, online, jobStage, cashed, setTab, toggleOnline, acceptJob, declineJob, advanceJob, cashOut } = useDriverStore();

  const todayEarn = 2010 + (jobStage >= 5 ? 840 : 0);
  const trips = jobStage >= 5 ? 4 : 3;
  const weekEarn = 11460 + (jobStage >= 5 ? 840 : 0);
  const showOffer = online && jobStage === 0;
  const showWaiting = !online || jobStage > 0;
  const isJob = tab === 'job' && jobStage > 0;
  const showPod = jobStage === 4;
  const navHint = jobStage <= 1 ? 'Navigate: 2.1 km to pickup' : jobStage === 2 ? 'Navigate: 12 km to Yaba' : 'Almost there';

  return (
    <main className="min-h-screen flex justify-center p-3 min-[500px]:p-6" style={{ background: '#E9E6DD' }}>
      <div className="w-full max-w-[430px] bg-sand rounded-3xl overflow-hidden shadow-[0_24px_60px_rgba(10,61,44,.18)] flex flex-col min-h-[780px]">
        <div className="bg-emerald px-5 py-4 flex items-center gap-3">
          <span className="font-display font-bold text-lg text-sand tracking-tight">
            speed<span className="text-lime">+</span> <span className="font-medium text-xs text-sand/60">RIDER</span>
          </span>
          <span className="flex-1" />
          <button
            onClick={toggleOnline}
            className={`flex items-center gap-2 rounded-full px-3.5 py-1.5 ${online ? 'bg-lime' : 'bg-sand/[.14]'}`}
          >
            <span className={`w-2 h-2 rounded-full ${online ? 'bg-emerald' : 'bg-[#9A968D]'}`} />
            <span className={`text-xs font-bold ${online ? 'text-emerald' : 'text-sand/70'}`}>{online ? 'Online' : 'Offline'}</span>
          </button>
        </div>

        {tab === 'home' && (
          <div className="flex-1 px-5 py-4.5 flex flex-col gap-3.5">
            <div className="flex gap-2.5">
              <div className="flex-1 bg-white border border-line rounded-[14px] p-3.5 flex flex-col gap-0.5">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">TODAY</span>
                <span className="font-display text-xl font-bold text-emerald">₦{todayEarn.toLocaleString()}</span>
              </div>
              <div className="flex-1 bg-white border border-line rounded-[14px] p-3.5 flex flex-col gap-0.5">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">TRIPS</span>
                <span className="font-display text-xl font-bold text-ink">{trips}</span>
              </div>
              <div className="flex-1 bg-white border border-line rounded-[14px] p-3.5 flex flex-col gap-0.5">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">RATING</span>
                <span className="font-display text-xl font-bold text-ink">★ 4.9</span>
              </div>
            </div>

            {showOffer && (
              <div className="bg-emerald rounded-[18px] p-4 flex flex-col gap-3 shadow-[0_12px_30px_rgba(10,61,44,.3)]">
                <div className="flex items-center gap-2.5">
                  <span className="w-2 h-2 rounded-full bg-lime animate-pulse" />
                  <span className="font-display text-[15px] font-semibold text-sand">New delivery — Package</span>
                  <span className="ml-auto font-display text-lg font-bold text-lime">₦840</span>
                </div>
                <div className="flex flex-col gap-1.5 bg-sand/[.08] rounded-xl px-3.5 py-2.5">
                  <span className="flex gap-2.5 text-xs text-sand">
                    <b className="text-lime">A</b> Lekki Phase 1 — 12 Admiralty Way <span className="ml-auto text-sand/55">2.1 km away</span>
                  </span>
                  <span className="flex gap-2.5 text-xs text-sand">
                    <b className="text-amber">B</b> Yaba — 5 Herbert Macaulay Way <span className="ml-auto text-sand/55">12 km trip</span>
                  </span>
                </div>
                <span className="text-[11px] text-sand/60">Medium box · under 10 kg · bike OK · cash on delivery</span>
                <div className="flex gap-2.5">
                  <button
                    onClick={declineJob}
                    className="flex-1 text-center font-display text-[13px] font-semibold text-sand border-[1.5px] border-sand/30 rounded-xl py-3 hover:bg-sand/10 transition-colors"
                  >
                    Decline
                  </button>
                  <button
                    onClick={acceptJob}
                    className="flex-[2] text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-xl py-3 hover:bg-lime-600 transition-colors"
                  >
                    Accept — ₦840
                  </button>
                </div>
              </div>
            )}

            {showWaiting && (
              <div className="border-[1.5px] border-dashed border-line rounded-2xl p-5.5 flex flex-col items-center gap-1.5 text-center">
                <span className="text-[13px] font-semibold text-mid">
                  {online ? (jobStage > 0 ? 'You have an active delivery' : 'Waiting for jobs…') : 'You are offline'}
                </span>
                <span className="text-[11px] text-[#9A968D]">
                  {online
                    ? jobStage > 0
                      ? 'Open the Delivery tab to continue it.'
                      : 'Stay near busy areas — Lekki and Yaba are hot right now.'
                    : 'Go online to receive delivery offers.'}
                </span>
              </div>
            )}

            <span className="text-[11px] font-semibold text-mid tracking-[.6px] mt-1">EARLIER TODAY</span>
            <div className="bg-white border border-line rounded-[14px] overflow-hidden">
              {[
                { emoji: '📦', label: 'Package → Surulere', meta: '14:22 · 16 km', amount: '₦920' },
                { emoji: '🔥', label: 'Gas 12.5kg → Lekki', meta: '11:05 · 4 km', amount: '₦480' },
                { emoji: '💊', label: 'Medicine → Ikoyi', meta: '09:40 · 7 km', amount: '₦610' },
              ].map((row, i) => (
                <div key={row.label} className={`flex items-center gap-2.5 px-3.5 py-3 ${i < 2 ? 'border-b border-[#EFECE3]' : ''}`}>
                  <span className="text-base">{row.emoji}</span>
                  <span className="flex-1 flex flex-col">
                    <span className="text-[12.5px] font-semibold">{row.label}</span>
                    <span className="text-[10.5px] text-mid">{row.meta}</span>
                  </span>
                  <b className="text-[12.5px] text-emerald">{row.amount}</b>
                </div>
              ))}
            </div>
          </div>
        )}

        {isJob && (
          <>
            <div className="relative h-[170px] flex-none overflow-hidden" style={{ background: 'linear-gradient(160deg,#0D4E38,#08301F)' }}>
              <span
                className="absolute rounded-full bg-lime border-[3px] border-emerald animate-pulse"
                style={{ left: 122, top: 42, width: 14, height: 14 }}
              />
              <span className="absolute rounded-full bg-sand border-[3px] border-emerald" style={{ left: 224, top: 109, width: 11, height: 11 }} />
              <div className="absolute left-3.5 bottom-3 bg-emerald-900/85 border border-lime/30 rounded-full px-3.5 py-1.5">
                <span className="text-[11px] font-semibold text-sand">{navHint}</span>
              </div>
            </div>
            <div className="flex-1 px-5 py-4 flex flex-col gap-3">
              <div className="flex items-center gap-3 bg-white border border-line rounded-[14px] px-3.5 py-3">
                <span className="w-10 h-10 rounded-full bg-tile flex items-center justify-center font-display font-bold text-emerald">K</span>
                <span className="flex-1 flex flex-col">
                  <span className="text-[13px] font-semibold">Ken A. — customer</span>
                  <span className="text-[10.5px] text-mid">Package R-7458 · pays ₦840 cash on delivery</span>
                </span>
                <button className="w-[38px] h-[38px] rounded-[11px] bg-tile flex items-center justify-center hover:bg-[#DCEDC2] transition-colors" aria-label="Call customer">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#0A3D2C" strokeWidth={1.9} strokeLinecap="round" strokeLinejoin="round">
                    <path d="M4 6c0 8 6 14 14 14l2.5-3-4-2-2 2c-2.5-1.2-4.3-3-5.5-5.5l2-2-2-4L6 4" />
                  </svg>
                </button>
              </div>

              <div className="bg-white border border-line rounded-[14px] px-3.5 py-3.5 flex flex-col gap-2">
                {STAGE_LABELS.map((label, i) => {
                  const done = jobStage > i;
                  const current = jobStage === i + 1;
                  return (
                    <span key={label} className="flex items-center gap-2.5">
                      <span
                        className="w-2.5 h-2.5 rounded-full border-[2.5px]"
                        style={{ background: done ? '#0A3D2C' : '#FFFFFF', borderColor: done ? '#C6F24E' : '#E4E0D6' }}
                      />
                      <span className="text-[12.5px] font-semibold" style={{ color: current ? '#0A3D2C' : done ? '#121216' : '#9A968D' }}>
                        {label}
                      </span>
                    </span>
                  );
                })}
              </div>

              {showPod && (
                <div className="bg-[#FFF7E6] border border-[#F0DFB4] rounded-[14px] px-3.5 py-3 flex flex-col gap-2">
                  <span className="text-[12.5px] font-bold">Proof of delivery</span>
                  <span className="text-[11px] text-mid">Ask Ken for the 4-digit code, or snap a photo of the handover.</span>
                  <div className="flex gap-2">
                    <span className="flex-1 text-center font-display tracking-[6px] text-base font-bold bg-white border-[1.5px] border-line rounded-[10px] py-2.5 text-emerald">
                      7 4 2 9
                    </span>
                    <button className="w-11 flex items-center justify-center bg-white border-[1.5px] border-line rounded-[10px] hover:border-emerald transition-colors" aria-label="Take photo">
                      📷
                    </button>
                  </div>
                </div>
              )}

              <div className="mt-auto flex flex-col gap-2">
                <button
                  onClick={advanceJob}
                  className="text-center font-display text-sm font-semibold text-emerald bg-lime rounded-[13px] py-3.5 hover:bg-lime-600 transition-colors"
                >
                  {CTAS[Math.min(jobStage, 5)]}
                </button>
                <span className="text-[10.5px] text-[#9A968D] text-center">{HINTS[Math.min(jobStage, 5)]}</span>
              </div>
            </div>
          </>
        )}

        {tab === 'earn' && (
          <div className="flex-1 px-5 py-4.5 flex flex-col gap-3.5">
            <div className="bg-emerald rounded-[18px] p-4.5 flex flex-col gap-1">
              <span className="text-[11px] font-semibold text-sand/60 tracking-[.6px]">THIS WEEK</span>
              <span className="font-display text-[34px] font-bold text-lime tracking-tight">₦{weekEarn.toLocaleString()}</span>
              <span className="text-[11.5px] text-sand/60">19 trips · ₦{todayEarn.toLocaleString()} today</span>
              <button
                onClick={cashOut}
                className="mt-2.5 text-center font-display text-[13px] font-semibold text-emerald bg-lime rounded-xl py-3 hover:bg-lime-600 transition-colors"
              >
                {cashed ? '✓ Sent to your bank' : 'Cash out ₦9,120 to bank'}
              </button>
            </div>

            <span className="text-[11px] font-semibold text-mid tracking-[.6px]">BY DAY</span>
            <div className="bg-white border border-line rounded-[14px] p-3.5 flex items-end gap-2 h-[110px]">
              {DAYS.map((d, i) => (
                <span key={i} className="flex-1 flex flex-col items-center gap-1 h-full justify-end">
                  <span
                    className="w-full rounded-t-[5px] rounded-b-[2px]"
                    style={{ height: `${d.h}%`, background: d.highlight ? '#C6F24E' : '#0A3D2C' }}
                  />
                  <span className="text-[9px] text-mid">{d.label}</span>
                </span>
              ))}
            </div>

            <div className="bg-white border border-line rounded-[14px] overflow-hidden">
              <div className="flex justify-between px-3.5 py-3 border-b border-[#EFECE3]">
                <span className="text-xs text-mid">Cash collected (to remit)</span>
                <b className="text-[12.5px] text-amber">₦2,340</b>
              </div>
              <div className="flex justify-between px-3.5 py-3">
                <span className="text-xs text-mid">Paid to wallet</span>
                <b className="text-[12.5px] text-emerald">₦9,120</b>
              </div>
            </div>
          </div>
        )}

        {tab === 'me' && (
          <div className="flex-1 px-5 py-4.5 flex flex-col gap-3.5">
            <div className="flex items-center gap-3.5 bg-white border border-line rounded-2xl p-4">
              <span className="w-[54px] h-[54px] rounded-full bg-emerald flex items-center justify-center text-lime font-display font-bold text-xl">M</span>
              <span className="flex flex-col gap-0.5">
                <span className="font-display text-base font-semibold">Musa Ibrahim</span>
                <span className="text-[11.5px] text-mid">Bike · Lagos · ★ 4.9 · 2,140 deliveries</span>
              </span>
            </div>

            <span className="text-[11px] font-semibold text-mid tracking-[.6px]">YOUR DOCUMENTS</span>
            <div className="bg-white border border-line rounded-[14px] overflow-hidden">
              {[
                { label: "Rider's licence", status: '✓ Verified', ok: true },
                { label: 'Background check', status: '✓ Passed', ok: true },
                { label: 'Insurance', status: 'Renews in 12 days', ok: false },
              ].map((doc, i) => (
                <div key={doc.label} className={`flex items-center gap-2.5 px-3.5 py-3 ${i < 2 ? 'border-b border-[#EFECE3]' : ''}`}>
                  <span className="flex-1 text-[12.5px] font-semibold">{doc.label}</span>
                  <span
                    className="text-[11px] font-bold rounded-full px-2.5 py-1"
                    style={doc.ok ? { color: '#0A3D2C', background: '#E9F3D8' } : { color: '#8A6A1B', background: '#FFF3D6' }}
                  >
                    {doc.status}
                  </span>
                </div>
              ))}
            </div>

            <span className="text-[11px] font-semibold text-mid tracking-[.6px]">TRAINED FOR</span>
            <div className="flex gap-2 flex-wrap">
              {[
                { label: '📦 Packages', trained: true },
                { label: '🔥 Gas handling', trained: true },
                { label: '💊 Sealed medicine', trained: true },
                { label: '🍛 Food (take course)', trained: false },
              ].map((tag) => (
                <span
                  key={tag.label}
                  className="text-[11.5px] font-semibold rounded-full px-3 py-1.5"
                  style={tag.trained ? { color: '#0A3D2C', background: '#E9F3D8' } : { color: '#9A968D', background: '#EFECE3' }}
                >
                  {tag.label}
                </span>
              ))}
            </div>
          </div>
        )}

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
                className={`flex flex-col items-center gap-0.5 text-[9.5px] ${active ? 'font-bold text-emerald' : 'font-medium text-mid'}`}
              >
                <Icon />
                {label}
              </button>
            );
          })}
        </div>
      </div>
    </main>
  );
}
