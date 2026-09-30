# Đầu vào cho F2: Trang quản trị bài học

Nguồn: [giai-doan-1.md › F2](../phases/giai-doan-1.md), [mvp-features.md › F2](../mvp-features.md).

## Khối 1: `/speckit-specify`

```text
F2 – Trang quản trị bài học (Giai đoạn 1) cho Luna.

Mục tiêu: quản trị viên tự soạn bài học và xếp lộ trình để mỗi ngày người học có một bài mới.

Kịch bản:
- Quản trị viên tạo bài: dán văn bản tiếng Anh, nhập tiêu đề, trình độ CEFR (A1–C2), chủ đề, nguồn, giấy phép → lưu.
- Sau khi lưu, hệ thống tự làm nền 3 việc: tách bài thành câu, tạo audio đọc từng câu, và nhờ AI chú thích các từ và cụm từ đáng học (dạng gốc, nghĩa tiếng Việt theo ngữ cảnh).
- Danh sách bài hiện trạng thái audio và chú thích của từng bài: đang chạy, xong, lỗi; bài lỗi có nút Chạy lại.
- Quản trị viên xem trước bài, xem và sửa từng chú thích (sửa nghĩa, thêm, xoá).
- Sửa nội dung bài → tách câu, audio, chú thích được làm lại; chú thích đã sửa tay bị thay mới (có cảnh báo trước khi lưu).
- Xếp lộ trình: thêm bài vào lộ trình, kéo thả đổi thứ tự, gỡ bài khỏi lộ trình.
- Lộ trình còn dưới 3 bài chưa học → trang quản trị hiện cảnh báo.
- Xoá bài: không xoá được bài đang nằm trong lộ trình (phải gỡ trước).

Quy tắc:
- AI lỗi hoặc hết hạn mức: bài vẫn lưu được, trạng thái chú thích là lỗi.
- Mỗi lần tạo hoặc sửa bài chỉ gọi AI một lần cho chú thích.
- Audio tạo một lần và dùng lại.
- Chỉ quản trị viên truy cập được.
- Dùng được trên điện thoại từ 360px.

Ngoài phạm vi: AI sinh bài (giai đoạn 2), nhập bài từ URL, nhiều lộ trình.

Tiêu chí nghiệm thu: theo mục F2 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/lesson, internal/job, internal/tts, internal/ai, internal/storage/mongo)
- Collection lessons: title, content, level, topic, source, license, sentences [{index, text, audioPath}], audioStatus, annotationStatus, annotations [{text, lemma, meaningVi, sentenceIndex, editedByAdmin}], createdAt, updatedAt.
- Collection roadmap: một document duy nhất {lessonIds: [...] theo thứ tự}.
- Tách câu: tự viết trong internal/lesson (quy tắc dấu . ! ? + danh sách viết tắt Mr. Mrs. Dr. e.g. i.e. …), có unit test.
- Tác vụ nền: collection jobs (type: tts | annotate, lessonId, status, attempts, error, runAt). Worker goroutine trong cùng process, lấy job bằng findOneAndUpdate; tối đa 3 lần thử, backoff; job đang chạy khi server tắt được trả về pending lúc khởi động lại.
- TTS: container Kokoro-FastAPI bản CPU (ghcr.io/remsky/kokoro-fastapi-cpu), gọi POST /v1/audio/speech (voice af_heart, mp3). internal/tts có interface Synthesizer để giả lập trong test. File lưu ở volume /data/audio/{lessonId}/{index}.mp3; backend phục vụ qua GET /api/audio/... (yêu cầu đăng nhập).
- AI: internal/ai định nghĩa interface Provider { Annotate(ctx, lessonText, level) ([]Annotation, error) }. Hiện thực Gemini qua REST generateContent với responseMimeType application/json và responseSchema; model và API key lấy từ biến môi trường (GEMINI_API_KEY, GEMINI_MODEL, mặc định flash-lite). Không có key thì provider trả lỗi "chưa cấu hình" và job ghi trạng thái lỗi.
- Endpoint (RequireAdmin): GET/POST /api/admin/lessons, GET/PUT/DELETE /api/admin/lessons/{id}, PUT /api/admin/lessons/{id}/annotations, POST /api/admin/lessons/{id}/retry?job=tts|annotate, GET/PUT /api/admin/roadmap.
- Test: tách câu, worker (thành công, thử lại, quá số lần), service lesson (xoá bài trong lộ trình bị chặn, sửa nội dung tạo lại job) với repository, Synthesizer, Provider giả.

Frontend (features/admin)
- Trang: danh sách bài (kèm chip trạng thái, nút Chạy lại), form bài, chi tiết bài (xem trước + bảng chú thích sửa được), lộ trình (kéo thả bằng @angular/cdk/drag-drop).
- Tự làm mới trạng thái bằng polling 5 giây khi còn job đang chạy.
- Route /admin/** dùng adminGuard.
- docker-compose: thêm service kokoro (CPU) và volume audio.
```
