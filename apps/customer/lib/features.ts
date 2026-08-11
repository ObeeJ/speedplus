// Feature flags — flip to true once the full order → settlement → refund path
// is tested end-to-end for that surface. Do not delete disabled code.
export const FEATURES = {
  loyalty: false,
  giftCards: false,
  ussdFunding: false,
  paymentLinks: false,
  grocery: false,
  food: false,
} as const;
