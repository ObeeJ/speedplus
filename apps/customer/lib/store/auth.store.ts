import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@speedplus/types';
import { setAuthToken, setRefreshToken } from '@speedplus/api-client';

// Access token lives in memory ONLY — never written to localStorage.
// isAuthenticated is derived from user !== null on every read, never persisted.
// This prevents the auth guard from passing on a hard refresh when the
// in-memory token is gone.

interface AuthState {
  user: (User & { referralCode?: string }) | null;
  isAuthenticated: boolean;
  setAuth: (user: User & { referralCode?: string }, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
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
      name: 'speedplus-auth',
      // Persist only the user profile. isAuthenticated is intentionally excluded:
      // on rehydration it defaults to false and is set to true only after a
      // successful token refresh, preventing the auth guard from passing when
      // the in-memory access token is absent.
      partialize: (s) => ({ user: s.user }),
    },
  ),
);
