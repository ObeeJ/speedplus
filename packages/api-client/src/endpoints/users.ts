import { apiClient } from '../client';
import type { ApiResponse, User } from '@speedplus/types';

export const usersApi = {
  async me() {
    const { data } = await apiClient.get<ApiResponse<User>>('/users/me');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async updateMe(payload: Partial<Pick<User, 'firstName' | 'lastName' | 'email'>>) {
    const { data } = await apiClient.put<ApiResponse<User>>('/users/me', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getVirtualAccount() {
    const { data } = await apiClient.get<ApiResponse<{ accountNumber: string; bankName: string; bankCode: string }>>(
      '/users/me/virtual-account',
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getTrustTier() {
    const { data } = await apiClient.get<ApiResponse<{ tier: number; label: string; nextTierAt?: number }>>(
      '/users/me/trust-tier',
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getCard() {
    const { data } = await apiClient.get<ApiResponse<{ qrPayload: string; userId: string }>>(
      '/users/me/card',
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
