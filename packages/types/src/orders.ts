import type { Money, Vertical } from './common';

export type OrderStatus =
  | 'pending'
  | 'confirmed'
  | 'preparing'
  | 'ready_for_pickup'
  | 'driver_assigned'
  | 'awaiting_collection'
  | 'empty_collected'
  | 'at_plant'
  | 'in_transit'
  | 'delivered'
  | 'cancelled'
  | 'refunded';

export interface Order {
  id: string;
  customerId: string;
  merchantId: string;
  driverId?: string;
  vertical: Vertical;
  status: OrderStatus;
  items: OrderItem[];
  // Backend returns deliveryAddressId (UUID string), not a full Address object
  deliveryAddressId: string;
  subtotal: Money;
  deliveryFee: Money;
  serviceFee: Money;
  total: Money;
  tip?: Money;
  paymentMethod: string;
  trackingRef?: string;
  declaredValueKobo?: number;
  scheduledFor?: string;
  createdAt: string;
  updatedAt: string;
  estimatedDeliveryAt?: string;
  deliveredAt?: string;
  cancellationReason?: string;
  prescriptionId?: string;
  // Driver enrichment — populated when a driver is assigned
  driverName?: string;
  driverPhone?: string;
  driverVehicle?: string;
  driverRating?: number;
}

export interface OrderItem {
  id: string;
  productId: string;
  name: string;
  quantity: number;
  unitPrice: Money;
  total: Money;
  customizations?: string;
  substitutionPreference?: string;
}

export interface CreateOrderPayload {
  merchantId: string;
  quoteId: string;
  vertical: Vertical;
  items: Array<{
    productId: string;
    name?: string;
    quantity: number;
    unitPriceKobo?: number;
    weightKg?: number;
    sizeCategory?: string;
    customizations?: string;
    substitutionPreference?: string;
  }>;
  deliveryAddressId: string;
  recipientName?: string;
  recipientPhone?: string;
  paymentMethod?: string;
  tipKobo?: number;
  declaredValueKobo?: number;
  scheduledFor?: string;
  prescriptionId?: string;
  gasMode?: 'swap' | 'refill' | 'new_cylinder';
  cylinderId?: string;
  stops?: Array<{
    sequence: number;
    addressId: string;
    recipientName?: string;
    recipientPhone?: string;
    notes?: string;
  }>;
}
