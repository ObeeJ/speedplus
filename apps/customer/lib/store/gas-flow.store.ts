import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { QuoteResult } from '@/lib/store/package-flow.store';

export type CylinderSize = '3' | '6' | '12.5' | '25';
export type GasMode = 'refill' | 'swap' | 'new_cylinder';

// Cylinder weight in kg by size — used only to pass weightKg to the quote API.
export const CYLINDER_KG: Record<CylinderSize, number> = { '3': 3, '6': 6, '12.5': 12.5, '25': 25 };

export interface GasAddress {
  id: string;
  label?: string;
  street: string;
  city: string;
  lat: number;
  lng: number;
}

interface GasFlowState {
  cylinder: CylinderSize | null;
  mode: GasMode | null;
  deliverToId: string | null;
  deliverToAddress: GasAddress | null; // full address for dest coords in quote
  quote: QuoteResult | null;
  orderId: string | null;
  setCylinder: (v: CylinderSize) => void;
  setMode: (v: GasMode) => void;
  setDeliverTo: (v: GasAddress) => void;
  setQuote: (v: QuoteResult) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
}

export const useGasFlowStore = create<GasFlowState>()(
  persist(
    (set) => ({
      cylinder: null,
      mode: null,
      deliverToId: null,
      deliverToAddress: null,
      quote: null,
      orderId: null,
      setCylinder: (v) => set({ cylinder: v, quote: null }),
      setMode: (v) => set({ mode: v, quote: null }),
      setDeliverTo: (v) => set({ deliverToId: v.id, deliverToAddress: v, quote: null }),
      setQuote: (v) => set({ quote: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () => set({ cylinder: null, mode: null, deliverToId: null, deliverToAddress: null, quote: null, orderId: null }),
    }),
    { name: 'fourdat-gas-flow' },
  ),
);
