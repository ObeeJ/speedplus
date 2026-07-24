import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export const paymentLinksApi = {
  async create(payload: { amountKobo: number; note?: string }, idempotencyKey?: string) {
    const { data } = await apiClient.post<ApiResponse<{
      slug: string; url: string; amountKobo: number; note?: string; expiresAt: string;
    }>>('/payment-links', payload, idempotencyKey ? { headers: { 'Idempotency-Key': idempotencyKey } } : undefined);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async resolve(slug: string) {
    const { data } = await apiClient.get<ApiResponse<{ amountKobo: number; note?: string; expiresAt: string }>>(
      `/pay/${slug}`,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async pay(slug: string, idempotencyKey: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(
      `/payment-links/${slug}/pay`,
      {},
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async guestPay(slug: string, payload: { email: string; callbackUrl: string }) {
    const { data } = await apiClient.post<ApiResponse<{ authorizationUrl: string; reference: string }>>(
      `/pay/${slug}/guest`,
      payload,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
