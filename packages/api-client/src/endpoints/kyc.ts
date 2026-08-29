import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export const kycApi = {
  async submitBVN(bvn: string) {
    const { data } = await apiClient.post<ApiResponse<{ status: string }>>('/kyc/check', {
      docType: 'bvn',
      params: { bvn },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async submitNIN(nin: string) {
    const { data } = await apiClient.post<ApiResponse<{ status: string }>>('/kyc/check', {
      docType: 'nin',
      params: { nin },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
