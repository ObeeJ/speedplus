import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@speedplus/types';
import { setAuthToken, setRefreshToken } from '@speedplus/api-client';
import type { MerchantProfile } from '@speedplus/api-client';

interface MerchantAuthState {
  user: User | null;
  merchant: MerchantProfile | null;
  isAuthenticated: boolean;
  setAuth: (user: User, accessToken: string, refreshToken: string) => void;
  setMerchant: (m: MerchantProfile) => void;
  clearAuth: () => void;
}

export const useMerchantAuthStore = create<MerchantAuthState>()(
  persist(
    (set) => ({
      user: null,
      merchant: null,
      isAuthenticated: false,
      setAuth: (user, accessToken, refreshToken) => {
        setAuthToken(accessToken);
        setRefreshToken(refreshToken);
        set({ user, isAuthenticated: true });
      },
      setMerchant: (m) => set({ merchant: m }),
      clearAuth: () => {
        setAuthToken(null);
        setRefreshToken(null);
        set({ user: null, merchant: null, isAuthenticated: false });
      },
    }),
    {
      name: 'speedplus-merchant-auth',
      // Never persist tokens — only user identity and merchant profile
      partialize: (s) => ({ user: s.user, merchant: s.merchant, isAuthenticated: s.isAuthenticated }),
    },
  ),
);
