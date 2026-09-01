'use client';

import { useMutation, useQuery } from '@tanstack/react-query';
import { ordersApi, catalogApi, walletApi, quotesApi } from '@fourdat/api-client';
import type { QuoteResult, QuotePayload, MultiStopQuotePayload } from '@fourdat/api-client';
import type { CreateOrderPayload } from '@fourdat/types';

export function useCreateOrder() {
  return useMutation({
    mutationFn: ({ payload, idempotencyKey }: { payload: CreateOrderPayload; idempotencyKey: string }) =>
      ordersApi.create(payload, idempotencyKey),
  });
}

export function useTrackOrder(orderId: string | null) {
  return useQuery({
    queryKey: ['order-track', orderId],
    queryFn: () => ordersApi.track(orderId as string),
    enabled: Boolean(orderId),
    refetchInterval: 5000,
    retry: false,
  });
}

// useUploadPrescription performs the real 3-step upload: presign → PUT the
// file bytes directly to R2 → create the prescription row with the
// server-derived key. Previously this flow was faked entirely — a fabricated
// key was sent straight to createPrescription with no bytes ever uploaded,
// so the pharmacist reviewed an image that didn't exist.
export function useUploadPrescription() {
  return useMutation({
    mutationFn: async ({ file, merchantId }: { file: File; merchantId: string }) => {
      const { uploadUrl, key } = await catalogApi.presignPrescription(file.type);
      const putRes = await fetch(uploadUrl, {
        method: 'PUT',
        headers: { 'Content-Type': file.type },
        body: file,
      });
      if (!putRes.ok) {
        throw new Error(`Upload to storage failed (${putRes.status})`);
      }
      return catalogApi.createPrescription(key, merchantId);
    },
  });
}

// usePrescriptionStatus polls the real server-side review status — replaces
// the client-side setTimeout that previously faked pharmacist approval.
// Stops polling once the prescription reaches a terminal state.
export function usePrescriptionStatus(prescriptionId: string | null) {
  return useQuery({
    queryKey: ['prescription-status', prescriptionId],
    queryFn: () => catalogApi.getPrescription(prescriptionId as string),
    enabled: Boolean(prescriptionId),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === 'approved' || status === 'rejected' || status === 'expired') {
        return false;
      }
      return 3000;
    },
    retry: false,
  });
}

export function useWalletBalance() {
  return useQuery({
    queryKey: ['wallet-balance'],
    queryFn: () => walletApi.getBalance(),
    staleTime: 30_000,
  });
}

export function useRequestQuote() {
  return useMutation({
    mutationFn: (payload: QuotePayload): Promise<QuoteResult> => quotesApi.quote(payload),
  });
}

/**
 * useRequestMultiStopQuote prices a one-pickup → N-dropoff package order via
 * /quotes/multistop. The returned quote's stop count is baked into its signed
 * hash server-side, so the order must be submitted with exactly these stops.
 */
export function useRequestMultiStopQuote() {
  return useMutation({
    mutationFn: (payload: MultiStopQuotePayload): Promise<QuoteResult> => quotesApi.multiStop(payload),
  });
}
