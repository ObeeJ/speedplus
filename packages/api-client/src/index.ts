export { apiClient, setAuthToken } from './client';
export { SpeedPlusError } from './errors';

// Auth & users
export { authApi } from './endpoints/auth';
export { usersApi } from './endpoints/users';
export { kycApi } from './endpoints/kyc';

// Catalog (replaces phantom productsApi + prescriptionsApi)
export { catalogApi } from './endpoints/catalog';
export type { MerchantSummary, ProductSummary, PrescriptionRecord } from './endpoints/catalog';

// Orders & dispatch
export { ordersApi } from './endpoints/orders';
export { dispatchApi } from './endpoints/dispatch';
export { paycodesApi } from './endpoints/paycodes';
export type { Paycode } from './endpoints/paycodes';

// Wallet & payments
export { walletApi } from './endpoints/wallet';
export { paymentLinksApi } from './endpoints/payment-links';
export { ussdApi } from './endpoints/ussd';
export type { USSDBank, USSDIntent } from './endpoints/ussd';
export { affordabilityApi } from './endpoints/affordability';
export type { AffordabilityResult } from './endpoints/affordability';

// Card, DVA, trust tier
export { cardApi } from './endpoints/card';
export type { VirtualAccount, TrustTier, SpeedPlusCard } from './endpoints/card';

// Growth
export { loyaltyApi } from './endpoints/loyalty';
export type { LoyaltyEvent } from './endpoints/loyalty';
export { giftCardsApi } from './endpoints/gift-cards';
export type { GiftCard } from './endpoints/gift-cards';
export { subscriptionsApi } from './endpoints/subscriptions';
export type { Subscription } from './endpoints/subscriptions';

// Driver
export { earningsApi } from './endpoints/earnings';

// Admin
export { adminApi } from './endpoints/admin';
export type {
  KYCCheck,
  MerchantRow,
  DriverRow,
  OrderSummary,
  OrderDetail,
  OrderEvent,
  CancellationRule,
  LedgerEntry,
} from './endpoints/admin';
