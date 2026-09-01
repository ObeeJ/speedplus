import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { QuoteResult } from '@/lib/store/package-flow.store';

export interface FoodAddress {
  id: string;
  label?: string;
  street: string;
  city: string;
  lat: number;
  lng: number;
}

interface FoodFlowState {
  // Merchant (kitchen) chosen from the list
  merchantId: string | null;
  merchantLat: number | null;
  merchantLng: number | null;
  // Single product selected from the merchant's menu
  productId: string | null;
  productPriceKobo: number | null;
  // Delivery destination
  deliverToId: string | null;
  deliverToAddress: FoodAddress | null;
  // Quote from server — set before navigating to price page
  quote: QuoteResult | null;
  orderId: string | null;
  setMerchant: (id: string, lat: number, lng: number) => void;
  setProduct: (id: string, priceKobo: number) => void;
  setDeliverTo: (v: FoodAddress) => void;
  setQuote: (v: QuoteResult) => void;
  setOrderId: (v: string | null) => void;
  reset: () => void;
}

export const useFoodFlowStore = create<FoodFlowState>()(
  persist(
    (set) => ({
      merchantId: null,
      merchantLat: null,
      merchantLng: null,
      productId: null,
      productPriceKobo: null,
      deliverToId: null,
      deliverToAddress: null,
      quote: null,
      orderId: null,
      setMerchant: (id, lat, lng) => set({ merchantId: id, merchantLat: lat, merchantLng: lng, productId: null, productPriceKobo: null, quote: null }),
      setProduct: (id, priceKobo) => set({ productId: id, productPriceKobo: priceKobo, quote: null }),
      setDeliverTo: (v) => set({ deliverToId: v.id, deliverToAddress: v, quote: null }),
      setQuote: (v) => set({ quote: v }),
      setOrderId: (v) => set({ orderId: v }),
      reset: () => set({ merchantId: null, merchantLat: null, merchantLng: null, productId: null, productPriceKobo: null, deliverToId: null, deliverToAddress: null, quote: null, orderId: null }),
    }),
    { name: 'fourdat-food-flow' },
  ),
);
