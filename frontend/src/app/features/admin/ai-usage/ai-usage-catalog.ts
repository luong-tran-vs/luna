/** Vietnamese names of the operations the server records (internal/ai/gemini). */
export const OP_LABELS: Readonly<Record<string, string>> = {
  generate: 'Sinh bài học',
  annotate: 'Chú thích bài',
  practice: 'Luyện tập từ vựng',
  image: 'Sinh ảnh từ vựng',
  lesson_review: 'Kiểm tra bài',
  lesson_fix: 'Đề xuất sửa lỗi bài',
  suggest_words: 'Gợi ý từ cho chủ đề',
  word_meanings: 'Điền nghĩa, phiên âm (kho từ)',
  grammar_lesson: 'Sinh bài ngữ pháp',
  grammar_solve: 'Kiểm tra bài tập ngữ pháp',
  explain: 'Người học hỏi nghĩa',
  grade: 'Chấm bài viết',
};

export function opLabel(op: string): string {
  return OP_LABELS[op] ?? op;
}

/** One feature and how many AI requests one use of it costs. */
export interface FeatureCost {
  feature: string;
  where: string;
  requests: string;
  /** Uses the paid image model. */
  paid?: boolean;
}

export interface FeatureGroup {
  title: string;
  rows: readonly FeatureCost[];
}

/** What each feature costs, as the code is written (kept in step with the server by hand). */
export const FEATURE_COSTS: readonly FeatureGroup[] = [
  {
    title: 'Quản trị viên bấm',
    rows: [
      { feature: 'Sinh bài học', where: 'Chủ đề → Sinh bài (1–5 bài)', requests: '1 cho cả lô' },
      { feature: 'Gợi ý từ cho chủ đề', where: 'Từ vựng chủ đề → Gợi ý bằng AI', requests: '1' },
      { feature: 'Kiểm tra bài', where: 'Chi tiết bài → Kiểm tra', requests: '1' },
      { feature: 'Đề xuất sửa lỗi bài', where: 'Từng mục bị đánh dấu', requests: '1 mỗi mục' },
      { feature: 'Sinh lại luyện tập', where: 'Chi tiết bài', requests: '1' },
      { feature: 'Chú thích lại / sửa nội dung bài', where: 'Chi tiết bài', requests: '1 (rồi tự chạy luyện tập, ảnh)' },
      { feature: 'Bật ảnh từ vựng cho bài', where: 'Chi tiết bài', requests: '1 mỗi từ chưa có ảnh (≤ 30)', paid: true },
      { feature: 'Sinh ảnh 1 từ', where: 'Kho từ → Sinh ảnh AI', requests: '1', paid: true },
      {
        feature: 'Điền nghĩa, phiên âm',
        where: 'Kho từ → lọc thiếu + chủ đề',
        requests: '1 cho cả chủ đề (0 khi không còn thiếu)',
      },
      { feature: 'Sinh bài ngữ pháp', where: 'Ngữ pháp → Sinh bài', requests: '1' },
      { feature: 'Kiểm tra bài tập ngữ pháp', where: 'Ngữ pháp → Kiểm tra', requests: '1 cho mọi bài tập' },
      { feature: 'Xác nhận, đăng, ẩn bài', where: '', requests: '0' },
    ],
  },
  {
    title: 'Tự chạy sau khi lưu bài',
    rows: [
      { feature: 'Chú thích bài (nghĩa, câu hỏi, ngữ pháp, đề viết)', where: 'Job nền', requests: '1' },
      { feature: 'Luyện tập từ vựng', where: 'Job nền', requests: '1' },
      { feature: 'Ảnh từ vựng (nếu bài bật ảnh)', where: 'Job nền', requests: '1 mỗi từ chưa có ảnh (≤ 30)', paid: true },
    ],
  },
  {
    title: 'Người học',
    rows: [
      { feature: 'Hỏi nghĩa từ trong bài', where: 'Bài đọc', requests: '1 lần đầu; 0 khi đã có người hỏi' },
      { feature: 'Nộp bài viết / Chấm lại', where: 'Bước Viết', requests: '1' },
      { feature: 'Đọc, nghe, ôn thẻ, bài tập, điền từ, dịch câu', where: '', requests: '0' },
    ],
  },
];
