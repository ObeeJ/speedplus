import axios, { type AxiosInstance } from 'axios';
import { FourdatError } from './errors';

let authToken: string | null = null;
let refreshToken: string | null = null;

export function setAuthToken(token: string | null): void {
  authToken = token;
}

export function getAuthToken(): string | null {
  return authToken;
}

export function setRefreshToken(token: string | null): void {
  refreshToken = token;
}

export function getRefreshToken(): string | null {
  return refreshToken;
}

export function createApiClient(baseURL: string): AxiosInstance {
  const client = axios.create({
    baseURL,
    timeout: 15_000,
    withCredentials: true,
    headers: { 'Content-Type': 'application/json' },
  });

  client.interceptors.request.use((config) => {
    if (authToken) config.headers.Authorization = `Bearer ${authToken}`;
    return config;
  });

  client.interceptors.response.use(
    (response) => response,
    async (error) => {
      const original = error.config as typeof error.config & { _retry?: boolean };
      if (error.response?.status === 401 && !original._retry) {
        original._retry = true;
        try {
          const storedRefresh = refreshToken;
          const { data } = await axios.post(
            `${baseURL}/auth/refresh`,
            storedRefresh ? { refreshToken: storedRefresh } : {},
            { headers: { 'Content-Type': 'application/json' }, withCredentials: true },
          );
          const tokens = (data as { data: { accessToken: string; refreshToken: string } }).data;
          setAuthToken(tokens.accessToken);
          setRefreshToken(tokens.refreshToken);
          original.headers.Authorization = `Bearer ${tokens.accessToken}`;
          return client(original);
        } catch {
          setAuthToken(null);
          setRefreshToken(null);
        }
      }
      throw FourdatError.fromAxios(error);
    },
  );

  return client;
}

const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {};

export const apiClient = createApiClient(
  env['NEXT_PUBLIC_API_URL'] ?? 'http://localhost:8000/api/v1',
);
