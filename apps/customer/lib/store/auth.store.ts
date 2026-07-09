import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { CustomerProfile } from '@speedplus/types';
import { setAuthToken } from '@speedplus/api-client';

interface AuthState {
  user: CustomerProfile | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  setAuth: (user: CustomerProfile, accessToken: string) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      setAuth: (user, accessToken) => {
        setAuthToken(accessToken);
        set({ user, accessToken, isAuthenticated: true });
      },
      clearAuth: () => {
        setAuthToken(null);
        set({ user: null, accessToken: null, isAuthenticated: false });
      },
    }),
    {
      name: 'speedplus-auth',
      partialize: (s) => ({ user: s.user, accessToken: s.accessToken, isAuthenticated: s.isAuthenticated }),
    },
  ),
);
