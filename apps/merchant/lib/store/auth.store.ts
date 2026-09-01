import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@fourdat/types';
import { setAuthToken, setRefreshToken } from '@fourdat/api-client';
import type { MerchantProfile } from '@fourdat/api-client';

interface MerchantAuthState {
  user: User | null;
  merchant: MerchantProfile | null;
  isAuthenticated: boolean;
  // _rt is the refresh token persisted to localStorage so the 401 interceptor
  // can exchange it for a new access token after a hard reload.
  _rt: string | null;
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
      _rt: null,
      setAuth: (user, accessToken, refreshToken) => {
        setAuthToken(accessToken);
        setRefreshToken(refreshToken);
        set({ user, isAuthenticated: true, _rt: refreshToken });
      },
      setMerchant: (m) => set({ merchant: m }),
      clearAuth: () => {
        setAuthToken(null);
        setRefreshToken(null);
        set({ user: null, merchant: null, isAuthenticated: false, _rt: null });
      },
    }),
    {
      name: 'fourdat-merchant-auth',
      partialize: (s) => ({ user: s.user, merchant: s.merchant }),
      onRehydrateStorage: () => (state) => {
        if (state?.user) {
          state.isAuthenticated = true;
        }
      },
    },
  ),
);
