# Đầu vào cho F8: Viết

Nguồn: [giai-doan-2.md › F8](../phases/giai-doan-2.md). Cần F15 (đề viết). Sửa L (thêm bước), F6 (thanh Viết, thống kê), F13 (xuất dữ liệu).

## Khối 1: `/speckit-specify`

```text
F8 – Viết (Giai đoạn 2) cho Luna.

Mục tiêu: người học luyện viết mỗi ngày về chính bài vừa học, và nhận nhận xét cụ thể để sửa.

Kịch bản:
- Bài hôm nay có thêm bước Viết (bắt buộc) sau bước Nghe: Ôn → Đọc → Nghe → Viết.
- Người học thấy đề viết của bài và độ dài gợi ý theo trình độ, gõ bài; bản nháp tự lưu trong lúc gõ.
- Bấm Nộp: bước Viết hoàn thành ngay, bài hôm nay xong (mục tiêu +1, streak +1); AI chấm ở nền.
- Có kết quả: app hiện thông báo; mở ra thấy điểm 1–5 và nhận xét tiếng Việt cho 4 tiêu chí (hoàn thành yêu cầu, ngữ pháp, từ vựng, mạch lạc), bản đã sửa và phần so sánh với bản gốc (chỗ thêm, chỗ bớt).
- Trang "Bài viết": danh sách bài đã nộp (ngày, bài học, điểm trung bình hoặc trạng thái đang chấm/lỗi), mở từng bài xem nhận xét.
- Chấm lỗi: bài viết vẫn được lưu, có nút Chấm lại.

Quy tắc:
- Bài viết tối thiểu 5 từ, tối đa 400 từ.
- Bài chưa có đề viết: dùng đề mặc định "Tóm tắt bài bằng 3–5 câu".
- Sau khi nộp không sửa được bài viết.
- Mở lại bài đã học (trang Bài học): xem bài viết và nhận xét cũ, không nộp lại.
- Thanh kỹ năng Viết trên màn hình chính đếm số bài đã nộp bài viết; thống kê thêm số bài viết và điểm trung bình.
- File xuất dữ liệu (F13) có thêm bài viết và nhận xét.
- So sánh bản gốc và bản sửa không truyền đạt chỉ bằng màu.

Ngoài phạm vi: viết tự do ngoài bài học, chấm theo thang IELTS, nhiều đề mỗi bài.

Tiêu chí nghiệm thu: theo mục F8 trong docs/phases/giai-doan-2.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/writing mới, internal/ai, internal/job, internal/progress)
- Package internal/writing: model Writing {id, userId, lessonId, revision, prompt, text, status (draft|submitted), grade {status pending|done|failed, error, criteria [{name, score 1–5, commentVi}], correctedText, overallVi}, createdAt, submittedAt, gradedAt}; service, handler, repository interface; hiện thực ở storage/mongo/writings.go.
- ai.Provider thêm GradeWriting(ctx, GradeRequest{level, lessonText, prompt, text}) (Grade, error): 1 request Gemini, responseSchema cố định 4 tiêu chí.
- Chấm nền: job type "grade" trong internal/job (thử lại tối đa 3 lần, như tts/annotate); ErrQuota → failed kèm thông báo hết lượt.
- Endpoint (RequireAuth): GET/PUT /api/lessons/{id}/writing (nháp, debounce phía client), POST /api/lessons/{id}/writing/submit, POST /api/writings/{id}/regrade (chỉ khi failed), GET /api/writings (danh sách), GET /api/writings/{id}, GET /api/writings/unseen-count (kết quả mới chưa xem), POST /api/writings/{id}/seen.
- progress: thêm bước write vào thứ tự bước; hoàn thành khi submit; dashboard/stats thêm write count và averageScore; export thêm writings.
- Test: service (5–400 từ, nộp 2 lần → 409, đề mặc định), job grade với Provider giả (thành công, lỗi, hết lượt), progress 4 bước, quyền (người khác 404).

Frontend
- features/lesson/writing: ô soạn (textarea, đếm từ, lưu nháp debounce 1 giây, báo "Đã lưu nháp"), nút Nộp, trạng thái sau nộp.
- features/writings: trang danh sách và trang chi tiết; so sánh bản gốc và bản sửa bằng diff theo từ (shared/utils/word-diff.ts, có unit test), phần thêm có gạch chân + nhãn ẩn "thêm", phần bớt gạch ngang.
- Thông báo: header có chấm báo + số kết quả chưa xem (poll unseen-count mỗi 30 giây khi có bài đang chấm); toast khi có kết quả mới.
- Header thêm liên kết "Bài viết"; /today thêm bước Viết; home thêm thanh kỹ năng Viết.
- Test Vitest: word-diff, lưu nháp, nộp, trạng thái chấm.
```
