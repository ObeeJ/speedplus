import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type AdminTab = 'kyc' | 'merchants' | 'drivers' | 'orders' | 'disputes' | 'cancellation-rules' | 'ledger';

interface AdminState {
  tab: AdminTab;
  setTab: (t: AdminTab) => void;
}

export const useAdminStore = create<AdminState>()(
  persist(
    (set) => ({
      tab: 'kyc',
      setTab: (t) => set({ tab: t }),
    }),
    { name: 'fourdat-admin-v2' },
  ),
);
