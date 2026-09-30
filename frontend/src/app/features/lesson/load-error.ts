import { ApiError } from '../../core/interceptors/error-interceptor';

/** Message for a lesson that could not be loaded (403: locked until its day, L). */
export function loadErrorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 404) {
    return 'Không tìm thấy bài học.';
  }
  if (err instanceof ApiError && err.status === 403) {
    return 'Bài này sẽ mở khi tới lượt.';
  }
  return 'Không tải được bài học.';
}
