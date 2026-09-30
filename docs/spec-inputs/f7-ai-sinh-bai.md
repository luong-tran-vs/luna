# Đầu vào cho F7: AI sinh bài học

Nguồn: [giai-doan-2.md › F7](../phases/giai-doan-2.md). Dựa trên F2 (quản trị bài học), F14 (chủ đề và lộ trình).

## Khối 1: `/speckit-specify`

```text
F7 – AI sinh bài học (Giai đoạn 2) cho Luna.

Mục tiêu: quản trị viên lấp đầy lộ trình của một chủ đề nhanh bằng bài do AI soạn, vẫn giữ quyền duyệt từng bài.

Kịch bản (quản trị viên):
- Ở trang lộ trình của một chủ đề (ví dụ "A1 · Gia đình"), bấm "Sinh bài bằng AI". Trình độ và chủ đề lấy theo lộ trình đang mở.
- Chọn: số bài (1–5), độ dài (số từ, gợi ý mặc định theo trình độ), dạng bài (bài đọc hoặc hội thoại), ý chính (không bắt buộc).
- Hệ thống sinh các bản nháp, mỗi bản gồm tiêu đề và nội dung. Trong lúc chờ có trạng thái đang sinh.
- Quản trị viên xem từng bản nháp: sửa tiêu đề, nội dung; bấm Lưu hoặc Bỏ từng bài; có Lưu tất cả.
- Bài đã lưu có nguồn "AI sinh", giấy phép "Nội dung do AI tạo", đi qua quy trình của F2 (tách câu, audio, chú thích) và được thêm vào cuối lộ trình của chủ đề.

Quy tắc:
- Bản nháp đúng trình độ của chủ đề; độ dài sai lệch không quá 20% so với yêu cầu.
- Các bài trong một lượt khác nội dung nhau và khác các bài đã có trong chủ đề.
- Không lưu tự động: bản nháp chưa lưu không xuất hiện trong danh sách bài hay lộ trình.
- AI lỗi, hết lượt hoặc trả về không dùng được: thông báo rõ ràng; giữ nguyên các bản nháp đã sinh và các lựa chọn đã nhập để thử lại.
- Rời trang khi còn bản nháp chưa lưu thì hỏi xác nhận.
- Chỉ quản trị viên dùng được; dùng được ở 360px.

Ngoài phạm vi: sinh bài theo lịch tự động, sinh ảnh minh hoạ, sinh bài cho nhiều chủ đề một lượt.

Tiêu chí nghiệm thu: theo mục F7 trong docs/phases/giai-doan-2.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/ai, internal/lesson, internal/topic)
- Mở rộng interface ai.Provider thêm GenerateLessons(ctx, GenerateRequest) ([]LessonDraft, error). GenerateRequest {level, topicName, count, words, kind (reading|dialogue), idea, existingTitles []string}. LessonDraft {title, content}.
- Hiện thực Gemini: 1 request cho cả lượt, responseMimeType application/json + responseSchema (mảng draft). Prompt nêu trình độ CEFR, độ dài, dạng bài, danh sách tiêu đề đã có để tránh lặp. Dialogue viết dạng "A: ... / B: ..." mỗi lượt một dòng.
- Kiểm tra kết quả: bỏ bản nháp rỗng hoặc tiêu đề trùng; không còn bản nào → lỗi "AI trả về nội dung không dùng được".
- Endpoint (RequireAdmin): POST /api/admin/topics/{id}/generate {count, words, kind, idea} → {drafts: [...]} (đồng bộ, timeout 60 giây). Bản nháp chỉ trả về client, không lưu DB.
- Lưu: dùng lại POST /api/admin/lessons với topicId, source "AI sinh", license "Nội dung do AI tạo", thêm cờ appendToRoadmap=true để thêm vào cuối lộ trình trong cùng thao tác.
- Lỗi AI ánh xạ: ErrNotConfigured → 503 ai_not_configured, ErrQuota → 429 ai_quota, lỗi khác → 502 ai_failed.
- Test: service với Provider giả (lọc bản rỗng/trùng, ánh xạ lỗi), handler (thành công, 400 count ngoài 1–5, 403 người học), gemini client với httptest (schema, lỗi 429).

Frontend (features/admin/roadmap)
- Nút "Sinh bài bằng AI" mở dialog: số bài, độ dài (mặc định A1 120, A2 160, B1 220, B2 300, C1–C2 400 từ), dạng bài, ý chính.
- Danh sách bản nháp giữ trong signal của trang: mỗi bản có ô sửa tiêu đề và nội dung, nút Lưu, Bỏ; nút Lưu tất cả; hiển thị số từ.
- Guard canDeactivate khi còn bản nháp chưa lưu.
- Sau khi lưu, lộ trình tải lại, bài mới ở cuối với trạng thái audio/chú thích "Đang chạy".
- Test Vitest: dialog kiểm tra đầu vào, lưu từng bài và lưu tất cả, lỗi AI giữ bản nháp.
```
