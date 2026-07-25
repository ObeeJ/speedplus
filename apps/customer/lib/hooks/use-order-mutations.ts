'use client';

import { useMutation, useQuery } from '@tanstack/react-query';
import { ordersApi, catalogApi, walletApi } from '@speedplus/api-client';
import type { CreateOrderPayload } from '@speedplus/types';
import { apiClient } from '@speedplus/api-client';
import type { ApiResponse } from '@speedplus/types';
import type { QuoteResult } from '@/lib/store/package-flow.store';

export function useCreateOrder() {
  return useMutation({
    mutationFn: (payload: CreateOrderPayload) => ordersApi.create(payload),
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

export function useUploadPrescription() {
  return useMutation({
    mutationFn: ({ r2Key, merchantId }: { r2Key: string; merchantId?: string }) =>
      catalogApi.createPrescription(r2Key, merchantId),
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
    mutationFn: async (payload: {
      merchantId: string;
      vertical: string;
      subtotalKobo: number;
      originLat: number;
      originLng: number;
      destLat: number;
      destLng: number;
      weightKg?: number;
      sizeCategory?: string;
    }): Promise<QuoteResult> => {
      const { data } = await apiClient.post<ApiResponse<QuoteResult>>('/quotes', payload);
      if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
      return data.data;
    },
  });
}

/**
 * useRequestMultiStopQuote prices a one-pickup → N-dropoff package order via
 * /quotes/multistop. The returned quote's stop count is baked into its signed
 * hash server-side, so the order must be submitted with exactly these stops.
 */
export function useRequestMultiStopQuote() {
  return useMutation({
    mutationFn: async (payload: {
      merchantId: string;
      vertical: string;
      subtotalKobo: number;
      originLat: number;
      originLng: number;
      stops: { lat: number; lng: number }[];
      weightKg?: number;
      sizeCategory?: string;
    }): Promise<QuoteResult> => {
      const { data } = await apiClient.post<ApiResponse<QuoteResult>>('/quotes/multistop', payload);
      if (!data.success) throw new Error((data as { error: { message: string } }).error.message);
      return data.data;
    },
  });
}
