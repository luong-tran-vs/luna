import { ApiError } from '../../core/interceptors/error-interceptor';

const AI_MESSAGES: Record<number, string> = {
  503: 'AI chưa được cấu hình. Liên hệ người vận hành.',
  429: 'Đã hết lượt AI, vui lòng thử lại sau.',
  502: 'AI trả về nội dung không dùng được, vui lòng thử lại.',
};

/** Message for a failed check / confirm / verify request on a lesson (Vietnamese). */
export function lessonCheckFailure(err: unknown, fallback: string, ai = false): string {
  if (!(err instanceof ApiError) || err.kind !== 'http') {
    return 'Không kết nối được máy chủ, vui lòng thử lại.';
  }
  const body = err.body as { error?: unknown; message?: unknown; count?: unknown } | null;
  const code = err.status === 409 ? body?.error : undefined;
  if (code === 'flags_unresolved') {
    const n = typeof body?.count === 'number' ? body.count : null;
    return n === null ? 'Còn chỗ bị AI gắn cờ chưa xác nhận. Hãy xem lại rồi bấm Giữ nguyên.' : `Còn ${n} chỗ bị AI gắn cờ chưa xác nhận. Hãy xem lại rồi bấm Giữ nguyên.`;
  }
  if (typeof body?.message === 'string' && body.message) {
    return body.message;
  }
  if (code === 'annotation_not_done') {
    return 'Chú thích chưa chạy xong, hãy đợi xong rồi kiểm tra.';
  }
  if (code === 'not_checked') {
    return 'Bài chưa được kiểm tra bằng AI. Hãy bấm Kiểm tra bằng AI trước.';
  }
  return (ai ? AI_MESSAGES[err.status] : undefined) ?? fallback;
}
