import { FEATURES } from '../features';

describe('FEATURES (packages/config)', () => {
  it('food is off', () => expect(FEATURES.food).toBe(false));
  it('grocery is off', () => expect(FEATURES.grocery).toBe(false));
  it('loyalty is off', () => expect(FEATURES.loyalty).toBe(false));
  it('giftCards is off', () => expect(FEATURES.giftCards).toBe(false));
  it('ussdFunding is off', () => expect(FEATURES.ussdFunding).toBe(false));
  it('paymentLinks is off', () => expect(FEATURES.paymentLinks).toBe(false));
});
