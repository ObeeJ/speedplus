import type { ApiResponse } from '@fourdat/types';
import { apiClient } from '../client';

export interface DeliveryRun {
  id: string;
  zoneId: string;
  driverId: string | null;
  status: 'pending' | 'dispatched' | 'in_progress' | 'completed' | 'cancelled';
  totalDistanceKm: number;
  orderCount: number;
  windowStart: string;
  windowEnd: string;
  createdAt: string;
}

export const runsApi = {
  async get(runId: string): Promise<DeliveryRun> {
    const { data } = await apiClient.get<ApiResponse<DeliveryRun>>(`/runs/${runId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
