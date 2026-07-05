import type { ApiResponse, Product, Vertical, PaginationMeta } from '@speedplus/types';
import { apiClient } from '../client';

export const productsApi = {
  async list(params: { vertical: Vertical; merchantId?: string; category?: string; page?: number }): Promise<{ products: Product[]; meta: PaginationMeta }> {
    const { data } = await apiClient.get<ApiResponse<{ products: Product[]; meta: PaginationMeta }>>('/products', { params });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getById(productId: string): Promise<Product> {
    const { data } = await apiClient.get<ApiResponse<Product>>(`/products/${productId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async search(query: string, vertical?: Vertical): Promise<Product[]> {
    const { data } = await apiClient.get<ApiResponse<Product[]>>('/products/search', { params: { q: query, vertical } });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
