# Feature Specification: Bài học 6 bước

**Feature Branch**: `019-six-step-lesson`

**Created**: 2026-10-22

**Status**: Implemented

**Input**: đánh giá hệ thống (2026-10-22) và bản phác thảo `design/de-xuat-quy-trinh-bai-hoc.png` đã được duyệt
("bản phác thảo ok, triển khai như bản phác thảo"). Đã chốt mặc định: Nghe trước Đọc; câu hỏi nghe ở A1–A2 viết bằng tiếng
Việt; bài cũ được sinh lại phần luyện tập tự động.

## Tóm tắt

Bài đang học đi qua 6 bước, đều lưu trên server: **Từ vựng → Nghe → Đọc → Ngữ pháp → Nói → Viết**. Bài hoàn thành khi xong
cả 6 (Viết vẫn bỏ qua được). Chi tiết từng bước ở `docs/mvp-features.md` R4.

## Yêu cầu

- **FR-001** Tiến độ: `progress.Steps` = vocab, listen, read, grammar, speak, write. Bài đã hoàn thành trước thay đổi vẫn hoàn
  thành; bài đang học dở cần thêm các bước mới.
- **FR-002** Nội dung bước mới (câu hỏi nghe, câu hỏi dẫn đọc, bước ngữ pháp, câu đọc theo, câu hỏi nói, gợi ý viết) sinh trong
  request luyện tập (`practice`), lưu ở `Practice.Steps`; phần nào không qua kiểm tra thì bỏ, bước vẫn chạy với phần còn lại.
  Bài chưa có điểm ngữ pháp được AI chọn một điểm trong giáo trình (lưu ở `Steps.GrammarPointID`).
- **FR-003** Khi khởi động, bài có luyện tập xong nhưng chưa có `Steps` được xếp hàng sinh lại một lần.
- **FR-004** Chép chính tả của bước Nghe dùng `lesson.Dictation`: 3 câu ví dụ ngắn (3–8 từ), thiếu thì lấy câu ngắn của bài;
  bài chưa có `Steps` giữ cách cũ (mọi câu).
- **FR-005** Câu hỏi hiểu bài có loại (`kind`: gist, detail, tfn, vocab), `promptVi`, `quote`; câu tfn có 3 lựa chọn cố định.
  Trình duyệt đảo thứ tự lựa chọn (trừ tfn) và gửi chỉ số gốc.
- **FR-006** Bài sinh mới có đoạn về người khác (he/she), câu phủ định.
- **FR-007** Chấm bài viết trả `mistakes` (gốc, sửa, loại, ghi chú); lỗi xếp đúng tiêu chí; `GET /api/writings/mistakes` cộng
  dồn theo loại (bỏ other, spelling). MySQL: migration `034_writing_mistakes`.
- **FR-008** `POST /api/lessons/{id}/speak/check`: 1–3 câu trả lời, 1 request AI, tối đa 5 lần mỗi người mỗi bài mỗi ngày
  (admin không giới hạn; lần AI lỗi không bị trừ).
- **FR-009** Bước Từ vựng không còn nút "Đã học": mọi từ trừ từ "Đã biết" vào ôn tập khi xong bước; từ nhớ sai đến hạn ngay.
- **FR-010** Luyện thêm (Hội thoại, Điền ô trống, Dịch câu) mở sau bước Đọc.

## Ngoài phạm vi

- Mở điểm ngữ pháp theo từng người học (điểm ngữ pháp mở khi admin đăng nội dung, như trước).
- Lưu kết quả câu hỏi nghe và bài tập ngữ pháp của bước lên server.
- Màn quản trị để sửa nội dung các bước mới (admin chỉ sinh lại luyện tập).
