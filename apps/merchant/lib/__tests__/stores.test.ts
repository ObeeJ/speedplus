// Merchant store tests — pure state logic, no React, no DOM.
// Navigation is now handled by Next.js routing — useMerchantStore removed.

import { useMerchantAuthStore } from '../store/auth.store';
import type { MerchantProfile } from '@fourdat/api-client';

// ─── useMerchantAuthStore ─────────────────────────────────────────────────────

describe('useMerchantAuthStore', () => {
  const merchantUser = {
    id: 'usr1',
    firstName: 'Chidi',
    lastName: 'Nwosu',
    phone: '08011111111',
    role: 'merchant' as const,
    createdAt: '2024-01-01T00:00:00Z',
    isVerified: true,
  };

  const merchantProfile: MerchantProfile = {
    id: 'mer1',
    businessName: 'Chidi Foods',
    vertical: 'food',
    status: 'active',
    isOpen: true,
    rating: 4.8,
    kycStatus: 'approved',
  };

  beforeEach(() => useMerchantAuthStore.getState().clearAuth());

  it('starts unauthenticated with no user or merchant', () => {
    const s = useMerchantAuthStore.getState();
    expect(s.isAuthenticated).toBe(false);
    expect(s.user).toBeNull();
    expect(s.merchant).toBeNull();
  });

  it('setAuth stores user and marks authenticated', () => {
    useMerchantAuthStore.getState().setAuth(merchantUser, 'tok', 'ref');
    const s = useMerchantAuthStore.getState();
    expect(s.isAuthenticated).toBe(true);
    expect(s.user?.id).toBe('usr1');
    expect(s.user?.role).toBe('merchant');
  });

  it('setMerchant stores merchant profile independently of auth', () => {
    useMerchantAuthStore.getState().setAuth(merchantUser, 'tok', 'ref');
    useMerchantAuthStore.getState().setMerchant(merchantProfile);
    const s = useMerchantAuthStore.getState();
    expect(s.merchant?.id).toBe('mer1');
    expect(s.merchant?.businessName).toBe('Chidi Foods');
    expect(s.merchant?.vertical).toBe('food');
  });

  it('clearAuth resets user, merchant, and auth state', () => {
    useMerchantAuthStore.getState().setAuth(merchantUser, 'tok', 'ref');
    useMerchantAuthStore.getState().setMerchant(merchantProfile);
    useMerchantAuthStore.getState().clearAuth();
    const s = useMerchantAuthStore.getState();
    expect(s.isAuthenticated).toBe(false);
    expect(s.user).toBeNull();
    expect(s.merchant).toBeNull();
  });

  it('setMerchant does not affect isAuthenticated', () => {
    // merchant profile can be loaded without being authenticated (edge case guard)
    useMerchantAuthStore.getState().setMerchant(merchantProfile);
    expect(useMerchantAuthStore.getState().isAuthenticated).toBe(false);
  });
});
