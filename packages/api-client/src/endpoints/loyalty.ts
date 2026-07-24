import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface LoyaltyEvent {
  id: string;
  userId: string;
  eventType: string;
  points: number;
  refId?: string;
  createdAt: string;
}

export const loyaltyApi = {
  async getBalance() {
    const { data } = await apiClient.get<ApiResponse<{ points: number }>>('/loyalty');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getHistory() {
    const { data } = await apiClient.get<ApiResponse<{ events: LoyaltyEvent[] }>>('/loyalty/history');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
