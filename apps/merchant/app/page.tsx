'use client';

import { useMerchantStore, type MerchantTab, type OrderPipelineState } from '../lib/store/merchant.store';

const ORDER_DEFS = [
  { id: 'o1', title: 'Ken A. — Amoxicillin 500mg ×14 (Rx approved)', sub: '#2841 · rider assigns after you confirm', total: '3,200' },
  { id: 'o2', title: 'Bisi O. — Malaria kit + Paracetamol', sub: '#2842 · OTC · pay on delivery', total: '5,300' },
  { id: 'o3', title: 'Chuka E. — First aid pack ×2', sub: '#2839 · rider Musa arrives ~6 min', total: '6,000' },
  { id: 'o4', title: 'Tola F. — Vitamin C 1000mg', sub: '#2833 · delivered 16:02 ✓', total: '2,100' },
];

const ORDER_STAGE_META: Record<
  OrderPipelineState,
  { status: string; chipC: string; chipB: string; dot: string; actLabel: string | null; next: OrderPipelineState | null; doneLabel: string | null }
> = {
  new: { status: 'NEW', chipC: '#0A3D2C', chipB: '#C6F24E', dot: '#C6F24E', actLabel: 'Confirm & prepare', next: 'preparing', doneLabel: null },
  preparing: { status: 'PREPARING', chipC: '#8A6A1B', chipB: '#FFF3D6', dot: '#E8B14E', actLabel: 'Mark ready for rider', next: 'ready', doneLabel: null },
  ready: { status: 'AWAITING RIDER', chipC: '#0A3D2C', chipB: '#E9F3D8', dot: '#0A3D2C', actLabel: null, next: null, doneLabel: 'Rider on the way' },
  done: { status: 'DELIVERED', chipC: '#63636E', chipB: '#EFECE3', dot: '#BDBAB2', actLabel: null, next: null, doneLabel: 'Completed' },
};

const PROD_DEFS = [
  { id: 'p1', emoji: '💊', name: 'Paracetamol 500mg ×24', cat: 'Pain & fever · OTC', stockN: 4, price: '800' },
  { id: 'p2', emoji: '🦟', name: 'Malaria kit (test + ACT)', cat: 'Anti-malarial · OTC', stockN: 9, price: '4,500' },
  { id: 'p3', emoji: '🩹', name: 'First aid pack', cat: 'First aid · OTC', stockN: 26, price: '3,000' },
  { id: 'p4', emoji: '💉', name: 'Amoxicillin 500mg ×14', cat: 'Antibiotic · Rx only', stockN: 41, price: '3,200' },
];

const NAV_ITEMS: { id: MerchantTab; label: string; icon: string }[] = [
  { id: 'dash', label: 'Dashboard', icon: '▦' },
  { id: 'orders', label: 'Orders', icon: '🛒' },
  { id: 'rx', label: 'Prescriptions', icon: '💊' },
  { id: 'prod', label: 'Products', icon: '📋' },
  { id: 'earn', label: 'Earnings', icon: '₦' },
  { id: 'set', label: 'Verification', icon: '⚙' },
];

export default function MerchantPortalPage() {
  const { tab, orderStates, rxState, prodOff, setTab, advanceOrder, approveRx, rejectRx, resetRx, toggleProduct } = useMerchantStore();
  const newCount = Object.values(orderStates).filter((v) => v === 'new').length;
  const rxCount = rxState === 'pending' ? 3 : 2;

  return (
    <main className="min-h-screen flex bg-sand">
      <aside className="w-[240px] flex-none bg-emerald p-6 flex flex-col gap-6 min-h-screen">
        <div className="px-2 flex flex-col gap-1">
          <span className="font-display font-bold text-xl text-sand tracking-tight">
            speed<span className="text-lime">+</span> <span className="font-medium text-[11px] text-sand/55">PARTNER</span>
          </span>
          <span className="text-[11px] text-sand/55">HealthPlus Pharmacy · Lekki</span>
        </div>

        <nav className="flex flex-col gap-1">
          {NAV_ITEMS.map((item) => {
            const active = tab === item.id;
            const count = item.id === 'orders' ? newCount : item.id === 'rx' ? rxCount : null;
            return (
              <button
                key={item.id}
                onClick={() => setTab(item.id)}
                className={`flex items-center gap-2.5 rounded-xl px-4 py-2.5 text-[13.5px] font-medium transition-colors ${
                  active ? 'bg-lime/[.14] text-lime font-semibold' : 'text-sand/70 hover:bg-sand/[.08] hover:text-sand'
                }`}
              >
                {item.icon} {item.label}
                {count !== null && count > 0 && (
                  <span
                    className="ml-auto text-[10.5px] font-bold rounded-full px-2 py-0.5"
                    style={item.id === 'rx' ? { color: '#3B2E10', background: '#E8B14E' } : { color: '#0A3D2C', background: '#C6F24E' }}
                  >
                    {count}
                  </span>
                )}
              </button>
            );
          })}
        </nav>

        <div className="mt-auto flex items-center gap-2.5 bg-sand/[.06] rounded-[13px] p-2.75">
          <span className="w-9 h-9 rounded-full bg-lime flex items-center justify-center text-emerald font-display font-bold text-[13px]">A</span>
          <span className="flex flex-col">
            <span className="text-[12.5px] font-semibold text-sand">Adaeze O.</span>
            <span className="text-[10px] text-sand/55">Pharmacist · PCN verified ✓</span>
          </span>
        </div>
      </aside>

      <div className="flex-1 min-w-0 px-8.5 py-7.5 flex flex-col gap-4.5">
        {tab === 'dash' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Good evening, HealthPlus</h1>
            <div className="grid grid-cols-4 gap-3.5">
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">TODAY&apos;S SALES</span>
                <span className="font-display text-2xl font-bold text-emerald">₦86,400</span>
                <span className="text-[11px] text-emerald">▲ 12% vs yesterday</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">ORDERS</span>
                <span className="font-display text-2xl font-bold text-ink">31</span>
                <span className="text-[11px] text-mid">{newCount} need action</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">RX QUEUE</span>
                <span className="font-display text-2xl font-bold" style={{ color: '#8A6A1B' }}>{rxCount}</span>
                <span className="text-[11px] text-mid">avg review 3 min</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">RATING</span>
                <span className="font-display text-2xl font-bold text-ink">★ 4.8</span>
                <span className="text-[11px] text-mid">last 30 days</span>
              </div>
            </div>

            <div className="grid gap-3.5" style={{ gridTemplateColumns: '1.4fr 1fr' }}>
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-3">
                <span className="text-[11px] font-semibold text-mid tracking-[.6px]">NEEDS YOUR ACTION NOW</span>
                <div className="flex items-center gap-2.5 px-3.25 py-2.75 rounded-xl" style={{ background: '#FFF7E6', border: '1px solid #F0DFB4' }}>
                  <span className="w-[7px] h-[7px] rounded-full bg-amber animate-pulse" />
                  <span className="flex-1 text-[12.5px] font-semibold">Prescription from Ken A. — waiting 2 min</span>
                  <button onClick={() => setTab('rx')} className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-3.5 py-1.75 hover:bg-lime-600 transition-colors">
                    Review
                  </button>
                </div>
                <div className="flex items-center gap-2.5 px-3.25 py-2.75 rounded-xl bg-tile">
                  <span className="w-[7px] h-[7px] rounded-full bg-emerald" />
                  <span className="flex-1 text-[12.5px] font-semibold">2 new orders to confirm</span>
                  <button
                    onClick={() => setTab('orders')}
                    className="font-display text-xs font-semibold text-emerald border-[1.5px] border-emerald rounded-[10px] px-3.5 py-1.75 hover:bg-emerald/[.07] transition-colors"
                  >
                    Open
                  </button>
                </div>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-2.5">
                <span className="text-[11px] font-semibold text-mid tracking-[.6px]">LOW STOCK</span>
                <span className="flex justify-between text-[12.5px]">
                  <span>Paracetamol 500mg</span>
                  <b style={{ color: '#B4231F' }}>4 left</b>
                </span>
                <span className="flex justify-between text-[12.5px]">
                  <span>Malaria kits (ACT)</span>
                  <b style={{ color: '#8A6A1B' }}>9 left</b>
                </span>
                <span className="flex justify-between text-[12.5px]">
                  <span>Vitamin C 1000mg</span>
                  <b style={{ color: '#8A6A1B' }}>11 left</b>
                </span>
              </div>
            </div>
          </>
        )}

        {tab === 'orders' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Orders</h1>
            {ORDER_DEFS.map((o) => {
              const state = orderStates[o.id] ?? 'new';
              const meta = ORDER_STAGE_META[state];
              return (
                <div key={o.id} className="bg-white border border-line rounded-2xl px-4.5 py-3.75 flex items-center gap-3.5">
                  <span className="w-2 h-2 flex-none rounded-full" style={{ background: meta.dot }} />
                  <span className="flex-1 flex flex-col min-w-0">
                    <span className="text-[13.5px] font-semibold">{o.title}</span>
                    <span className="text-[11px] text-mid">{o.sub}</span>
                  </span>
                  <span className="text-[11px] font-bold rounded-full px-2.75 py-1" style={{ color: meta.chipC, background: meta.chipB }}>
                    {meta.status}
                  </span>
                  <b className="font-display text-[15px] text-emerald w-[78px] text-right">₦{o.total}</b>
                  {meta.actLabel && meta.next ? (
                    <button
                      onClick={() => advanceOrder(o.id, meta.next as OrderPipelineState)}
                      className="w-[150px] font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] py-2 hover:bg-lime-600 transition-colors"
                    >
                      {meta.actLabel}
                    </button>
                  ) : (
                    <span className="w-[150px] text-center text-[11.5px]" style={{ color: '#9A968D' }}>
                      {meta.doneLabel}
                    </span>
                  )}
                </div>
              );
            })}
          </>
        )}

        {tab === 'rx' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Prescription review</h1>
            <div className="grid gap-4" style={{ gridTemplateColumns: '1fr 1.1fr' }}>
              <div className="bg-white border border-line rounded-2xl overflow-hidden">
                <div
                  className="h-[280px] flex items-center justify-center relative"
                  style={{ background: 'repeating-linear-gradient(0deg,#EFEDE6,#EFEDE6 26px,#E7E4DB 27px)' }}
                >
                  <span className="text-xs bg-white border border-line rounded-[9px] px-3.5 py-2" style={{ color: '#9A968D' }}>
                    📄 Rx photo — Dr. T. Balogun, Reddington Hospital
                  </span>
                </div>
                <div className="px-4 py-3.25 flex justify-between items-center border-t border-line">
                  <span className="text-xs text-mid">
                    Uploaded by <b className="text-ink">Ken A.</b> · 2 min ago
                  </span>
                  <span className="text-[11.5px] font-semibold text-emerald cursor-pointer">Zoom ⤢</span>
                </div>
              </div>

              <div className="flex flex-col gap-3">
                {rxState === 'pending' ? (
                  <>
                    <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
                      <span className="text-[11px] font-semibold text-mid tracking-[.6px]">WHAT THE DOCTOR PRESCRIBED</span>
                      <div className="flex justify-between items-center px-3 py-2.5 bg-sand rounded-[10px]">
                        <span className="text-[13px] font-semibold">Amoxicillin 500mg</span>
                        <span className="text-xs text-mid">14 capsules · 2× daily</span>
                        <b className="text-[13px] text-emerald">₦3,200</b>
                      </div>
                      <span className="text-[11.5px] text-mid">In stock ✓ · No interaction flags · Dosage within adult range</span>
                    </div>
                    <div className="bg-white border border-line rounded-2xl p-4 flex flex-col gap-2.5">
                      <span className="text-[11px] font-semibold text-mid tracking-[.6px]">YOUR DECISION — LOGGED UNDER YOUR PCN LICENCE</span>
                      <div className="flex gap-2.5">
                        <button onClick={approveRx} className="flex-[2] font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] py-3.25 hover:bg-lime-600 transition-colors">
                          ✓ Approve &amp; prepare — ₦3,200
                        </button>
                        <button
                          onClick={rejectRx}
                          className="flex-1 font-display text-xs font-semibold rounded-[10px] py-3.25 border-[1.5px] transition-colors"
                          style={{ color: '#B4231F', borderColor: '#E5B5B3' }}
                        >
                          Reject…
                        </button>
                      </div>
                      <span className="text-[11px]" style={{ color: '#9A968D' }}>
                        Need to talk to the patient first? <a href="#" className="font-semibold text-emerald">Start a live consult</a>
                      </span>
                    </div>
                  </>
                ) : (
                  <div className="rounded-2xl p-4.5 flex flex-col gap-2 bg-tile" style={{ border: '1px solid #C9E0A8' }}>
                    <span className="text-[15px] font-bold text-emerald">
                      {rxState === 'approved' ? '✓ Approved — order created for Ken A.' : 'Rejected — customer notified kindly'}
                    </span>
                    <span className="text-xs text-mid">
                      {rxState === 'approved'
                        ? 'Logged under PCN #A-48812. Prepare and seal it — a rider is being matched.'
                        : '“We couldn’t verify this prescription. Please ask your doctor to re-issue it.”'}
                    </span>
                    <button
                      onClick={resetRx}
                      className="self-start mt-1 font-display text-xs font-semibold text-emerald border-[1.5px] border-emerald rounded-[10px] px-3.5 py-1.75 hover:bg-emerald/[.07] transition-colors"
                    >
                      Next in queue ({rxCount})
                    </button>
                  </div>
                )}

                <div className="bg-white border border-line rounded-2xl px-4 py-3.5 flex flex-col gap-2">
                  <span className="text-[11px] font-semibold text-mid tracking-[.6px]">UP NEXT</span>
                  <span className="flex justify-between text-[12.5px]">
                    <span>Metformin 850mg — Bisi O.</span>
                    <span style={{ color: '#9A968D' }}>2 min</span>
                  </span>
                  <span className="flex justify-between text-[12.5px]">
                    <span>Ventolin inhaler — Chuka E.</span>
                    <span style={{ color: '#9A968D' }}>6 min</span>
                  </span>
                </div>
              </div>
            </div>
          </>
        )}

        {tab === 'prod' && (
          <>
            <div className="flex items-center justify-between">
              <h1 className="font-display font-semibold text-[26px] tracking-tight">Products</h1>
              <button className="font-display text-xs font-semibold text-emerald bg-lime rounded-[10px] px-4 py-2.25 hover:bg-lime-600 transition-colors">
                + Add product
              </button>
            </div>
            {PROD_DEFS.map((p) => {
              const off = Boolean(prodOff[p.id]);
              const stockColor = p.stockN < 5 ? '#B4231F' : p.stockN < 12 ? '#8A6A1B' : '#63636E';
              return (
                <div key={p.id} className="bg-white border border-line rounded-2xl px-4.5 py-3.25 flex items-center gap-3.5">
                  <span className="text-lg">{p.emoji}</span>
                  <span className="flex-1 flex flex-col">
                    <span className="text-[13px] font-semibold">{p.name}</span>
                    <span className="text-[11px] text-mid">{p.cat}</span>
                  </span>
                  <span className="text-xs w-[70px]" style={{ color: stockColor }}>
                    {p.stockN} in stock
                  </span>
                  <b className="font-display text-sm w-20 text-right">₦{p.price}</b>
                  <button
                    onClick={() => toggleProduct(p.id)}
                    className="w-11 h-6 flex-none rounded-full relative transition-colors"
                    style={{ background: off ? '#D5D2C8' : '#0A3D2C' }}
                    aria-label={off ? 'Enable product' : 'Disable product'}
                  >
                    <span
                      className="absolute top-[3px] w-[18px] h-[18px] rounded-full bg-white shadow-sm transition-all"
                      style={{ left: off ? 3 : 23 }}
                    />
                  </button>
                </div>
              );
            })}
          </>
        )}

        {tab === 'earn' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Earnings</h1>
            <div className="grid grid-cols-3 gap-3.5">
              <div className="bg-emerald rounded-2xl p-4.5 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-sand/60 tracking-[.5px]">THIS WEEK</span>
                <span className="font-display text-[28px] font-bold text-lime">₦512,300</span>
                <span className="text-[11px] text-sand/60">168 orders</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">PENDING PAYOUT</span>
                <span className="font-display text-[28px] font-bold text-ink">₦86,400</span>
                <span className="text-[11px] text-mid">pays out tonight, 11pm</span>
              </div>
              <div className="bg-white border border-line rounded-2xl p-4.5 flex flex-col gap-1">
                <span className="text-[10.5px] font-semibold text-mid tracking-[.5px]">SPEEDPLUS FEE</span>
                <span className="font-display text-[28px] font-bold text-ink">8%</span>
                <span className="text-[11px] text-mid">flat, no hidden charges</span>
              </div>
            </div>
            <div className="bg-white border border-line rounded-2xl overflow-hidden">
              <div className="flex justify-between px-4.5 py-3.25 border-b border-[#EFECE3] text-[11px] font-semibold text-mid tracking-[.5px]">
                <span>PAYOUT</span>
                <span>AMOUNT</span>
              </div>
              {[
                { date: 'Thu 9 Jul · GTBank ••4821', amount: '₦94,150 ✓' },
                { date: 'Wed 8 Jul · GTBank ••4821', amount: '₦71,900 ✓' },
                { date: 'Tue 7 Jul · GTBank ••4821', amount: '₦88,220 ✓' },
              ].map((row, i, arr) => (
                <div key={row.date} className={`flex justify-between px-4.5 py-3.25 text-[13px] ${i < arr.length - 1 ? 'border-b border-[#EFECE3]' : ''}`}>
                  <span>{row.date}</span>
                  <b className="text-emerald">{row.amount}</b>
                </div>
              ))}
            </div>
          </>
        )}

        {tab === 'set' && (
          <>
            <h1 className="font-display font-semibold text-[26px] tracking-tight">Verification &amp; onboarding</h1>
            <span className="text-[13px] text-mid" style={{ maxWidth: '56ch' }}>
              Pharmacies must pass all checks before selling on SpeedPlus. Your customers see the badges — trust is the product.
            </span>
            <div className="bg-white border border-line rounded-2xl overflow-hidden" style={{ maxWidth: 760 }}>
              {[
                { icon: '🏥', label: 'Premises licence (PCN)', sub: 'HealthPlus Lekki · exp. Mar 2027', chip: '✓ Verified', ok: true },
                { icon: '👩🏾‍⚕️', label: 'Superintendent pharmacist licence', sub: 'Adaeze O. · PCN #A-48812 · annual renewal', chip: '✓ Verified', ok: true },
                { icon: '🔎', label: 'Background check', sub: 'Directors + superintendent · renewed yearly', chip: '✓ Passed', ok: true },
                { icon: '🧊', label: 'Cold-chain capability', sub: 'Needed to sell insulin & vaccines', chip: 'Inspection booked — 18 Jul', ok: false },
              ].map((row) => (
                <div key={row.label} className="flex items-center gap-3.25 px-4.5 py-3.75 border-b border-[#EFECE3]">
                  <span className="text-[17px]">{row.icon}</span>
                  <span className="flex-1 flex flex-col">
                    <span className="text-[13.5px] font-semibold">{row.label}</span>
                    <span className="text-[11px] text-mid">{row.sub}</span>
                  </span>
                  <span
                    className="text-[11px] font-bold rounded-full px-2.75 py-1"
                    style={row.ok ? { color: '#0A3D2C', background: '#E9F3D8' } : { color: '#8A6A1B', background: '#FFF3D6' }}
                  >
                    {row.chip}
                  </span>
                </div>
              ))}
              <div className="flex items-center gap-3.25 px-4.5 py-3.75">
                <span className="text-[17px]">🎥</span>
                <span className="flex-1 flex flex-col">
                  <span className="text-[13.5px] font-semibold">Live consult setup</span>
                  <span className="text-[11px] text-mid">Video consults — pharmacist can prescribe after live diagnosis</span>
                </span>
                <button className="font-display text-xs font-semibold text-emerald border-[1.5px] border-emerald rounded-[10px] px-3.5 py-1.75 hover:bg-emerald/[.07] transition-colors">
                  Set up
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </main>
  );
}
