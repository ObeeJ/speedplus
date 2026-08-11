import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface Subscription {
  id: string;
  customerId: string;
  merchantId: string;
  vertical: string;
  cadence: 'weekly' | 'biweekly' | 'monthly';
  addressId: string;
  paymentMethod: 'wallet' | 'card';
  status: 'active' | 'paused' | 'cancelled';
  nextChargeAt: string;
  dunningCount: number;
  createdAt: string;
}

export const subscriptionsApi = {
  async list(): Promise<Subscription[]> {
    const { data } = await apiClient.get<ApiResponse<{ subscriptions: Subscription[] }>>('/subscriptions');
    if (!data.success) throw new Error(data.error.message);
    return data.data.subscriptions;
  },

  async create(payload: {
    merchantId: string;
    vertical: string;
    cadence: 'weekly' | 'biweekly' | 'monthly';
    addressId: string;
    paymentMethod: 'wallet' | 'card';
  }) {
    const { data } = await apiClient.post<ApiResponse<Subscription>>('/subscriptions', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async pause(id: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/subscriptions/${id}/pause`, {});
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async cancel(id: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/subscriptions/${id}/cancel`, {});
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
