import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type MerchantTab = 'dash' | 'orders' | 'rx' | 'prod' | 'earn' | 'set';
export type OrderPipelineState = 'new' | 'preparing' | 'ready' | 'done';
export type RxState = 'pending' | 'approved' | 'rejected';

interface MerchantState {
  tab: MerchantTab;
  orderStates: Record<string, OrderPipelineState>;
  rxState: RxState;
  prodOff: Record<string, boolean>;
  setTab: (t: MerchantTab) => void;
  advanceOrder: (id: string, next: OrderPipelineState) => void;
  approveRx: () => void;
  rejectRx: () => void;
  resetRx: () => void;
  toggleProduct: (id: string) => void;
}

export const useMerchantStore = create<MerchantState>()(
  persist(
    (set) => ({
      tab: 'dash',
      orderStates: { o1: 'new', o2: 'new', o3: 'preparing', o4: 'done' },
      rxState: 'pending',
      prodOff: {},
      setTab: (t) => set({ tab: t }),
      advanceOrder: (id, next) => set((s) => ({ orderStates: { ...s.orderStates, [id]: next } })),
      approveRx: () => set({ rxState: 'approved' }),
      rejectRx: () => set({ rxState: 'rejected' }),
      resetRx: () => set({ rxState: 'pending' }),
      toggleProduct: (id) => set((s) => ({ prodOff: { ...s.prodOff, [id]: !s.prodOff[id] } })),
    }),
    { name: 'speedplus-merchant' },
  ),
);
