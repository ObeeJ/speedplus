import type { Coordinates } from './common';

export type DeliveryStatus =
  | 'searching_driver'
  | 'driver_assigned'
  | 'heading_to_merchant'
  | 'at_merchant'
  | 'heading_to_customer'
  | 'at_customer'
  | 'delivered'
  | 'failed';

export interface Delivery {
  id: string;
  orderId: string;
  driverId: string;
  status: DeliveryStatus;
  currentLocation?: Coordinates;
  estimatedArrivalMinutes?: number;
  route?: Route;
  proofPhotoUrl?: string;
  failureReason?: string;
}

export interface Route {
  polyline: string;
  distanceKm: number;
  durationMinutes: number;
  waypoints: Coordinates[];
}

export interface DriverLocation {
  driverId: string;
  coordinates: Coordinates;
  heading?: number;
  updatedAt: string;
}
