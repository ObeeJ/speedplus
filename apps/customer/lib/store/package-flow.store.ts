import { create } from 'zustand';

export type PackageSize = 'small' | 'medium' | 'large';
export type PackageWeight = 'light' | 'medium' | 'heavy' | 'very_heavy';
export type PaymentMethod = 'wallet' | 'pay_on_arrival';

export interface AddressOption {
  id: string;
  label: string;
  street: string;
  city: string;
  lat: number;
  lng: number;
}

export interface StopInput {
  sequence: number;
  address: AddressOption;
  recipientName: string;
  recipientPhone: string;
  notes: string;
}

export interface QuoteResult {
  id: string;
  totalKobo: number;
  deliveryKobo: number;
  serviceKobo: number;
  subtotalKobo: number;
  distanceKm: number;
  etaMinutes: number;
  weatherAdvisory: string;
  expiresAt: string;
}

interface PackageFlowState {
  // Step 1: Where
  pickup: AddressOption | null;
  // Single drop-off (simple mode)
  dropoff: AddressOption | null;
  recipientName: string;
  recipientPhone: string;
  // Multi-drop mode
  isMultiDrop: boolean;
  stops: StopInput[];

  // Step 2: What
  size: PackageSize | null;
  weight: PackageWeight | null;

  // Step 3: Price
  quote: QuoteResult | null;
  paymentMethod: PaymentMethod;

  // Post-order
  orderId: string | null;

  // Setters
  setPickup: (v: AddressOption) => void;
  setDropoff: (v: AddressOption) => void;
  setRecipientName: (v: string) => void;
  setRecipientPhone: (v: string) => void;
  setIsMultiDrop: (v: boolean) => void;
  addStop: (stop: StopInput) => void;
  removeStop: (sequence: number) => void;
  setSize: (v: PackageSize) => void;
  setWeight: (v: PackageWeight) => void;
  setQuote: (v: QuoteResult) => void;
  setPaymentMethod: (v: PaymentMethod) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
}

export const usePackageFlowStore = create<PackageFlowState>()((set, get) => ({
  pickup: null,
  dropoff: null,
  recipientName: '',
  recipientPhone: '',
  isMultiDrop: false,
  stops: [],
  size: null,
  weight: null,
  quote: null,
  paymentMethod: 'wallet',
  orderId: null,
  setPickup: (v) => set({ pickup: v }),
  setDropoff: (v) => set({ dropoff: v }),
  setRecipientName: (v) => set({ recipientName: v }),
  setRecipientPhone: (v) => set({ recipientPhone: v }),
  setIsMultiDrop: (v) => set({ isMultiDrop: v, stops: v ? get().stops : [] }),
  addStop: (stop) => set((s) => ({ stops: [...s.stops.filter((x) => x.sequence !== stop.sequence), stop].sort((a, b) => a.sequence - b.sequence) })),
  removeStop: (seq) => set((s) => ({ stops: s.stops.filter((x) => x.sequence !== seq) })),
  setSize: (v) => set({ size: v }),
  setWeight: (v) => set({ weight: v }),
  setQuote: (v) => set({ quote: v }),
  setPaymentMethod: (v) => set({ paymentMethod: v }),
  setOrderId: (v) => set({ orderId: v }),
  reset: () => set({
    pickup: null, dropoff: null, recipientName: '', recipientPhone: '',
    isMultiDrop: false, stops: [], size: null, weight: null,
    quote: null, paymentMethod: 'wallet', orderId: null,
  }),
}));
