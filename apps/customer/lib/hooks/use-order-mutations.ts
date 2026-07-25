'use client';

import { useMutation, useQuery } from '@tanstack/react-query';
import { ordersApi, catalogApi } from '@speedplus/api-client';
import type { CreateOrderPayload } from '@speedplus/types';

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
    refetchInterval: 8000,
    retry: false,
  });
}

export function useUploadPrescription() {
  return useMutation({
    // r2Key is the object key returned by the R2 direct-upload flow.
    // merchantId is optional — set when uploading for a specific pharmacy.
    mutationFn: ({ r2Key, merchantId }: { r2Key: string; merchantId?: string }) =>
      catalogApi.createPrescription(r2Key, merchantId),
  });
}
