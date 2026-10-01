# Research: AI sinh bài học (F7)

Không còn mục NEEDS CLARIFICATION. Mỗi mục: quyết định, lý do, phương án đã cân nhắc.

## R1. Sinh đồng bộ trong một request, bản nháp chỉ ở client

- **Quyết định**: `POST /api/admin/topics/{id}/generate` gọi AI ngay trong request (tối đa 60 giây) và trả bản nháp về client; không
  tạo job, không lưu bản nháp vào DB. Bản nháp sống trong signal của trang lộ trình.
- **Lý do**: Quản trị viên đang ngồi chờ trên trang để duyệt; spec yêu cầu không lưu tự động (FR-008) và bản nháp mất khi rời trang
  là chấp nhận được (Assumptions). Một lượt = 1 request AI (FR-020). Không cần collection, job type, polling.
- **Phương án loại**: (a) job nền + collection `lesson_drafts` + polling: đúng chữ nguyên tắc III nhưng thêm collection, job type,
  màn hình trạng thái và dọn bản nháp cũ cho một thao tác quản trị vài lần mỗi tuần; (b) lưu bài ngay với trạng thái "nháp" trong
  `lessons`: vi phạm FR-008 và phải lọc khắp nơi. Ngoại lệ với nguyên tắc III ghi ở Complexity Tracking của plan.

## R2. Giới hạn thời gian 60 giây xuyên qua nginx và http.Server

- **Quyết định**:
  - Service bọc lời gọi AI bằng `context.WithTimeout(ctx, 60*time.Second)` (`generateTimeout`).
  - Handler gia hạn ghi cho riêng request này: `http.NewResponseController(w).SetWriteDeadline(time.Now().Add(75*time.Second))`
    (server đang có `WriteTimeout: 15s`).
  - `frontend/nginx.conf` thêm `location ~ ^/api/admin/topics/[^/]+/generate$` với `proxy_read_timeout 75s` (location `/api/`
    đang là 15s; location regex thắng location tiền tố không có `^~`).
  - http.Client của Gemini giữ `aiTimeout = 90s` (lớn hơn 60s, context cắt trước).
  - Frontend không đặt timeout riêng (trình duyệt chờ lâu hơn 75s).
- **Lý do**: Không đổi thì request bị cắt ở 15 giây (cả nginx lẫn Go) trong khi Gemini Flash-Lite sinh 5 × 400 từ mất 10–30 giây.
  Chỉ nới cho đúng một route, các route khác giữ 15 giây.
- **Phương án loại**: tăng `WriteTimeout`/`proxy_read_timeout` toàn cục (nới cho mọi request, kể cả request lỗi treo); streaming
  (phức tạp, không cần).

## R3. Hợp đồng ai.Provider và client Gemini

- **Quyết định**: Thêm vào `ai.Provider`:
  `GenerateLessons(ctx, GenerateRequest) ([]LessonDraft, error)` với
  `GenerateRequest{Level, TopicName string; Count, Words int; Kind LessonKind; Idea string; ExistingTitles []string}`,
  `LessonDraft{Title, Content string}`, `LessonKind` = `"reading" | "dialogue"`. `ai.Disabled` trả `ErrNotConfigured`.
  Gemini: tách phần gửi request/đọc lỗi/log của `Annotate` thành `c.generate(ctx, op, prompt, schema, temperature, attrs)` dùng chung;
  `GenerateLessons` gửi 1 request với `responseMimeType: application/json`, `responseSchema` là mảng `{title, content}` (cả hai
  `required`), `temperature: 0.9` (đa dạng nội dung; Annotate giữ 0.2). Log 1 dòng `ai request` với `op=generate`, `count`, `words`,
  không ghi nội dung hay khoá.
- **Prompt** (tiếng Anh, vì đầu ra là tiếng Anh): nêu học viên Việt Nam trình độ CEFR `{level}`, chủ đề `{topicName}`, đúng
  `{count}` bài khác nhau, mỗi bài khoảng `{words}` từ (không dưới 0.9×, không quá 1.1× để có biên trong ±20%), chỉ dùng từ vựng và
  ngữ pháp của trình độ; dạng `reading` = đoạn văn liền mạch; `dialogue` = hai đến ba nhân vật có tên, mỗi lượt một dòng
  `Name: câu nói`, không dòng trống, không ghi chú sân khấu; tiêu đề ngắn (≤ 8 từ), không trùng nhau, không trùng/không gần giống các
  tiêu đề đã có (liệt kê); ý chính nếu có: "use this idea as inspiration: …"; không markdown.
- **Lý do**: Constitution III (mọi lời gọi AI qua interface); responseSchema của Gemini đã dùng ở F2 và trả JSON đáng tin cậy.
- **Phương án loại**: một request mỗi bài (tốn hạn mức ×5, khó tránh trùng giữa các bài); interface riêng `ai.Generator` (thêm một
  chỗ cấu hình cho cùng nhà cung cấp, kiến trúc chỉ cho phép interface Provider).

## R4. Lọc bản nháp phía server

- **Quyết định**: `lesson.Service.Generate` lọc kết quả AI theo thứ tự, mỗi bản bị loại cộng vào `dropped`:
  1. Trim tiêu đề và nội dung; nội dung chuẩn hoá xuống dòng `\r\n` → `\n`, bỏ dòng trống thừa ở đầu/cuối.
  2. Rỗng tiêu đề hoặc nội dung → loại.
  3. Tiêu đề > 200 ký tự hoặc nội dung > 10000 ký tự (giới hạn F2) → loại.
  4. Khoá tiêu đề (`lower` + gộp khoảng trắng + bỏ dấu câu cuối) trùng với bài đã có trong chủ đề hoặc với bản trước đó trong lượt → loại.
  5. Số từ (`len(strings.Fields(content))`) ngoài `[ceil(0.8·N), floor(1.2·N)]` → loại.
  6. Nhiều hơn `count` bản → cắt bớt (không tính là loại).
  Còn 0 bản → `ErrUnusableDraft` ("AI trả về nội dung không dùng được").
- **Lý do**: FR-005/FR-006/FR-007 và SC-002 đòi 100% bản nháp đúng độ dài và không trùng tiêu đề; kiểm ở server để có test bằng
  Provider giả và client không phải tin AI. Prompt nhắm 0.9–1.1× nên tỷ lệ bị loại thấp.
- **Phương án loại**: giữ bản sai độ dài và gắn cảnh báo (trái SC-002); gọi AI lại để bù bản bị loại (tốn thêm request, có thể vượt
  60 giây). Người dùng thấy "đã loại k bản" và có thể sinh thêm.
- **Bài đã có**: lấy tiêu đề mọi bài của chủ đề (`Lessons.List(Filter{TopicID})`), không chỉ bài trong lộ trình, vì bài ngoài lộ
  trình vẫn có thể được thêm vào sau.

## R5. Kiểm tra đầu vào

- **Quyết định**: `count` 1–5; `words` 50–800; `kind` ∈ {reading, dialogue}; `idea` tối đa 500 ký tự (rune, sau trim). Lỗi trả 400
  `validation_failed` với `fields` tiếng Việt như các form khác. Chủ đề không tồn tại → 404 `not_found` ("Không tìm thấy chủ đề").
  Frontend kiểm tra trùng các quy tắc này trước khi gửi (FR-003); mặc định độ dài A1 120, A2 160, B1 220, B2 300, C1/C2 400; số bài 3.
- **Lý do**: khối 2 và spec; 800 từ × 5 bài vẫn nằm trong giới hạn kích thước response (≤ 1 MB đang đọc) và giới hạn nội dung F2.

## R6. Lưu bài và thêm vào cuối lộ trình trong cùng thao tác

- **Quyết định**: `POST /api/admin/lessons` nhận thêm `appendToRoadmap` (bool, mặc định false; PUT bỏ qua). `lesson.Input` thêm trường
  `AppendToRoadmap`. `Service.Create`: kiểm tra → `Lessons.Create` → nếu cờ bật: `Topics.AppendLesson(topicID, lessonID)`; lỗi thì
  `Lessons.Delete` bài vừa tạo (bù trừ, best-effort, ghi log nếu xoá cũng lỗi) và trả lỗi → cuối cùng mới enqueue TTS + chú thích.
  Port `lesson.Topics` thêm `AppendLesson`, adapter `lessonTopicsPort` gọi `topic.Service.AppendLesson` (mới, bọc `repo.AppendLesson`).
  Nguồn "AI sinh" và giấy phép "Nội dung do AI tạo" do frontend gửi (hằng số trong `core/models/generate.ts`), backend không đặc cách.
- **Lý do**: FR-012 không cho phép trạng thái "đã tạo bài mà chưa vào lộ trình". MongoDB chạy standalone (không transaction) nên dùng
  bù trừ; enqueue sau cùng để không có job cho bài đã bị xoá.
- **Phương án loại**: client gọi POST lesson rồi PUT roadmap (2 request, lỗi giữa chừng để lại bài lẻ, đua với thao tác kéo thả);
  endpoint riêng `POST /api/admin/topics/{id}/lessons` (lặp lại kiểm tra và JSON của F2).

## R7. Ánh xạ lỗi AI

| Lỗi | HTTP | `error` | Thông báo |
| --- | --- | --- | --- |
| `ai.ErrNotConfigured` | 503 | `ai_not_configured` | AI chưa được cấu hình. Liên hệ người vận hành. |
| `ai.ErrInvalidKey` | 503 | `ai_not_configured` | Khoá AI không hợp lệ. Liên hệ người vận hành. |
| `ai.ErrQuota` | 429 | `ai_quota` | Đã hết lượt AI, vui lòng thử lại sau. |
| `ErrUnusableDraft` | 502 | `ai_unusable` | AI trả về nội dung không dùng được, vui lòng thử lại. |
| `context.DeadlineExceeded` và lỗi khác | 502 | `ai_failed` | Sinh bài thất bại, vui lòng thử lại. |

- **Lý do**: khối 2 (503/429/502); khoá sai cũng là lỗi cấu hình, thử lại không giúp gì nên gộp vào 503 với thông báo riêng;
  `ai_unusable` tách khỏi `ai_failed` để frontend báo đúng loại (SC-005). Frontend hiển thị `message` của server.

## R8. Giao diện: dialog, danh sách bản nháp, chặn rời trang

- **Quyết định**:
  - Component `lu-generate-dialog` (`features/admin/generate-dialog`) dùng `<dialog>` gốc với `showModal()` (khoá focus, Escape đóng
    có sẵn). Form reactive: số bài, độ dài, dạng bài (radio), ý chính (textarea). Trong lúc sinh: nút "Đang sinh…", `aria-busy`, mọi
    ô khoá, không đóng bằng Escape. Thành công → đóng. Lỗi → `role="alert"` trong dialog, giá trị đã nhập giữ nguyên.
  - Lựa chọn lần trước lưu trong signal của trang để mở lại dialog thấy giá trị cũ (spec US3).
  - Danh sách bản nháp: component `lu-draft-list` (`features/admin/draft-list`) hiển thị; trạng thái `drafts = signal<DraftState[]>`
    ở trang lộ trình (`DraftState {key, title, content, saving, error, fields}`); mỗi bản có ô tiêu đề, textarea nội dung, số từ trực
    tiếp (`aria-live="polite"` thưa: chỉ đọc khi rời ô), nút Lưu, Bỏ; "Lưu tất cả" khi ≥ 2 bản.
  - Lưu tất cả: lưu tuần tự theo thứ tự hiển thị (giữ thứ tự lộ trình), bản lỗi ở lại, tiếp tục bản sau; tải lại lộ trình một lần ở
    cuối. Lưu một bản: tải lại lộ trình sau khi lưu. Nút Lưu khoá khi đang lưu (chống tạo trùng).
  - Chặn rời trang: `unsavedChangesGuard` (`core/guards/unsaved-changes.guard.ts`, `CanDeactivateFn<CanLeave>` gọi
    `component.canLeave(): boolean | Promise<boolean>`) gắn vào route `admin/roadmap`; trang hiện `lu-confirm-dialog` có sẵn và trả
    Promise. Đổi chủ đề trong ô chọn (cùng route, chỉ đổi query) cũng hỏi qua cùng hàm. Tải lại/đóng tab: `beforeunload` khi còn bản nháp.
- **Lý do**: dùng lại `ConfirmDialog` và phong cách form hiện có; `<dialog>` gốc dễ tiếp cận ở 360px và bàn phím mà không cần CDK
  Overlay; giữ state ở trang theo khối 2.
- **Phương án loại**: `window.confirm` cho guard (không theo design system, khó test); CDK Dialog (thêm phụ thuộc module, không cần).

## R9. Kiểm thử

- Go: `ai/gemini` với httptest (path, header khoá, `responseSchema` mảng `{title, content}`, temperature, prompt chứa trình độ/tiêu
  đề đã có/dạng hội thoại, 429 → `ErrQuota`, JSON hỏng → lỗi); `lesson` service với Provider giả (lọc rỗng/trùng/độ dài/quá giới
  hạn, cắt bớt, `ErrUnusableDraft`, truyền tiêu đề đã có, kiểm tra đầu vào, timeout context); `Create` với `appendToRoadmap` (thêm
  cuối lộ trình, lỗi append → xoá bài và không enqueue); handler (200, 400 count ngoài 1–5, 403 người học, 404 chủ đề, ánh xạ 503/429/502).
- Vitest: dialog (mặc định theo trình độ, lỗi đầu vào không gửi, đang sinh, lỗi giữ giá trị), trang lộ trình (lưu từng bài, lưu tất
  cả theo thứ tự và giữ bản lỗi, bỏ bản nháp, lỗi AI giữ bản nháp cũ, lượt mới nối thêm), guard (không bản nháp → cho đi; có → hỏi).
