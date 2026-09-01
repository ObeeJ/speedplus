import { apiClient } from '../client';
import type { ApiResponse } from '@fourdat/types';

export interface GiftCard {
  id: string;
  amountKobo: number;
  issuerId: string;
  redeemedBy?: string;
  redeemedAt?: string;
  expiresAt?: string;
  createdAt: string;
}

export const giftCardsApi = {
  async issue(amountKobo: number, expiryDays = 365) {
    const { data } = await apiClient.post<ApiResponse<{ code: string; card: GiftCard }>>(
      '/gift-cards',
      { amountKobo, expiryDays },
      { headers: { 'Idempotency-Key': crypto.randomUUID() } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async redeem(code: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(
      '/gift-cards/redeem',
      { code },
      { headers: { 'Idempotency-Key': crypto.randomUUID() } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
