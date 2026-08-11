import { apiClient } from '../client';
import type { ApiResponse } from '@speedplus/types';

export interface DriverBankAccount {
  bankCode: string;
  bankName: string;
  accountNumber: string;
  accountName: string;
}

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

  async getBankAccount(): Promise<DriverBankAccount | null> {
    const { data } = await apiClient.get<ApiResponse<DriverBankAccount | null>>('/drivers/bank-account');
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async resolveAccount(bankCode: string, accountNumber: string): Promise<{ accountName: string }> {
    const { data } = await apiClient.post<ApiResponse<{ accountName: string }>>(
      '/drivers/bank-account/resolve',
      { bankCode, accountNumber },
    );
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async saveBankAccount(input: { bankCode: string; bankName: string; accountNumber: string }): Promise<DriverBankAccount> {
    const { data } = await apiClient.post<ApiResponse<DriverBankAccount>>('/drivers/bank-account', input);
    if (!data.success) throw new Error(data.error.message);
    return data.data;
  },

  async listBanks(): Promise<Array<{ code: string; name: string }>> {
    const { data } = await apiClient.get<ApiResponse<{ banks: Array<{ code: string; name: string }> }>>('/banks');
    if (!data.success) throw new Error(data.error.message);
    return data.data.banks;
  },
};
