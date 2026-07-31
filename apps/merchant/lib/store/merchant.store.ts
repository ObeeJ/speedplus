import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type MerchantTab = 'dash' | 'orders' | 'rx' | 'prod' | 'earn' | 'set' | 'gas' | 'pay';

interface MerchantState {
  tab: MerchantTab;
  setTab: (t: MerchantTab) => void;
}

export const useMerchantStore = create<MerchantState>()(
  persist(
    (set) => ({
      tab: 'dash',
      setTab: (t) => set({ tab: t }),
    }),
    { name: 'speedplus-merchant' },
  ),
);
