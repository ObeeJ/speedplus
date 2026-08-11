import type { ApiResponse } from '@speedplus/types';
import { apiClient } from '../client';

export interface QuoteResult {
  id: string;
  totalKobo: number;
  deliveryKobo: number;
  serviceKobo: number;
  subtotalKobo: number;
  weatherSurchargeKobo: number; // 0 when surcharge is off
  distanceKm: number;
  etaMinutes: number;
  weatherAdvisory: string;
  expiresAt: string;
}

export interface QuotePayload {
  merchantId: string;
  vertical: string;
  subtotalKobo: number;
  originLat: number;
  originLng: number;
  destLat: number;
  destLng: number;
  weightKg?: number;
  sizeCategory?: string;
}

export interface MultiStopQuotePayload {
  merchantId: string;
  vertical: string;
  subtotalKobo: number;
  originLat: number;
  originLng: number;
  stops: { lat: number; lng: number }[];
  weightKg?: number;
  sizeCategory?: string;
}

export const quotesApi = {
  async quote(payload: QuotePayload): Promise<QuoteResult> {
    const { data } = await apiClient.post<ApiResponse<QuoteResult>>('/quotes', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async multiStop(payload: MultiStopQuotePayload): Promise<QuoteResult> {
    const { data } = await apiClient.post<ApiResponse<QuoteResult>>('/quotes/multistop', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
