import type { Address, Money, Vertical } from './common';

export type OrderStatus =
  | 'pending'
  | 'confirmed'
  | 'preparing'
  | 'ready_for_pickup'
  | 'driver_assigned'
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
  deliveryAddress: Address;
  subtotal: Money;
  deliveryFee: Money;
  serviceFee: Money;
  total: Money;
  tip?: Money;
  scheduledFor?: string;
  createdAt: string;
  updatedAt: string;
  estimatedDeliveryAt?: string;
  deliveredAt?: string;
  cancellationReason?: string;
  proofOfDeliveryUrl?: string;
}

export interface OrderItem {
  id: string;
  productId: string;
  name: string;
  quantity: number;
  unitPrice: Money;
  total: Money;
  customizations?: string;
  substitutionPreference?: 'allow' | 'deny' | 'contact';
}

export interface CreateOrderPayload {
  merchantId: string;
  vertical: Vertical;
  items: Array<{ productId: string; quantity: number; customizations?: string }>;
  deliveryAddressId: string;
  tip?: number;
  scheduledFor?: string;
  prescriptionId?: string;
}
