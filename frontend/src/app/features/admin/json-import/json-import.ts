import { ApiError } from '../../../core/interceptors/error-interceptor';

/**
 * Helpers of the import dialogs (FR-030, added 2026-10-26): a lesson written outside the app, by an
 * admin or an agent, is pasted or picked as JSON, checked on the server (dry run), then saved.
 */

/** The JSON of text, or why it is not JSON (in Vietnamese). */
export function parseJson(text: string): { value: unknown } | { error: string } {
  if (!text.trim()) {
    return { error: 'Chưa có nội dung: dán JSON hoặc chọn file .json.' };
  }
  try {
    const value: unknown = JSON.parse(text);
    if (value === null || typeof value !== 'object' || Array.isArray(value)) {
      return { error: 'JSON phải là một đối tượng { … }.' };
    }
    return { value };
  } catch (err) {
    return { error: `Không đọc được JSON: ${err instanceof Error ? err.message : String(err)}` };
  }
}

/** The message, field errors and other parts of an API error body, when the server sent them. */
export function errorBody(err: unknown): {
  message: string | null;
  error: string | null;
  fields: Record<string, string> | null;
  body: Record<string, unknown> | null;
} {
  if (
    !(err instanceof ApiError) ||
    err.kind !== 'http' ||
    !err.body ||
    typeof err.body !== 'object'
  ) {
    return { message: null, error: null, fields: null, body: null };
  }
  const body = err.body as Record<string, unknown>;
  const fields = body['fields'];
  return {
    message: typeof body['message'] === 'string' ? body['message'] : null,
    error: typeof body['error'] === 'string' ? body['error'] : null,
    fields: fields && typeof fields === 'object' ? (fields as Record<string, string>) : null,
    body,
  };
}

/** Field errors as lines, sorted by field. */
export function fieldLines(fields: Record<string, string>): { key: string; text: string }[] {
  return Object.keys(fields)
    .sort()
    .map((key) => ({ key, text: fields[key] }));
}

/** Lets the browser save data as a pretty JSON file. */
export function downloadJson(name: string, data: unknown): void {
  const blob = new Blob([JSON.stringify(data, null, 2) + '\n'], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

/** The controls of a dialog that Tab cycles through. */
export function focusables(root: HTMLElement | undefined): HTMLElement[] {
  return Array.from(
    root?.querySelectorAll<HTMLElement>(
      'button:not(:disabled), textarea, input:not([type="radio"]), input[type="radio"]:checked, a[href], summary',
    ) ?? [],
  ).filter((el) => !el.closest('[hidden]'));
}

/** Keeps Tab inside root. */
export function trapTab(event: KeyboardEvent, root: HTMLElement | undefined): void {
  if (event.key !== 'Tab') {
    return;
  }
  const items = focusables(root);
  const first = items[0];
  const last = items[items.length - 1];
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last?.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first?.focus();
  }
}
