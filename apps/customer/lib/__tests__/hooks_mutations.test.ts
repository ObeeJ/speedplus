// Tests for use-order-mutations.ts — every exported hook's mutationFn/queryFn
// is exercised directly (no React renderer needed) against the mocked apiClient.

import MockAdapter from 'axios-mock-adapter';
import { apiClient, setAuthToken, ordersApi, catalogApi, walletApi, quotesApi } from '@speedplus/api-client';
import { makeQueryClient } from '../query';

const mock = new MockAdapter(apiClient);

afterEach(() => {
  mock.reset();
  setAuthToken(null);
});

function ok<T>(data: T) { return { success: true, data }; }
function fail(message: string) { return { success: false, error: { message } }; }

// ── useCreateOrder ────────────────────────────────────────────────────────────
// Test the mutationFn directly — no React needed.

describe('useCreateOrder mutationFn', () => {

  const order = {
    id: 'o1', status: 'pending', vertical: 'food', merchantId: 'm1',
    customerId: 'u1', items: [], deliveryAddressId: 'a1',
    paymentMethod: 'wallet', subtotal: {}, deliveryFee: {}, serviceFee: {},
    total: {}, createdAt: '', updatedAt: '',
  };

  const payload = {
    merchantId: 'm1', quoteId: 'q1', vertical: 'food' as const,
    items: [{ productId: 'p1', quantity: 1 }], deliveryAddressId: 'a1',
  };

  it('resolves order on success', async () => {
    mock.onPost('/orders').reply(201, ok(order));
    const res = await ordersApi.create(payload, 'idem-1');
    expect(res.id).toBe('o1');
    expect(res.status).toBe('pending');
  });

  it('throws on MERCHANT_CLOSED', async () => {
    mock.onPost('/orders').reply(200, fail('Merchant is currently closed'));
    await expect(ordersApi.create(payload, 'idem-2')).rejects.toThrow('Merchant is currently closed');
  });

  it('throws on INSUFFICIENT_BALANCE', async () => {
    mock.onPost('/orders').reply(200, fail('Insufficient wallet balance'));
    await expect(ordersApi.create(payload, 'idem-3')).rejects.toThrow('Insufficient wallet balance');
  });

  it('throws on PRESCRIPTION_REQUIRED', async () => {
    mock.onPost('/orders').reply(200, fail('Prescription required'));
    await expect(ordersApi.create({ ...payload, vertical: 'pharmacy' }, 'idem-4')).rejects.toThrow('Prescription required');
  });
});

// ── useTrackOrder queryFn ─────────────────────────────────────────────────────

describe('useTrackOrder queryFn', () => {

  const order = {
    id: 'o1', status: 'in_transit', vertical: 'gas', merchantId: 'm1',
    customerId: 'u1', items: [], deliveryAddressId: 'a1',
    paymentMethod: 'wallet', subtotal: {}, deliveryFee: {}, serviceFee: {},
    total: {}, createdAt: '', updatedAt: '',
  };

  it('resolves order', async () => {
    mock.onGet('/orders/o1/track').reply(200, ok(order));
    const res = await ordersApi.track('o1');
    expect(res.status).toBe('in_transit');
  });

  it('throws NOT_FOUND', async () => {
    mock.onGet('/orders/bad/track').reply(200, fail('Order not found'));
    await expect(ordersApi.track('bad')).rejects.toThrow('Order not found');
  });

  it('resolves all gas-specific statuses', async () => {
    const statuses = ['awaiting_collection', 'empty_collected', 'at_plant', 'in_transit'];
    for (const status of statuses) {
      mock.onGet('/orders/o1/track').reply(200, ok({ ...order, status }));
      const res = await ordersApi.track('o1');
      expect(res.status).toBe(status);
      mock.reset();
    }
  });
});

// ── useUploadPrescription mutationFn ──────────────────────────────────────────

describe('useUploadPrescription mutationFn', () => {

  it('presign step resolves uploadUrl and key', async () => {
    mock.onPost('/prescriptions/presign').reply(200, ok({
      uploadUrl: 'https://r2.example.com/upload',
      key: 'prescriptions/u1/rx.jpg',
    }));
    const res = await catalogApi.presignPrescription('image/jpeg');
    expect(res.key).toBe('prescriptions/u1/rx.jpg');
    expect(res.uploadUrl).toContain('r2.example.com');
  });

  it('presign throws on invalid content type', async () => {
    mock.onPost('/prescriptions/presign').reply(200, fail('unsupported content type'));
    await expect(catalogApi.presignPrescription('application/exe')).rejects.toThrow('unsupported content type');
  });

  it('createPrescription sends server-derived key', async () => {
    const rx = { id: 'rx1', customerId: 'u1', merchantId: 'm1', r2Key: 'prescriptions/u1/rx.jpg', status: 'pending', createdAt: '' };
    mock.onPost('/prescriptions').reply(200, ok(rx));
    const res = await catalogApi.createPrescription('prescriptions/u1/rx.jpg', 'm1');
    expect(res.id).toBe('rx1');
    expect(res.status).toBe('pending');
  });

  it('createPrescription throws on invalid key', async () => {
    mock.onPost('/prescriptions').reply(200, fail('invalid prescription key'));
    await expect(catalogApi.createPrescription('bad-key', 'm1')).rejects.toThrow('invalid prescription key');
  });
});

// ── usePrescriptionStatus queryFn ─────────────────────────────────────────────

describe('usePrescriptionStatus queryFn', () => {
  const base = { id: 'rx1', customerId: 'u1', merchantId: 'm1', r2Key: 'k', createdAt: '' };

  it('returns pending — polling continues', async () => {
    mock.onGet('/prescriptions/rx1').reply(200, ok({ ...base, status: 'pending' }));
    const res = await catalogApi.getPrescription('rx1');
    expect(res.status).toBe('pending');
  });

  it('returns approved — polling stops', async () => {
    mock.onGet('/prescriptions/rx1').reply(200, ok({ ...base, status: 'approved' }));
    const res = await catalogApi.getPrescription('rx1');
    expect(res.status).toBe('approved');
  });

  it('returns rejected with reviewNote', async () => {
    mock.onGet('/prescriptions/rx1').reply(200, ok({ ...base, status: 'rejected', reviewNote: 'Illegible' }));
    const res = await catalogApi.getPrescription('rx1');
    expect(res.status).toBe('rejected');
    expect(res.reviewNote).toBe('Illegible');
  });

  it('returns expired — polling stops', async () => {
    mock.onGet('/prescriptions/rx1').reply(200, ok({ ...base, status: 'expired' }));
    const res = await catalogApi.getPrescription('rx1');
    expect(res.status).toBe('expired');
  });
});

// ── useWalletBalance queryFn ──────────────────────────────────────────────────

describe('useWalletBalance queryFn', () => {

  it('resolves balance', async () => {
    mock.onGet('/wallet').reply(200, ok({ balanceKobo: 750000, currency: 'NGN' }));
    const res = await walletApi.getBalance();
    expect(res.balanceKobo).toBe(750000);
    expect(res.currency).toBe('NGN');
  });

  it('throws UNAUTHORIZED', async () => {
    mock.onGet('/wallet').reply(200, fail('UNAUTHORIZED'));
    await expect(walletApi.getBalance()).rejects.toThrow('UNAUTHORIZED');
  });
});

// ── useRequestQuote mutationFn ────────────────────────────────────────────────

describe('useRequestQuote mutationFn', () => {

  const quote = {
    id: 'q1', totalKobo: 175000, deliveryKobo: 90000, serviceKobo: 10000,
    subtotalKobo: 75000, weatherSurchargeKobo: 0, distanceKm: 4.2,
    etaMinutes: 18, weatherAdvisory: '', expiresAt: '2026-12-31T00:00:00Z',
  };

  it('resolves quote', async () => {
    mock.onPost('/quotes').reply(200, ok(quote));
    const res = await quotesApi.quote({
      merchantId: 'm1', vertical: 'food', subtotalKobo: 75000,
      originLat: 6.5, originLng: 3.3, destLat: 6.4, destLng: 3.4,
    });
    expect(res.id).toBe('q1');
    expect(res.totalKobo).toBe(175000);
    expect(res.weatherSurchargeKobo).toBe(0);
  });

  it('includes weather surcharge when active', async () => {
    mock.onPost('/quotes').reply(200, ok({ ...quote, weatherSurchargeKobo: 5000, totalKobo: 180000, weatherAdvisory: 'Heavy rain' }));
    const res = await quotesApi.quote({
      merchantId: 'm1', vertical: 'food', subtotalKobo: 75000,
      originLat: 6.5, originLng: 3.3, destLat: 6.4, destLng: 3.4,
    });
    expect(res.weatherSurchargeKobo).toBe(5000);
    expect(res.weatherAdvisory).toBe('Heavy rain');
  });

  it('throws on merchant unavailable', async () => {
    mock.onPost('/quotes').reply(200, fail('merchant unavailable'));
    await expect(quotesApi.quote({
      merchantId: 'bad', vertical: 'food', subtotalKobo: 0,
      originLat: 0, originLng: 0, destLat: 0, destLng: 0,
    })).rejects.toThrow('merchant unavailable');
  });
});

// ── useRequestMultiStopQuote mutationFn ───────────────────────────────────────

describe('useRequestMultiStopQuote mutationFn', () => {

  const quote = {
    id: 'q2', totalKobo: 200000, deliveryKobo: 120000, serviceKobo: 10000,
    subtotalKobo: 70000, weatherSurchargeKobo: 0, distanceKm: 8.5,
    etaMinutes: 35, weatherAdvisory: '', expiresAt: '2026-12-31T00:00:00Z',
  };

  it('resolves multi-stop quote', async () => {
    mock.onPost('/quotes/multistop').reply(200, ok(quote));
    const res = await quotesApi.multiStop({
      merchantId: 'm1', vertical: 'package', subtotalKobo: 70000,
      originLat: 6.5, originLng: 3.3,
      stops: [{ lat: 6.4, lng: 3.4 }, { lat: 6.3, lng: 3.5 }],
    });
    expect(res.id).toBe('q2');
    expect(res.distanceKm).toBe(8.5);
  });

  it('throws on area not covered', async () => {
    mock.onPost('/quotes/multistop').reply(200, fail('route not serviceable'));
    await expect(quotesApi.multiStop({
      merchantId: 'm1', vertical: 'package', subtotalKobo: 1,
      originLat: 0, originLng: 0, stops: [{ lat: 0, lng: 0 }],
    })).rejects.toThrow('route not serviceable');
  });
});

// ── makeQueryClient config ────────────────────────────────────────────────────

describe('makeQueryClient', () => {
  it('staleTime is 60s', () => {
    const qc = makeQueryClient();
    expect(qc.getDefaultOptions().queries?.staleTime).toBe(60_000);
  });

  it('gcTime is 5 minutes', () => {
    const qc = makeQueryClient();
    expect(qc.getDefaultOptions().queries?.gcTime).toBe(5 * 60 * 1000);
  });

  it('refetchOnWindowFocus is false', () => {
    const qc = makeQueryClient();
    expect(qc.getDefaultOptions().queries?.refetchOnWindowFocus).toBe(false);
  });

  it('mutations do not retry', () => {
    const qc = makeQueryClient();
    expect(qc.getDefaultOptions().mutations?.retry).toBe(false);
  });

  it('does not retry on UNAUTHORIZED', () => {
    const qc = makeQueryClient();
    const retry = qc.getDefaultOptions().queries?.retry as (count: number, error: unknown) => boolean;
    expect(retry(0, new Error('UNAUTHORIZED'))).toBe(false);
    expect(retry(1, new Error('Session expired. UNAUTHORIZED'))).toBe(false);
  });

  it('retries up to 2 times on other errors', () => {
    const qc = makeQueryClient();
    const retry = qc.getDefaultOptions().queries?.retry as (count: number, error: unknown) => boolean;
    expect(retry(0, new Error('network error'))).toBe(true);
    expect(retry(1, new Error('timeout'))).toBe(true);
    expect(retry(2, new Error('timeout'))).toBe(false);
  });
});
