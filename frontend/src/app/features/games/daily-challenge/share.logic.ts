import { shortDate } from './daily.logic';

/** The row of squares: 🟩 right, 🟥 wrong. */
export function squares(answers: readonly boolean[]): string {
  return answers.map((r) => (r ? '🟩' : '🟥')).join('');
}

/**
 * The text "Chia sẻ" copies (FR-030): "Luna 13/11 · 9/10 🔥13" and the row of squares. No answer,
 * no word, nothing about the learner.
 */
export function shareText(day: string, answers: readonly boolean[], streak: number): string {
  const right = answers.filter(Boolean).length;
  return `Luna ${shortDate(day)} · ${right}/${answers.length} 🔥${streak}\n${squares(answers)}`;
}
