/** Vietnamese names of the operations the server records (internal/ai/gemini). */
export const OP_LABELS: Readonly<Record<string, string>> = {
  generate: 'Sinh bài học',
  annotate: 'Chú thích bài',
  practice: 'Luyện tập và 6 bước của bài',
  glosses: 'Nghĩa trong bài (tra từ)',
  image: 'Sinh ảnh từ vựng',
  lesson_review: 'Kiểm tra bài',
  lesson_fix: 'Đề xuất sửa lỗi bài',
  lesson_simplify: 'Thay từ khó trong bài mới sinh',
  suggest_words: 'Gợi ý từ cho chủ đề',
  word_meanings: 'Điền nghĩa, phiên âm (kho từ)',
  grammar_lesson: 'Sinh bài ngữ pháp',
  grammar_solve: 'Kiểm tra bài tập ngữ pháp',
  explain: 'Người học hỏi nghĩa',
  grade: 'Chấm bài viết',
  talk: 'Luyện nói với AI (lượt đáp)',
  talk_review: 'Luyện nói với AI (nhận xét)',
  speak_check: 'Nhận xét bước Nói',
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
      {
        feature: 'Sinh bài học',
        where: 'Chủ đề → Sinh bài (1–5 bài)',
        requests: '1 cho cả lô, +1 nếu có bài dùng từ khó hơn trình độ',
      },
      {
        feature: 'Sinh bài theo giáo trình',
        where: 'Giáo trình → Sinh bài này / Sinh các ô trống',
        requests: '1 mỗi bài, +1 nếu bài dùng từ khó hơn trình độ',
      },
      { feature: 'Gợi ý từ cho chủ đề', where: 'Từ vựng chủ đề → Gợi ý bằng AI', requests: '1' },
      { feature: 'Kiểm tra bài', where: 'Chi tiết bài → Kiểm tra', requests: '1' },
      { feature: 'Đề xuất sửa lỗi bài', where: 'Từng mục bị đánh dấu', requests: '1 mỗi mục' },
      { feature: 'Sinh lại luyện tập', where: 'Chi tiết bài', requests: '1' },
      { feature: 'Chú thích lại / sửa nội dung bài', where: 'Chi tiết bài', requests: '1 (rồi tự chạy luyện tập, nghĩa trong bài, ảnh)' },
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
      {
        feature: 'Luyện tập và nội dung 6 bước (câu hỏi nghe, ngữ pháp, nói, gợi ý viết)',
        where: 'Job nền; bài cũ được sinh lại khi khởi động server',
        requests: '1',
      },
      {
        feature: 'Nghĩa trong bài của mọi từ (popup tra từ)',
        where: 'Job nền; bài cũ được bổ sung khi khởi động server',
        requests: '1',
      },
      { feature: 'Ảnh từ vựng (nếu bài bật ảnh)', where: 'Job nền', requests: '1 mỗi từ chưa có ảnh (≤ 30)', paid: true },
    ],
  },
  {
    title: 'Người học',
    rows: [
      { feature: 'Hỏi nghĩa từ trong bài', where: 'Bài đọc', requests: '1 lần đầu; 0 khi đã có người hỏi' },
      { feature: 'Nộp bài viết / Chấm lại', where: 'Bước Viết', requests: '1' },
      { feature: 'Nhận xét các câu trả lời nói', where: 'Bước Nói', requests: '1 (tối đa 5 lần mỗi bài mỗi ngày)' },
      { feature: 'Luyện nói với AI: mở buổi, mỗi lượt đáp', where: 'Ôn tập', requests: '1 mỗi lượt (tối đa 21 mỗi buổi)' },
      { feature: 'Luyện nói với AI: nhận xét cuối buổi', where: 'Ôn tập', requests: '1' },
      { feature: 'Đọc, nghe, ôn thẻ, bài tập, điền từ, dịch câu', where: '', requests: '0' },
    ],
  },
];
