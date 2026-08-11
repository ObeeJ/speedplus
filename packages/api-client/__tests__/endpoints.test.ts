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
