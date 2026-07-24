import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface Paycode {
  id: string;
  orderId: string;
  payload: string;
  expiresAt: string;
  confirmedAt?: string;
}

export const paycodesApi = {
  async generate(orderId: string) {
    const { data } = await apiClient.post<ApiResponse<Paycode>>('/paycodes/generate', { orderId });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async resolve(payload: string) {
    const { data } = await apiClient.post<ApiResponse<{ order: unknown }>>('/paycodes/resolve', { payload });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async confirmByCode(orderId: string, code: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>('/paycodes/confirm-code', {
      orderId,
      code,
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async confirm(paycodeId: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/paycodes/${paycodeId}/confirm`, {});
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async scanCard(cardPayload: string, pin: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>('/paycodes/scan-card', {
      cardPayload,
      pin,
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
