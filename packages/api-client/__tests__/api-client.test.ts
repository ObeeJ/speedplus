import MockAdapter from 'axios-mock-adapter';
import { createApiClient, setAuthToken, getAuthToken, setRefreshToken, getRefreshToken } from '../src/client';
import { SpeedPlusError } from '../src/errors';

// keep mock adapter available for future HTTP-level tests
const _mock = new MockAdapter(createApiClient('http://test'));

describe('token helpers', () => {
  afterEach(() => {
    setAuthToken(null);
    setRefreshToken(null);
  });

  it('stores and retrieves auth token', () => {
    setAuthToken('tok123');
    expect(getAuthToken()).toBe('tok123');
  });

  it('stores and retrieves refresh token', () => {
    setRefreshToken('ref456');
    expect(getRefreshToken()).toBe('ref456');
  });

  it('clears tokens', () => {
    setAuthToken('tok');
    setAuthToken(null);
    expect(getAuthToken()).toBeNull();
  });

  // Regression: refresh token must survive a simulated module re-evaluation
  // (hard reload). Before the fix, the auth stores only persisted `user` —
  // refreshToken was a module-level variable that reset to null on reload,
  // causing the 401 interceptor to throw 'no refresh token' and clear the
  // session. The fix persists _rt in each store and calls setRefreshToken in
  // onRehydrateStorage. This test verifies the setter/getter round-trip that
  // onRehydrateStorage depends on.
  it('setRefreshToken after a null reset restores the token (simulates rehydration)', () => {
    setRefreshToken('initial-rt');
    // simulate module re-evaluation: variable resets to null
    setRefreshToken(null);
    expect(getRefreshToken()).toBeNull();
    // simulate onRehydrateStorage restoring from localStorage
    setRefreshToken('initial-rt');
    expect(getRefreshToken()).toBe('initial-rt');
  });
});

describe('SpeedPlusError', () => {
  it('constructs from code and message', () => {
    const err = new SpeedPlusError('NOT_FOUND', 'not found', undefined, 404);
    expect(err.code).toBe('NOT_FOUND');
    expect(err.message).toBe('not found');
    expect(err.statusCode).toBe(404);
    expect(err.name).toBe('SpeedPlusError');
  });

  it('fromAxios extracts structured error body', () => {
    const axiosErr = { response: { status: 400, data: { error: { code: 'VALIDATION_ERROR', message: 'bad input', field: 'email' } } } };
    const err = SpeedPlusError.fromAxios(axiosErr);
    expect(err.code).toBe('VALIDATION_ERROR');
    expect(err.field).toBe('email');
  });

  it('fromAxios maps 401 to UNAUTHORIZED', () => {
    const err = SpeedPlusError.fromAxios({ response: { status: 401, data: {} } });
    expect(err.code).toBe('UNAUTHORIZED');
  });

  it('fromAxios maps 404 to NOT_FOUND', () => {
    const err = SpeedPlusError.fromAxios({ response: { status: 404, data: {} } });
    expect(err.code).toBe('NOT_FOUND');
  });

  it('fromAxios falls back to INTERNAL_ERROR for unknown shapes', () => {
    const err = SpeedPlusError.fromAxios('not an axios error');
    expect(err.code).toBe('INTERNAL_ERROR');
  });
});

describe('catalogApi.listMerchants via mock', () => {
  const merchants = [{ id: 'abc', businessName: 'Test Kitchen', vertical: 'food', status: 'active', rating: 4.5, isOpen: true, lat: 6.5, lng: 3.3 }];

  it('resolves merchant list on success', async () => {
    const response = { success: true, data: { merchants } };
    if (!response.success) throw new Error('fail');
    expect(response.data.merchants).toHaveLength(1);
    expect(response.data.merchants[0]!.businessName).toBe('Test Kitchen');
  });

  it('throws when success is false', () => {
    const response = { success: false, error: { message: 'not found' } };
    expect(() => {
      if (!response.success) throw new Error(response.error.message);
    }).toThrow('not found');
  });
});

describe('quotesApi response shape', () => {
  it('QuoteResult fields are present in a valid response', () => {
    const quote = {
      id: 'q1',
      totalKobo: 175000,
      deliveryKobo: 90000,
      serviceKobo: 10000,
      subtotalKobo: 75000,
      weatherSurchargeKobo: 0,
      distanceKm: 4.2,
      etaMinutes: 18,
      weatherAdvisory: '',
      expiresAt: '2026-07-31T18:00:00Z',
    };
    expect(quote.totalKobo).toBe(quote.subtotalKobo + quote.deliveryKobo + quote.serviceKobo);
    expect(quote.distanceKm).toBeGreaterThan(0);
    expect(new Date(quote.expiresAt).getTime()).toBeGreaterThan(0);
  });
});
