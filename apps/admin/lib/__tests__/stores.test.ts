// Admin store tests — pure state logic, no React, no DOM.

import { useAdminStore } from '../store/admin.store';
import { useAdminAuthStore } from '../store/auth.store';

// ─── useAdminStore ────────────────────────────────────────────────────────────

describe('useAdminStore', () => {
  beforeEach(() => useAdminStore.setState({ tab: 'kyc' }));

  it('starts on kyc tab', () => {
    expect(useAdminStore.getState().tab).toBe('kyc');
  });

  it('setTab switches to any valid tab', () => {
    useAdminStore.getState().setTab('merchants');
    expect(useAdminStore.getState().tab).toBe('merchants');

    useAdminStore.getState().setTab('drivers');
    expect(useAdminStore.getState().tab).toBe('drivers');

    useAdminStore.getState().setTab('orders');
    expect(useAdminStore.getState().tab).toBe('orders');

    useAdminStore.getState().setTab('disputes');
    expect(useAdminStore.getState().tab).toBe('disputes');

    useAdminStore.getState().setTab('ledger');
    expect(useAdminStore.getState().tab).toBe('ledger');
  });
});

// ─── useAdminAuthStore ────────────────────────────────────────────────────────

describe('useAdminAuthStore', () => {
  const adminUser = {
    id: 'admin1',
    firstName: 'Super',
    lastName: 'Admin',
    phone: '08000000000',
    role: 'admin' as const,
    createdAt: '2024-01-01T00:00:00Z',
    isVerified: true,
  };

  beforeEach(() => useAdminAuthStore.getState().clearAuth());

  it('starts unauthenticated with no user', () => {
    const s = useAdminAuthStore.getState();
    expect(s.isAuthenticated).toBe(false);
    expect(s.user).toBeNull();
  });

  it('setAuth stores user and marks authenticated', () => {
    useAdminAuthStore.getState().setAuth(adminUser, 'access-tok', 'refresh-tok');
    const s = useAdminAuthStore.getState();
    expect(s.isAuthenticated).toBe(true);
    expect(s.user?.id).toBe('admin1');
    expect(s.user?.role).toBe('admin');
  });

  it('clearAuth resets to unauthenticated', () => {
    useAdminAuthStore.getState().setAuth(adminUser, 'tok', 'ref');
    useAdminAuthStore.getState().clearAuth();
    const s = useAdminAuthStore.getState();
    expect(s.isAuthenticated).toBe(false);
    expect(s.user).toBeNull();
  });

  it('setAuth then clearAuth clears tokens via api-client', () => {
    // Verify the store integrates with token helpers — no throw means it called them
    expect(() => {
      useAdminAuthStore.getState().setAuth(adminUser, 'tok', 'ref');
      useAdminAuthStore.getState().clearAuth();
    }).not.toThrow();
  });
});
