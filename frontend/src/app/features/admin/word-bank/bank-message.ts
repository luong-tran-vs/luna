import { ApiError } from '../../../core/interceptors/error-interceptor';

/** The server's Vietnamese message for a word bank request, when the error has one. */
export function bankMessage(err: unknown): string | null {
  if (err instanceof ApiError && err.kind === 'http') {
    const body = err.body as { message?: unknown; fields?: Record<string, string> } | null;
    const fields = body?.fields ?? {};
    const field =
      fields['lemma'] ??
      fields['meaningVi'] ??
      fields['ipa'] ??
      fields['image'] ??
      fields['url'] ??
      fields['style'] ??
      fields['topicId'] ??
      fields['lemmas'] ??
      fields['engine'];
    if (field) {
      return field;
    }
    return typeof body?.message === 'string' && body.message ? body.message : null;
  }
  return null;
}

/** Said when the admin draws before choosing what draws: there is no default (asked 2026-11-10). */
export const NO_ENGINE = 'Hãy chọn "Sinh ảnh bằng" Gemini hay Codex CLI trước.';
