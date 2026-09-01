import { apiClient } from '../client';
import type { ApiResponse } from '@fourdat/types';

export interface AffordabilityResult {
  vertical: string;
  avgOrderKobo: number;
  minOrderKobo: number;
  sampleSize: number;
}

export const affordabilityApi = {
  async get(lat: number, lng: number, vertical?: string) {
    const { data } = await apiClient.get<ApiResponse<{ results: AffordabilityResult[] }>>(
      '/wallet/affordability',
      { params: { lat, lng, ...(vertical ? { vertical } : {}) } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
