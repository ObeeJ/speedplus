import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@speedplus/types';
import { setAuthToken, setRefreshToken } from '@speedplus/api-client';

// Access token lives in memory ONLY — never written to localStorage.
// This prevents XSS from reading it. The refresh token is also memory-only;
// the server issues a new pair on every /auth/refresh call.
// We persist only the user profile so the UI can render without a round-trip.

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
        // Tokens go to the in-memory api-client store, NOT to Zustand persist.
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
      // Only persist the user profile — never tokens.
      partialize: (s) => ({ user: s.user, isAuthenticated: s.isAuthenticated }),
    },
  ),
);
