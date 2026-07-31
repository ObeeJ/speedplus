import { getAuthToken } from './client';

const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {};

/**
 * buildWsUrl returns the WebSocket base URL (no token in the URL).
 * The access token is passed via the Sec-WebSocket-Protocol subprotocol header:
 *   new WebSocket(url, ['bearer', token])
 * This keeps the token out of proxy access logs entirely.
 * Usage: const ws = new WebSocket(buildWsUrl(), buildWsProtocols());
 */
export function buildWsUrl(): string {
  return env['NEXT_PUBLIC_WS_URL'] ?? 'ws://localhost:8000/api/v1/ws';
}

/**
 * buildWsProtocols returns the subprotocol array for WebSocket authentication.
 * Pass this as the second argument to new WebSocket(url, protocols).
 * Returns undefined when no token is available (unauthenticated).
 */
export function buildWsProtocols(): string[] | undefined {
  const token = getAuthToken();
  if (!token) return undefined;
  return ['bearer', token];
}
