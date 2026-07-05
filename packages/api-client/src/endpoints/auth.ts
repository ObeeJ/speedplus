import type { ApiResponse, User } from '@speedplus/types';
import { apiClient, setAuthToken } from '../client';

interface LoginPayload { phone: string; password: string }
interface RegisterPayload { firstName: string; lastName: string; phone: string; password: string }
interface AuthTokens { accessToken: string; refreshToken: string; user: User }

export const authApi = {
  async login(payload: LoginPayload): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/login', payload);
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    return data.data;
  },

  async register(payload: RegisterPayload): Promise<AuthTokens> {
    const { data } = await apiClient.post<ApiResponse<AuthTokens>>('/auth/register', payload);
    if (!data.success) throw new Error(data.error.message);
    setAuthToken(data.data.accessToken);
    return data.data;
  },

  async logout(): Promise<void> {
    await apiClient.post('/auth/logout');
    setAuthToken(null);
  },

  async verifyOtp(phone: string, otp: string): Promise<{ verified: boolean }> {
    const { data } = await apiClient.post<ApiResponse<{ verified: boolean }>>('/auth/verify-otp', { phone, otp });
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
