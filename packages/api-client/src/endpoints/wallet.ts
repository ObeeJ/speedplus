import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export const walletApi = {
  async getBalance() {
    const { data } = await apiClient.get<ApiResponse<{ balanceKobo: number; currency: string }>>('/wallet');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getTransactions(cursor?: string) {
    const { data } = await apiClient.get<ApiResponse<{ transactions: unknown[] }>>('/wallet/transactions', {
      params: cursor ? { cursor } : undefined,
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async fund(payload: { amountKobo: number; email: string; callbackUrl: string }, idempotencyKey: string) {
    const { data } = await apiClient.post<ApiResponse<{ authorizationUrl: string; reference: string }>>(
      '/wallet/fund',
      payload,
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async fundCrypto(payload: { amountKobo: number; email: string; fullName: string; callbackUrl: string }, idempotencyKey: string) {
    const { data } = await apiClient.post<ApiResponse<{ authorizationUrl: string; reference: string }>>(
      '/wallet/fund/crypto',
      payload,
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async transfer(
    payload: { recipientId?: string; username?: string; phone?: string; amountKobo: number; pin: string },
    idempotencyKey: string,
  ) {
    const { data } = await apiClient.post<ApiResponse<{ message: string; recipientName: string }>>(
      '/wallet/transfer',
      payload,
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
