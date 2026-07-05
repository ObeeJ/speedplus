import type { ApiResponse, Order, CreateOrderPayload, PaginationMeta } from '@speedplus/types';
import { apiClient } from '../client';

export const ordersApi = {
  async create(payload: CreateOrderPayload): Promise<Order> {
    const { data } = await apiClient.post<ApiResponse<Order>>('/orders', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getById(orderId: string): Promise<Order> {
    const { data } = await apiClient.get<ApiResponse<Order>>(`/orders/${orderId}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async list(params?: { page?: number; status?: string }): Promise<{ orders: Order[]; meta: PaginationMeta }> {
    const { data } = await apiClient.get<ApiResponse<{ orders: Order[]; meta: PaginationMeta }>>('/orders', { params });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async cancel(orderId: string, reason: string): Promise<Order> {
    const { data } = await apiClient.post<ApiResponse<Order>>(`/orders/${orderId}/cancel`, { reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async track(orderId: string): Promise<Order> {
    const { data } = await apiClient.get<ApiResponse<Order>>(`/orders/${orderId}/track`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
