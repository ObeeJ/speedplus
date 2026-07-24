import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type PharmacyTab = 'otc' | 'rx';
export type RxStatus = 'uploaded' | 'under_review' | 'approved';

export interface OtcItem {
  id: string;
  name: string;
  description: string;
  price: number;
}

export const OTC_ITEMS: OtcItem[] = [
  { id: 'paracetamol', name: 'Paracetamol (20 tabs)', description: 'Pain & fever relief', price: 800 },
  { id: 'ors', name: 'ORS + Zinc', description: 'Rehydration sachets', price: 1200 },
  { id: 'vitc', name: 'Vitamin C (30 tabs)', description: 'Immune support', price: 1500 },
  { id: 'malariakit', name: 'Malaria test kit', description: 'Rapid diagnostic kit', price: 2000 },
];

const BASE_FARE = 100;
const PER_KM = 45;
const DEMO_KM = 5.1;
const RX_ITEMS_PRICE = 6500;

interface PharmacyFlowState {
  tab: PharmacyTab;
  otcItemId: string | null;
  rxStatus: RxStatus | null;
  deliverTo: string | null;
  orderId: string | null;
  prescriptionId: string | null;
  setTab: (v: PharmacyTab) => void;
  setOtcItemId: (v: string) => void;
  uploadRx: () => void;
  setRxStatus: (v: RxStatus | null) => void;
  setPrescriptionId: (v: string | null) => void;
  setDeliverTo: (v: string) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
  km: () => number;
  canContinueItems: () => boolean;
  priceBreakdown: () => { base: number; distance: number; item: number; total: number };
}

export const usePharmacyFlowStore = create<PharmacyFlowState>()(
  persist(
    (set, get) => ({
      tab: 'otc',
      otcItemId: null,
      rxStatus: null,
      deliverTo: null,
      orderId: null,
      prescriptionId: null,
      setTab: (v) => set({ tab: v }),
      setOtcItemId: (v) => set({ otcItemId: v }),
      uploadRx: () => {
        set({ rxStatus: 'uploaded' });
        setTimeout(() => set({ rxStatus: 'under_review' }), 300);
        setTimeout(() => set({ rxStatus: 'approved' }), 3000);
      },
      setRxStatus: (v) => set({ rxStatus: v }),
      setPrescriptionId: (v) => set({ prescriptionId: v }),
      setDeliverTo: (v) => set({ deliverTo: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () => set({ tab: 'otc', otcItemId: null, rxStatus: null, deliverTo: null, orderId: null, prescriptionId: null }),
      km: () => DEMO_KM,
      canContinueItems: () => {
        const { tab, otcItemId, rxStatus } = get();
        return tab === 'otc' ? Boolean(otcItemId) : rxStatus === 'approved';
      },
      priceBreakdown: () => {
        const { tab, otcItemId } = get();
        const distance = Math.round(get().km() * PER_KM);
        const item = tab === 'otc' ? (OTC_ITEMS.find((o) => o.id === otcItemId)?.price ?? 0) : RX_ITEMS_PRICE;
        return { base: BASE_FARE, distance, item, total: BASE_FARE + distance + item };
      },
    }),
    { name: 'speedplus-pharmacy-flow' },
  ),
);
