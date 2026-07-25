import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type DriverTab = 'home' | 'job' | 'earn' | 'me';

interface DriverState {
  tab: DriverTab;
  online: boolean;
  jobStage: number; // 0 none/offer, 1 accepted->pickup, 2 picked up, 3 at dropoff, 4 pod, 5 delivered
  cashed: boolean;
  setTab: (t: DriverTab) => void;
  toggleOnline: () => void;
  acceptJob: () => void;
  declineJob: () => void;
  advanceJob: () => void;
  cashOut: () => void;
}

export const useDriverStore = create<DriverState>()(
  persist(
    (set, get) => ({
      tab: 'home',
      online: true,
      jobStage: 0,
      cashed: false,
      setTab: (t) => set({ tab: t === 'job' && get().jobStage === 0 ? 'home' : t }),
      toggleOnline: () => set((s) => ({ online: !s.online })),
      acceptJob: () => set({ jobStage: 1, tab: 'job' }),
      declineJob: () => set({ jobStage: 0 }),
      advanceJob: () =>
        set((s) => (s.jobStage >= 5 ? { jobStage: 0, tab: 'home' } : { jobStage: s.jobStage + 1 })),
      cashOut: () => set({ cashed: true }),
    }),
    { name: 'speedplus-driver' },
  ),
);
