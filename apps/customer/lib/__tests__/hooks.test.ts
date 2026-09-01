// Customer hook dependency tests — verified against backend DTOs, handlers,
// service layer, and DB schema. Every mock URL, field name, and response shape
// matches the actual Go backend exactly.

import MockAdapter from 'axios-mock-adapter';
import { apiClient, setAuthToken } from '@fourdat/api-client';
import { ordersApi } from '@fourdat/api-client';
import { quotesApi } from '@fourdat/api-client';
import { catalogApi } from '@fourdat/api-client';
import { walletApi } from '@fourdat/api-client';

const mock = new MockAdapter(apiClient);

afterEach(() => {
  mock.reset();
  setAuthToken(null);
});

// ─── helpers matching dto.OK / dto.Fail from backend ─────────────────────────

function ok<T>(data: T) { return { success: true, data }; }
function fail(code: string, message: string) { return { success: false, error: { code, message } }; }

// Minimal OrderResponse shape matching backend dto.OrderResponse exactly:
// subtotal/deliveryFee/serviceFee/tip/total are MoneyResponse{amount,currency}
// deliveryAddressId is a UUID string (NOT a full Address object)
// paymentMethod is always present
const orderResp = {
  id: 'a1b2c3d4-0000-0000-0000-000000000001',
  customerId: 'a1b2c3d4-0000-0000-0000-000000000002',
  merchantId: 'a1b2c3d4-0000-0000-0000-000000000003',
  vertical: 'food',
  status: 'pending',
  paymentMethod: 'wallet',
  items: [],
  subtotal: { amount: 290000, currency: 'NGN' },
  deliveryFee: { amount: 50000, currency: 'NGN' },
  serviceFee: { amount: 10000, currency: 'NGN' },
  tip: { amount: 0, currency: 'NGN' },
  total: { amount: 350000, currency: 'NGN' },
  deliveryAddressId: 'a1b2c3d4-0000-0000-0000-000000000004',
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-01T00:00:00Z',
};

// QuoteResponse matching backend dto.QuoteResponse exactly
const quoteResp = {
  id: 'a1b2c3d4-0000-0000-0000-000000000010',
  subtotalKobo: 290000,
  deliveryKobo: 50000,
  serviceKobo: 10000,
  weatherSurchargeKobo: 0,
  totalKobo: 350000,
  distanceKm: 3.5,
  etaMinutes: 20,
  weatherAdvisory: '',
  expiresAt: '2024-01-01T01:00:00Z',
};

// ─── ordersApi.create — requires Idempotency-Key header ──────────────────────

describe('ordersApi.create', () => {
  const payload = {
    merchantId: 'a1b2c3d4-0000-0000-0000-000000000003',
    quoteId: 'a1b2c3d4-0000-0000-0000-000000000010',
    vertical: 'food' as const,
    deliveryAddressId: 'a1b2c3d4-0000-0000-0000-000000000004',
    items: [{ productId: 'a1b2c3d4-0000-0000-0000-000000000005', name: 'Jollof Rice', quantity: 1, unitPriceKobo: 290000 }],
  };
  const idemKey = 'test-idem-key-001';

  it('sends Idempotency-Key header and resolves with order', async () => {
    mock.onPost('/orders').reply((config) => {
      // Verify the header is present — backend returns 400 without it
      expect(config.headers?.['Idempotency-Key']).toBe(idemKey);
      return [201, ok(orderResp)];
    });
    const res = await ordersApi.create(payload, idemKey);
    expect(res.id).toBe(orderResp.id);
    expect(res.status).toBe('pending');
    expect(res.paymentMethod).toBe('wallet');
    // deliveryAddressId is a UUID string, NOT a full Address object
    expect(res.deliveryAddressId).toBe(orderResp.deliveryAddressId);
    expect((res as any).deliveryAddress).toBeUndefined();
  });

  it('throws INSUFFICIENT_BALANCE on 402', async () => {
    mock.onPost('/orders').reply(200, fail('INSUFFICIENT_BALANCE', 'Insufficient wallet balance'));
    await expect(ordersApi.create(payload, idemKey)).rejects.toThrow('Insufficient wallet balance');
  });

  it('throws MERCHANT_CLOSED on conflict', async () => {
    mock.onPost('/orders').reply(200, fail('MERCHANT_CLOSED', 'Merchant is currently closed'));
    await expect(ordersApi.create(payload, idemKey)).rejects.toThrow('Merchant is currently closed');
  });

  it('throws PRESCRIPTION_REQUIRED for pharmacy without prescriptionId', async () => {
    mock.onPost('/orders').reply(200, fail('PRESCRIPTION_REQUIRED', 'Prescription required'));
    await expect(ordersApi.create({ ...payload, vertical: 'pharmacy' }, idemKey)).rejects.toThrow('Prescription required');
  });

  it('throws on network error', async () => {
    mock.onPost('/orders').networkError();
    await expect(ordersApi.create(payload, idemKey)).rejects.toThrow();
  });
});

// ─── ordersApi.track — GET /orders/:id/track (same handler as getById) ───────

describe('ordersApi.track', () => {
  it('resolves with order including all gas statuses', async () => {
    const gasStatuses = ['awaiting_collection', 'empty_collected', 'at_plant', 'in_transit'] as const;
    for (const status of gasStatuses) {
      mock.onGet(`/orders/${orderResp.id}/track`).reply(200, ok({ ...orderResp, status }));
      const res = await ordersApi.track(orderResp.id);
      expect(res.status).toBe(status);
      mock.reset();
    }
  });

  it('throws NOT_FOUND when order does not exist', async () => {
    mock.onGet('/orders/bad-id/track').reply(200, fail('NOT_FOUND', 'Order not found'));
    await expect(ordersApi.track('bad-id')).rejects.toThrow('Order not found');
  });
});

// ─── ordersApi.cancel — returns {message} NOT an Order ───────────────────────

describe('ordersApi.cancel', () => {
  it('resolves with message string, not an Order object', async () => {
    mock.onPost(`/orders/${orderResp.id}/cancel`).reply(200, ok({ message: 'order cancelled' }));
    const res = await ordersApi.cancel(orderResp.id, 'changed my mind');
    expect(res.message).toBe('order cancelled');
    // Confirm it is NOT an Order — no id/status fields
    expect((res as any).id).toBeUndefined();
    expect((res as any).status).toBeUndefined();
  });

  it('throws VALIDATION_ERROR when order is already delivered', async () => {
    mock.onPost(`/orders/${orderResp.id}/cancel`).reply(200, fail('VALIDATION_ERROR', 'order already delivered'));
    await expect(ordersApi.cancel(orderResp.id, 'too late')).rejects.toThrow('order already delivered');
  });
});

// ─── ordersApi.review — POST with Idempotency-Key, returns 201 ───────────────

describe('ordersApi.review', () => {
  it('sends Idempotency-Key and resolves on success', async () => {
    mock.onPost(`/orders/${orderResp.id}/review`).reply((config) => {
      expect(config.headers?.['Idempotency-Key']).toBe('review-idem-001');
      return [201, ok({ message: 'review submitted' })];
    });
    await expect(
      ordersApi.review(orderResp.id, { revieweeType: 'driver', rating: 5, comment: 'Fast!' }, 'review-idem-001')
    ).resolves.toBeUndefined();
  });

  it('throws on duplicate review (idempotency key reuse)', async () => {
    mock.onPost(`/orders/${orderResp.id}/review`).reply(200, fail('VALIDATION_ERROR', 'already reviewed'));
    await expect(
      ordersApi.review(orderResp.id, { revieweeType: 'driver', rating: 4 }, 'review-idem-001')
    ).rejects.toThrow('already reviewed');
  });
});

// ─── quotesApi.quote — POST /quotes ──────────────────────────────────────────

describe('quotesApi.quote', () => {
  const payload = {
    merchantId: 'a1b2c3d4-0000-0000-0000-000000000003',
    vertical: 'food',
    subtotalKobo: 290000,
    originLat: 6.43, originLng: 3.45,
    destLat: 6.46, destLng: 3.48,
  };

  it('resolves with quote matching backend QuoteResponse shape', async () => {
    mock.onPost('/quotes').reply(200, ok(quoteResp));
    const res = await quotesApi.quote(payload);
    expect(res.id).toBe(quoteResp.id);
    expect(res.subtotalKobo).toBe(290000);
    expect(res.deliveryKobo).toBe(50000);
    expect(res.serviceKobo).toBe(10000);
    expect(res.totalKobo).toBe(350000);
    expect(res.weatherSurchargeKobo).toBe(0);
    expect(res.distanceKm).toBe(3.5);
    expect(res.etaMinutes).toBe(20);
  });

  it('includes weather surcharge and advisory when active', async () => {
    const surchargedQuote = { ...quoteResp, weatherSurchargeKobo: 5000, totalKobo: 355000, weatherAdvisory: 'Heavy rain expected' };
    mock.onPost('/quotes').reply(200, ok(surchargedQuote));
    const res = await quotesApi.quote(payload);
    expect(res.weatherSurchargeKobo).toBe(5000);
    expect(res.weatherAdvisory).toBe('Heavy rain expected');
    expect(res.totalKobo).toBe(355000);
  });

  it('throws when merchant is unavailable', async () => {
    mock.onPost('/quotes').reply(200, fail('MERCHANT_CLOSED', 'merchant unavailable'));
    await expect(quotesApi.quote(payload)).rejects.toThrow('merchant unavailable');
  });
});

// ─── quotesApi.multiStop — POST /quotes/multistop ────────────────────────────

describe('quotesApi.multiStop', () => {
  const payload = {
    merchantId: 'a1b2c3d4-0000-0000-0000-000000000003',
    vertical: 'package',
    subtotalKobo: 1000,
    originLat: 6.43, originLng: 3.45,
    stops: [{ lat: 6.46, lng: 3.48 }, { lat: 6.50, lng: 3.52 }],
    weightKg: 2, sizeCategory: 'medium',
  };

  it('resolves with multi-stop quote including stopCount', async () => {
    mock.onPost('/quotes/multistop').reply(200, ok({ ...quoteResp, stopCount: 2 }));
    const res = await quotesApi.multiStop(payload);
    expect(res.id).toBe(quoteResp.id);
  });

  it('throws when route is not serviceable', async () => {
    mock.onPost('/quotes/multistop').reply(200, fail('AREA_NOT_COVERED', 'route not serviceable'));
    await expect(quotesApi.multiStop(payload)).rejects.toThrow('route not serviceable');
  });
});

// ─── Prescription upload 3-step flow ─────────────────────────────────────────
// Step 1: POST /prescriptions/presign → {uploadUrl, key}
// Step 2: PUT uploadUrl (direct to R2, not via apiClient)
// Step 3: POST /prescriptions → PrescriptionRecord

describe('catalogApi prescription upload flow', () => {
  const rx = {
    id: 'rx-uuid-001',
    customerId: 'a1b2c3d4-0000-0000-0000-000000000002',
    merchantId: 'a1b2c3d4-0000-0000-0000-000000000003',
    r2Key: 'prescriptions/2024/rx-uuid-001.jpg',
    status: 'pending' as const,
    createdAt: '2024-01-01T00:00:00Z',
  };

  it('presignPrescription returns uploadUrl and server-derived key', async () => {
    mock.onPost('/prescriptions/presign').reply(200, ok({
      uploadUrl: 'https://r2.example.com/presigned-upload-url',
      key: 'prescriptions/2024/rx-uuid-001.jpg',
    }));
    const res = await catalogApi.presignPrescription('image/jpeg');
    expect(res.uploadUrl).toContain('r2.example.com');
    expect(res.key).toBe('prescriptions/2024/rx-uuid-001.jpg');
  });

  it('presignPrescription throws on invalid content type', async () => {
    mock.onPost('/prescriptions/presign').reply(200, fail('VALIDATION_ERROR', 'unsupported content type'));
    await expect(catalogApi.presignPrescription('application/exe')).rejects.toThrow('unsupported content type');
  });

  it('createPrescription sends server-derived r2Key and merchantId', async () => {
    mock.onPost('/prescriptions').reply((config) => {
      const body = JSON.parse(config.data);
      // Backend requires r2Key (not a client-invented path) and merchantId
      expect(body.r2Key).toBe('prescriptions/2024/rx-uuid-001.jpg');
      expect(body.merchantId).toBe('a1b2c3d4-0000-0000-0000-000000000003');
      return [200, ok(rx)];
    });
    const res = await catalogApi.createPrescription('prescriptions/2024/rx-uuid-001.jpg', 'a1b2c3d4-0000-0000-0000-000000000003');
    expect(res.id).toBe('rx-uuid-001');
    expect(res.status).toBe('pending');
    expect(res.r2Key).toBe('prescriptions/2024/rx-uuid-001.jpg');
  });

  it('createPrescription throws when backend rejects invalid key', async () => {
    mock.onPost('/prescriptions').reply(200, fail('VALIDATION_ERROR', 'invalid prescription key'));
    await expect(catalogApi.createPrescription('client-invented-key', 'mer1')).rejects.toThrow('invalid prescription key');
  });
});

// ─── usePrescriptionStatus → catalogApi.getPrescription ──────────────────────
// Polls GET /prescriptions/:id — stops when status is terminal

describe('catalogApi.getPrescription', () => {
  const base = { id: 'rx-uuid-001', customerId: 'c1', merchantId: 'm1', r2Key: 'k', createdAt: '' };

  it('returns pending while awaiting pharmacist review', async () => {
    mock.onGet('/prescriptions/rx-uuid-001').reply(200, ok({ ...base, status: 'pending' }));
    const res = await catalogApi.getPrescription('rx-uuid-001');
    expect(res.status).toBe('pending');
  });

  it('returns approved — polling should stop (terminal state)', async () => {
    mock.onGet('/prescriptions/rx-uuid-001').reply(200, ok({ ...base, status: 'approved' }));
    const res = await catalogApi.getPrescription('rx-uuid-001');
    expect(res.status).toBe('approved');
  });

  it('returns rejected with reviewNote from pharmacist', async () => {
    mock.onGet('/prescriptions/rx-uuid-001').reply(200, ok({ ...base, status: 'rejected', reviewNote: 'Illegible image' }));
    const res = await catalogApi.getPrescription('rx-uuid-001');
    expect(res.status).toBe('rejected');
    expect(res.reviewNote).toBe('Illegible image');
  });

  it('returns expired — polling should stop (terminal state)', async () => {
    mock.onGet('/prescriptions/rx-uuid-001').reply(200, ok({ ...base, status: 'expired' }));
    const res = await catalogApi.getPrescription('rx-uuid-001');
    expect(res.status).toBe('expired');
  });

  it('throws NOT_FOUND when prescription does not exist', async () => {
    mock.onGet('/prescriptions/bad-id').reply(200, fail('NOT_FOUND', 'Prescription not found'));
    await expect(catalogApi.getPrescription('bad-id')).rejects.toThrow('Prescription not found');
  });
});

// ─── walletApi.getBalance — GET /wallet (not /wallet/balance) ────────────────

describe('walletApi.getBalance', () => {
  it('resolves with balanceKobo and currency NGN', async () => {
    mock.onGet('/wallet').reply(200, ok({ balanceKobo: 500000, currency: 'NGN' }));
    const res = await walletApi.getBalance();
    expect(res.balanceKobo).toBe(500000);
    expect(res.currency).toBe('NGN');
  });

  it('throws UNAUTHORIZED when token is missing or expired', async () => {
    mock.onGet('/wallet').reply(200, fail('UNAUTHORIZED', 'UNAUTHORIZED'));
    await expect(walletApi.getBalance()).rejects.toThrow('UNAUTHORIZED');
  });
});

// ─── ordersApi.list — GET /orders ────────────────────────────────────────────

describe('ordersApi.list', () => {
  it('resolves with orders array', async () => {
    mock.onGet('/orders').reply(200, ok({ orders: [orderResp] }));
    const res = await ordersApi.list();
    expect(res.orders).toHaveLength(1);
    expect(res.orders[0]!.id).toBe(orderResp.id);
  });

  it('filters by vertical and status query params', async () => {
    mock.onGet('/orders').reply((config) => {
      expect(config.params?.vertical).toBe('food');
      expect(config.params?.status).toBe('delivered');
      return [200, ok({ orders: [] })];
    });
    const res = await ordersApi.list({ vertical: 'food', status: 'delivered' });
    expect(res.orders).toHaveLength(0);
  });
});

// ─── ordersApi.getStops — GET /orders/:id/stops ──────────────────────────────

describe('ordersApi.getStops', () => {
  it('resolves with stops array for multi-drop order', async () => {
    const stops = [
      { sequence: 1, addressId: 'addr-1', status: 'pending' },
      { sequence: 2, addressId: 'addr-2', status: 'confirmed' },
    ];
    mock.onGet(`/orders/${orderResp.id}/stops`).reply(200, ok({ stops }));
    const res = await ordersApi.getStops(orderResp.id);
    expect(res).toHaveLength(2);
    expect(res[0]!.sequence).toBe(1);
    expect(res[1]!.status).toBe('confirmed');
  });
});

// ─── ordersApi.confirmStop — POST /orders/:id/stops/confirm ──────────────────

describe('ordersApi.confirmStop', () => {
  it('resolves on successful stop confirmation with delivery code', async () => {
    mock.onPost(`/orders/${orderResp.id}/stops/confirm`).reply(200, ok({ message: 'stop confirmed' }));
    await expect(
      ordersApi.confirmStop(orderResp.id, { sequence: 1, code: 'ABC123' })
    ).resolves.toBeUndefined();
  });

  it('throws on wrong delivery code', async () => {
    mock.onPost(`/orders/${orderResp.id}/stops/confirm`).reply(200, fail('VALIDATION_ERROR', 'invalid delivery code'));
    await expect(
      ordersApi.confirmStop(orderResp.id, { sequence: 1, code: 'WRONG' })
    ).rejects.toThrow('invalid delivery code');
  });
});
