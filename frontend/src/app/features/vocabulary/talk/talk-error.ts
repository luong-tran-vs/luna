import { ApiError } from '../../../core/interceptors/error-interceptor';

/**
 * The message for a failed talk request: the server's own (AI out of quota, too many turns…), or a
 * fallback. A network error says so.
 */
export function talkError(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    if (err.kind === 'network') {
      return 'Không kết nối được máy chủ. Kiểm tra mạng rồi thử lại.';
    }
    const body = err.body as { message?: string; fields?: Record<string, string> } | null;
    return body?.fields?.['text'] ?? body?.message ?? fallback;
  }
  return fallback;
}
