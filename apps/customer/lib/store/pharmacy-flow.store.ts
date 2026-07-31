import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { QuoteResult } from '@/lib/store/package-flow.store';

export type PharmacyTab = 'otc' | 'rx';
// Mirrors the backend's real prescription.status values (service/catalog.go)
// plus the client-only 'uploading' transient state for the upload-in-flight UI.
export type RxStatus = 'uploading' | 'pending' | 'approved' | 'rejected' | 'expired';

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

export interface PharmacyAddress {
  id: string;
  label?: string;
  street: string;
  city: string;
  lat: number;
  lng: number;
}

interface PharmacyFlowState {
  tab: PharmacyTab;
  otcItemId: string | null;
  rxStatus: RxStatus | null;
  // The pharmacy the Rx was submitted to — required by the backend as of the
  // integrity fix (a prescription with no target pharmacy could never be
  // reviewed). Must be chosen before upload. Lat/lng are the quote origin.
  merchantId: string | null;
  merchantLat: number | null;
  merchantLng: number | null;
  deliverToId: string | null;
  deliverToAddress: PharmacyAddress | null;
  quote: QuoteResult | null;
  orderId: string | null;
  prescriptionId: string | null;
  setTab: (v: PharmacyTab) => void;
  setOtcItemId: (v: string) => void;
  setMerchant: (id: string, lat: number, lng: number) => void;
  setRxStatus: (v: RxStatus | null) => void;
  setPrescriptionId: (v: string | null) => void;
  setDeliverTo: (v: PharmacyAddress) => void;
  setQuote: (v: QuoteResult) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
  canContinueItems: () => boolean;
}

export const usePharmacyFlowStore = create<PharmacyFlowState>()(
  persist(
    (set, get) => ({
      tab: 'otc',
      otcItemId: null,
      rxStatus: null,
      merchantId: null,
      merchantLat: null,
      merchantLng: null,
      deliverToId: null,
      deliverToAddress: null,
      quote: null,
      orderId: null,
      prescriptionId: null,
      setTab: (v) => set({ tab: v }),
      setOtcItemId: (v) => set({ otcItemId: v, quote: null }),
      setMerchant: (id, lat, lng) => set({ merchantId: id, merchantLat: lat, merchantLng: lng, quote: null }),
      setRxStatus: (v) => set({ rxStatus: v }),
      setPrescriptionId: (v) => set({ prescriptionId: v }),
      setDeliverTo: (v) => set({ deliverToId: v.id, deliverToAddress: v, quote: null }),
      setQuote: (v) => set({ quote: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () =>
        set({
          tab: 'otc',
          otcItemId: null,
          rxStatus: null,
          merchantId: null,
          merchantLat: null,
          merchantLng: null,
          deliverToId: null,
          deliverToAddress: null,
          quote: null,
          orderId: null,
          prescriptionId: null,
        }),
      canContinueItems: () => {
        const { tab, otcItemId, rxStatus } = get();
        return tab === 'otc' ? Boolean(otcItemId) : rxStatus === 'approved';
      },
    }),
    { name: 'speedplus-pharmacy-flow' },
  ),
);
