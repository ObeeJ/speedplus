import { apiClient } from '../client';
import type { ApiResponse } from '@fourdat/types';

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

export type FillStatus = 'good' | 'warned' | 'probation' | 'delisted';
export type LaunchStatus = 'piloting' | 'live' | 'paused';

export interface GasMerchantRow {
  id: string;
  businessName: string;
  fillAccuracyPct: number | null;
  fillSampleCount: number;
  fillStatus: FillStatus;
}

export interface ZoneRow {
  id: string;
  name: string;
  launchStatus: LaunchStatus;
  isActive: boolean;
  windowStart: number;
  windowEnd: number;
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

export interface OperationalMetrics {
  ordersToday: number;
  gmvKobo: number;
  revenueKobo: number;
  activeDrivers: number;
  activeMerchants: number;
  failedPayments: number;
  cancellations: number;
  cancellationRate: number;
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

  // Gas: fill_status
  async listGasMerchants(fillStatus?: FillStatus, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ merchants: GasMerchantRow[] }>>('/admin/gas/merchants', {
      params: { ...(fillStatus ? { fillStatus } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async setMerchantFillStatus(id: string, status: FillStatus, reason: string) {
    const { data } = await apiClient.put<ApiResponse<{ message: string }>>(`/admin/gas/merchants/${id}/fill-status`, { status, reason });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Gas: zones / launch_status
  async listZones(launchStatus?: LaunchStatus, page = 0) {
    const { data } = await apiClient.get<ApiResponse<{ zones: ZoneRow[] }>>('/admin/gas/zones', {
      params: { ...(launchStatus ? { launchStatus } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async setZoneLaunchStatus(id: string, status: LaunchStatus, reason: string) {
    const { data } = await apiClient.put<ApiResponse<{ message: string }>>(`/admin/gas/zones/${id}/launch-status`, { status, reason });
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

  // LPG price index
  async recordLPGPrice(payload: { region: string; pricePerKgKobo: number; source: string }) {
    const { data } = await apiClient.post<ApiResponse<{ entry: unknown; suggestion: string | null }>>('/admin/gas/price-index', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // Weather surcharge settings
  async getWeatherSurcharge(): Promise<{ enabled: boolean; amountKobo: number }> {
    const { data } = await apiClient.get<ApiResponse<{ enabled: boolean; amountKobo: number }>>('/admin/settings/weather');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async setWeatherSurcharge(payload: { enabled: boolean; amountKobo: number; reason: string }): Promise<void> {
    const { data } = await apiClient.put<ApiResponse<unknown>>('/admin/settings/weather', payload);
    if (!data.success) throw new Error(data.error.message);
  },

  // Operational metrics
  async getMetrics(): Promise<OperationalMetrics> {
    const { data } = await apiClient.get<ApiResponse<OperationalMetrics>>('/admin/metrics');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // ── Routes pending backend implementation ────────────────────────────────
  // These are called by admin UI pages. Backend routes will be added in the
  // next sprint. Until then the pages show empty states gracefully.

  /** GET /admin/users — list all users with optional role/search filter. */
  async listUsers(role?: string, q?: string, page = 0): Promise<{ users: Array<{ id: string; role: string; firstName: string; lastName: string; phone: string; email?: string; isVerified: boolean; isActive: boolean; createdAt: string }> }> {
    const { data } = await apiClient.get<ApiResponse<{ users: Array<{ id: string; role: string; firstName: string; lastName: string; phone: string; email?: string; isVerified: boolean; isActive: boolean; createdAt: string }> }>>('/admin/users', {
      params: { ...(role ? { role } : {}), ...(q ? { q } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  /** GET /admin/runs — list delivery runs with optional status filter. */
  async listRuns(status?: string, page = 0): Promise<{ runs: Array<{ id: string; zoneId: string; driverId?: string; windowStart: string; windowEnd: string; status: string; totalDistanceKm: number; orderCount?: number }> }> {
    const { data } = await apiClient.get<ApiResponse<{ runs: Array<{ id: string; zoneId: string; driverId?: string; windowStart: string; windowEnd: string; status: string; totalDistanceKm: number; orderCount?: number }> }>>('/admin/runs', {
      params: { ...(status ? { status } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  /** GET /admin/subscriptions — list all subscriptions with optional status filter. */
  async listSubscriptions(status?: string, page = 0): Promise<{ subscriptions: Array<{ id: string; customerId: string; merchantId: string; merchantName?: string; vertical: string; frequency: string; status: string; nextRunAt?: string; createdAt: string }> }> {
    const { data } = await apiClient.get<ApiResponse<{ subscriptions: Array<{ id: string; customerId: string; merchantId: string; merchantName?: string; vertical: string; frequency: string; status: string; nextRunAt?: string; createdAt: string }> }>>('/admin/subscriptions', {
      params: { ...(status ? { status } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  /** GET /admin/prescriptions — list all prescriptions with optional status filter. */
  async listPrescriptions(status?: string, page = 0): Promise<{ prescriptions: Array<{ id: string; customerId: string; merchantId: string; merchantName?: string; status: string; reviewNote?: string; createdAt: string; expiresAt?: string }> }> {
    const { data } = await apiClient.get<ApiResponse<{ prescriptions: Array<{ id: string; customerId: string; merchantId: string; merchantName?: string; status: string; reviewNote?: string; createdAt: string; expiresAt?: string }> }>>('/admin/prescriptions', {
      params: { ...(status ? { status } : {}), page },
    });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
