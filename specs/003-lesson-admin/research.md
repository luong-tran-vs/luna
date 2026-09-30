# Research: Trang quản trị bài học (F2)

Khối 2 trong `docs/spec-inputs/f2-quan-tri-bai-hoc.md` đã chốt kiến trúc. File này chốt các chi tiết còn mở.
Không còn mục NEEDS CLARIFICATION.

## R1. Chống kết quả cũ ghi đè khi sửa nội dung (FR-013)

- **Decision**: `lessons.revision` (int, bắt đầu 1) tăng mỗi khi nội dung đổi. Job lưu `revision`. Khi worker
  ghi kết quả, dùng update có điều kiện `{_id, revision}`; không khớp → coi job xong nhưng bỏ kết quả. Sửa nội
  dung cũng xoá job `pending` cũ của bài.
- **Audio theo revision**: file ở `{AUDIO_DIR}/{lessonId}/{revision}/{index}.mp3`. Revision mới không đụng file
  cũ; sau khi audio revision mới xong thì xoá thư mục revision cũ. Xoá bài xoá cả `{lessonId}/`.
- **Rationale**: Không cần khoá; đúng với một worker và cả khi sau này nhiều worker.

## R2. Hàng đợi việc nền

- **Decision**:
  - Collection `jobs`: `{type: "tts"|"annotate", lessonId, revision, status: "pending"|"running"|"done"|"failed",
    attempts, error, runAt, createdAt, updatedAt}`.
  - Lấy việc: `findOneAndUpdate({status:"pending", runAt ≤ now}, {$set:{status:"running"}, $inc:{attempts:1}},
    sort runAt)`.
  - Một worker goroutine (TTS nặng CPU); ngủ 2 giây khi rảnh, hoặc thức ngay khi có việc mới (channel `Notify`).
  - Lỗi: `attempts < 3` → `pending`, `runAt = now + backoff` (30 giây, 2 phút); `attempts = 3` → `failed`, ghi
    trạng thái lỗi lên bài. Lỗi "không thử lại được" (`ai.ErrNotConfigured`, bài không tồn tại) → `failed` ngay.
  - Khởi động: `running` → `pending` (FR-012).
  - Job `done`/`failed` giữ lại để xem log; xoá khi xoá bài.
  - Tắt máy chủ: worker dừng theo context; job đang chạy bị huỷ context → trở về `pending` khi khởi động lại.
- **Mapping trạng thái bài**: `audioStatus`, `annotationStatus` ∈ `running | done | failed` (+ `audioError`,
  `annotationError`). Tạo/sửa nội dung/chạy lại → `running`.
- **Alternatives considered**: goroutine gọi trực tiếp không lưu job (mất việc khi tắt máy); Redis/queue (thêm dịch vụ).

## R3. Tạo audio (Kokoro-FastAPI)

- **Decision**: `POST {TTS_URL}/v1/audio/speech` body
  `{"model":"kokoro","input":<câu>,"voice":"af_heart","response_format":"mp3","speed":1.0}` → body là mp3.
  Timeout 60 giây/câu. Một job `tts` làm tuần tự mọi câu; **bỏ qua câu đã có file** (tiếp tục sau lỗi/khởi động
  lại, FR-016). Ghi file tạm rồi `rename` để không có file dở. Sau khi xong mọi câu: set
  `sentences[i].audioPath`, `audioStatus=done` (có điều kiện revision).
- **Image**: `ghcr.io/remsky/kokoro-fastapi-cpu` — ghim tag phiên bản cụ thể khi implement (README chỉ nêu
  `latest`). Cổng 8880, chỉ trong mạng Docker; mở `127.0.0.1:8880` cho chạy backend local.
- **License**: code Apache 2.0, model Kokoro-82M Apache 2.0 → nguyên tắc II.
- **Rationale**: Khối 2; API kiểu OpenAI đơn giản, không cần SDK.

## R4. Chú thích bằng AI (Gemini)

- **Decision**: `POST https://generativelanguage.googleapis.com/v1beta/models/{GEMINI_MODEL}:generateContent`,
  header `x-goog-api-key`. `generationConfig.responseMimeType = "application/json"` và `responseSchema` kiểu
  ARRAY các OBJECT `{text, lemma, meaningVi, sentenceIndex}` (required cả 4). Prompt tiếng Anh gồm trình độ CEFR,
  danh sách câu đánh số, yêu cầu 8–25 mục đáng học với trình độ đó, `text` phải chép nguyên văn từ câu, nghĩa
  tiếng Việt theo ngữ cảnh. Timeout 90 giây. `temperature` 0.2.
- **Model mặc định**: `gemini-3.5-flash-lite` (trang models hiện liệt kê là Flash-Lite ổn định). Đổi bằng
  `GEMINI_MODEL`. Hạn mức gói miễn phí xem trong AI Studio; ~1–3 request/ngày là rất thấp.
- **Endpoint**: `generateContent` vẫn được hỗ trợ (tài liệu mới giới thiệu thêm Interactions API; không cần).
- **Lỗi**: không có `GEMINI_API_KEY` hoặc `AI_PROVIDER=none` → `ai.ErrNotConfigured` (không thử lại). 429 → lỗi
  "hết hạn mức" (thử lại theo backoff). 5xx/timeout → thử lại. Body không parse được → lỗi, thử lại.
- **Đúng 1 request mỗi lần** (FR-018): mỗi job `annotate` gọi `Annotate` một lần mỗi attempt; tạo/sửa/chạy lại
  tạo đúng 1 job. Thử lại tự động khi lỗi tạm thời là lần gọi mới của cùng job (log ghi rõ attempt) — chấp nhận
  vì lần trước không có kết quả. Log `ai request` mỗi lần gọi để đếm (SC-004).
- **Log**: không log API key; không log toàn bộ nội dung bài.

## R5. Làm sạch kết quả AI (FR-021)

- Trim mọi trường; bỏ mục thiếu `text`, `lemma` hoặc `meaningVi`.
- Tìm `text` (không phân biệt hoa thường) trong câu `sentenceIndex`; không thấy thì tìm trong mọi câu và lấy câu
  đầu tiên chứa nó; vẫn không thấy → bỏ.
- Bỏ trùng (cùng `text` không phân biệt hoa thường + cùng câu).
- Còn 0 mục → lỗi "AI không trả về chú thích hợp lệ" (thử lại).
- Chú thích sửa tay (PUT annotations) dùng cùng hàm tìm câu; không thấy → 400 "Cụm từ không có trong bài".

## R6. Tách câu (FR-002)

- **Decision**: tự viết `SplitSentences(text) []string`:
  1. Chuẩn hoá: `\r\n` → `\n`; đoạn cách nhau bởi dòng trống luôn tách câu; trong đoạn, gộp khoảng trắng.
  2. Ứng viên kết thúc câu: một chuỗi `.`, `!`, `?`, `…` (có thể kèm `"`, `'`, `”`, `’`, `)` phía sau), theo sau là
     khoảng trắng và ký tự bắt đầu câu (chữ hoa, chữ số, ngoặc/nháy mở) hoặc hết đoạn.
  3. Không tách nếu từ trước dấu chấm là **danh xưng** (Mr, Mrs, Ms, Dr, Prof, St, Sr, Jr, Mt, Capt, Gen, Rev),
     **viết tắt luôn đi tiếp** (e.g, i.e, vs, cf, approx, Fig, No khi sau là chữ số) hoặc
     **chữ cái viết tắt tên** (một chữ hoa: "J. K. Rowling").
  4. Viết tắt có thể kết thúc câu (a.m, p.m, etc, U.S, U.K, Inc, Ltd, Co): tách nếu từ sau viết hoa.
  5. Số thập phân (9.30, 3.5) và URL/email không có khoảng trắng sau dấu chấm nên không bị tách.
- **Test mẫu** (SC-006): ví dụ trong spec, "J. K. Rowling wrote it.", "It costs $3.50. Cheap!", "Wait... what?",
  "\"Stop!\" she said. He stopped.", "Wait… Really?", "e.g. apples", "U.S. troops left. They…", đoạn nhiều dòng.
- **Alternatives considered**: thư viện Go tách câu (thêm phụ thuộc, khó chỉnh cho quy tắc riêng).

## R7. Quy tắc lộ trình

- Một document `roadmap {_id:"main", lessonIds:[ObjectID], updatedAt}`.
- `PUT /api/admin/roadmap {lessonIds}`: thay toàn bộ thứ tự; từ chối nếu có id trùng hoặc bài không tồn tại. Thêm,
  gỡ, đổi chỗ trên frontend đều gửi danh sách đầy đủ (đơn giản, lưu ngay sau mỗi thay đổi — FR-026).
- `GET` trả `lessons` (tóm tắt theo thứ tự: id, tiêu đề, trình độ, trạng thái), `remaining` (= số bài; F2 chưa có
  tiến độ) và `warning` (`remaining < 3`).
- Xoá bài: nếu id có trong roadmap → 409 `lesson_in_roadmap`.

## R8. Giới hạn và kiểm tra bài

- `title` 1–200 ký tự, `content` 1–10.000 ký tự và ≥ 1 câu sau khi tách, `level` ∈ A1…C2, `topic` ≤ 60,
  `source` 1–200, `license` 1–100. Trim mọi trường. Lỗi → 400 `validation_failed` + `fields` (tiếng Việt).
- Tối đa 200 câu mỗi bài (10.000 ký tự hiếm vượt; tránh job TTS quá dài).

## R9. Phục vụ audio (FR-017)

- `GET /api/audio/{lessonId}/{revision}/{index}` sau `RequireAuth` (người học cũng nghe được ở F4).
- Kiểm tra `lessonId` là hex 24 ký tự, `revision` và `index` là số nguyên ≥ 0 → ghép đường dẫn bằng
  `filepath.Join(AUDIO_DIR, …)` (không nhận chuỗi tự do → không path traversal). `http.ServeFile` với
  `Content-Type: audio/mpeg`, `Cache-Control: private, max-age=31536000, immutable` (đường dẫn có revision nên bất
  biến).
- `sentence.audioPath` lưu dạng `"/api/audio/{id}/{rev}/{i}"` để frontend dùng thẳng.

## R10. Frontend

- Route `admin` → `loadChildren: admin.routes` với `canActivate: [authGuard, adminGuard]` áp cho cả nhánh:
  `''` danh sách, `lessons/new`, `lessons/:id`, `lessons/:id/edit`, `roadmap`.
- `pollWhile(source, predicate, 5000)`: gọi lại mỗi 5 giây khi `predicate(kết quả)` đúng (còn bài `running`), dừng
  khi không còn, dừng khi rời trang (`takeUntilDestroyed`). Trang chi tiết cũng dùng.
- Kéo thả: `CdkDropList` + `CdkDrag` với `cdkDragHandle` (tay nắm 44px) để cuộn trang trên điện thoại không bị
  kéo nhầm; nút Lên/Xuống cho bàn phím; sau mỗi thay đổi gọi PUT, lỗi thì hoàn tác và báo.
- Form sửa: nếu nội dung đổi và bài có chú thích `editedByAdmin` → `confirm` bằng hộp thoại trong trang (không
  dùng `window.confirm` để test được): "Các chú thích đã sửa tay sẽ bị thay mới".
- Nghe câu: một `<audio>` ẩn dùng chung, nút ▶ theo từng câu; audio đi cùng cookie phiên (cùng origin).
- Chip trạng thái: chữ + biểu tượng (vòng xoay, ✓, !) + màu token (`--color-ok`, `--color-bad`,
  `--color-text-muted`), chữ luôn `--color-text` (như F0 R6).

## R11. Docker

- Service `kokoro`: image CPU (tag ghim), `restart: unless-stopped`, cổng `127.0.0.1:8880:8880`. Backend
  `depends_on: kokoro` (không chờ healthy — TTS lỗi thì job tự thử lại).
- Volume `audio-data` mount `/data/audio` cho backend. Image distroless `nonroot` (uid 65532): volume mới do Docker
  tạo thuộc root → dùng thư mục có sẵn quyền: Dockerfile tạo `/data/audio` và `chown 65532` ở stage build rồi copy
  sang (`COPY --chown`), để volume khởi tạo từ image với đúng quyền.
- Backend env: `TTS_URL=http://kokoro:8880`, `AUDIO_DIR=/data/audio`, `AI_PROVIDER`, `GEMINI_API_KEY` (rỗng mặc
  định), `GEMINI_MODEL`.
