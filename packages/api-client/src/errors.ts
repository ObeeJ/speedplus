import type { ErrorCode } from '@fourdat/types';

export class FourdatError extends Error {
  constructor(
    public readonly code: ErrorCode,
    message: string,
    public readonly field?: string,
    public readonly statusCode?: number,
  ) {
    super(message);
    this.name = 'FourdatError';
  }

  static fromAxios(error: unknown): FourdatError {
    if (isAxiosLike(error)) {
      const data = error.response?.data as { error?: { code?: ErrorCode; message?: string; field?: string } } | undefined;
      if (data?.error) {
        return new FourdatError(
          data.error.code ?? 'INTERNAL_ERROR',
          data.error.message ?? 'An unexpected error occurred',
          data.error.field,
          error.response?.status,
        );
      }
      if (error.response?.status === 401) return new FourdatError('UNAUTHORIZED', 'Session expired. Please log in again.', undefined, 401);
      if (error.response?.status === 403) return new FourdatError('FORBIDDEN', 'You do not have permission to do this.', undefined, 403);
      if (error.response?.status === 404) return new FourdatError('NOT_FOUND', 'Resource not found.', undefined, 404);
    }
    return new FourdatError('INTERNAL_ERROR', 'An unexpected error occurred');
  }
}

function isAxiosLike(e: unknown): e is { response?: { data?: unknown; status?: number } } {
  return typeof e === 'object' && e !== null && 'response' in e;
}
