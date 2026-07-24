import axios, { type AxiosInstance } from 'axios';
import { SpeedPlusError } from './errors';

let authToken: string | null = null;

export function setAuthToken(token: string | null): void {
  authToken = token;
}

export function createApiClient(baseURL: string): AxiosInstance {
  const client = axios.create({
    baseURL,
    timeout: 15_000,
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
          const { data } = await axios.post(`${baseURL}/auth/refresh`, {}, { headers: original.headers });
          const newToken = (data as { data: { accessToken: string } }).data.accessToken;
          setAuthToken(newToken);
          original.headers.Authorization = `Bearer ${newToken}`;
          return client(original);
        } catch {
          setAuthToken(null);
        }
      }
      throw SpeedPlusError.fromAxios(error);
    },
  );

  return client;
}

const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {};

export const apiClient = createApiClient(
  env['NEXT_PUBLIC_API_URL'] ?? 'http://localhost:8000/api/v1',
);
