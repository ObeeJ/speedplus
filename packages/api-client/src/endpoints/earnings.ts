import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export const earningsApi = {
  async cashout(amountKobo: number, idempotencyKey: string) {
    const { data } = await apiClient.post<ApiResponse<{ message: string }>>(
      '/earnings/cashout',
      { amountKobo },
      { headers: { 'Idempotency-Key': idempotencyKey } },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },
};
