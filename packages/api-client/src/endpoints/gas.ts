import type { ApiResponse } from '@speedplus/types';
import { apiClient } from '../client';

export interface CylinderSpec {
  id: string;
  sizeKg: number;
  label: string;
  valveType: string;
  tareKg: number;
}

export interface CustomerCylinder {
  id: string;
  specId: string;
  serial: string;
  manufactureYear: number;
  lastRecertAt: string | null;
  status: 'active' | 'retired' | 'in_custody';
}

export interface RegisterCylinderInput {
  specId: string;
  serial: string;
  manufactureYear: number;
  lastRecertAt?: string;
}

export interface LPGPriceEntry {
  id: string;
  region: string;
  pricePerKgKobo: number;
  source: string;
  effectiveAt: string;
}

export const gasApi = {
  async listSpecs(): Promise<CylinderSpec[]> {
    const { data } = await apiClient.get<ApiResponse<CylinderSpec[]>>('/gas/specs');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getPriceIndex(region = 'Lagos'): Promise<LPGPriceEntry> {
    const { data } = await apiClient.get<ApiResponse<LPGPriceEntry>>('/gas/price-index', { params: { region } });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};

export const cylindersApi = {
  async list(): Promise<CustomerCylinder[]> {
    const { data } = await apiClient.get<ApiResponse<CustomerCylinder[]>>('/cylinders');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async register(input: RegisterCylinderInput): Promise<CustomerCylinder> {
    const { data } = await apiClient.post<ApiResponse<CustomerCylinder>>('/cylinders', input);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async retire(cylinderId: string): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<null>>(`/cylinders/${cylinderId}/retire`);
    if (!data.success) throw new Error(data.error.message);
  },
};
