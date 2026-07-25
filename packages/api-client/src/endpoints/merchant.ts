import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface MerchantProfile {
  id: string;
  businessName: string;
  vertical: string;
  status: 'pending' | 'active' | 'suspended';
  isOpen: boolean;
  rating: number;
  kycStatus: 'not_started' | 'pending' | 'under_review' | 'approved' | 'rejected';
}

export interface MerchantProduct {
  id: string;
  merchantId: string;
  name: string;
  description?: string;
  priceKobo: number;
  category: string;
  isAvailable: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface MerchantOrder {
  id: string;
  customerId: string;
  vertical: string;
  status: string;
  subtotal: { amount: number; currency: string };
  deliveryFee: { amount: number; currency: string };
  total: { amount: number; currency: string };
  paymentMethod: string;
  recipientName?: string;
  recipientPhone?: string;
  createdAt: string;
  updatedAt: string;
  items: Array<{
    id: string;
    name: string;
    quantity: number;
    unitPrice: { amount: number; currency: string };
    total: { amount: number; currency: string };
  }>;
}

export interface ProductInput {
  name: string;
  description?: string;
  priceKobo: number;
  category: string;
  isAvailable: boolean;
}

export const merchantApi = {
  async getProfile(): Promise<MerchantProfile> {
    const { data } = await apiClient.get<ApiResponse<MerchantProfile>>('/merchant/profile');
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },

  async setOpen(isOpen: boolean): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ isOpen: boolean }>>('/merchant/status', { isOpen });
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
  },

  async listOrders(status?: string, cursor?: string): Promise<{ orders: MerchantOrder[] }> {
    const { data } = await apiClient.get<ApiResponse<{ orders: MerchantOrder[] }>>('/merchant/orders', {
      params: { ...(status && { status }), ...(cursor && { cursor }) },
    });
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },

  async transitionOrder(orderId: string, to: string, note?: string): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(
      `/merchant/orders/${orderId}/transition`,
      { to, note },
    );
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
  },

  async listProducts(): Promise<{ products: MerchantProduct[] }> {
    const { data } = await apiClient.get<ApiResponse<{ products: MerchantProduct[] }>>('/merchant/products');
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },

  async createProduct(input: ProductInput): Promise<MerchantProduct> {
    const { data } = await apiClient.post<ApiResponse<MerchantProduct>>('/merchant/products', input);
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },

  async updateProduct(productId: string, input: ProductInput): Promise<MerchantProduct> {
    const { data } = await apiClient.put<ApiResponse<MerchantProduct>>(`/merchant/products/${productId}`, input);
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },

  async setProductAvailability(productId: string, available: boolean): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ available: boolean }>>(
      `/merchant/products/${productId}/availability`,
      { available },
    );
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
  },

  async getWallet(): Promise<{ balanceKobo: number; currency: string }> {
    const { data } = await apiClient.get<ApiResponse<{ balanceKobo: number; currency: string }>>('/merchant/wallet');
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },

  async getTransactions(cursor?: string) {
    const { data } = await apiClient.get<ApiResponse<{ transactions: unknown[] }>>('/merchant/wallet/transactions', {
      params: cursor ? { cursor } : undefined,
    });
    if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
    return data.data;
  },
};
