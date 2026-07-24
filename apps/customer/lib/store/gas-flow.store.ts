import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type CylinderSize = '3' | '6' | '12.5' | '25';
export type GasMode = 'refill' | 'swap';

export const CYLINDER_PRICE: Record<CylinderSize, number> = { '3': 3200, '6': 6100, '12.5': 11800, '25': 23000 };
const SWAP_FEE = 500;
const BASE_FARE = 100;
const PER_KM = 45;
const DEMO_KM = 4.8;

interface GasFlowState {
  cylinder: CylinderSize | null;
  mode: GasMode | null;
  deliverTo: string | null;
  orderId: string | null;
  setCylinder: (v: CylinderSize) => void;
  setMode: (v: GasMode) => void;
  setDeliverTo: (v: string) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
  km: () => number;
  priceBreakdown: () => { base: number; distance: number; item: number; swap: number; total: number };
}

export const useGasFlowStore = create<GasFlowState>()(
  persist(
    (set, get) => ({
      cylinder: null,
      mode: null,
      deliverTo: null,
      orderId: null,
      setCylinder: (v) => set({ cylinder: v }),
      setMode: (v) => set({ mode: v }),
      setDeliverTo: (v) => set({ deliverTo: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () => set({ cylinder: null, mode: null, deliverTo: null, orderId: null }),
      km: () => DEMO_KM,
      priceBreakdown: () => {
        const { cylinder, mode, km } = get();
        const distance = Math.round(km() * PER_KM);
        const item = cylinder ? CYLINDER_PRICE[cylinder] : 0;
        const swap = mode === 'swap' ? SWAP_FEE : 0;
        return { base: BASE_FARE, distance, item, swap, total: BASE_FARE + distance + item + swap };
      },
    }),
    { name: 'speedplus-gas-flow' },
  ),
);
