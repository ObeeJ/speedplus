export type ApiResponse<T> =
  | { success: true; data: T; meta?: PaginationMeta }
  | { success: false; error: ApiError };

export interface PaginationMeta {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

export interface ApiError {
  code: ErrorCode;
  message: string;
  field?: string;
}

export type ErrorCode =
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'VALIDATION_ERROR'
  | 'RATE_LIMITED'
  | 'INTERNAL_ERROR'
  | 'PRESCRIPTION_REQUIRED'
  | 'OUT_OF_STOCK'
  | 'AREA_NOT_COVERED'
  | 'MERCHANT_CLOSED';

export type Vertical = 'package' | 'gas' | 'grocery' | 'food' | 'pharmacy';

export type Currency = 'NGN';

export interface Money {
  amount: number;
  currency: Currency;
}

export interface Address {
  id: string;
  label?: string;
  street: string;
  city: string;
  state: string;
  country: string;
  coordinates: Coordinates;
  deliveryInstructions?: string;
}

export interface Coordinates {
  lat: number;
  lng: number;
}

export interface TimeSlot {
  id: string;
  startTime: string;
  endTime: string;
  available: boolean;
}
