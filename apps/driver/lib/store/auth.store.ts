import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@speedplus/types';
import { setAuthToken, setRefreshToken } from '@speedplus/api-client';

interface DriverAuthState {
  user: User | null;
  isAuthenticated: boolean;
  setAuth: (user: User, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
}

export const useDriverAuthStore = create<DriverAuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      setAuth: (user, accessToken, refreshToken) => {
        setAuthToken(accessToken);
        setRefreshToken(refreshToken);
        set({ user, isAuthenticated: true });
      },
      clearAuth: () => {
        setAuthToken(null);
        setRefreshToken(null);
        set({ user: null, isAuthenticated: false });
      },
    }),
    {
      name: 'speedplus-driver-auth',
      partialize: (s) => ({ user: s.user }),
    },
  ),
);
