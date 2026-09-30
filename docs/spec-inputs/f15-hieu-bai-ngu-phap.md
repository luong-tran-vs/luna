# Đầu vào cho F15: Câu hỏi hiểu bài và ghi chú ngữ pháp

Nguồn: [giai-doan-2.md › F15](../phases/giai-doan-2.md). Sửa F2 (chú thích), F3 (bước Đọc), L (hoàn thành bước Đọc), F6 (thống kê).

## Khối 1: `/speckit-specify`

```text
F15 – Câu hỏi hiểu bài và ghi chú ngữ pháp (Giai đoạn 2) cho Luna.

Mục tiêu: bước Đọc kiểm tra được người học có hiểu bài không, và dạy thêm một điểm ngữ pháp của bài.

Kịch bản:
- Khi chú thích bài, AI trả về thêm trong cùng một lần gọi: 3–5 câu hỏi trắc nghiệm hiểu bài (4 lựa chọn, 1 đáp án, giải thích tiếng Việt), một ghi chú ngữ pháp tiếng Việt về một điểm nổi bật trong bài kèm ví dụ lấy từ bài, và một đề viết (dùng ở F8).
- Quản trị viên xem và sửa câu hỏi (thêm, xoá, đổi đáp án), ghi chú ngữ pháp và đề viết ở trang chi tiết bài.
- Bài có từ trước: bấm "Chạy lại chú thích" để sinh thêm các phần này. Chú thích từ đã sửa tay được cảnh báo trước khi bị thay.
- Người học ở bước Đọc: đọc bài, rồi trả lời lần lượt các câu hỏi; chọn xong mỗi câu thì thấy ngay đúng/sai, đáp án đúng và giải thích; không đổi được câu đã trả lời.
- Mục "Ngữ pháp" hiện trong bước Đọc, đọc được bất cứ lúc nào.
- Trả lời hết câu hỏi thì bước Đọc hoàn thành, kể cả khi có câu sai; hiện số câu đúng.

Quy tắc:
- Chú thích một bài vẫn chỉ tốn 1 request AI.
- Bài chưa có câu hỏi (AI lỗi hoặc bài cũ chưa chạy lại): bước Đọc dùng nút "Đã đọc xong" như giai đoạn 1.
- Đúng/sai phân biệt được không cần nhìn màu.
- Tỷ lệ trả lời đúng được lưu để thống kê (F6 hiện thêm).
- Mở lại bài đã học (trang Bài học) xem được câu trả lời cũ; không tính lại.
- Tiến độ trả lời được lưu: thoát ra vào lại tiếp tục đúng câu.

Ngoài phạm vi: câu hỏi tự luận, nhiều điểm ngữ pháp mỗi bài, bài tập ngữ pháp riêng.

Tiêu chí nghiệm thu: theo mục F15 trong docs/phases/giai-doan-2.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/ai, internal/lesson, internal/progress)
- Đổi ai.Provider.Annotate trả về LessonExtras {annotations, questions [{prompt, options[4], answerIndex, explanationVi}], grammarNote {title, bodyVi, examples[]}, writingPrompt}. Cùng 1 request Gemini, mở rộng responseSchema. Kiểm tra: câu hỏi đủ 4 lựa chọn khác nhau, answerIndex 0–3; bỏ câu không hợp lệ; ví dụ ngữ pháp phải có trong bài; thiếu phần nào thì phần đó để trống (không làm hỏng chú thích từ).
- lessons: thêm questions, grammarNote, writingPrompt, extrasEditedByAdmin. Endpoint admin: PUT /api/admin/lessons/{id}/extras.
- Người học: GET /api/lessons/{id} trả questions (không kèm answerIndex, explanation) và grammarNote. POST /api/lessons/{id}/answers {questionIndex, choice} → {correct, answerIndex, explanationVi}; lưu collection reading_answers {userId, lessonId, revision, questionIndex, choice, correct}; trả lời lại câu đã có → 409.
- progress: bước read hoàn thành khi số câu đã trả lời = số câu hỏi (server kiểm tra); bài không có câu hỏi giữ cách cũ.
- Sửa nội dung bài (revision mới) → câu trả lời cũ không còn áp dụng.
- dashboard/stats: thêm readingAccuracy.
- Test: CleanExtras (câu hỏi hỏng, ví dụ không có trong bài), answers (đúng/sai, 409, người khác), progress read với và không có câu hỏi.

Frontend
- features/admin/lesson-detail: phần Câu hỏi hiểu bài (thêm, xoá, sửa, chọn đáp án), Ngữ pháp, Đề viết.
- features/lesson/reading: component comprehension-quiz (từng câu, khoá sau khi chọn, ✓/✗ + chữ "Đúng"/"Sai" + giải thích), mục grammar-note (thu gọn được), bỏ nút "Đã đọc xong" khi bài có câu hỏi.
- Test Vitest: quiz (chọn, khoá, tiếp tục sau khi tải lại), fallback nút Đã đọc xong.
```
