import type { Address } from './common';

export type UserRole = 'customer' | 'driver' | 'merchant' | 'admin';

export interface User {
  id: string;
  role: UserRole;
  firstName: string;
  lastName: string;
  phone: string;
  email?: string;
  avatarUrl?: string;
  createdAt: string;
  isVerified: boolean;
}

export interface CustomerProfile extends User {
  role: 'customer';
  savedAddresses: Address[];
  isFlexPlus: boolean;
}

export type DriverStatus = 'pending' | 'under_review' | 'approved' | 'suspended';
export type VehicleType = 'bicycle' | 'motorcycle' | 'car' | 'van';

export interface DriverProfile extends User {
  role: 'driver';
  status: DriverStatus;
  vehicleType: VehicleType;
  vehiclePlate: string;
  rating: number;
  totalDeliveries: number;
  isOnline: boolean;
}

export type MerchantVertical = 'gas' | 'grocery' | 'food' | 'pharmacy';
export type MerchantStatus = 'pending' | 'active' | 'suspended';

export interface MerchantProfile extends User {
  role: 'merchant';
  businessName: string;
  vertical: MerchantVertical;
  status: MerchantStatus;
  rating: number;
  isOpen: boolean;
  openingHours: OpeningHours;
  licenceNumber?: string;
}

export interface OpeningHours {
  [day: string]: { open: string; close: string } | null;
}
