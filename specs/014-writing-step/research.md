# Research: Bước Viết (F8)

Không còn mục NEEDS CLARIFICATION. Mỗi mục: quyết định, lý do, phương án đã cân nhắc.

## R1. Bước thứ tư trong `progress`

- **Quyết định**:
  - `StepWrite = "write"` thêm vào cuối `progress.Steps` (Ôn → Đọc → Nghe → Viết).
  - Phần lưu Mongo (`steps.write`) và `NextStep` tự theo danh sách, không phải sửa.
  - `completeStep` có thêm cổng cho `StepWrite`, giống cổng Đọc và Nghe. Port mới `progress.Writings.Submitted(ctx, userID, lessonID) (bool, error)`; chưa nộp thì trả `ErrWriteIncomplete` (409 `write_incomplete`).
- **Bài cũ**:
  - Bài đã có `CompletedAt` vẫn là "xong", vì `TodayLesson` và mục tiêu dựa trên `Progress.Completed`, không dựa trên `Done[write]`.
  - Bài đang dở (chưa `CompletedAt`) có `NextStep = write` sau Nghe.
  - Không cần migration.
- **Luồng nộp**: client gọi `POST /api/lessons/{id}/writing/submit`, thành công thì gọi ngay `POST /api/today/steps/write/complete`. Server kiểm tra bài đã nộp.
  - Lần gọi thứ hai lỗi (mất mạng): vào lại thấy bài đã nộp và nút **Tiếp tục**, như F15.
- **Lý do**: Giữ chiều phụ thuộc hiện có (`progress` không import gói khác, `main` nối port). Dùng lại thứ tự bước, tính idempotent, cập nhật `study_days`, streak và mục tiêu của `CompleteStep`. Người dùng vẫn thấy "xong ngay" (FR-005) vì hai request nối tiếp nhau trong cùng một lần bấm.
- **Phương án loại**: route submit tự gọi `progress.CompleteStep` (gói `writing` phụ thuộc `progress`, và 2 nơi cùng ghi tiến độ). Gộp Viết vào `progress` (trái kiến trúc theo domain).

## R2. Bước Viết chỉ mở cho bài hôm nay đang ở bước Viết

- **Quyết định**:
  - Port `writing.Steps.CanWrite(ctx, userID, lessonID) (bool, error)`. `progress.StudyService.CanWrite` trả true khi bài là bài hôm nay của người học và `NextStep(Done) == StepWrite`.
  - `PUT /writing` (nháp) và `POST /submit` đều kiểm tra; sai thì trả 409 `write_locked`.
  - `GET /writing` luôn được (để xem lại) và trả `canWrite` cho frontend.
  - Route `/api/lessons/{id}/writing*` đặt sau guard L (`CanOpen`), như các route của bài.
- **Lý do**: FR-001 (chỉ mở sau Nghe). Spec yêu cầu bài cũ mở lại chỉ được xem, không viết bù.

## R3. Gói `internal/writing` và collection `writings`

- **Quyết định**:
  - `Writing {ID, UserID, LessonID, LessonRevision, LessonTitle, Prompt, Text, Status (draft|submitted), Grade, CreatedAt, UpdatedAt, SubmittedAt}`.
  - `Grade {Status (pending|done|failed), Error, Criteria [4]Criterion{Name, Score, CommentVi}, OverallVi, CorrectedText, GradedAt, Seen bool}`.
  - Collection `writings` có index unique `(userId, lessonId)` (FR-006: tối đa một bài viết mỗi bài học).
  - Index `(userId, submittedAt -1)` cho danh sách.
  - Index `(userId, grade.status, grade.seen)` cho đếm chưa xem.
  - `LessonTitle` được lưu lúc nộp, để danh sách không phải tra bài và vẫn hiện tên khi bài bị xoá.
- **Thao tác**:
  - **Nháp**: `SaveDraft(user, lesson, text)` upsert khi `status != submitted`. Bài đã nộp → `ErrSubmitted` (409 `already_submitted`).
  - **Nộp**: `Submit(user, lesson, text)`.
    1. Đếm từ 5–400.
    2. Đề lấy từ `lesson.writingPrompt`, rỗng thì dùng `DefaultPrompt = "Tóm tắt bài bằng 3–5 câu."`.
    3. Lưu `status = submitted`, `submittedAt`, `grade = {pending, seen: false}`.
    4. Enqueue job `grade`.
    5. Cập nhật có điều kiện `status != submitted` để hai lần nộp đua nhau chỉ thắng một lần; lần sau → `ErrSubmitted`.
- **Lý do**: Khối 2. Lưu đề, revision và tên bài lúc nộp theo FR-007.

## R4. Việc chấm nền (job `grade`)

- **Quyết định**:
  - `job.TypeGrade = "grade"`. `job.Job` có thêm `TargetID string` (bson `targetId`, omitempty) là id bài viết. `LessonID` để rỗng: Mongo `Enqueue` chỉ ghi `lessonId` khi có, đọc ngược lại cũng vậy.
  - Nhờ vậy xoá bài học (`DeleteForLesson`) không xoá job chấm, và bài viết không kẹt ở "đang chấm".
  - Worker dùng chung (`MaxAttempts = 3`, backoff sẵn có).
  - `onFailed` trong `main.go` chia theo loại: `grade` → `writingSvc.JobFailed`, còn lại → `lessonSvc.JobFailed`.
- **Xử lý** (`writing.Service.ProcessGrade`):
  1. Tải bài viết theo `TargetID`. Không có hoặc không còn `pending` → `job.Permanent` hoặc bỏ qua.
  2. Tải bài học để lấy trình độ và nội dung. Bài đã xoá → vẫn chấm, chỉ không có nội dung bài.
  3. Gọi `ai.GradeWriting`. `ErrNotConfigured`, `ErrInvalidKey` → `job.Permanent`.
  4. Kiểm tra kết quả (R5). Không dùng được → lỗi thường (thử lại).
  5. Lưu `grade = done` (`seen: false`, `gradedAt`).
- **`JobFailed`**: lưu `grade = failed` (`seen: false`) với lý do tiếng Việt:
  - chưa cấu hình → "AI chưa được cấu hình";
  - khoá sai → "Khoá API của AI không hợp lệ";
  - `ErrQuota` → "AI hết lượt, vui lòng chấm lại sau";
  - kết quả hỏng → "AI trả về kết quả không dùng được";
  - lỗi khác → "Không chấm được bài".
- **Chấm lại** (`POST /api/writings/{id}/regrade`): chỉ khi `grade = failed` (khác thì 409 `not_failed`). Đặt `pending`, `seen = true`, rồi enqueue lại.
- **Lý do**: Nguyên tắc III (chạy nền, có trạng thái, chạy lại được, lưu kết quả). Dùng lại hàng đợi của F2.
- **Phương án loại**: Collection job riêng cho bài viết (lặp lại worker). Nhét id bài viết vào `LessonID` (sai nghĩa, lại bị xoá theo bài học).

## R5. `ai.GradeWriting` và schema 4 tiêu chí

- **Quyết định**:
  - Hàm: `GradeWriting(ctx, GradeRequest{Level, LessonText, Prompt, Text}) (Grade, error)`.
  - Kết quả: `ai.Grade {Task, Grammar, Vocabulary, Coherence Criterion; OverallVi, CorrectedText string}` với `Criterion {Score int; CommentVi string}`.
  - Gemini: 1 request, `temperature 0.3`. `responseSchema` là OBJECT, cả 6 key đều bắt buộc; mỗi tiêu chí là OBJECT `{score INTEGER, commentVi STRING}`.
- **Prompt** (tiếng Anh):
  - Vai trò giáo viên chấm học viên Việt Nam trình độ CEFR `{level}`, có đề, bài học (để chấm "hoàn thành yêu cầu") và bài viết.
  - Chấm 1–5 cho từng tiêu chí, nhận xét tiếng Việt ngắn, cụ thể, có ví dụ từ bài viết.
  - `correctedText`: sửa tối thiểu, giữ ý và cấu trúc của người học.
  - Không markdown.
- **Kiểm tra** (`writing.CheckGrade`): mỗi điểm nguyên 1–5, mỗi nhận xét không rỗng, `overallVi` và `correctedText` không rỗng (cắt bớt nếu quá dài: nhận xét 600, bản sửa 4000 ký tự). Sai → `ErrUnusableGrade`.
- **Lưu**: `Criteria` theo thứ tự cố định với tên `task`, `grammar`, `vocabulary`, `coherence`. Frontend hiện "Hoàn thành yêu cầu", "Ngữ pháp", "Từ vựng", "Mạch lạc".
- **Điểm trung bình**: trung bình 4 điểm, làm tròn 1 chữ số; tính khi đọc, không lưu.
- **Lý do**: Schema cố định bảo đảm đủ 4 tiêu chí (FR-009). Dữ liệu người học gửi AI chỉ là bài viết, không kèm danh tính (nguyên tắc V).

## R6. Thông báo trong app

- **Quyết định**:
  - Route `GET /api/writings/unseen-count` trả `{unseen, pending}`:
    - `unseen` = bài đã nộp có `grade.status ∈ {done, failed}` và `seen = false`;
    - `pending` = số bài đang chấm.
  - Route `POST /api/writings/{id}/seen` đặt `seen = true`. Trang chi tiết gọi khi mở bài có kết quả.
- **Frontend `WritingNotifier`** (`core/services/writing-notifier.service.ts`, root):
  - Sau đăng nhập và mỗi lần chuyển trang, nếu chưa có số liệu thì gọi `unseen-count` một lần.
  - Khi `pending > 0` thì poll mỗi 30 giây, hết `pending` thì dừng.
  - `submitted()` (gọi sau khi nộp) bắt đầu poll.
  - Mỗi lần `unseen` tăng, đẩy một toast.
- **Header**: liên kết "Bài viết" có dấu báo số (ví dụ "2"), `aria-label="Bài viết, 2 kết quả mới"`.
- **Toast** `lu-toast` (`shared/components/toast`):
  - Một vùng `role="status"` `aria-live="polite"` cố định ở dưới cùng trang.
  - Nội dung "Bài viết đã có kết quả" (hoặc "Chấm bài viết bị lỗi"), nút "Xem" mở `/writings/{id}`, nút đóng.
  - Tự ẩn sau 8 giây, không che nút ở 360px (đặt trên chân trang, rộng tối đa 100vw − 32px).
- **Chọn bài cho toast**: `unseen-count` trả thêm `latest: {id, status}` của bài vừa có kết quả gần nhất.
- **Lý do**: SC-004 (≤ 1 phút, poll 30 s). Không poll khi không có gì đang chấm. Không cần SSE hay websocket.
- **Phương án loại**: SSE (thêm kết nối dài qua nginx). Poll liên tục mọi lúc (lãng phí).

## R7. So sánh bản gốc và bản sửa theo từ

- **Quyết định**:
  - `shared/utils/word-diff.ts` có `wordDiff(a, b): DiffPart[]`, với `DiffPart {kind: 'same'|'add'|'del', text}`.
  - Tách token gồm từ kèm khoảng trắng phía sau. Dùng LCS quy hoạch động O(n·m); 400 × 450 từ là đủ nhanh.
  - Gộp các phần liền kề cùng loại. So sánh phân biệt hoa thường và dấu câu: dấu câu sửa cũng là thay đổi.
- **Hiển thị**:
  - `add` → `<ins>` gạch chân, có `<span class="visually-hidden">thêm: </span>`.
  - `del` → `<del>` gạch ngang, có nhãn ẩn "bớt: ".
  - Màu chỉ phụ.
  - Có chú thích "Gạch chân: thêm · Gạch ngang: bớt" hiện ra.
- **Lý do**: FR-014, SC-006. Không thêm thư viện diff (nguyên tắc VII).

## R8. Frontend

- **Bước Viết `lu-writing`** (`features/lesson/writing`): dùng trong `/today` khi `currentStep === 'write'`, và ở route `lessons/:id/write` (xem lại).
  - Hiện đề, gợi ý độ dài theo trình độ (A1 30–60, A2 50–80, B1 80–120, B2 120–180, C1/C2 150–250), textarea, đếm từ `countWords`.
  - Lưu nháp debounce 1 s; trạng thái "Đang lưu…", "Đã lưu nháp", "Chưa lưu được nháp" (`aria-live`).
  - **Nộp** khoá khi ngoài 5–400 từ, có hint.
  - Sau khi nộp hiện "Đã nộp, AI đang chấm" và liên kết tới bài viết. Phát `completed` để Today gọi `complete('write')`.
  - Bài đã nộp hoặc `canWrite = false`: chỉ đọc, kèm tóm tắt kết quả và liên kết chi tiết.
- **Trang `features/writings`**: danh sách (`/writings`) và chi tiết (`/writings/:id`).
  - Chi tiết có đề, bài gốc, trạng thái.
  - Khi đã chấm: 4 thẻ tiêu chí (điểm "4/5" + nhận xét), điểm trung bình, nhận xét chung, bản sửa, phần so sánh.
  - Chấm lỗi: lý do và nút **Chấm lại**.
  - Đang chấm: tự tải lại khi notifier báo có kết quả cho bài này (theo dõi `latest`) hoặc poll 30 s trong lúc mở.
- **Các chỗ sửa khác**:
  - `core/models/study.ts`: `Step` có thêm `'write'`.
  - `step-indicator`: nhãn "Viết", lưới 4 cột (giữ 1 hàng ở 360px).
  - Home: thanh "Viết" (`tone="write"`, token `--color-skill-write` có sẵn).
  - My-lessons: liên kết "Bài viết" mở `lessons/:id/write` cho bài đã xong.
  - Stats: thẻ "Bài viết".
  - Header: liên kết và dấu báo.
  - Routes `writings`, `writings/:id`, `lessons/:id/write`.

## R9. Thống kê, màn hình chính, xuất dữ liệu

- **Quyết định**:
  - `SkillCounts`, `StepCounts` có thêm `Write` (Mongo đếm `steps.write`).
  - `StatsView` có thêm `Writing {Submitted int; AverageScore *float64}` lấy từ port `progress.Writings.Stats(ctx, userID)`. Mongo aggregate trung bình của trung bình 4 điểm các bài `grade.status = done`.
  - JSON `/api/dashboard` `skills.write`; `/api/stats` thêm `writing: {submitted, averageScore}`.
  - Export: `CollWritings = "writings"`, trường `writings` (cả nháp).
- **Lý do**: FR-018, FR-019. `Submitted` đếm mọi bài đã nộp; điểm chỉ tính bài đã chấm.

## R10. Kiểm thử

- **Go**:
  - `writing` service:
    - nháp lưu và đọc lại;
    - 4 / 5 / 400 / 401 từ;
    - nộp 2 lần → `ErrSubmitted`, nháp sau khi nộp → `ErrSubmitted`;
    - đề mặc định;
    - `CanWrite` false → `ErrLocked`;
    - đề và tên bài lưu lúc nộp;
    - enqueue `grade`.
  - `ProcessGrade` với Provider giả: thành công, kết quả hỏng → lỗi thường, hết lượt / chưa cấu hình → `JobFailed` lưu lý do.
  - `Regrade` chỉ khi failed; `unseen-count`, `seen`.
  - Quyền: người khác → `ErrNotFound`, 404.
  - Handler đủ các mã lỗi.
  - Gemini httptest: schema, prompt, 429.
  - `progress`:
    - 4 bước, write gate;
    - bài đã xong trước F8 vẫn xong;
    - `CanWrite`;
    - dashboard `write`, stats `writing`.
  - Mongo `jobs`: `targetId`, `lessonId` rỗng.
  - Export `writings`.
- **Vitest**:
  - `word-diff` (thêm, bớt, thay, giống hệt, rỗng);
  - `lu-writing`: debounce lưu nháp, báo lỗi lưu, khoá Nộp theo số từ, nộp → `completed`, chỉ đọc khi đã nộp;
  - trang danh sách, trang chi tiết (đang chấm / xong / lỗi + Chấm lại, nhãn thêm/bớt);
  - notifier: poll khi `pending`, dừng khi hết, toast khi `unseen` tăng;
  - header có dấu báo;
  - today có bước Viết;
  - home có thanh Viết.
