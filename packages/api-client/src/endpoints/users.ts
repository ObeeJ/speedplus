import { apiClient } from '../client';
import type { ApiResponse, User } from '@fourdat/types';

export interface SavedAddress {
  id: string;
  label?: string;
  street: string;
  city: string;
  state: string;
  country: string;
  lat: number;
  lng: number;
  deliveryInstructions?: string;
  isDefault: boolean;
}

export interface CreateAddressPayload {
  label?: string;
  street: string;
  city: string;
  state: string;
  lat: number;
  lng: number;
  deliveryInstructions?: string;
  isDefault?: boolean;
}

export interface DriverProfileData {
  userId: string;
  status: string;
  vehicleType: string;
  vehiclePlate: string;
  rating: number;
  totalDeliveries: number;
  isOnline: boolean;
  hazmatCertified: boolean;
}

export interface DriverBadge {
  badgeType: string;
  awardedAt: string;
}

export const usersApi = {
  async me() {
    const { data } = await apiClient.get<ApiResponse<User>>('/users/me');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async updateMe(payload: Partial<Pick<User, 'firstName' | 'lastName' | 'email'>>) {
    const { data } = await apiClient.put<ApiResponse<User>>('/users/me', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async listAddresses(): Promise<SavedAddress[]> {
    const { data } = await apiClient.get<ApiResponse<{ addresses: SavedAddress[] }>>('/users/me/addresses');
    if (!data.success) throw new Error(data.error.message);
    return data.data.addresses;
  },

  async createAddress(payload: CreateAddressPayload): Promise<SavedAddress> {
    const { data } = await apiClient.post<ApiResponse<SavedAddress>>('/users/me/addresses', payload);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getDriverProfile(): Promise<DriverProfileData> {
    const { data } = await apiClient.get<ApiResponse<DriverProfileData>>('/users/me/driver-profile');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  /** Returns the merchant profile for the authenticated merchant user.
   *  Distinct from `/merchant/profile` which returns the full merchant record. */
  async getMerchantProfile(): Promise<{ userId: string; businessName: string; vertical: string; status: string; rating: number; isOpen: boolean }> {
    const { data } = await apiClient.get<ApiResponse<{ userId: string; businessName: string; vertical: string; status: string; rating: number; isOpen: boolean }>>('/users/me/merchant-profile');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getDriverBadges(driverId: string): Promise<DriverBadge[]> {
    const { data } = await apiClient.get<ApiResponse<{ badges: DriverBadge[] }>>(`/drivers/${driverId}/badges`);
    if (!data.success) throw new Error(data.error.message);
    return data.data.badges;
  },

  async getVirtualAccount() {
    const { data } = await apiClient.get<ApiResponse<{ accountNumber: string; bankName: string; bankCode: string }>>(
      '/users/me/virtual-account',
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getTrustTier() {
    const { data } = await apiClient.get<ApiResponse<{ tier: number; label: string; nextTierAt?: number }>>(
      '/users/me/trust-tier',
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getCard() {
    const { data } = await apiClient.get<ApiResponse<{ qrPayload: string; userId: string }>>(
      '/users/me/card',
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
