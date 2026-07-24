import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export const kycApi = {
  async submitBVN(bvn: string) {
    const { data } = await apiClient.post<ApiResponse<{ status: string }>>('/kyc/check', {
      type: 'bvn',
      value: bvn,
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async submitNIN(nin: string) {
    const { data } = await apiClient.post<ApiResponse<{ status: string }>>('/kyc/check', {
      type: 'nin',
      value: nin,
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
