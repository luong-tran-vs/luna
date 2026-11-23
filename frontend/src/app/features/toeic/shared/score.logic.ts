import { Outcome, TestKind, ToeicPart } from '../../../core/models/toeic';

/** "1:58:12" or "18:05" for a duration in ms; "—" for none. */
export function durationLabel(ms: number): string {
  if (!ms || ms < 0) {
    return '—';
  }
  const s = Math.round(ms / 1000);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = String(s % 60).padStart(2, '0');
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${sec}` : `${m}:${sec}`;
}

/** "64,5%" (Vietnamese decimal comma). */
export function percentLabel(value: number): string {
  return `${String(Math.round(value * 10) / 10).replace('.', ',')}%`;
}

/** The accuracy of right of total, one decimal. */
export function accuracyOf(right: number, total: number): number {
  return total > 0 ? Math.round((right * 1000) / total) / 10 : 0;
}

/** The letter of an option index ("" for none). */
export function letter(i: number): string {
  return i >= 0 && i < 26 ? String.fromCharCode(65 + i) : '';
}

/** The sign of an outcome, so it is not told by colour alone. */
export function outcomeMark(o: Outcome): string {
  return o === 'right' ? '✓' : o === 'wrong' ? '✗' : '–';
}

/** The Vietnamese of an outcome. */
export function outcomeLabel(o: Outcome): string {
  return o === 'right' ? 'đúng' : o === 'wrong' ? 'sai' : 'bỏ qua';
}

/** The name of a part. */
export const PART_NAMES: Record<ToeicPart, string> = {
  1: 'Mô tả tranh',
  2: 'Hỏi – đáp',
  3: 'Đoạn hội thoại',
  4: 'Bài nói ngắn',
  5: 'Điền câu',
  6: 'Điền đoạn văn',
  7: 'Đọc hiểu',
};

/**
 * The line shown above a question that has no printed prompt: Part 1 and 2 questions are only
 * heard; undefined for the other parts (their prompt, or the Part 6 blank line, is used).
 */
export function listeningPrompt(part: ToeicPart): string | undefined {
  if (part === 1) {
    return 'Nghe 4 câu mô tả, chọn câu đúng với bức ảnh.';
  }
  if (part === 2) {
    return 'Mark your answer on your answer sheet.';
  }
  return undefined;
}

/** "Full test" or "Mini Test 50 câu". */
export function kindLabel(kind: TestKind): string {
  return kind === 'full' ? 'Full test' : kind === 'mini50' ? 'Mini Test 50 câu' : 'Mini Test 100 câu';
}

/** The suggested time for some questions, ~0.6 min each in the reading, rounded up to 5 min. */
export function suggestedMinutes(questions: number): number {
  return questions > 0 ? Math.max(5, Math.ceil((questions * 0.6) / 5) * 5) : 0;
}

/** "12/11" date of an RFC 3339 time, in the device's zone. */
export function shortDate(at: string): string {
  const d = new Date(at);
  return Number.isNaN(d.getTime())
    ? ''
    : `${String(d.getDate()).padStart(2, '0')}/${String(d.getMonth() + 1).padStart(2, '0')}`;
}
