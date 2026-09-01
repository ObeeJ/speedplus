import type { ApiResponse, User } from '@fourdat/types';
import { apiClient, setAuthToken, setRefreshToken } from '../client';

interface LoginPayload { phone: string; password: string }
interface RegisterPayload { firstName: string; lastName: string; phone: string; password: string; referralCode?: string; role?: 'customer' | 'driver' | 'merchant'; vehicleType?: string; vehiclePlate?: string }
interface AuthTokens { accessToken: string; refreshToken: string; user: User }

export const authApi = {
  async login(payload: LoginPayload): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/login', payload);
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    setRefreshToken(data.data.refreshToken);
    return data.data;
  },

  async register(payload: RegisterPayload): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/register', payload);
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    setRefreshToken(data.data.refreshToken);
    return data.data;
  },

  async logout(): Promise<void> {
    await apiClient.post('/auth/logout');
    setAuthToken(null);
    setRefreshToken(null);
  },

  async setPin(pin: string): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>('/auth/pin/set', { pin });
    if (!data.success) throw new Error(data.error.message);
  },

  async verifyPin(pin: string): Promise<{ verified: boolean }> {
    const { data } = await apiClient.post<ApiResponse<{ verified: boolean }>>('/auth/pin/verify', { pin });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  // purpose scopes the OTP lookup on the backend (e.g. "phone_verification") —
  // request and verify must pass the same purpose or verification will 422.
  async requestOtp(phone: string, purpose: string): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>('/otp/request', { phone, purpose });
    if (!data.success) throw new Error(data.error.message);
  },

  // Fresh tokens are returned on success (purpose === 'phone_verification'
  // flips users.is_verified server-side, and the old token would still read
  // unverified since that claim is baked in at issue time) — stored here so
  // the caller doesn't have to remember to.
  async verifyOtpCode(phone: string, otp: string, purpose: string): Promise<{ verified: boolean }> {
    const { data } = await apiClient.post<ApiResponse<{ verified: boolean; accessToken?: string; refreshToken?: string }>>(
      '/otp/verify',
      { phone, otp, purpose },
    );
    if (!data.success) throw new Error(data.error.message);
    if (data.data.accessToken) setAuthToken(data.data.accessToken);
    if (data.data.refreshToken) setRefreshToken(data.data.refreshToken);
    return { verified: data.data.verified };
  },

  /** Explicit token refresh — the axios interceptor calls this automatically,
   *  but expose it for manual use (e.g. background tab wake-up). */
  async refresh(refreshToken: string): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/refresh', { refreshToken });
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    setRefreshToken(data.data.refreshToken);
    return data.data;
  },

  async resetPassword(payload: { phone: string; otp: string; newPassword: string }): Promise<void> {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>('/auth/reset-password', payload);
    if (!data.success) throw new Error(data.error.message);
  },
};
