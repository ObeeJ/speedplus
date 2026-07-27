import { getAuthToken } from './client';

const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {};

/**
 * buildWsUrl returns the authenticated WebSocket URL.
 * Browsers cannot send Authorization headers on WebSocket upgrades,
 * so the access token is passed as ?token= query param.
 * The server's Auth middleware accepts this for WS upgrade requests only.
 */
export function buildWsUrl(): string {
  const base = env['NEXT_PUBLIC_WS_URL'] ?? 'ws://localhost:8000/api/v1/ws';
  const token = getAuthToken();
  if (!token) return base;
  const sep = base.includes('?') ? '&' : '?';
  return `${base}${sep}token=${encodeURIComponent(token)}`;
}
