// Tests for every api-client endpoint module.
// Strategy: mock the shared apiClient axios instance via axios-mock-adapter,
// then call the real exported functions and assert success/error behaviour.
// This covers the only testable logic in these files: the success/error branch.

import MockAdapter from 'axios-mock-adapter';
import { apiClient } from '../src/client';
import { authApi } from '../src/endpoints/auth';
import { usersApi } from '../src/endpoints/users';
import { walletApi } from '../src/endpoints/wallet';
import { ordersApi } from '../src/endpoints/orders';
import { quotesApi } from '../src/endpoints/quotes';
import { catalogApi } from '../src/endpoints/catalog';
import { gasApi, cylindersApi } from '../src/endpoints/gas';
import { proofApi, sha256Hex } from '../src/endpoints/proof';
import { buildWsUrl, buildWsProtocols } from '../src/ws';
import { setAuthToken } from '../src/client';
import { kycApi } from '../src/endpoints/kyc';
import { dispatchApi } from '../src/endpoints/dispatch';
import { paycodesApi } from '../src/endpoints/paycodes';
import { paymentLinksApi } from '../src/endpoints/payment-links';
import { ussdApi } from '../src/endpoints/ussd';
import { loyaltyApi } from '../src/endpoints/loyalty';
import { giftCardsApi } from '../src/endpoints/gift-cards';
import { subscriptionsApi } from '../src/endpoints/subscriptions';
import { merchantApi } from '../src/endpoints/merchant';
import { earningsApi } from '../src/endpoints/earnings';
import { affordabilityApi } from '../src/endpoints/affordability';
import { cardApi } from '../src/endpoints/card';
import { runsApi } from '../src/endpoints/runs';
import { adminApi } from '../src/endpoints/admin';

const mock = new MockAdapter(apiClient);

afterEach(() => {
  mock.reset();
  setAuthToken(null);
});

// ─── helpers ─────────────────────────────────────────────────────────────────

function ok<T>(data: T) {
  return { success: true, data };
}
function fail(message: string) {
  return { success: false, error: { message } };
}

// ─── authApi ─────────────────────────────────────────────────────────────────

describe('authApi', () => {
  const tokens = { accessToken: 'a', refreshToken: 'r', user: { id: 'u1', firstName: 'A', lastName: 'B', phone: '080', role: 'customer', createdAt: '', isVerified: true } };

  it('login resolves tokens', async () => {
    mock.onPost('/auth/login').reply(200, ok(tokens));
    const res = await authApi.login({ phone: '080', password: 'pw' });
    expect(res.accessToken).toBe('a');
  });

  it('login throws on failure', async () => {
    mock.onPost('/auth/login').reply(200, fail('invalid credentials'));
    await expect(authApi.login({ phone: '080', password: 'bad' })).rejects.toThrow('invalid credentials');
  });

  it('register resolves tokens', async () => {
    mock.onPost('/auth/register').reply(200, ok(tokens));
    const res = await authApi.register({ firstName: 'A', lastName: 'B', phone: '080', password: 'pw' });
    expect(res.user.id).toBe('u1');
  });

  it('verifyPin resolves verified flag', async () => {
    mock.onPost('/auth/pin/verify').reply(200, ok({ verified: true }));
    const res = await authApi.verifyPin('1234');
    expect(res.verified).toBe(true);
  });

  it('requestOtp throws on failure', async () => {
    mock.onPost('/otp/request').reply(200, fail('rate limited'));
    await expect(authApi.requestOtp('080', 'phone_verification')).rejects.toThrow('rate limited');
  });

  it('verifyOtpCode resolves', async () => {
    mock.onPost('/otp/verify').reply(200, ok({ verified: true }));
    const res = await authApi.verifyOtpCode('080', '123456', 'phone_verification');
    expect(res.verified).toBe(true);
  });
});

// ─── usersApi ─────────────────────────────────────────────────────────────────

describe('usersApi', () => {
  const user = { id: 'u1', firstName: 'A', lastName: 'B', phone: '080', role: 'customer', createdAt: '', isVerified: true };
  const addr = { id: 'a1', street: '1 Test St', city: 'Lagos', state: 'Lagos', country: 'NG', lat: 6.5, lng: 3.3, isDefault: false };

  it('me resolves user', async () => {
    mock.onGet('/users/me').reply(200, ok(user));
    const res = await usersApi.me();
    expect(res.id).toBe('u1');
  });

  it('listAddresses resolves array', async () => {
    mock.onGet('/users/me/addresses').reply(200, ok({ addresses: [addr] }));
    const res = await usersApi.listAddresses();
    expect(res).toHaveLength(1);
    expect(res[0]!.id).toBe('a1');
  });

  it('createAddress resolves new address', async () => {
    mock.onPost('/users/me/addresses').reply(200, ok(addr));
    const res = await usersApi.createAddress({ street: '1 Test St', city: 'Lagos', state: 'Lagos', lat: 6.5, lng: 3.3 });
    expect(res.id).toBe('a1');
  });

  it('listAddresses throws on failure', async () => {
    mock.onGet('/users/me/addresses').reply(200, fail('unauthorized'));
    await expect(usersApi.listAddresses()).rejects.toThrow('unauthorized');
  });
});

// ─── walletApi ────────────────────────────────────────────────────────────────

describe('walletApi', () => {
  it('getBalance resolves', async () => {
    mock.onGet('/wallet').reply(200, ok({ balanceKobo: 500000, currency: 'NGN' }));
    const res = await walletApi.getBalance();
    expect(res.balanceKobo).toBe(500000);
  });

  it('getTransactions resolves', async () => {
    mock.onGet('/wallet/transactions').reply(200, ok({ transactions: [] }));
    const res = await walletApi.getTransactions();
    expect(res.transactions).toEqual([]);
  });

  it('fund resolves authorization url', async () => {
    mock.onPost('/wallet/fund').reply(200, ok({ authorizationUrl: 'https://pay.example.com', reference: 'ref1' }));
    const res = await walletApi.fund({ amountKobo: 100000, email: 'a@b.com', callbackUrl: 'https://cb' }, 'idem-1');
    expect(res.authorizationUrl).toBe('https://pay.example.com');
  });

  it('transfer throws on failure', async () => {
    mock.onPost('/wallet/transfer').reply(200, fail('insufficient balance'));
    await expect(walletApi.transfer({ amountKobo: 999999999, pin: '1234' }, 'idem-2')).rejects.toThrow('insufficient balance');
  });
});

// ─── quotesApi ────────────────────────────────────────────────────────────────

describe('quotesApi', () => {
  const quote = { id: 'q1', totalKobo: 175000, deliveryKobo: 90000, serviceKobo: 10000, subtotalKobo: 75000, weatherSurchargeKobo: 0, distanceKm: 4.2, etaMinutes: 18, weatherAdvisory: '', expiresAt: '2026-07-31T18:00:00Z' };

  it('quote resolves', async () => {
    mock.onPost('/quotes').reply(200, ok(quote));
    const res = await quotesApi.quote({ merchantId: 'm1', vertical: 'food', subtotalKobo: 75000, originLat: 6.5, originLng: 3.3, destLat: 6.4, destLng: 3.4 });
    expect(res.id).toBe('q1');
    expect(res.totalKobo).toBe(175000);
  });

  it('quote throws on failure', async () => {
    mock.onPost('/quotes').reply(200, fail('merchant not found'));
    await expect(quotesApi.quote({ merchantId: 'bad', vertical: 'food', subtotalKobo: 0, originLat: 0, originLng: 0, destLat: 0, destLng: 0 })).rejects.toThrow('merchant not found');
  });

  it('multiStop resolves', async () => {
    mock.onPost('/quotes/multistop').reply(200, ok(quote));
    const res = await quotesApi.multiStop({ merchantId: 'm1', vertical: 'package', subtotalKobo: 1, originLat: 6.5, originLng: 3.3, stops: [{ lat: 6.4, lng: 3.4 }] });
    expect(res.distanceKm).toBe(4.2);
  });
});

// ─── ordersApi ────────────────────────────────────────────────────────────────

describe('ordersApi', () => {
  const order = { id: 'o1', status: 'pending', vertical: 'package', merchantId: 'm1', customerId: 'u1', items: [], deliveryAddressId: 'a1', paymentMethod: 'wallet', subtotal: {}, deliveryFee: {}, serviceFee: {}, total: {}, createdAt: '', updatedAt: '' };

  it('create resolves order', async () => {
    mock.onPost('/orders').reply(201, ok(order));
    const res = await ordersApi.create({ merchantId: 'm1', quoteId: 'q1', vertical: 'package', items: [], deliveryAddressId: 'a1' }, 'idem-key-1');
    expect(res.id).toBe('o1');
  });

  it('create throws on failure', async () => {
    mock.onPost('/orders').reply(200, fail('quote expired'));
    await expect(ordersApi.create({ merchantId: 'm1', quoteId: 'q1', vertical: 'package', items: [], deliveryAddressId: 'a1' }, 'idem-key-2')).rejects.toThrow('quote expired');
  });

  it('getById resolves', async () => {
    mock.onGet('/orders/o1').reply(200, ok(order));
    const res = await ordersApi.getById('o1');
    expect(res.status).toBe('pending');
  });

  it('list resolves', async () => {
    mock.onGet('/orders').reply(200, ok({ orders: [order] }));
    const res = await ordersApi.list();
    expect(res.orders).toHaveLength(1);
  });

  it('cancel resolves with message', async () => {
    mock.onPost('/orders/o1/cancel').reply(200, ok({ message: 'order cancelled' }));
    const res = await ordersApi.cancel('o1', 'changed mind');
    expect(res.message).toBe('order cancelled');
  });

  it('track resolves', async () => {
    mock.onGet('/orders/o1/track').reply(200, ok(order));
    const res = await ordersApi.track('o1');
    expect(res.id).toBe('o1');
  });

  it('getStops resolves', async () => {
    mock.onGet('/orders/o1/stops').reply(200, ok({ stops: [{ sequence: 1, addressId: 'a1', status: 'pending' }] }));
    const res = await ordersApi.getStops('o1');
    expect(res[0]!.sequence).toBe(1);
  });

  it('confirmStop resolves without throwing', async () => {
    mock.onPost('/orders/o1/stops/confirm').reply(200, ok({ message: 'ok' }));
    await expect(ordersApi.confirmStop('o1', { sequence: 1, code: '1234' })).resolves.toBeUndefined();
  });
});

// ─── catalogApi ───────────────────────────────────────────────────────────────

describe('catalogApi', () => {
  const merchant = { id: 'm1', businessName: 'Test Kitchen', vertical: 'food', status: 'active', rating: 4.5, isOpen: true, lat: 6.5, lng: 3.3 };
  const product = { id: 'p1', merchantId: 'm1', name: 'Jollof', priceKobo: 350000, category: 'main', isAvailable: true };

  it('listMerchants resolves', async () => {
    mock.onGet('/merchants').reply(200, ok({ merchants: [merchant] }));
    const res = await catalogApi.listMerchants('food');
    expect(res.merchants[0]!.id).toBe('m1');
  });

  it('getMerchant resolves', async () => {
    mock.onGet('/merchants/m1').reply(200, ok(merchant));
    const res = await catalogApi.getMerchant('m1');
    expect(res.businessName).toBe('Test Kitchen');
  });

  it('listProducts resolves', async () => {
    mock.onGet('/products').reply(200, ok({ products: [product] }));
    const res = await catalogApi.listProducts('m1');
    expect(res.products[0]!.priceKobo).toBe(350000);
  });

  it('listProducts throws on failure', async () => {
    mock.onGet('/products').reply(200, fail('merchant not found'));
    await expect(catalogApi.listProducts('bad')).rejects.toThrow('merchant not found');
  });

  it('searchProducts resolves', async () => {
    mock.onGet('/products/search').reply(200, ok({ products: [product] }));
    const res = await catalogApi.searchProducts('jollof', 'food');
    expect(res.products).toHaveLength(1);
  });

  it('getPrescription resolves', async () => {
    const rx = { id: 'rx1', customerId: 'u1', merchantId: 'm1', r2Key: 'key', status: 'pending', createdAt: '' };
    mock.onGet('/prescriptions/rx1').reply(200, ok(rx));
    const res = await catalogApi.getPrescription('rx1');
    expect(res.status).toBe('pending');
  });
});

// ─── gasApi / cylindersApi ────────────────────────────────────────────────────

describe('gasApi', () => {
  it('listSpecs resolves', async () => {
    const spec = { id: 's1', sizeKg: 12.5, label: '12.5kg', valveType: 'standard', tareKg: 8 };
    mock.onGet('/gas/specs').reply(200, ok([spec]));
    const res = await gasApi.listSpecs();
    expect(res[0]!.sizeKg).toBe(12.5);
  });

  it('getPriceIndex resolves', async () => {
    const entry = { id: 'pi1', region: 'Lagos', pricePerKgKobo: 120000, source: 'PPPRA', effectiveAt: '' };
    mock.onGet('/gas/price-index').reply(200, ok(entry));
    const res = await gasApi.getPriceIndex('Lagos');
    expect(res.pricePerKgKobo).toBe(120000);
  });

  it('getPriceIndex throws on failure', async () => {
    mock.onGet('/gas/price-index').reply(200, fail('region not found'));
    await expect(gasApi.getPriceIndex('Unknown')).rejects.toThrow('region not found');
  });
});

describe('cylindersApi', () => {
  it('list resolves', async () => {
    const cyl = { id: 'c1', specId: 's1', serial: 'SN001', manufactureYear: 2020, lastRecertAt: null, status: 'active' };
    mock.onGet('/cylinders').reply(200, ok([cyl]));
    const res = await cylindersApi.list();
    expect(res[0]!.serial).toBe('SN001');
  });

  it('register resolves', async () => {
    const cyl = { id: 'c2', specId: 's1', serial: 'SN002', manufactureYear: 2021, lastRecertAt: null, status: 'active' };
    mock.onPost('/cylinders').reply(200, ok(cyl));
    const res = await cylindersApi.register({ specId: 's1', serial: 'SN002', manufactureYear: 2021 });
    expect(res.id).toBe('c2');
  });
});

// ─── proofApi ─────────────────────────────────────────────────────────────────

describe('proofApi', () => {
  it('presign resolves upload url and key', async () => {
    mock.onPost('/orders/o1/proof/presign').reply(200, ok({ uploadUrl: 'https://r2.example.com/upload', key: 'k1' }));
    const res = await proofApi.presign('o1', { kind: 'weight_photo', contentType: 'image/jpeg' });
    expect(res.key).toBe('k1');
  });

  it('confirm resolves id', async () => {
    mock.onPost('/orders/o1/proof/confirm').reply(200, ok({ id: 'pm1' }));
    const res = await proofApi.confirm('o1', { kind: 'weight_photo', key: 'k1', sha256: 'abc' });
    expect(res.id).toBe('pm1');
  });

  it('getMedia resolves array', async () => {
    const media = [{ id: 'pm1', kind: 'weight_photo', viewUrl: 'https://r2.example.com/view', sha256: 'abc', capturedAt: '' }];
    mock.onGet('/orders/o1/proof').reply(200, ok({ media }));
    const res = await proofApi.getMedia('o1');
    expect(res[0]!.id).toBe('pm1');
  });

  it('presign throws on failure', async () => {
    mock.onPost('/orders/o1/proof/presign').reply(200, fail('order not found'));
    await expect(proofApi.presign('o1', { kind: 'dropoff_photo', contentType: 'image/jpeg' })).rejects.toThrow('order not found');
  });
});

describe('sha256Hex', () => {
  it('returns a 64-char lowercase hex string for a blob', async () => {
    const blob = new Blob(['hello world']);
    const hex = await sha256Hex(blob);
    expect(hex).toHaveLength(64);
    expect(hex).toMatch(/^[0-9a-f]{64}$/);
    // same input always produces same digest
    expect(await sha256Hex(blob)).toBe(hex);
  });

  it('produces different digests for different inputs', async () => {
    const h1 = await sha256Hex(new Blob(['a']));
    const h2 = await sha256Hex(new Blob(['b']));
    expect(h1).not.toBe(h2);
  });
});

// ─── ws helpers ───────────────────────────────────────────────────────────────

describe('buildWsUrl / buildWsProtocols', () => {
  it('buildWsUrl returns a ws:// url', () => {
    expect(buildWsUrl()).toMatch(/^ws/);
  });

  it('buildWsProtocols returns undefined when no token', () => {
    setAuthToken(null);
    expect(buildWsProtocols()).toBeUndefined();
  });

  it('buildWsProtocols returns bearer array when token is set', () => {
    setAuthToken('tok123');
    const protocols = buildWsProtocols();
    expect(protocols).toEqual(['bearer', 'tok123']);
  });
});

// ─── kycApi ───────────────────────────────────────────────────────────────────

describe('kycApi', () => {
  it('submitBVN resolves status', async () => {
    mock.onPost('/kyc/check').reply(200, ok({ status: 'pending' }));
    const res = await kycApi.submitBVN('12345678901');
    expect(res.status).toBe('pending');
  });

  it('submitNIN resolves status', async () => {
    mock.onPost('/kyc/check').reply(200, ok({ status: 'under_review' }));
    const res = await kycApi.submitNIN('98765432100');
    expect(res.status).toBe('under_review');
  });

  it('submitBVN throws on failure', async () => {
    mock.onPost('/kyc/check').reply(200, fail('invalid bvn'));
    await expect(kycApi.submitBVN('bad')).rejects.toThrow('invalid bvn');
  });
});

// ─── dispatchApi ──────────────────────────────────────────────────────────────

describe('dispatchApi', () => {
  it('setOnline resolves', async () => {
    mock.onPatch('/drivers/online').reply(200, ok({ message: 'online' }));
    const res = await dispatchApi.setOnline(true);
    expect(res.message).toBe('online');
  });

  it('updateLocation resolves', async () => {
    mock.onPost('/drivers/location').reply(200, ok({ message: 'ok' }));
    const res = await dispatchApi.updateLocation(6.5, 3.3);
    expect(res.message).toBe('ok');
  });

  it('acceptOffer resolves', async () => {
    mock.onPost('/drivers/offers/off1/accept').reply(200, ok({ message: 'accepted' }));
    const res = await dispatchApi.acceptOffer('off1');
    expect(res.message).toBe('accepted');
  });

  it('rejectOffer resolves', async () => {
    mock.onPost('/drivers/offers/off1/reject').reply(200, ok({ message: 'rejected' }));
    const res = await dispatchApi.rejectOffer('off1');
    expect(res.message).toBe('rejected');
  });

  it('setOnline throws on failure', async () => {
    mock.onPatch('/drivers/online').reply(200, fail('not a driver'));
    await expect(dispatchApi.setOnline(true)).rejects.toThrow('not a driver');
  });
});

// ─── paycodesApi ──────────────────────────────────────────────────────────────

describe('paycodesApi', () => {
  const paycode = { id: 'pc1', orderId: 'o1', payload: 'QR_DATA', expiresAt: '2026-12-31T00:00:00Z' };

  it('generate resolves paycode', async () => {
    mock.onPost('/paycodes/generate').reply(200, ok(paycode));
    const res = await paycodesApi.generate('o1');
    expect(res.id).toBe('pc1');
  });

  it('resolve resolves order', async () => {
    mock.onPost('/paycodes/resolve').reply(200, ok({ order: { id: 'o1' } }));
    const res = await paycodesApi.resolve('QR_DATA');
    expect((res.order as { id: string }).id).toBe('o1');
  });

  it('confirmByCode resolves', async () => {
    mock.onPost('/paycodes/confirm-code').reply(200, ok({ message: 'confirmed' }));
    const res = await paycodesApi.confirmByCode('o1', '4321');
    expect(res.message).toBe('confirmed');
  });

  it('confirm resolves', async () => {
    mock.onPost('/paycodes/pc1/confirm').reply(200, ok({ message: 'confirmed' }));
    const res = await paycodesApi.confirm('pc1');
    expect(res.message).toBe('confirmed');
  });

  it('scanCard resolves', async () => {
    mock.onPost('/paycodes/scan-card').reply(200, ok({ message: 'paid' }));
    const res = await paycodesApi.scanCard('CARD_PAYLOAD', '1234');
    expect(res.message).toBe('paid');
  });

  it('generate throws on failure', async () => {
    mock.onPost('/paycodes/generate').reply(200, fail('order not found'));
    await expect(paycodesApi.generate('bad')).rejects.toThrow('order not found');
  });
});

// ─── paymentLinksApi ──────────────────────────────────────────────────────────

describe('paymentLinksApi', () => {
  const link = { slug: 'abc123', url: 'https://pay.fourdat.app/abc123', amountKobo: 50000, expiresAt: '2026-12-31T00:00:00Z' };

  it('create resolves link', async () => {
    mock.onPost('/payment-links').reply(200, ok(link));
    const res = await paymentLinksApi.create({ amountKobo: 50000 });
    expect(res.slug).toBe('abc123');
  });

  it('resolve resolves link details', async () => {
    mock.onGet('/pay/abc123').reply(200, ok({ amountKobo: 50000, expiresAt: '2026-12-31T00:00:00Z' }));
    const res = await paymentLinksApi.resolve('abc123');
    expect(res.amountKobo).toBe(50000);
  });

  it('pay resolves message', async () => {
    mock.onPost('/payment-links/abc123/pay').reply(200, ok({ message: 'paid' }));
    const res = await paymentLinksApi.pay('abc123', 'idem-pl-1');
    expect(res.message).toBe('paid');
  });

  it('guestPay resolves authorization url', async () => {
    mock.onPost('/pay/abc123/guest').reply(200, ok({ authorizationUrl: 'https://paystack.com/pay/ref', reference: 'ref1' }));
    const res = await paymentLinksApi.guestPay('abc123', { email: 'g@test.com', callbackUrl: 'https://cb' });
    expect(res.authorizationUrl).toContain('paystack');
  });

  it('create throws on failure', async () => {
    mock.onPost('/payment-links').reply(200, fail('amount too low'));
    await expect(paymentLinksApi.create({ amountKobo: 1 })).rejects.toThrow('amount too low');
  });
});

// ─── ussdApi ──────────────────────────────────────────────────────────────────

describe('ussdApi', () => {
  const intent: import('../src/endpoints/ussd').USSDIntent = {
    id: 'ui1', ussdCode: '*737*000*50000#', bankName: 'GTBank', amountKobo: 50000,
    expiresAt: '2026-12-31T00:00:00Z', status: 'pending',
  };

  it('getBanks resolves list', async () => {
    mock.onGet('/wallet/ussd/banks').reply(200, ok({ banks: [{ code: '058', name: 'GTBank' }] }));
    const res = await ussdApi.getBanks();
    expect(res.banks[0]!.code).toBe('058');
  });

  it('initiate resolves intent', async () => {
    mock.onPost('/wallet/ussd/initiate').reply(200, ok(intent));
    const res = await ussdApi.initiate({ bankCode: '058', amountKobo: 50000, email: 'a@b.com' }, 'idem-ussd-1');
    expect(res.ussdCode).toBe('*737*000*50000#');
  });

  it('getIntentStatus resolves', async () => {
    mock.onGet('/wallet/ussd/intents/ui1').reply(200, ok({ id: 'ui1', status: 'paid', paidAt: '2026-09-01T10:00:00Z' }));
    const res = await ussdApi.getIntentStatus('ui1');
    expect(res.status).toBe('paid');
  });

  it('initiate throws on failure', async () => {
    mock.onPost('/wallet/ussd/initiate').reply(200, fail('bank not supported'));
    await expect(ussdApi.initiate({ bankCode: 'bad', amountKobo: 1, email: 'a@b.com' }, 'idem-ussd-2')).rejects.toThrow('bank not supported');
  });
});

// ─── loyaltyApi ───────────────────────────────────────────────────────────────

describe('loyaltyApi', () => {
  it('getBalance resolves points', async () => {
    mock.onGet('/loyalty').reply(200, ok({ points: 420 }));
    const res = await loyaltyApi.getBalance();
    expect(res.points).toBe(420);
  });

  it('getHistory resolves events', async () => {
    const event = { id: 'le1', userId: 'u1', eventType: 'order_delivered', points: 10, createdAt: '' };
    mock.onGet('/loyalty/history').reply(200, ok({ events: [event] }));
    const res = await loyaltyApi.getHistory();
    expect(res.events[0]!.points).toBe(10);
  });

  it('getBalance throws on failure', async () => {
    mock.onGet('/loyalty').reply(200, fail('unauthorized'));
    await expect(loyaltyApi.getBalance()).rejects.toThrow('unauthorized');
  });
});

// ─── giftCardsApi ─────────────────────────────────────────────────────────────

describe('giftCardsApi', () => {
  const card = { id: 'gc1', amountKobo: 100000, issuerId: 'u1', createdAt: '' };

  it('issue resolves code and card', async () => {
    mock.onPost('/gift-cards').reply(200, ok({ code: 'GIFT-XXXX', card }));
    const res = await giftCardsApi.issue(100000);
    expect(res.code).toBe('GIFT-XXXX');
    expect(res.card.amountKobo).toBe(100000);
  });

  it('redeem resolves message', async () => {
    mock.onPost('/gift-cards/redeem').reply(200, ok({ message: 'redeemed' }));
    const res = await giftCardsApi.redeem('GIFT-XXXX');
    expect(res.message).toBe('redeemed');
  });

  it('issue throws on failure', async () => {
    mock.onPost('/gift-cards').reply(200, fail('insufficient balance'));
    await expect(giftCardsApi.issue(999999999)).rejects.toThrow('insufficient balance');
  });

  it('redeem throws on failure', async () => {
    mock.onPost('/gift-cards/redeem').reply(200, fail('code already used'));
    await expect(giftCardsApi.redeem('USED')).rejects.toThrow('code already used');
  });
});

// ─── subscriptionsApi ─────────────────────────────────────────────────────────

describe('subscriptionsApi', () => {
  const sub = {
    id: 'sub1', customerId: 'u1', merchantId: 'm1', vertical: 'gas',
    cadence: 'weekly' as const, addressId: 'a1', paymentMethod: 'wallet' as const,
    status: 'active' as const, nextChargeAt: '2026-09-08T06:00:00Z', dunningCount: 0, createdAt: '',
  };

  it('list resolves subscriptions', async () => {
    mock.onGet('/subscriptions').reply(200, ok({ subscriptions: [sub] }));
    const res = await subscriptionsApi.list();
    expect(res[0]!.id).toBe('sub1');
  });

  it('create resolves subscription', async () => {
    mock.onPost('/subscriptions').reply(200, ok(sub));
    const res = await subscriptionsApi.create({ merchantId: 'm1', vertical: 'gas', cadence: 'weekly', addressId: 'a1', paymentMethod: 'wallet' });
    expect(res.cadence).toBe('weekly');
  });

  it('pause resolves message', async () => {
    mock.onPost('/subscriptions/sub1/pause').reply(200, ok({ message: 'paused' }));
    const res = await subscriptionsApi.pause('sub1');
    expect(res.message).toBe('paused');
  });

  it('cancel resolves message', async () => {
    mock.onPost('/subscriptions/sub1/cancel').reply(200, ok({ message: 'cancelled' }));
    const res = await subscriptionsApi.cancel('sub1');
    expect(res.message).toBe('cancelled');
  });

  it('create throws on failure', async () => {
    mock.onPost('/subscriptions').reply(200, fail('merchant not found'));
    await expect(subscriptionsApi.create({ merchantId: 'bad', vertical: 'gas', cadence: 'weekly', addressId: 'a1', paymentMethod: 'wallet' })).rejects.toThrow('merchant not found');
  });
});

// ─── merchantApi ──────────────────────────────────────────────────────────────

describe('merchantApi', () => {
  const profile = { id: 'm1', businessName: 'Gas Co', vertical: 'gas', status: 'active' as const, isOpen: true, rating: 4.8, kycStatus: 'approved' as const };
  const product = { id: 'p1', merchantId: 'm1', name: '12.5kg Cylinder', priceKobo: 1500000, category: 'gas', isAvailable: true, createdAt: '', updatedAt: '' };
  const order = { id: 'o1', customerId: 'u1', vertical: 'gas', status: 'pending', subtotal: { amount: 0, currency: 'NGN' }, deliveryFee: { amount: 0, currency: 'NGN' }, total: { amount: 0, currency: 'NGN' }, paymentMethod: 'wallet', createdAt: '', updatedAt: '', items: [] };

  it('getProfile resolves', async () => {
    mock.onGet('/merchant/profile').reply(200, ok(profile));
    const res = await merchantApi.getProfile();
    expect(res.businessName).toBe('Gas Co');
  });

  it('setOpen resolves', async () => {
    mock.onPost('/merchant/status').reply(200, ok({ isOpen: false }));
    await expect(merchantApi.setOpen(false)).resolves.toBeUndefined();
  });

  it('listOrders resolves', async () => {
    mock.onGet('/merchant/orders').reply(200, ok({ orders: [order] }));
    const res = await merchantApi.listOrders();
    expect(res.orders[0]!.id).toBe('o1');
  });

  it('transitionOrder resolves', async () => {
    mock.onPost('/merchant/orders/o1/transition').reply(200, ok({ message: 'transitioned' }));
    await expect(merchantApi.transitionOrder('o1', 'confirmed')).resolves.toBeUndefined();
  });

  it('listProducts resolves', async () => {
    mock.onGet('/merchant/products').reply(200, ok({ products: [product] }));
    const res = await merchantApi.listProducts();
    expect(res.products[0]!.name).toBe('12.5kg Cylinder');
  });

  it('createProduct resolves', async () => {
    mock.onPost('/merchant/products').reply(200, ok(product));
    const res = await merchantApi.createProduct({ name: '12.5kg Cylinder', priceKobo: 1500000, category: 'gas', isAvailable: true });
    expect(res.id).toBe('p1');
  });

  it('updateProduct resolves', async () => {
    mock.onPut('/merchant/products/p1').reply(200, ok({ ...product, priceKobo: 1600000 }));
    const res = await merchantApi.updateProduct('p1', { name: '12.5kg Cylinder', priceKobo: 1600000, category: 'gas', isAvailable: true });
    expect(res.priceKobo).toBe(1600000);
  });

  it('setProductAvailability resolves', async () => {
    mock.onPost('/merchant/products/p1/availability').reply(200, ok({ available: false }));
    await expect(merchantApi.setProductAvailability('p1', false)).resolves.toBeUndefined();
  });

  it('getWallet resolves', async () => {
    mock.onGet('/merchant/wallet').reply(200, ok({ balanceKobo: 250000, currency: 'NGN' }));
    const res = await merchantApi.getWallet();
    expect(res.balanceKobo).toBe(250000);
  });

  it('getBankAccount resolves', async () => {
    const bank = { bankCode: '058', bankName: 'GTBank', accountNumber: '0123456789', accountName: 'Gas Co Ltd' };
    mock.onGet('/merchant/bank-account').reply(200, ok(bank));
    const res = await merchantApi.getBankAccount();
    expect(res!.accountName).toBe('Gas Co Ltd');
  });

  it('saveBankAccount resolves', async () => {
    const bank = { bankCode: '058', bankName: 'GTBank', accountNumber: '0123456789', accountName: 'Gas Co Ltd' };
    mock.onPost('/merchant/bank-account').reply(200, ok(bank));
    const res = await merchantApi.saveBankAccount(bank);
    expect(res.bankCode).toBe('058');
  });

  it('withdraw resolves', async () => {
    mock.onPost('/merchant/withdraw').reply(200, ok({ message: 'withdrawal initiated' }));
    await expect(merchantApi.withdraw(100000, '1234', 'idem-wd-1')).resolves.toBeUndefined();
  });

  it('listPrescriptions resolves', async () => {
    const rx = { id: 'rx1', customerId: 'u1', viewUrl: 'https://r2.example.com/rx1', status: 'pending' as const, createdAt: '' };
    mock.onGet('/merchant/prescriptions').reply(200, ok({ prescriptions: [rx] }));
    const res = await merchantApi.listPrescriptions();
    expect(res.prescriptions[0]!.id).toBe('rx1');
  });

  it('reviewPrescription resolves', async () => {
    mock.onPost('/merchant/prescriptions/rx1/review').reply(200, ok({ id: 'rx1', status: 'approved' }));
    const res = await merchantApi.reviewPrescription('rx1', true);
    expect(res.status).toBe('approved');
  });

  it('getProfile throws on failure', async () => {
    mock.onGet('/merchant/profile').reply(200, fail('not a merchant'));
    await expect(merchantApi.getProfile()).rejects.toThrow('not a merchant');
  });
});

// ─── earningsApi ──────────────────────────────────────────────────────────────

describe('earningsApi', () => {
  it('cashout resolves message', async () => {
    mock.onPost('/earnings/cashout').reply(200, ok({ message: 'cashout queued' }));
    const res = await earningsApi.cashout(50000, 'idem-ewa-1');
    expect(res.message).toBe('cashout queued');
  });

  it('getBankAccount resolves', async () => {
    const bank = { bankCode: '058', bankName: 'GTBank', accountNumber: '0123456789', accountName: 'Driver A' };
    mock.onGet('/drivers/bank-account').reply(200, ok(bank));
    const res = await earningsApi.getBankAccount();
    expect(res!.accountName).toBe('Driver A');
  });

  it('resolveAccount resolves account name', async () => {
    mock.onPost('/drivers/bank-account/resolve').reply(200, ok({ accountName: 'Driver A' }));
    const res = await earningsApi.resolveAccount('058', '0123456789');
    expect(res.accountName).toBe('Driver A');
  });

  it('saveBankAccount resolves', async () => {
    const bank = { bankCode: '058', bankName: 'GTBank', accountNumber: '0123456789', accountName: 'Driver A' };
    mock.onPost('/drivers/bank-account').reply(200, ok(bank));
    const res = await earningsApi.saveBankAccount({ bankCode: '058', bankName: 'GTBank', accountNumber: '0123456789' });
    expect(res.bankCode).toBe('058');
  });

  it('listBanks resolves', async () => {
    mock.onGet('/banks').reply(200, ok({ banks: [{ code: '058', name: 'GTBank' }] }));
    const res = await earningsApi.listBanks();
    expect(res[0]!.code).toBe('058');
  });

  it('cashout throws on failure', async () => {
    mock.onPost('/earnings/cashout').reply(200, fail('no bank account'));
    await expect(earningsApi.cashout(50000, 'idem-ewa-2')).rejects.toThrow('no bank account');
  });
});

// ─── affordabilityApi ─────────────────────────────────────────────────────────

describe('affordabilityApi', () => {
  it('get resolves results', async () => {
    const result = { vertical: 'food', avgOrderKobo: 350000, minOrderKobo: 150000, sampleSize: 42 };
    mock.onGet('/wallet/affordability').reply(200, ok({ results: [result] }));
    const res = await affordabilityApi.get(6.5, 3.3, 'food');
    expect(res.results[0]!.vertical).toBe('food');
    expect(res.results[0]!.sampleSize).toBe(42);
  });

  it('get throws on failure', async () => {
    mock.onGet('/wallet/affordability').reply(200, fail('unauthorized'));
    await expect(affordabilityApi.get(0, 0)).rejects.toThrow('unauthorized');
  });
});

// ─── cardApi ──────────────────────────────────────────────────────────────────

describe('cardApi', () => {
  it('getVirtualAccount resolves', async () => {
    const va = { accountNumber: '9900001234', bankName: 'Wema Bank', bankCode: '035', provider: 'monnify' };
    mock.onGet('/users/me/virtual-account').reply(200, ok(va));
    const res = await cardApi.getVirtualAccount();
    expect(res.accountNumber).toBe('9900001234');
  });

  it('getTrustTier resolves', async () => {
    const tier = { tier: 2, tierName: 'Silver', completedOrders: 15, ordersToNext: 10, nextTierName: 'Gold', canPayOnArrival: true, frozen: false };
    mock.onGet('/users/me/trust-tier').reply(200, ok(tier));
    const res = await cardApi.getTrustTier();
    expect(res.tier).toBe(2);
    expect(res.canPayOnArrival).toBe(true);
  });

  it('getCard resolves', async () => {
    const card = { payload: 'QR_PAYLOAD_BASE64', createdAt: '' };
    mock.onGet('/users/me/card').reply(200, ok(card));
    const res = await cardApi.getCard();
    expect(res.payload).toBe('QR_PAYLOAD_BASE64');
  });

  it('getVirtualAccount throws on failure', async () => {
    mock.onGet('/users/me/virtual-account').reply(200, fail('not onboarded'));
    await expect(cardApi.getVirtualAccount()).rejects.toThrow('not onboarded');
  });
});

// ─── runsApi ──────────────────────────────────────────────────────────────────

describe('runsApi', () => {
  const run = { id: 'r1', zoneId: 'z1', driverId: 'd1', status: 'in_progress' as const, totalDistanceKm: 12.4, orderCount: 3, windowStart: '', windowEnd: '', createdAt: '' };

  it('get resolves run', async () => {
    mock.onGet('/runs/r1').reply(200, ok(run));
    const res = await runsApi.get('r1');
    expect(res.id).toBe('r1');
    expect(res.orderCount).toBe(3);
  });

  it('get throws on failure', async () => {
    mock.onGet('/runs/bad').reply(200, fail('run not found'));
    await expect(runsApi.get('bad')).rejects.toThrow('run not found');
  });
});

// ─── adminApi ─────────────────────────────────────────────────────────────────

describe('adminApi', () => {
  it('getKYCQueue resolves', async () => {
    const check = { id: 'k1', userId: 'u1', docType: 'bvn', status: 'pending' as const, createdAt: '' };
    mock.onGet('/admin/kyc/queue').reply(200, ok({ checks: [check] }));
    const res = await adminApi.getKYCQueue();
    expect(res.checks[0]!.docType).toBe('bvn');
  });

  it('approveKYC resolves', async () => {
    mock.onPost('/admin/kyc/k1/approve').reply(200, ok({ message: 'approved' }));
    const res = await adminApi.approveKYC('k1');
    expect(res.message).toBe('approved');
  });

  it('rejectKYC resolves', async () => {
    mock.onPost('/admin/kyc/k1/reject').reply(200, ok({ message: 'rejected' }));
    const res = await adminApi.rejectKYC('k1', 'mismatch');
    expect(res.message).toBe('rejected');
  });

  it('listMerchants resolves', async () => {
    const merchant = { id: 'm1', userId: 'u1', businessName: 'Gas Co', vertical: 'gas', status: 'active' as const, rating: 4.5, createdAt: '' };
    mock.onGet('/admin/merchants').reply(200, ok({ merchants: [merchant] }));
    const res = await adminApi.listMerchants();
    expect(res.merchants[0]!.id).toBe('m1');
  });

  it('setMerchantStatus resolves', async () => {
    mock.onPost('/admin/merchants/m1/status').reply(200, ok({ message: 'updated' }));
    const res = await adminApi.setMerchantStatus('m1', 'suspended', 'fraud');
    expect(res.message).toBe('updated');
  });

  it('listDrivers resolves', async () => {
    const driver = { id: 'd1', userId: 'u2', status: 'approved' as const, vehicleType: 'bike', vehiclePlate: 'LAG-001', rating: 4.7, totalDeliveries: 120, createdAt: '' };
    mock.onGet('/admin/drivers').reply(200, ok({ drivers: [driver] }));
    const res = await adminApi.listDrivers();
    expect(res.drivers[0]!.totalDeliveries).toBe(120);
  });

  it('setDriverStatus resolves', async () => {
    mock.onPost('/admin/drivers/d1/status').reply(200, ok({ message: 'updated' }));
    const res = await adminApi.setDriverStatus('d1', 'suspended');
    expect(res.message).toBe('updated');
  });

  it('searchOrders resolves', async () => {
    const order = { id: 'o1', customerId: 'u1', merchantId: 'm1', vertical: 'gas', status: 'delivered', totalKobo: 1500000, createdAt: '' };
    mock.onGet('/admin/orders').reply(200, ok({ orders: [order] }));
    const res = await adminApi.searchOrders();
    expect(res.orders[0]!.id).toBe('o1');
  });

  it('getOrderDetail resolves', async () => {
    const detail = { id: 'o1', customerId: 'u1', merchantId: 'm1', vertical: 'gas', status: 'delivered', totalKobo: 1500000, createdAt: '', items: [], events: [] };
    mock.onGet('/admin/orders/o1').reply(200, ok(detail));
    const res = await adminApi.getOrderDetail('o1');
    expect(res.events).toEqual([]);
  });

  it('assignDriver resolves', async () => {
    mock.onPost('/admin/dispatch/o1/assign').reply(200, ok({ message: 'assigned' }));
    const res = await adminApi.assignDriver('o1', 'd1');
    expect(res.message).toBe('assigned');
  });

  it('freezeEscrow resolves', async () => {
    mock.onPost('/admin/disputes/o1/freeze').reply(200, ok({ message: 'frozen' }));
    const res = await adminApi.freezeEscrow('o1', 'dispute raised');
    expect(res.message).toBe('frozen');
  });

  it('releaseEscrow resolves', async () => {
    mock.onPost('/admin/disputes/o1/release').reply(200, ok({ message: 'released' }));
    const res = await adminApi.releaseEscrow('o1', 'customer', 'resolved in favour');
    expect(res.message).toBe('released');
  });

  it('listCancellationRules resolves', async () => {
    const rule = { id: 'cr1', vertical: 'gas', orderStatusAtCancel: 'confirmed', merchantCompKobo: 0, merchantCompPct: 0, riderCompPctOfDelivery: 0, fullRefund: true };
    mock.onGet('/admin/settings/cancellation-rules').reply(200, ok({ rules: [rule] }));
    const res = await adminApi.listCancellationRules();
    expect(res.rules[0]!.fullRefund).toBe(true);
  });

  it('upsertCancellationRule resolves', async () => {
    const rule = { id: 'cr1', vertical: 'gas', orderStatusAtCancel: 'confirmed', merchantCompKobo: 0, merchantCompPct: 0, riderCompPctOfDelivery: 0, fullRefund: true };
    mock.onPut('/admin/settings/cancellation-rules').reply(200, ok(rule));
    const res = await adminApi.upsertCancellationRule({ vertical: 'gas', orderStatusAtCancel: 'confirmed', merchantCompKobo: 0, merchantCompPct: 0, riderCompPctOfDelivery: 0, fullRefund: true });
    expect(res.id).toBe('cr1');
  });

  it('deleteCancellationRule resolves', async () => {
    mock.onDelete('/admin/settings/cancellation-rules/cr1').reply(200, ok({ message: 'deleted' }));
    const res = await adminApi.deleteCancellationRule('cr1');
    expect(res.message).toBe('deleted');
  });

  it('listFeeConfigs resolves', async () => {
    const cfg = { id: 'fc1', vertical: 'gas', baseFeeKobo: 50000, perKmKobo: 5000, perKgKobo: 0, servicePct: 5, merchantTakeRate: 80, driverTakeRate: 15, platformTakeRate: 5, fuelPriceRefKobo: 120000, effectiveAt: '', updatedBy: 'admin', reason: 'init', createdAt: '' };
    mock.onGet('/admin/settings/fees').reply(200, ok({ configs: [cfg] }));
    const res = await adminApi.listFeeConfigs();
    expect(res.configs[0]!.vertical).toBe('gas');
  });

  it('upsertFeeConfig resolves', async () => {
    const cfg = { id: 'fc1', vertical: 'gas', baseFeeKobo: 50000, perKmKobo: 5000, perKgKobo: 0, servicePct: 5, merchantTakeRate: 80, driverTakeRate: 15, platformTakeRate: 5, fuelPriceRefKobo: 120000, effectiveAt: '', updatedBy: 'admin', reason: 'update', createdAt: '' };
    mock.onPut('/admin/settings/fees').reply(200, ok({ config: cfg, fuelSuggestion: null }));
    const res = await adminApi.upsertFeeConfig({ vertical: 'gas', baseFeeKobo: 50000, perKmKobo: 5000, perKgKobo: 0, servicePct: 5, merchantTakeRate: 80, driverTakeRate: 15, platformTakeRate: 5, fuelPriceRefKobo: 120000, reason: 'update' });
    expect(res.config.id).toBe('fc1');
    expect(res.fuelSuggestion).toBeNull();
  });

  it('getMetrics resolves', async () => {
    const metrics = { ordersToday: 42, gmvKobo: 6300000, revenueKobo: 315000, activeDrivers: 8, activeMerchants: 5, failedPayments: 1, cancellations: 2, cancellationRate: 4.76 };
    mock.onGet('/admin/metrics').reply(200, ok(metrics));
    const res = await adminApi.getMetrics();
    expect(res.ordersToday).toBe(42);
  });

  it('getWeatherSurcharge resolves', async () => {
    mock.onGet('/admin/settings/weather').reply(200, ok({ enabled: true, amountKobo: 20000 }));
    const res = await adminApi.getWeatherSurcharge();
    expect(res.amountKobo).toBe(20000);
  });

  it('setWeatherSurcharge resolves', async () => {
    mock.onPut('/admin/settings/weather').reply(200, ok({}));
    await expect(adminApi.setWeatherSurcharge({ enabled: false, amountKobo: 0, reason: 'clear skies' })).resolves.toBeUndefined();
  });

  it('listGasMerchants resolves', async () => {
    const row = { id: 'm1', businessName: 'Gas Co', fillAccuracyPct: 98.5, fillSampleCount: 40, fillStatus: 'good' as const };
    mock.onGet('/admin/gas/merchants').reply(200, ok({ merchants: [row] }));
    const res = await adminApi.listGasMerchants();
    expect(res.merchants[0]!.fillStatus).toBe('good');
  });

  it('setMerchantFillStatus resolves', async () => {
    mock.onPut('/admin/gas/merchants/m1/fill-status').reply(200, ok({ message: 'updated' }));
    const res = await adminApi.setMerchantFillStatus('m1', 'warned', 'low accuracy');
    expect(res.message).toBe('updated');
  });

  it('listZones resolves', async () => {
    const zone = { id: 'z1', name: 'Lekki', launchStatus: 'live' as const, isActive: true, windowStart: 6, windowEnd: 18 };
    mock.onGet('/admin/gas/zones').reply(200, ok({ zones: [zone] }));
    const res = await adminApi.listZones();
    expect(res.zones[0]!.name).toBe('Lekki');
  });

  it('setZoneLaunchStatus resolves', async () => {
    mock.onPut('/admin/gas/zones/z1/launch-status').reply(200, ok({ message: 'updated' }));
    const res = await adminApi.setZoneLaunchStatus('z1', 'paused', 'maintenance');
    expect(res.message).toBe('updated');
  });

  it('getLedger resolves entries', async () => {
    const entry = { id: 'le1', journalId: 'j1', accountId: 'acc1', amountKobo: 100000, description: 'wallet fund', refType: 'wallet_fund', createdAt: '' };
    mock.onGet('/admin/ledger').reply(200, ok({ entries: [entry] }));
    const res = await adminApi.getLedger('u1');
    expect(res.entries[0]!.amountKobo).toBe(100000);
  });

  it('recordLPGPrice resolves', async () => {
    mock.onPost('/admin/gas/price-index').reply(200, ok({ entry: {}, suggestion: null }));
    const res = await adminApi.recordLPGPrice({ region: 'Lagos', pricePerKgKobo: 125000, source: 'PPPRA' });
    expect(res.suggestion).toBeNull();
  });

  it('listUsers resolves', async () => {
    const user = { id: 'u1', role: 'customer', firstName: 'A', lastName: 'B', phone: '080', isVerified: true, isActive: true, createdAt: '' };
    mock.onGet('/admin/users').reply(200, ok({ users: [user] }));
    const res = await adminApi.listUsers();
    expect(res.users[0]!.role).toBe('customer');
  });

  it('listRuns resolves', async () => {
    const run = { id: 'r1', zoneId: 'z1', windowStart: '', windowEnd: '', status: 'completed', totalDistanceKm: 10 };
    mock.onGet('/admin/runs').reply(200, ok({ runs: [run] }));
    const res = await adminApi.listRuns();
    expect(res.runs[0]!.id).toBe('r1');
  });

  it('listSubscriptions resolves', async () => {
    const sub = { id: 'sub1', customerId: 'u1', merchantId: 'm1', vertical: 'gas', frequency: 'weekly', status: 'active', createdAt: '' };
    mock.onGet('/admin/subscriptions').reply(200, ok({ subscriptions: [sub] }));
    const res = await adminApi.listSubscriptions();
    expect(res.subscriptions[0]!.status).toBe('active');
  });

  it('listPrescriptions resolves', async () => {
    const rx = { id: 'rx1', customerId: 'u1', merchantId: 'm1', status: 'pending', createdAt: '' };
    mock.onGet('/admin/prescriptions').reply(200, ok({ prescriptions: [rx] }));
    const res = await adminApi.listPrescriptions();
    expect(res.prescriptions[0]!.id).toBe('rx1');
  });

  it('getKYCQueue throws on failure', async () => {
    mock.onGet('/admin/kyc/queue').reply(200, fail('forbidden'));
    await expect(adminApi.getKYCQueue()).rejects.toThrow('forbidden');
  });
});
