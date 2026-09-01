import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User } from '@fourdat/types';
import { setAuthToken, setRefreshToken } from '@fourdat/api-client';

interface AdminAuthState {
  user: User | null;
  isAuthenticated: boolean;
  // _rt is the refresh token persisted to localStorage so the 401 interceptor
  // can exchange it for a new access token after a hard reload. It is never
  // exposed in the public interface — consumers use setAuth / clearAuth only.
  _rt: string | null;
  setAuth: (user: User, accessToken: string, refreshToken: string) => void;
  clearAuth: () => void;
}

export const useAdminAuthStore = create<AdminAuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      _rt: null,
      setAuth: (user, accessToken, refreshToken) => {
        setAuthToken(accessToken);
        setRefreshToken(refreshToken);
        set({ user, isAuthenticated: true, _rt: refreshToken });
      },
      clearAuth: () => {
        setAuthToken(null);
        setRefreshToken(null);
        set({ user: null, isAuthenticated: false, _rt: null });
      },
    }),
    {
      name: 'fourdat-admin-auth',
      partialize: (s) => ({ user: s.user }),
      onRehydrateStorage: () => (state) => {
        if (state?.user) {
          state.isAuthenticated = true;
        }
      },
    },
  ),
);
