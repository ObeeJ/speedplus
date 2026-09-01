import { apiClient } from '../client';
import type { ApiResponse } from '@fourdat/types';

export interface USSDBank {
  code: string;
  name: string;
}

export interface USSDIntent {
  id: string;
  ussdCode: string;
  bankName: string;
  amountKobo: number;
  expiresAt: string;
  status: 'pending' | 'paid' | 'expired' | 'failed';
}

export const ussdApi = {
  async getBanks() {
    const { data } = await apiClient.get<ApiResponse<{ banks: USSDBank[] }>>('/wallet/ussd/banks');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async initiate(
    payload: { bankCode: string; amountKobo: number; email: string },
    idempotencyKey: string,
  ) {
    const { data } = await apiClient.post<ApiResponse<USSDIntent>>(
      '/wallet/ussd/initiate',
      payload,
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getIntentStatus(intentId: string) {
    const { data } = await apiClient.get<ApiResponse<{ id: string; status: string; paidAt?: string }>>(
      `/wallet/ussd/intents/${intentId}`,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
