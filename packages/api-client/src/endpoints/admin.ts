import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

// ── Shared types ──────────────────────────────────────────────────────────────

export interface KYCCheck {
  id: string;
  userId: string;
  docType: string;
  status: 'pending' | 'under_review' | 'approved' | 'rejected';
  providerRef?: string;
  reviewNote?: string;
  createdAt: string;
}

export interface MerchantRow {
  id: string;
  userId: string;
  businessName: string;
  vertical: string;
  status: 'pending' | 'active' | 'suspended';
  rating: number;
  createdAt: string;
}

export interface DriverRow {
  id: string;
  userId: string;
  status: 'pending' | 'under_review' | 'approved' | 'suspended';
  vehicleType: string;
  vehiclePlate: string;
  rating: number;
  totalDeliveries: number;
  createdAt: string;
}

export interface OrderSummary {
  id: string;
  customerId: string;
  merchantId: string;
  driverId?: string;
  vertical: string;
  status: string;
  totalKobo: number;
  createdAt: string;
}

export interface OrderEvent {
  id: string;
  orderId: string;
  fromStatus: string;
  toStatus: string;
  actorId: string;
  actorRole: string;
  note?: string;
  createdAt: string;
}

export interface OrderDetail extends OrderSummary {
  items: Array<{
    id: string;
    name: string;
    quantity: number;
    unitPriceKobo: number;
    totalKobo: number;
  }>;
  events: OrderEvent[];
}

export interface CancellationRule {
  id: string;
  vertical: string;
  orderStatusAtCancel: string;
  merchantCompKobo: number;
  merchantCompPct: number;
  riderCompPctOfDelivery: number;
  fullRefund: boolean;
}

export interface FeeConfig {
  id: string;
  vertical: string;
  baseFeeKobo: number;
  perKmKobo: number;
  perKgKobo: number;
  servicePct: number;
  merchantTakeRate: number;
  driverTakeRate: number;
  platformTakeRate: number;
  fuelPriceRefKobo: number;
  effectiveAt: string;
  updatedBy: string;
  reason: string;
  createdAt: string;
}

export interface FuelSuggestion {
  prevFuelKobo: number;
  newFuelKobo: number;
  currentPerKmKobo: number;
  suggestedPerKmKobo: number;
}

export interface LedgerEntry {
  id: string;
  journalId: string;
  accountId: string;
  amountKobo: number;
  description: string;
  refType: string;
  refId?: string;
  createdAt: string;
}

// ── KYC ───────────────────────────────────────────────────────────────────────

export const adminApi = {
  // KYC
  async getKYCQueue() {
    const { data } = await apiClient.get<ApiResponse<{ checks: KYCCheck[] }>>('/admin/kyc/queue');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async approveKYC(id: string, note?: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/kyc/${id}/approve`, { note });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async rejectKYC(id: string, note: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/kyc/${id}/reject`, { note });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Merchants
  async listMerchants(status?: string, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ merchants: MerchantRow[] }>>('/admin/merchants', {
      params: { ...(status ? { status } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async setMerchantStatus(id: string, status: 'active' | 'suspended', reason?: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/merchants/${id}/status`, { status, reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Drivers
  async listDrivers(status?: string, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ drivers: DriverRow[] }>>('/admin/drivers', {
      params: { ...(status ? { status } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async setDriverStatus(id: string, status: 'approved' | 'suspended' | 'under_review', reason?: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/drivers/${id}/status`, { status, reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Orders
  async searchOrders(q?: string, status?: string, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ orders: OrderSummary[] }>>('/admin/orders', {
      params: { ...(q ? { q } : {}), ...(status ? { status } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getOrderDetail(id: string) {
    const { data } = await apiClient.get<ApiResponse<OrderDetail>>(`/admin/orders/${id}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async assignDriver(orderId: string, driverId: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/dispatch/${orderId}/assign`, { driverId });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Disputes
  async freezeEscrow(orderId: string, reason: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/disputes/${orderId}/freeze`, { reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async releaseEscrow(orderId: string, recipient: 'customer' | 'merchant', reason: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(`/admin/disputes/${orderId}/release`, { recipient, reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Cancellation rules
  async listCancellationRules() {
    const { data } = await apiClient.get<ApiResponse<{ rules: CancellationRule[] }>>('/admin/settings/cancellation-rules');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async upsertCancellationRule(rule: Omit<CancellationRule, 'id'>) {
    const { data } = await apiClient.put<ApiResponse<CancellationRule>>('/admin/settings/cancellation-rules', rule);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async deleteCancellationRule(id: string) {
    const { data } = await apiClient.delete<ApiResponse<{ message: string }>>(`/admin/settings/cancellation-rules/${id}`);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Fee configs (pricing engine)
  async listFeeConfigs() {
    const { data } = await apiClient.get<ApiResponse<{ configs: FeeConfig[] }>>('/admin/settings/fees');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async upsertFeeConfig(config: Omit<FeeConfig, 'id' | 'effectiveAt' | 'updatedBy' | 'createdAt'>) {
    const { data } = await apiClient.put<ApiResponse<{ config: FeeConfig; fuelSuggestion: FuelSuggestion | null }>>(
      '/admin/settings/fees',
      config,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Ledger viewer
  async getLedger(userId: string, cursor?: string) {
    const { data } = await apiClient.get<ApiResponse<{ entries: LedgerEntry[] }>>('/admin/ledger', {
      params: { userId, ...(cursor ? { cursor } : {}) },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
