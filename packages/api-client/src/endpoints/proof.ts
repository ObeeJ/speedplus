import type { ApiResponse } from '@fourdat/types';
import { apiClient } from '../client';

export type ProofKind = 'pickup_photo' | 'pickup_video' | 'dropoff_photo' | 'dropoff_video' | 'weight_photo';

export interface ProofMediaView {
  id: string;
  stopId?: string;
  kind: ProofKind;
  viewUrl: string;
  sha256: string;
  sealSerial?: string;
  measuredKg?: number;
  capturedLat?: number;
  capturedLng?: number;
  capturedAt: string;
}

/**
 * proofApi drives the chain-of-custody media flow:
 *   1. presign(kind, contentType) -> { uploadUrl, key }
 *   2. the caller PUTs the file bytes directly to uploadUrl (never through our API)
 *   3. confirm({ key, sha256, ... }) records the append-only evidence row
 * Viewing (getMedia) returns short-lived presigned GET URLs, not raw keys.
 */
export const proofApi = {
  async presign(
    orderId: string,
    body: { kind: ProofKind; contentType: string; stopId?: string },
  ): Promise<{ uploadUrl: string; key: string }> {
    const { data } = await apiClient.post<ApiResponse<{ uploadUrl: string; key: string }>>(
      `/orders/${orderId}/proof/presign`,
      body,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  /** Upload the raw file bytes straight to R2 via the presigned URL. */
  async uploadToPresignedUrl(uploadUrl: string, file: Blob, contentType: string): Promise<void> {
    const res = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': contentType },
      body: file,
    });
    if (!res.ok) throw new Error(`upload failed: ${res.status}`);
  },

  async confirm(
    orderId: string,
    body: {
      kind: ProofKind;
      key: string;
      sha256: string;
      stopId?: string;
      sealSerial?: string;
      measuredKg?: number;
      capturedLat?: number;
      capturedLng?: number;
    },
  ): Promise<{ id: string }> {
    const { data } = await apiClient.post<ApiResponse<{ id: string }>>(
      `/orders/${orderId}/proof/confirm`,
      body,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async getMedia(orderId: string): Promise<ProofMediaView[]> {
    const { data } = await apiClient.get<ApiResponse<{ media: ProofMediaView[] }>>(
      `/orders/${orderId}/proof`,
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data.media;
  },
};

/** sha256Hex computes the hex SHA-256 of a file in the browser (Web Crypto). */
export async function sha256Hex(file: Blob): Promise<string> {
  const buf = await file.arrayBuffer();
  const digest = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}
