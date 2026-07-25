import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface MerchantSummary {
  id: string;
  businessName: string;
  vertical: string;
  status: string;
  rating: number;
  isOpen: boolean;
  lat: number;
  lng: number;
}

export interface ProductSummary {
  id: string;
  merchantId: string;
  name: string;
  description?: string;
  priceKobo: number;
  category: string;
  isAvailable: boolean;
}

export interface PrescriptionRecord {
  id: string;
  customerId: string;
  merchantId?: string;
  r2Key: string;
  status: 'pending' | 'approved' | 'rejected';
  reviewNote?: string;
  createdAt: string;
}

export const catalogApi = {
  // Merchants
  async listMerchants(vertical?: string, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ merchants: MerchantSummary[] }>>('/merchants', {
      params: { ...(vertical ? { vertical } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getMerchant(id: string) {
    const { data } = await apiClient.get<ApiResponse<MerchantSummary>>(`/merchants/${id}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Products
  async listProducts(merchantId: string, category?: string, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ products: ProductSummary[] }>>('/products', {
      params: { merchantId, ...(category ? { category } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getProduct(id: string) {
    const { data } = await apiClient.get<ApiResponse<ProductSummary>>(`/products/${id}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async searchProducts(q: string, vertical?: string) {
    const { data } = await apiClient.get<ApiResponse<{ products: ProductSummary[] }>>('/products/search', {
      params: { q, ...(vertical ? { vertical } : {}) },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Prescriptions
  async createPrescription(r2Key: string, merchantId?: string) {
    const { data } = await apiClient.post<ApiResponse<PrescriptionRecord>>('/prescriptions', {
      r2Key,
      ...(merchantId ? { merchantId } : {}),
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getPrescription(id: string) {
    const { data } = await apiClient.get<ApiResponse<PrescriptionRecord>>(`/prescriptions/${id}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async listPrescriptions() {
    const { data } = await apiClient.get<ApiResponse<{ prescriptions: PrescriptionRecord[] }>>('/prescriptions');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
