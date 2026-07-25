import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface VirtualAccount {
  accountNumber: string;
  bankName: string;
  bankCode: string;
  provider: string;
}

export interface TrustTier {
  tier: number;
  tierName: string;
  completedOrders: number;
  ordersToNext: number;
  nextTierName: string;
  canPayOnArrival: boolean;
  frozen: boolean;
}

export interface SpeedPlusCard {
  payload: string;
  createdAt: string;
}

export const cardApi = {
  async getVirtualAccount() {
    const { data } = await apiClient.get<ApiResponse<VirtualAccount>>('/users/me/virtual-account');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getTrustTier() {
    const { data } = await apiClient.get<ApiResponse<TrustTier>>('/users/me/trust-tier');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getCard() {
    const { data } = await apiClient.get<ApiResponse<SpeedPlusCard>>('/users/me/card');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
