import type { ApiResponse, Prescription } from '@speedplus/types';
import { apiClient } from '../client';

export const prescriptionsApi = {
  async upload(pharmacyId: string, imageFile: File): Promise<Prescription> {
    const form = new FormData();
    form.append('pharmacyId', pharmacyId);
    form.append('image', imageFile);
    const { data } = await apiClient.post<ApiResponse<Prescription>>('/prescriptions', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getById(prescriptionId: string): Promise<Prescription> {
    const { data } = await apiClient.get<ApiResponse<Prescription>>(`/prescriptions/${prescriptionId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async list(): Promise<Prescription[]> {
    const { data } = await apiClient.get<ApiResponse<Prescription[]>>('/prescriptions');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
