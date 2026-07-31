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
  merchantId: string;
  r2Key: string;
  status: 'pending' | 'approved' | 'rejected' | 'consumed' | 'expired';
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
  // presignPrescription must be called first — it returns a short-lived R2
  // upload URL and the server-derived object key. The caller PUTs the file
  // bytes to uploadUrl, then passes the returned key into createPrescription.
  // There is no other way for r2Key to become valid: the backend never
  // accepts a client-invented key.
  async presignPrescription(contentType: string) {
    const { data } = await apiClient.post<ApiResponse<{ uploadUrl: string; key: string }>>(
      '/prescriptions/presign',
      { contentType },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async createPrescription(r2Key: string, merchantId: string) {
    const { data } = await apiClient.post<ApiResponse<PrescriptionRecord>>('/prescriptions', {
      r2Key,
      merchantId,
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
