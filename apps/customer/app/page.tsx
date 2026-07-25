'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  SearchIcon,
  PackageIcon,
  GasIcon,
  FoodIcon,
  MedicineIcon,
  HomeIcon,
  OrdersIcon,
  MoneyIcon,
  MeIcon,
} from './icons';

type ServiceId = 'package' | 'gas' | 'food' | 'medicine';

const FLOW_HREF: Partial<Record<ServiceId, string>> = {
  package: '/package/where',
  gas: '/gas/cylinder',
  food: '/food/menu',
  medicine: '/pharmacy/items',
};

const services: { id: ServiceId; label: string; Icon: (p: { size?: number; active?: boolean }) => React.JSX.Element; hint: string }[] = [
  { id: 'package', label: 'Package', Icon: PackageIcon, hint: 'From ₦100' },
  { id: 'gas', label: 'Gas', Icon: GasIcon, hint: 'Same day' },
  { id: 'food', label: 'Food', Icon: FoodIcon, hint: 'Live prep times' },
  { id: 'medicine', label: 'Medicine', Icon: MedicineIcon, hint: 'Checked by a pharmacist' },
];

const hintFor: Record<ServiceId, string> = {
  package: 'next we ask where it’s going',
  gas: 'next we ask which cylinder',
  food: 'next we show meals nearby',
  medicine: 'next: everyday items or a prescription',
};

export default function HomePage() {
  const router = useRouter();
  const [active, setActive] = useState<ServiceId>('package');
  const activeLabel = services.find((s) => s.id === active)!.label;
  const tapAgainHint = FLOW_HREF[active] ? ' Tap it again to continue.' : '';

  const handleServiceTap = (id: ServiceId) => {
    if (active === id && FLOW_HREF[id]) {
      router.push(FLOW_HREF[id]!);
      return;
    }
    setActive(id);
  };

  return (
    <main>
      {/* ============ MOBILE (<700px) ============ */}
      <div className="flex flex-col min-h-screen min-[700px]:hidden">
        <div className="bg-emerald px-5 pt-[18px] pb-[18px] flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <span className="font-display font-bold text-[19px] text-sand tracking-tight">
              speed<span className="text-lime">+</span>
            </span>
            <div className="flex items-center gap-2.5">
              <span className="text-[11px] text-sand/70">Your area: Lekki Phase 1 ▾</span>
              <span className="w-8 h-8 rounded-full bg-sand/[.12] flex items-center justify-center text-lime font-display font-semibold text-xs">K</span>
            </div>
          </div>
          <span className="font-display font-semibold text-[23px] leading-[1.1] text-sand tracking-tight">
            What do you need moved?
          </span>
          <div className="flex items-center gap-2.5 bg-sand rounded-[14px] px-[15px] py-3.5 shadow-[0_8px_24px_rgba(0,0,0,.25)]">
            <SearchIcon />
            <input type="text" placeholder="Tell us — e.g. “gas to Surulere”" className="flex-1 min-w-0 bg-transparent outline-none text-ink placeholder:text-mid" />
            <button className="font-display text-xs font-semibold text-emerald bg-lime rounded-[9px] px-[13px] py-[7px] hover:bg-lime-600 transition-colors">
              Send
            </button>
          </div>
          <span className="text-[11px] text-sand/55">…or just tap what you need below 👇</span>
        </div>

        <div className="px-5 pt-3.5 flex flex-col gap-2">
          <div className="grid grid-cols-4 gap-[9px]">
            {services.map(({ id, label, Icon }) => {
              const isActive = active === id;
              return (
                <button
                  key={id}
                  onClick={() => handleServiceTap(id)}
                  className={`flex flex-col items-center gap-1.5 rounded-[14px] border-2 px-1 py-2.5 transition-all ${
                    isActive ? 'bg-emerald border-lime shadow-[0_6px_16px_rgba(10,61,44,.3)]' : 'bg-tile border-transparent hover:border-emerald'
                  }`}
                >
                  <Icon active={isActive} />
                  <span className={`text-[10.5px] font-semibold ${isActive ? 'text-lime font-bold' : 'text-ink'}`}>
                    {isActive ? `✓ ${label}` : label}
                  </span>
                </button>
              );
            })}
          </div>
          <span className="text-[11px] text-mid text-center">
            You tapped <b className="text-emerald">{activeLabel}</b> — {hintFor[active]}. Nothing is paid yet.
          </span>
        </div>

        <div className="flex-1 px-5 pt-3.5 pb-2 flex flex-col gap-[11px]">
          <span className="text-[11px] font-semibold text-mid tracking-[.6px]">YOUR ORDERS RIGHT NOW</span>
          <div className="rounded-[18px] bg-emerald flex items-center gap-[11px] px-[15px] py-3.5">
            <span className="w-[7px] h-[7px] rounded-full bg-lime shadow-[0_0_0_3px_rgba(198,242,78,.3)]" />
            <span className="flex-1 flex flex-col">
              <span className="text-[12.5px] font-semibold text-sand">Your package is on its way to Yaba</span>
              <span className="text-[10.5px] text-sand/60">Musa is bringing it · arrives in about 4 minutes</span>
            </span>
            <span className="font-display text-[11px] font-semibold text-emerald bg-lime rounded-[9px] px-[11px] py-1.5">See where</span>
          </div>
          <div className="flex items-center gap-[11px] bg-white border border-line rounded-[14px] px-3.5 py-3">
            <span className="w-[7px] h-[7px] rounded-full bg-amber" />
            <span className="flex-1 flex flex-col">
              <span className="text-xs font-semibold">Your medicine is being checked</span>
              <span className="text-[10.5px] text-mid">A licensed pharmacist confirms it's right for you · ~8 min</span>
            </span>
            <span className="text-[11px] font-semibold text-emerald cursor-pointer">Open</span>
          </div>
          <span className="text-[11px] font-semibold text-mid tracking-[.6px] mt-0.5">ORDER AGAIN — SAME AS LAST TIME</span>
          <div className="flex gap-[9px]">
            <button className="flex-1 flex items-center gap-2 bg-white border border-line rounded-xl px-3 py-2.5 hover:border-emerald transition-colors text-left">
              <GasIcon size={16} />
              <span className="flex flex-col">
                <span className="text-xs font-semibold">Gas — 12.5kg</span>
                <span className="text-[9.5px] text-mid">Tap once, we do the rest</span>
              </span>
            </button>
            <button className="flex-1 flex items-center gap-2 bg-white border border-line rounded-xl px-3 py-2.5 hover:border-emerald transition-colors text-left">
              <MedicineIcon size={16} />
              <span className="flex flex-col">
                <span className="text-xs font-semibold">Malaria kit</span>
                <span className="text-[9.5px] text-mid">Tap once, we do the rest</span>
              </span>
            </button>
          </div>
        </div>

        <div className="sticky bottom-0 bg-white border-t border-line px-5 pt-2 pb-[calc(10px+env(safe-area-inset-bottom))] flex justify-around items-center">
          <span className="flex flex-col items-center gap-0.5 text-[9.5px] font-bold text-emerald">
            <span className="w-[34px] h-6 rounded-xl bg-tile flex items-center justify-center"><HomeIcon /></span>
            Home
          </span>
          <span className="flex flex-col items-center gap-0.5 text-[9.5px] font-medium text-mid cursor-pointer">
            <span className="w-[34px] h-6 flex items-center justify-center"><OrdersIcon /></span>
            My orders
          </span>
          <span className="flex flex-col items-center gap-0.5 text-[9.5px] font-medium text-mid cursor-pointer">
            <span className="w-[34px] h-6 flex items-center justify-center"><MoneyIcon /></span>
            My money
          </span>
          <span className="flex flex-col items-center gap-0.5 text-[9.5px] font-medium text-mid cursor-pointer">
            <span className="w-[34px] h-6 flex items-center justify-center"><MeIcon /></span>
            Me
          </span>
        </div>
      </div>

      {/* ============ TABLET (700–1023px) ============ */}
      <div className="hidden min-[700px]:flex lg:hidden flex-col min-h-screen">
        <div className="flex items-center justify-between px-[26px] py-4 border-b border-line">
          <span className="font-display font-bold text-lg text-ink tracking-tight">
            speed<span className="text-emerald">+</span>
          </span>
          <div className="flex items-center gap-3.5">
            <span className="text-xs text-mid">Your area: Lekki Phase 1 ▾</span>
            <span className="w-[34px] h-[34px] rounded-full bg-emerald flex items-center justify-center text-lime font-display font-semibold text-[13px]">K</span>
          </div>
        </div>
        <div className="flex-1 flex">
          <div className="flex-1 bg-emerald p-7 flex flex-col gap-4">
            <span className="text-[11px] font-semibold text-sand/50 tracking-[1px]">DO</span>
            <span className="font-display font-semibold text-[28px] leading-[1.1] text-sand tracking-tight">
              What do you<br />need moved?
            </span>
            <div className="flex items-center gap-2.5 bg-sand rounded-[14px] px-[15px] py-3.5 shadow-[0_8px_24px_rgba(0,0,0,.25)]">
              <SearchIcon />
              <input type="text" placeholder="Tell us — e.g. “gas to Surulere”" className="flex-1 min-w-0 bg-transparent outline-none text-ink placeholder:text-mid" />
              <button className="font-display text-xs font-semibold text-emerald bg-lime rounded-[9px] px-[13px] py-[7px] hover:bg-lime-600 transition-colors">
                Send
              </button>
            </div>
            <span className="text-[11px] text-sand/55">…or tap what you need:</span>
            <div className="grid grid-cols-2 gap-[11px]">
              {services.map(({ id, label, Icon, hint }) => {
                const isActive = active === id;
                return (
                  <button
                    key={id}
                    onClick={() => handleServiceTap(id)}
                    className={`flex flex-row items-center justify-start gap-[11px] rounded-[14px] border-2 px-3.5 py-[13px] transition-all ${
                      isActive ? 'bg-emerald-600 border-lime shadow-[0_6px_16px_rgba(10,61,44,.3)]' : 'bg-tile border-transparent hover:border-emerald'
                    }`}
                  >
                    <Icon active={isActive} size={21} />
                    <span className="flex flex-col text-left">
                      <span className={`text-[13px] font-semibold ${isActive ? 'text-lime font-bold' : 'text-ink'}`}>
                        {isActive ? `✓ ${label}` : label}
                      </span>
                      <span className={`text-[10.5px] ${isActive ? 'text-lime/70' : 'text-mid'}`}>{hint}</span>
                    </span>
                  </button>
                );
              })}
            </div>
            <span className="text-[11px] text-sand/60">
              You tapped <b className="text-lime">{activeLabel}</b> — {hintFor[active]}. Nothing is paid yet.
            </span>
            <div className="mt-auto flex gap-[9px]">
              <button className="flex-1 text-left text-[11.5px] font-semibold text-sand bg-sand/[.08] rounded-[11px] px-3 py-2.5">↻ Gas — 12.5kg, same as last time</button>
              <button className="flex-1 text-left text-[11.5px] font-semibold text-sand bg-sand/[.08] rounded-[11px] px-3 py-2.5">↻ Malaria kit</button>
            </div>
          </div>
          <div className="w-[330px] flex-none p-6 flex flex-col gap-[13px]">
            <span className="text-[11px] font-semibold text-mid tracking-[1px]">HAPPENING</span>
            <div className="rounded-2xl bg-emerald flex items-center gap-2.5 px-3.5 py-[13px]">
              <span className="w-[7px] h-[7px] rounded-full bg-lime shadow-[0_0_0_3px_rgba(198,242,78,.3)]" />
              <span className="flex-1 flex flex-col">
                <span className="text-xs font-semibold text-sand">Package on its way to Yaba</span>
                <span className="text-[10px] text-sand/60">Musa is bringing it</span>
              </span>
              <span className="font-display text-[15px] font-bold text-lime">4 min</span>
            </div>
            <div className="flex items-center gap-2.5 bg-white border border-line rounded-[14px] px-3.5 py-3">
              <span className="w-[7px] h-[7px] rounded-full bg-amber" />
              <span className="flex-1 flex flex-col">
                <span className="text-xs font-semibold">Medicine being checked</span>
                <span className="text-[10.5px] text-mid">Pharmacist confirming · ~8 min</span>
              </span>
              <span className="text-[11px] font-semibold text-emerald cursor-pointer">Open</span>
            </div>
            <div className="flex items-center gap-2.5 bg-white border border-line rounded-[14px] px-3.5 py-3">
              <span className="w-[7px] h-[7px] rounded-full bg-[#BDBAB2]" />
              <span className="flex-1 flex flex-col">
                <span className="text-xs font-semibold">Jollof from Kilimanjaro</span>
                <span className="text-[10.5px] text-mid">Delivered yesterday · Rate it</span>
              </span>
              <span className="text-[11px] font-semibold text-emerald">★★★★☆</span>
            </div>
          </div>
        </div>
      </div>

      {/* ============ DESKTOP (>=1024px) ============ */}
      <div className="hidden lg:flex flex-col min-h-screen bg-emerald relative overflow-hidden">
        <svg width="100%" height="100%" viewBox="0 0 1180 720" fill="none" preserveAspectRatio="xMidYMid slice" className="absolute inset-0 opacity-35">
          <g stroke="#0D4E38" strokeWidth={22}>
            <path d="M-20 180 H1200" />
            <path d="M-20 460 H1200" />
            <path d="M200 -20 V740" />
            <path d="M560 -20 V740" />
            <path d="M880 -20 V740" />
          </g>
          <g stroke="#0D4E38" strokeWidth={8}>
            <path d="M-20 320 H1200" />
            <path d="M380 -20 V740" />
            <path d="M720 -20 V740" />
          </g>
        </svg>

        <div className="relative flex items-center justify-between px-9 py-5">
          <span className="font-display font-bold text-xl text-sand tracking-tight">
            speed<span className="text-lime">+</span>
          </span>
          <div className="flex gap-6.5 items-center">
            <span className="text-[13px] font-semibold text-lime cursor-pointer">Home</span>
            <span className="text-[13px] font-medium text-sand/65 cursor-pointer">My orders</span>
            <span className="text-[13px] font-medium text-sand/65 cursor-pointer">My money</span>
            <span className="text-xs text-sand/55">Lekki Phase 1 ▾</span>
            <span className="w-9 h-9 rounded-full bg-sand/[.12] flex items-center justify-center text-lime font-display font-semibold text-[13px]">K</span>
          </div>
        </div>

        <div className="relative flex-1 flex flex-col items-center justify-center gap-6 px-9">
          <span className="font-display font-bold text-[clamp(34px,4vw,46px)] leading-[1.05] text-sand tracking-tight text-center">
            What do you need moved?
          </span>
          <div className="flex items-center gap-2.5 bg-sand rounded-[18px] px-[22px] py-[18px] shadow-[0_24px_60px_rgba(0,0,0,.4)] w-[640px] max-w-full">
            <SearchIcon size={20} />
            <input
              type="text"
              placeholder="Tell us — e.g. “12kg gas to Surulere” or “documents to Ikeja by 3pm”"
              className="flex-1 min-w-0 bg-transparent outline-none text-ink placeholder:text-mid"
            />
            <button className="font-display text-[13px] font-semibold text-emerald bg-lime rounded-[11px] px-[18px] py-2.5 hover:bg-lime-600 transition-colors">
              Send
            </button>
          </div>
          <div className="flex gap-[11px] items-center">
            {services.map(({ id, label, Icon }) => {
              const isActive = active === id;
              return (
                <button
                  key={id}
                  onClick={() => handleServiceTap(id)}
                  className={`flex flex-row items-center gap-2 rounded-full border-2 px-[18px] py-2.5 transition-all ${
                    isActive ? 'bg-emerald-600 border-lime shadow-[0_6px_16px_rgba(10,61,44,.3)]' : 'bg-tile border-transparent hover:border-emerald'
                  }`}
                >
                  <Icon active={isActive} size={16} />
                  <span className={`text-[13px] font-semibold ${isActive ? 'text-lime font-bold' : 'text-ink'}`}>
                    {isActive ? `✓ ${label}` : label}
                  </span>
                </button>
              );
            })}
          </div>
          <span className="text-xs text-sand/55">
            You tapped <b className="text-lime">{activeLabel}</b> — {hintFor[active]}. Price shown before you pay. No surprises.
          </span>
        </div>

        <div className="relative flex gap-3.5 px-9 pb-6.5">
          <button className="flex-1 flex items-center gap-3 bg-sand rounded-2xl px-4.5 py-3.5 shadow-[0_10px_30px_rgba(0,0,0,.3)] text-left">
            <span className="w-[7px] h-[7px] rounded-full bg-emerald shadow-[0_0_0_4px_rgba(198,242,78,.6)]" />
            <span className="flex-1 flex flex-col">
              <span className="text-[13px] font-semibold text-ink">Your package is on its way to Yaba</span>
              <span className="text-[11px] text-mid">Musa is bringing it · Bike · picked up 22:18</span>
            </span>
            <span className="font-display text-xl font-bold text-emerald">4 min</span>
          </button>
          <button className="flex-1 flex items-center gap-3 bg-sand/[.08] border border-sand/15 rounded-2xl px-4.5 py-3.5 text-left">
            <span className="w-[7px] h-[7px] rounded-full bg-amber" />
            <span className="flex-1 flex flex-col">
              <span className="text-[13px] font-semibold text-sand">Your medicine is being checked</span>
              <span className="text-[11px] text-sand/55">A licensed pharmacist confirms it · ~8 min</span>
            </span>
            <span className="text-xs font-semibold text-lime">Open</span>
          </button>
          <button className="flex-none flex items-center bg-sand/[.08] border border-sand/15 rounded-2xl px-4.5 py-3.5">
            <span className="text-xs font-semibold text-sand">↻ Gas — same as last time</span>
          </button>
        </div>
      </div>
    </main>
  );
}
