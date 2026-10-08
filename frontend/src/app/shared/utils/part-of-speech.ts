/** Vietnamese names of the parts of speech the server sends for lesson words (lesson.POSValues). */
const POS_LABELS: Record<string, string> = {
  noun: 'danh từ',
  verb: 'động từ',
  adjective: 'tính từ',
  adverb: 'trạng từ',
  pronoun: 'đại từ',
  preposition: 'giới từ',
  conjunction: 'liên từ',
  determiner: 'từ hạn định',
  interjection: 'thán từ',
  'phrasal verb': 'cụm động từ',
  phrase: 'cụm từ',
};

/** The Vietnamese name of a part of speech, '' when unknown or missing. */
export function posLabel(pos: string | null | undefined): string {
  return (pos && POS_LABELS[pos]) || '';
}
