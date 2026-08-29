import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export const dispatchApi = {
  async setOnline(online: boolean) {
    const { data } = await apiClient.patch<ApiResponse<{ message: string }>>('/drivers/online', { online });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async updateLocation(lat: number, lng: number) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>('/drivers/location', { lat, lng });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async acceptOffer(offerId: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/drivers/offers/${offerId}/accept`, {});
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async rejectOffer(offerId: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/drivers/offers/${offerId}/reject`, {});
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
