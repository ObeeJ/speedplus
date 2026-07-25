import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type PackageSize = 'small' | 'medium' | 'large';
export type PackageWeight = 'light' | 'medium' | 'heavy' | 'very_heavy';

export const SIZE_FEE: Record<PackageSize, number> = { small: 100, medium: 250, large: 600 };
export const WEIGHT_FEE: Record<PackageWeight, number> = { light: 0, medium: 100, heavy: 300, very_heavy: 700 };
const BASE_FARE = 100;
const PER_KM = 45;
const DEMO_KM = 6.4;

interface PackageFlowState {
  pickup: string | null;
  dropoff: string | null;
  size: PackageSize | null;
  weight: PackageWeight | null;
  orderId: string | null;
  setPickup: (v: string) => void;
  setDropoff: (v: string) => void;
  setSize: (v: PackageSize) => void;
  setWeight: (v: PackageWeight) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
  km: () => number;
  priceBreakdown: () => { base: number; distance: number; item: number; total: number };
}

export const usePackageFlowStore = create<PackageFlowState>()(
  persist(
    (set, get) => ({
      pickup: null,
      dropoff: null,
      size: null,
      weight: null,
      orderId: null,
      setPickup: (v) => set({ pickup: v }),
      setDropoff: (v) => set({ dropoff: v }),
      setSize: (v) => set({ size: v }),
      setWeight: (v) => set({ weight: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () => set({ pickup: null, dropoff: null, size: null, weight: null, orderId: null }),
      km: () => DEMO_KM,
      priceBreakdown: () => {
        const { size, weight, km } = get();
        const distance = Math.round(km() * PER_KM);
        const item = (size ? SIZE_FEE[size] : 0) + (weight ? WEIGHT_FEE[weight] : 0);
        return { base: BASE_FARE, distance, item, total: BASE_FARE + distance + item };
      },
    }),
    { name: 'speedplus-package-flow' },
  ),
);
