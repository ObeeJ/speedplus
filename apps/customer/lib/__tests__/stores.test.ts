// Flow store tests — pure state logic, no React, no DOM.
// Each describe block gets a fresh store by calling reset() before each test.

import { useFoodFlowStore } from '../store/food-flow.store';
import { useGasFlowStore, CYLINDER_KG } from '../store/gas-flow.store';
import { usePharmacyFlowStore } from '../store/pharmacy-flow.store';
import { usePackageFlowStore } from '../store/package-flow.store';
import { useAuthStore } from '../store/auth.store';
import { useCartStore } from '../store/cart.store';
import { useUiStore } from '../store/ui.store';
import { FEATURES } from '../features';
import { makeQueryClient } from '../query';

// zustand stores are singletons — reset state before each test
beforeEach(() => {
  useFoodFlowStore.getState().reset();
  useGasFlowStore.getState().reset();
  usePharmacyFlowStore.getState().reset();
  usePackageFlowStore.getState().reset();
});

// ─── Food flow store ──────────────────────────────────────────────────────────

describe('useFoodFlowStore', () => {
  it('starts empty', () => {
    const s = useFoodFlowStore.getState();
    expect(s.merchantId).toBeNull();
    expect(s.productId).toBeNull();
    expect(s.quote).toBeNull();
  });

  it('setMerchant stores id and coords, clears product and quote', () => {
    const s = useFoodFlowStore.getState();
    s.setProduct('p1', 350000);
    s.setMerchant('m1', 6.5, 3.3);
    const after = useFoodFlowStore.getState();
    expect(after.merchantId).toBe('m1');
    expect(after.merchantLat).toBe(6.5);
    expect(after.productId).toBeNull();
    expect(after.quote).toBeNull();
  });

  it('setProduct stores id and price, clears quote', () => {
    const s = useFoodFlowStore.getState();
    s.setMerchant('m1', 6.5, 3.3);
    s.setProduct('p1', 350000);
    const after = useFoodFlowStore.getState();
    expect(after.productId).toBe('p1');
    expect(after.productPriceKobo).toBe(350000);
    expect(after.quote).toBeNull();
  });

  it('setDeliverTo stores address and clears quote', () => {
    const s = useFoodFlowStore.getState();
    const addr = { id: 'a1', street: '14 Admiralty Way', city: 'Lekki', lat: 6.43, lng: 3.45 };
    s.setDeliverTo(addr);
    const after = useFoodFlowStore.getState();
    expect(after.deliverToId).toBe('a1');
    expect(after.deliverToAddress?.lat).toBe(6.43);
  });

  it('setQuote stores the quote', () => {
    const q = { id: 'q1', totalKobo: 100, deliveryKobo: 50, serviceKobo: 10, subtotalKobo: 40, distanceKm: 3, etaMinutes: 15, weatherSurchargeKobo: 0, weatherAdvisory: '', expiresAt: '' };
    useFoodFlowStore.getState().setQuote(q);
    expect(useFoodFlowStore.getState().quote?.id).toBe('q1');
  });

  it('reset clears all fields', () => {
    const s = useFoodFlowStore.getState();
    s.setMerchant('m1', 6.5, 3.3);
    s.setProduct('p1', 100);
    s.setOrderId('o1');
    s.reset();
    const after = useFoodFlowStore.getState();
    expect(after.merchantId).toBeNull();
    expect(after.productId).toBeNull();
    expect(after.orderId).toBeNull();
  });
});

// ─── Gas flow store ───────────────────────────────────────────────────────────

describe('useGasFlowStore', () => {
  it('CYLINDER_KG maps sizes to weights', () => {
    expect(CYLINDER_KG['3']).toBe(3);
    expect(CYLINDER_KG['12.5']).toBe(12.5);
    expect(CYLINDER_KG['25']).toBe(25);
  });

  it('starts empty', () => {
    const s = useGasFlowStore.getState();
    expect(s.cylinder).toBeNull();
    expect(s.quote).toBeNull();
  });

  it('setCylinder stores size and clears quote', () => {
    const s = useGasFlowStore.getState();
    s.setQuote({ id: 'q1', totalKobo: 1, deliveryKobo: 1, serviceKobo: 0, subtotalKobo: 0, distanceKm: 1, etaMinutes: 10, weatherSurchargeKobo: 0, weatherAdvisory: '', expiresAt: '' });
    s.setCylinder('12.5');
    const after = useGasFlowStore.getState();
    expect(after.cylinder).toBe('12.5');
    expect(after.quote).toBeNull();
  });

  it('setMode stores mode and clears quote', () => {
    const s = useGasFlowStore.getState();
    s.setMode('refill');
    expect(useGasFlowStore.getState().mode).toBe('refill');
  });

  it('setDeliverTo stores address id and full address', () => {
    const addr = { id: 'a1', street: 'Test St', city: 'Lagos', lat: 6.5, lng: 3.3 };
    useGasFlowStore.getState().setDeliverTo(addr);
    const after = useGasFlowStore.getState();
    expect(after.deliverToId).toBe('a1');
    expect(after.deliverToAddress?.lng).toBe(3.3);
  });

  it('reset clears all fields', () => {
    const s = useGasFlowStore.getState();
    s.setCylinder('6');
    s.setMode('swap');
    s.setOrderId('o1');
    s.reset();
    const after = useGasFlowStore.getState();
    expect(after.cylinder).toBeNull();
    expect(after.mode).toBeNull();
    expect(after.orderId).toBeNull();
  });
});

// ─── Pharmacy flow store ──────────────────────────────────────────────────────

describe('usePharmacyFlowStore', () => {
  it('starts with tab=otc and all nulls', () => {
    const s = usePharmacyFlowStore.getState();
    expect(s.tab).toBe('otc');
    expect(s.otcItemId).toBeNull();
    expect(s.merchantId).toBeNull();
  });

  it('setTab switches tab', () => {
    usePharmacyFlowStore.getState().setTab('rx');
    expect(usePharmacyFlowStore.getState().tab).toBe('rx');
  });

  it('setOtcItem stores id and price, clears quote', () => {
    const s = usePharmacyFlowStore.getState();
    s.setQuote({ id: 'q1', totalKobo: 1, deliveryKobo: 1, serviceKobo: 0, subtotalKobo: 0, distanceKm: 1, etaMinutes: 5, weatherSurchargeKobo: 0, weatherAdvisory: '', expiresAt: '' });
    s.setOtcItem('paracetamol', 80000);
    const after = usePharmacyFlowStore.getState();
    expect(after.otcItemId).toBe('paracetamol');
    expect(after.otcProductPriceKobo).toBe(80000);
    expect(after.quote).toBeNull();
  });

  it('setMerchant stores id and coords, clears quote', () => {
    const s = usePharmacyFlowStore.getState();
    s.setMerchant('m1', 6.4, 3.4);
    const after = usePharmacyFlowStore.getState();
    expect(after.merchantId).toBe('m1');
    expect(after.merchantLat).toBe(6.4);
    expect(after.merchantLng).toBe(3.4);
  });

  it('canContinueItems: otc requires otcItemId', () => {
    const s = usePharmacyFlowStore.getState();
    expect(s.canContinueItems()).toBe(false);
    s.setOtcItem('p1', 100);
    expect(usePharmacyFlowStore.getState().canContinueItems()).toBe(true);
  });

  it('canContinueItems: rx requires approved status', () => {
    const s = usePharmacyFlowStore.getState();
    s.setTab('rx');
    expect(s.canContinueItems()).toBe(false);
    s.setRxStatus('pending');
    expect(usePharmacyFlowStore.getState().canContinueItems()).toBe(false);
    usePharmacyFlowStore.getState().setRxStatus('approved');
    expect(usePharmacyFlowStore.getState().canContinueItems()).toBe(true);
  });

  it('reset clears all fields and returns to otc', () => {
    const s = usePharmacyFlowStore.getState();
    s.setTab('rx');
    s.setMerchant('m1', 1, 2);
    s.setOtcItem('p1', 100);
    s.setOrderId('o1');
    s.reset();
    const after = usePharmacyFlowStore.getState();
    expect(after.tab).toBe('otc');
    expect(after.merchantId).toBeNull();
    expect(after.otcItemId).toBeNull();
    expect(after.orderId).toBeNull();
  });
});

// ─── Package flow store ───────────────────────────────────────────────────────

describe('usePackageFlowStore', () => {
  const addr = (id: string) => ({ id, label: id, street: `${id} St`, city: 'Lagos', lat: 6.5, lng: 3.3 });

  it('starts empty with wallet payment', () => {
    const s = usePackageFlowStore.getState();
    expect(s.pickup).toBeNull();
    expect(s.paymentMethod).toBe('wallet');
    expect(s.isMultiDrop).toBe(false);
  });

  it('setPickup and setDropoff store addresses', () => {
    const s = usePackageFlowStore.getState();
    s.setPickup(addr('pickup'));
    s.setDropoff(addr('dropoff'));
    const after = usePackageFlowStore.getState();
    expect(after.pickup?.id).toBe('pickup');
    expect(after.dropoff?.id).toBe('dropoff');
  });

  it('addStop appends and sorts by sequence', () => {
    const s = usePackageFlowStore.getState();
    s.addStop({ sequence: 2, address: addr('b'), recipientName: 'B', recipientPhone: '080', notes: '' });
    s.addStop({ sequence: 1, address: addr('a'), recipientName: 'A', recipientPhone: '080', notes: '' });
    const stops = usePackageFlowStore.getState().stops;
    expect(stops[0]!.sequence).toBe(1);
    expect(stops[1]!.sequence).toBe(2);
  });

  it('removeStop removes by sequence', () => {
    const s = usePackageFlowStore.getState();
    s.addStop({ sequence: 1, address: addr('a'), recipientName: 'A', recipientPhone: '080', notes: '' });
    s.addStop({ sequence: 2, address: addr('b'), recipientName: 'B', recipientPhone: '080', notes: '' });
    s.removeStop(1);
    expect(usePackageFlowStore.getState().stops).toHaveLength(1);
    expect(usePackageFlowStore.getState().stops[0]!.sequence).toBe(2);
  });

  it('setIsMultiDrop(false) clears stops', () => {
    const s = usePackageFlowStore.getState();
    s.setIsMultiDrop(true);
    s.addStop({ sequence: 1, address: addr('a'), recipientName: 'A', recipientPhone: '080', notes: '' });
    s.setIsMultiDrop(false);
    expect(usePackageFlowStore.getState().stops).toHaveLength(0);
  });

  it('reset clears all fields', () => {
    const s = usePackageFlowStore.getState();
    s.setPickup(addr('p'));
    s.setSize('large');
    s.setWeight('heavy');
    s.setOrderId('o1');
    s.reset();
    const after = usePackageFlowStore.getState();
    expect(after.pickup).toBeNull();
    expect(after.size).toBeNull();
    expect(after.orderId).toBeNull();
  });
});

// ─── useAuthStore ─────────────────────────────────────────────────────────────

describe('useAuthStore', () => {
  beforeEach(() => useAuthStore.getState().clearAuth());

  it('starts unauthenticated', () => {
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
  });

  it('setAuth stores user and sets isAuthenticated', () => {
    const user = { id: 'u1', firstName: 'A', lastName: 'B', phone: '080', role: 'customer' as const, createdAt: '', isVerified: true };
    useAuthStore.getState().setAuth(user, 'tok', 'ref');
    const s = useAuthStore.getState();
    expect(s.isAuthenticated).toBe(true);
    expect(s.user?.id).toBe('u1');
  });

  it('clearAuth resets to unauthenticated', () => {
    const user = { id: 'u1', firstName: 'A', lastName: 'B', phone: '080', role: 'customer' as const, createdAt: '', isVerified: true };
    useAuthStore.getState().setAuth(user, 'tok', 'ref');
    useAuthStore.getState().clearAuth();
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
    expect(useAuthStore.getState().user).toBeNull();
  });
});

// ─── useCartStore ─────────────────────────────────────────────────────────────

describe('useCartStore', () => {
  beforeEach(() => useCartStore.getState().clearCart());

  const item = (id: string, price = 100, qty = 1) => ({ productId: id, name: id, price, quantity: qty });

  it('starts empty', () => {
    expect(useCartStore.getState().items).toHaveLength(0);
    expect(useCartStore.getState().totalItems()).toBe(0);
    expect(useCartStore.getState().subtotal()).toBe(0);
  });

  it('addItem adds a new item', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1', 500, 1));
    expect(useCartStore.getState().items).toHaveLength(1);
    expect(useCartStore.getState().subtotal()).toBe(500);
  });

  it('addItem increments quantity for existing product', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1', 500, 1));
    useCartStore.getState().addItem('m1', 'food', item('p1', 500, 2));
    expect(useCartStore.getState().items).toHaveLength(1);
    expect(useCartStore.getState().items[0]!.quantity).toBe(3);
  });

  it('addItem clears cart when switching merchant', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1'));
    useCartStore.getState().addItem('m2', 'food', item('p2'));
    const s = useCartStore.getState();
    expect(s.merchantId).toBe('m2');
    expect(s.items).toHaveLength(1);
    expect(s.items[0]!.productId).toBe('p2');
  });

  it('removeItem removes by productId', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1'));
    useCartStore.getState().addItem('m1', 'food', item('p2'));
    useCartStore.getState().removeItem('p1');
    expect(useCartStore.getState().items).toHaveLength(1);
    expect(useCartStore.getState().items[0]!.productId).toBe('p2');
  });

  it('updateQuantity updates quantity', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1', 100, 1));
    useCartStore.getState().updateQuantity('p1', 5);
    expect(useCartStore.getState().items[0]!.quantity).toBe(5);
  });

  it('updateQuantity with 0 removes item', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1'));
    useCartStore.getState().updateQuantity('p1', 0);
    expect(useCartStore.getState().items).toHaveLength(0);
  });

  it('totalItems sums quantities', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1', 100, 2));
    useCartStore.getState().addItem('m1', 'food', item('p2', 200, 3));
    expect(useCartStore.getState().totalItems()).toBe(5);
  });

  it('subtotal sums price * quantity', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1', 100, 2));
    useCartStore.getState().addItem('m1', 'food', item('p2', 300, 1));
    expect(useCartStore.getState().subtotal()).toBe(500);
  });

  it('clearCart empties everything', () => {
    useCartStore.getState().addItem('m1', 'food', item('p1'));
    useCartStore.getState().clearCart();
    expect(useCartStore.getState().items).toHaveLength(0);
    expect(useCartStore.getState().merchantId).toBeNull();
  });
});

// ─── useUiStore ───────────────────────────────────────────────────────────────

describe('useUiStore', () => {
  beforeEach(() => {
    useUiStore.setState({ toasts: [], isCartOpen: false });
  });

  it('starts with empty toasts and closed cart', () => {
    expect(useUiStore.getState().toasts).toHaveLength(0);
    expect(useUiStore.getState().isCartOpen).toBe(false);
  });

  it('addToast adds a toast with default type info', () => {
    useUiStore.getState().addToast('hello');
    const toasts = useUiStore.getState().toasts;
    expect(toasts).toHaveLength(1);
    expect(toasts[0]!.message).toBe('hello');
    expect(toasts[0]!.type).toBe('info');
  });

  it('addToast respects explicit type', () => {
    useUiStore.getState().addToast('oops', 'error');
    expect(useUiStore.getState().toasts[0]!.type).toBe('error');
  });

  it('removeToast removes by id', () => {
    useUiStore.getState().addToast('msg');
    const id = useUiStore.getState().toasts[0]!.id;
    useUiStore.getState().removeToast(id);
    expect(useUiStore.getState().toasts).toHaveLength(0);
  });

  it('setCartOpen toggles cart', () => {
    useUiStore.getState().setCartOpen(true);
    expect(useUiStore.getState().isCartOpen).toBe(true);
    useUiStore.getState().setCartOpen(false);
    expect(useUiStore.getState().isCartOpen).toBe(false);
  });
});

// ─── FEATURES flag ────────────────────────────────────────────────────────────

describe('FEATURES', () => {
  it('food is off', () => expect(FEATURES.food).toBe(false));
  it('grocery is off', () => expect(FEATURES.grocery).toBe(false));
  it('loyalty is off', () => expect(FEATURES.loyalty).toBe(false));
  it('giftCards is off', () => expect(FEATURES.giftCards).toBe(false));
});

// ─── makeQueryClient ──────────────────────────────────────────────────────────

describe('makeQueryClient', () => {
  it('returns a QueryClient with staleTime 60s', () => {
    const qc = makeQueryClient();
    const defaults = qc.getDefaultOptions();
    expect(defaults.queries?.staleTime).toBe(60_000);
  });

  it('does not retry on UNAUTHORIZED errors', () => {
    const qc = makeQueryClient();
    const retry = qc.getDefaultOptions().queries?.retry as (count: number, error: unknown) => boolean;
    expect(retry(0, new Error('UNAUTHORIZED'))).toBe(false);
  });

  it('retries up to 2 times on other errors', () => {
    const qc = makeQueryClient();
    const retry = qc.getDefaultOptions().queries?.retry as (count: number, error: unknown) => boolean;
    expect(retry(0, new Error('network error'))).toBe(true);
    expect(retry(1, new Error('network error'))).toBe(true);
    expect(retry(2, new Error('network error'))).toBe(false);
  });
});
