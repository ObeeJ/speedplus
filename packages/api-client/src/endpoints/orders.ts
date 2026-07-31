import type { ApiResponse, Order, CreateOrderPayload, PaginationMeta } from '@speedplus/types';
import { apiClient } from '../client';

export interface OrderStop {
  sequence: number;
  addressId: string;
  recipientName?: string;
  recipientPhone?: string;
  notes?: string;
  status: 'pending' | 'confirmed';
}

export interface ConfirmStopInput {
  sequence: number;
  code: string;
  emptyCollected?: boolean;
  emptyCylinderSerial?: string;
  capturedLat?: number;
  capturedLng?: number;
}

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

  async list(params?: { page?: number; status?: string; vertical?: string; cursor?: string }): Promise<{ orders: Order[]; meta: PaginationMeta }> {
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

  async getStops(orderId: string): Promise<OrderStop[]> {
    const { data } = await apiClient.get<ApiResponse<{ stops: OrderStop[] }>>(`/orders/${orderId}/stops`);
    if (!data.success) throw new Error(data.error.message);
    return data.data.stops;
  },

  async confirmStop(orderId: string, input: ConfirmStopInput): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/orders/${orderId}/stops/confirm`, input);
    if (!data.success) throw new Error(data.error.message);
  },

  async review(orderId: string, payload: { revieweeType: string; rating: number; comment?: string }, idempotencyKey: string): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(
      `/orders/${orderId}/review`,
      payload,
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
  },
};
