export { apiClient, setAuthToken, getAuthToken, setRefreshToken, getRefreshToken } from './client';
export { buildWsUrl, buildWsProtocols } from './ws';
export { SpeedPlusError } from './errors';

// Auth & users
export { authApi } from './endpoints/auth';
export { usersApi } from './endpoints/users';
export type { SavedAddress, CreateAddressPayload, DriverProfileData, DriverBadge } from './endpoints/users';
export { kycApi } from './endpoints/kyc';

// Catalog (replaces phantom productsApi + prescriptionsApi)
export { catalogApi } from './endpoints/catalog';
export type { MerchantSummary, ProductSummary, PrescriptionRecord } from './endpoints/catalog';

// Orders & dispatch
export { ordersApi } from './endpoints/orders';
export type { OrderStop, ConfirmStopInput, OrderReceipt } from './endpoints/orders';
export { quotesApi } from './endpoints/quotes';
export type { QuoteResult, QuotePayload, MultiStopQuotePayload } from './endpoints/quotes';
export { dispatchApi } from './endpoints/dispatch';
export { paycodesApi } from './endpoints/paycodes';
export { proofApi, sha256Hex } from './endpoints/proof';
export type { ProofKind, ProofMediaView } from './endpoints/proof';
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

// Merchant self-service
export { merchantApi } from './endpoints/merchant';
export type { MerchantProfile, MerchantProduct, MerchantOrder, ProductInput, BankAccount, MerchantPrescription } from './endpoints/merchant';

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
  FeeConfig,
  FuelSuggestion,
  GasMerchantRow,
  ZoneRow,
  FillStatus,
  LaunchStatus,
  OperationalMetrics,
} from './endpoints/admin';

// Gas vertical
export { gasApi, cylindersApi } from './endpoints/gas';
export type { CylinderSpec, CustomerCylinder, RegisterCylinderInput, LPGPriceEntry } from './endpoints/gas';
export { runsApi } from './endpoints/runs';
export type { DeliveryRun } from './endpoints/runs';
