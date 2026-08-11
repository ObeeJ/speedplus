// Driver store tests — covers the full job lifecycle including multi-drop.

import { useDriverStore } from '../store/driver.store';
import { useDriverAuthStore } from '../store/auth.store';
import type { ActiveJob, DriverOffer } from '../store/driver.store';

// ─── helpers ─────────────────────────────────────────────────────────────────

function makeJob(overrides: Partial<ActiveJob> = {}): ActiveJob {
  return {
    orderId: 'ord1',
    vertical: 'package',
    stage: 1,
    customerName: 'Tunde Bello',
    customerPhone: '08012345678',
    pickupAddress: '14 Admiralty Way, Lekki',
    dropoffAddress: '5 Ozumba Mbadiwe, VI',
    totalKobo: 250000,
    deliveryCode: 'ABC123',
    paymentMethod: 'wallet',
    stops: [],
    currentStopIndex: 0,
    ...overrides,
  };
}

function makeOffer(overrides: Partial<DriverOffer> = {}): DriverOffer {
  return {
    offerId: 'offer1',
    orderId: 'ord1',
    vertical: 'food',
    totalKobo: 150000,
    pickupAddress: '14 Admiralty Way',
    dropoffAddress: '5 Ozumba Mbadiwe',
    distanceKm: 4.2,
    ...overrides,
  };
}

// ─── useDriverStore ───────────────────────────────────────────────────────────

describe('useDriverStore', () => {
  beforeEach(() => {
    useDriverStore.setState({
      tab: 'home',
      online: false,
      pendingOffer: null,
      activeJob: null,
    });
  });

  it('starts offline on home tab with no job or offer', () => {
    const s = useDriverStore.getState();
    expect(s.tab).toBe('home');
    expect(s.online).toBe(false);
    expect(s.pendingOffer).toBeNull();
    expect(s.activeJob).toBeNull();
  });

  it('setOnline toggles availability', () => {
    useDriverStore.getState().setOnline(true);
    expect(useDriverStore.getState().online).toBe(true);
    useDriverStore.getState().setOnline(false);
    expect(useDriverStore.getState().online).toBe(false);
  });

  it('setTab switches tab', () => {
    useDriverStore.getState().setTab('earn');
    expect(useDriverStore.getState().tab).toBe('earn');
    useDriverStore.getState().setTab('me');
    expect(useDriverStore.getState().tab).toBe('me');
  });

  it('setPendingOffer stores an incoming job offer', () => {
    const offer = makeOffer();
    useDriverStore.getState().setPendingOffer(offer);
    expect(useDriverStore.getState().pendingOffer?.offerId).toBe('offer1');
    expect(useDriverStore.getState().pendingOffer?.vertical).toBe('food');
  });

  it('setPendingOffer(null) clears the offer (driver rejected)', () => {
    useDriverStore.getState().setPendingOffer(makeOffer());
    useDriverStore.getState().setPendingOffer(null);
    expect(useDriverStore.getState().pendingOffer).toBeNull();
  });

  it('setActiveJob stores the accepted job', () => {
    const job = makeJob();
    useDriverStore.getState().setActiveJob(job);
    expect(useDriverStore.getState().activeJob?.orderId).toBe('ord1');
    expect(useDriverStore.getState().activeJob?.stage).toBe(1);
  });

  // ─── Single-drop job lifecycle (stages 1→6) ───────────────────────────────

  it('advanceJobStage increments stage from 1 to 2', () => {
    useDriverStore.getState().setActiveJob(makeJob({ stage: 1 }));
    useDriverStore.getState().advanceJobStage();
    expect(useDriverStore.getState().activeJob?.stage).toBe(2);
  });

  it('advanceJobStage increments through all stages up to 5', () => {
    useDriverStore.getState().setActiveJob(makeJob({ stage: 1 }));
    for (let expected = 2; expected <= 5; expected++) {
      useDriverStore.getState().advanceJobStage();
      expect(useDriverStore.getState().activeJob?.stage).toBe(expected);
    }
  });

  it('advanceJobStage at stage 6 clears job and returns to home tab', () => {
    useDriverStore.getState().setActiveJob(makeJob({ stage: 6 }));
    useDriverStore.getState().advanceJobStage();
    expect(useDriverStore.getState().activeJob).toBeNull();
    expect(useDriverStore.getState().tab).toBe('home');
  });

  it('advanceJobStage does nothing when no active job', () => {
    useDriverStore.getState().advanceJobStage();
    expect(useDriverStore.getState().activeJob).toBeNull();
  });

  // ─── Multi-drop job lifecycle ─────────────────────────────────────────────

  it('confirmStop marks a stop as confirmed', () => {
    const stops = [
      { sequence: 1, addressId: 'a1', status: 'pending' as const },
      { sequence: 2, addressId: 'a2', status: 'pending' as const },
    ];
    useDriverStore.getState().setActiveJob(makeJob({ stops, stage: 4 }));
    useDriverStore.getState().confirmStop(1);
    const job = useDriverStore.getState().activeJob!;
    expect(job.stops[0]!.status).toBe('confirmed');
    expect(job.stops[1]!.status).toBe('pending');
  });

  it('confirmStop does nothing when no active job', () => {
    expect(() => useDriverStore.getState().confirmStop(1)).not.toThrow();
  });

  it('advanceJobStage at stage 5 with more stops advances to next stop (stage 4)', () => {
    const stops = [
      { sequence: 1, addressId: 'a1', status: 'confirmed' as const },
      { sequence: 2, addressId: 'a2', status: 'pending' as const },
    ];
    useDriverStore.getState().setActiveJob(makeJob({ stops, stage: 5, currentStopIndex: 0 }));
    useDriverStore.getState().advanceJobStage();
    const job = useDriverStore.getState().activeJob!;
    expect(job.stage).toBe(4);
    expect(job.currentStopIndex).toBe(1);
  });

  it('advanceJobStage at stage 5 with no more stops advances normally', () => {
    const stops = [{ sequence: 1, addressId: 'a1', status: 'confirmed' as const }];
    useDriverStore.getState().setActiveJob(makeJob({ stops, stage: 5, currentStopIndex: 0 }));
    useDriverStore.getState().advanceJobStage();
    // currentStopIndex 1 >= stops.length 1 → normal advance to stage 6
    expect(useDriverStore.getState().activeJob?.stage).toBe(6);
  });

  // ─── clearJob ─────────────────────────────────────────────────────────────

  it('clearJob removes active job and pending offer', () => {
    useDriverStore.getState().setActiveJob(makeJob());
    useDriverStore.getState().setPendingOffer(makeOffer());
    useDriverStore.getState().clearJob();
    expect(useDriverStore.getState().activeJob).toBeNull();
    expect(useDriverStore.getState().pendingOffer).toBeNull();
  });
});

// ─── useDriverAuthStore ───────────────────────────────────────────────────────

describe('useDriverAuthStore', () => {
  const driverUser = {
    id: 'drv1',
    firstName: 'Emeka',
    lastName: 'Okafor',
    phone: '08099999999',
    role: 'driver' as const,
    createdAt: '2024-01-01T00:00:00Z',
    isVerified: true,
  };

  beforeEach(() => useDriverAuthStore.getState().clearAuth());

  it('starts unauthenticated', () => {
    const s = useDriverAuthStore.getState();
    expect(s.isAuthenticated).toBe(false);
    expect(s.user).toBeNull();
  });

  it('setAuth stores driver user and marks authenticated', () => {
    useDriverAuthStore.getState().setAuth(driverUser, 'tok', 'ref');
    const s = useDriverAuthStore.getState();
    expect(s.isAuthenticated).toBe(true);
    expect(s.user?.id).toBe('drv1');
    expect(s.user?.role).toBe('driver');
  });

  it('clearAuth resets to unauthenticated', () => {
    useDriverAuthStore.getState().setAuth(driverUser, 'tok', 'ref');
    useDriverAuthStore.getState().clearAuth();
    expect(useDriverAuthStore.getState().isAuthenticated).toBe(false);
    expect(useDriverAuthStore.getState().user).toBeNull();
  });
});
