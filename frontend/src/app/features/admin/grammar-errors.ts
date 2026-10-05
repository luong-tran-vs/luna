import { ApiError } from '../../core/interceptors/error-interceptor';

/** What went wrong with a grammar lesson request, as lines to show (Vietnamese). */
export interface GrammarFailure {
  message: string;
  /** "đường dẫn: thông báo" lines for a 400 with `fields`. */
  fields: string[];
}

const AI_MESSAGES: Record<number, string> = {
  503: 'AI chưa được cấu hình. Liên hệ người vận hành.',
  429: 'Đã hết lượt AI, vui lòng thử lại sau.',
  502: 'AI trả về nội dung không dùng được, vui lòng thử lại.',
};

const NOT_CHECKED = 'Bài chưa được kiểm tra bằng AI. Hãy bấm Kiểm tra bằng AI trước.';
const FLAGS_UNRESOLVED = 'Còn câu bị AI gắn cờ chưa xử lý. Hãy xem lại các câu đó rồi đăng lại.';

/** Turns an API error into a message; the server's own Vietnamese message wins over the defaults. */
export function grammarFailure(err: unknown, fallback: string, ai = false): GrammarFailure {
  if (!(err instanceof ApiError) || err.kind !== 'http') {
    return { message: 'Không kết nối được máy chủ, vui lòng thử lại.', fields: [] };
  }
  const body = err.body as { message?: unknown; fields?: unknown } | null;
  const fields =
    err.status === 400 && body?.fields && typeof body.fields === 'object'
      ? Object.entries(body.fields as Record<string, unknown>).map(([path, text]) => `${path}: ${String(text)}`)
      : [];
  const serverMessage = typeof body?.message === 'string' && body.message ? body.message : null;
  const flagged = err.status === 409 && (err.body as { error?: unknown } | null)?.error === 'grammar_flags_unresolved';
  const notChecked = err.status === 409 && (err.body as { error?: unknown } | null)?.error === 'grammar_not_checked';
  const message =
    serverMessage ?? (ai ? AI_MESSAGES[err.status] : undefined) ?? (flagged ? FLAGS_UNRESOLVED : undefined) ??
    (notChecked ? NOT_CHECKED : undefined) ??
    fallback;
  return { message, fields };
}
