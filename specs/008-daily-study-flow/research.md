# Research: Luồng một ngày học (L)

Không còn mục NEEDS CLARIFICATION.

## R1. Logic thuần trong `internal/progress/rules.go`

- **Decision**: Mọi quy tắc ngày/bài/streak là hàm thuần, nhận dữ liệu đã đọc và `now`, `loc`; service chỉ đọc/ghi DB và gọi các
  hàm này. Hàm (theo khối 2, tên Go xuất khẩu):
  - `DayKey(now, loc) string` — `YYYY-MM-DD` theo múi giờ; `NextDayKey(key)`.
  - `EffectiveGoal(goals, today) (Goal, bool)` — goal `active` (tối đa một); `effectiveFrom > today` nghĩa là chủ đề mới chỉ có
    hiệu lực từ ngày đó (dùng để hiện thông báo; bài hôm nay đã bắt đầu vẫn cố định bởi `study_days`).
  - `TodayLesson(goal, roadmap, completed, todayDay) TodayState` — `studying{lessonId}` (bài đã cố định hôm nay, hoặc bài chưa
    hoàn thành đầu tiên của lộ trình), `doneToday`, `noNewLesson` (hết bài), `noGoal`.
  - `CanStartNewLesson(todayDay)` — hôm nay chưa có bài nào hoàn thành.
  - `Streak(days, today) int` — số ngày liên tiếp có `completed`, kết thúc hôm nay (nếu hôm nay xong) hoặc hôm qua.
  - `ReviewQuota(limit, reviewedToday) int` — `max(0, limit - reviewedToday)`.
- **Rationale**: Nguyên tắc VI: phủ test cho nửa đêm, đổi múi giờ, nghỉ nhiều ngày, hết lộ trình, đổi chủ đề trước/sau khi bắt đầu,
  quay lại chủ đề cũ mà không cần DB. Đồng hồ tiêm vào service (`now func() time.Time`, như các gói khác).

## R2. Cố định bài hôm nay và hiệu lực đổi chủ đề

- `study_days {userId, dayKey, lessonId, reviewedCount, completed}` được tạo (upsert) **khi bước đầu tiên hoàn thành**, ghi
  `lessonId` → bài hôm nay cố định từ lúc đó (spec Assumptions). Trước đó, bài hôm nay tính lại mỗi lần từ lộ trình hiện tại.
- Đổi mục tiêu `POST /api/goals {topicId}`: goal cũ → `paused`, goal của chủ đề (tạo mới hoặc goal `paused` cũ) → `active`
  ngay, `effectiveFrom = today` nếu hôm nay chưa có bước nào xong, ngược lại `= NextDayKey(today)`. Vì `study_days` đã cố định bài
  hôm nay, hôm nay vẫn học tiếp bài đang dở (của chủ đề cũ); từ ngày mai bài hôm nay tính theo goal mới. Phản hồi mang
  `effectiveFrom` để frontend hiện "Chủ đề mới bắt đầu từ ngày mai".
- Tiến độ của goal **không lưu số đếm**: = số bài trong lộ trình hiện tại của chủ đề có `lesson_progress.completedAt` của người
  học. Nhờ đó quay lại chủ đề cũ tự tiếp tục bài dở (FR-003), bài quản trị viên thêm sau tự vào lộ trình (khối 2).
- Goal `completed` khi mọi bài của lộ trình (khác rỗng) đã xong; phản hồi của bước cuối có `goalCompleted: true` để hiện chúc
  mừng. Quản trị viên thêm bài sau đó → trạng thái tính lại thành chưa xong (tính khi đọc, không cần job).
- **Alternatives considered**: giữ goal cũ `active` tới ngày mai (hai goal active, trái khối 2); chụp danh sách bài vào goal
  (không nhận bài thêm sau).

## R3. Giới hạn 30 thẻ và đếm thẻ đã ôn ở bước Ôn

- **Decision**: `review_logs` (F5) thêm trường `context: "daily" | "free"`; `POST /api/vocab/cards/{id}/review` nhận thêm
  `context` (tuỳ chọn, mặc định `free`). `review-session` có input `context`. Số thẻ đã ôn ở bước Ôn hôm nay = đếm
  `review_logs {userId, context: "daily", reviewedAt ≥ 0h hôm nay}` (port `vocab` trong `main.go`).
- `GET /api/today/review-cards` = `vocab.Due(limit = ReviewQuota(30, reviewedToday))`. Số thẻ cần ôn trong `GET /api/today` =
  `min(total due, quota)`. Bằng 0 khi mở bước Ôn → server tự đánh dấu bước Ôn xong (FR-009).
- **Rationale**: Đếm theo log chính xác cả khi người học thoát giữa phiên (không phụ thuộc client báo số); `study_days.
  reviewedCount` vẫn lưu bản sao để F6 thống kê, cập nhật khi hoàn thành bước.
- Giới hạn: hằng `DefaultReviewLimit = 30`, đọc qua `Deps.ReviewLimit func(ctx, userID) int` (mặc định trả 30; F12 thay).
- **Alternatives considered**: client gửi số thẻ đã ôn khi xong bước (sai khi thoát giữa chừng); đếm mọi lần ôn kể cả ôn tự do
  (spec Assumptions: ôn tự do không tính).

## R4. Vị trí và thứ tự bước

- `lesson_progress {userId, lessonId, dayKey, steps{review, read, listen}, currentStep, sentenceIndex, completedAt}`; unique
  `(userId, lessonId)` (mỗi bài học một lần; xem lại không ghi).
- `PUT /api/today/position {step, sentenceIndex}`: chỉ cho bước hiện tại của bài hôm nay. Frontend gửi sau khi đổi câu, debounce
  1 giây (khối 2). Bước Đọc: câu đầu tiên đang nhìn thấy (IntersectionObserver trên các câu, lấy câu trên cùng); bước Nghe: câu đang
  làm.
- `POST /api/today/steps/{step}/complete`: 409 `step_locked` nếu bước trước chưa xong; idempotent (xong rồi thì trả trạng thái
  hiện tại, không cộng lần nữa). Bước `listen` chỉ nhận khi tổng kết chép chính tả của bài (F4, cùng gói `progress`) đã
  `completed`; bước `read` tin client ("Đã đọc xong" chỉ bật khi đã cuộn tới cuối, F3).
- Bước cuối xong → `lesson_progress.completedAt`, `study_days.completed = true` (streak +1), tiến độ goal +1 (tính lại).

## R5. Chặn bài chưa tới lượt (FR-019)

- `progress.Access.CanOpen(ctx, principal, lessonID)`: quản trị viên luôn được; người học được mở bài hôm nay (kể cả chưa bắt
  đầu), bài đã có `lesson_progress` (đã học xong hoặc đang học).
- Middleware `progress.Access.Guard` (dùng `r.PathValue("id")`) bọc các route cần nội dung bài của người học: `GET
  /api/lessons/{id}`, `/lookup`, `/vocabulary` (F3, F5), `/dictation`, `/dictation/summary` (F4). Bị chặn → 403 `lesson_locked`
  "Bài này sẽ mở khi tới lượt". `lesson.ReadingHandler.Register` và `progress.Handler.Register` nhận thêm middleware `guard`.
- Audio file (`/api/audio/...`) không chặn: đường dẫn chứa id + revision, chỉ lộ khi đã mở được bài (nguyên tắc VII).

## R6. Nhúng các bước vào trang `/today`

- `Reading` (F3) và `Listening` (F4) thêm input tuỳ chọn `lessonId` (mặc định lấy từ route như hiện nay), `startSentence`,
  `mode: 'today' | 'review'`, output `position` (số câu). `mode = 'review'`: Listening không POST kết quả, không khôi phục kết quả
  cũ; Reading không hiện nút "Đã đọc xong".
- Trang `/today` (`features/lesson/today`): `shared/components/step-indicator` (Ôn · Đọc · Nghe; trạng thái xong ✓ / hiện tại ● /
  khoá 🔒 kèm chữ ẩn cho trình đọc màn hình) + bước hiện tại; nối `review-session.finished`, `reading.completed`,
  `listening.completed` → `POST complete`, rồi tải lại `GET /api/today`.
- Trang `/goal` (`features/lesson/goal`): bước 1 chọn trình độ, bước 2 chọn chủ đề (`GET /api/topics?level=` + tiến độ từ `GET
  /api/goals`), màn chúc mừng khi `goalCompleted`.
- Trang `/lessons` (`features/lesson/my-lessons`): `GET /api/lessons/mine` → hôm nay, đã học (link `/lessons/:id/read`,
  `/listen` với `?review=1`), sắp tới (khoá, không link).
- Trang chủ F0: nút "Học hôm nay" → `/today`; header thêm "Bài học" (`/lessons`) cạnh "Sổ từ".

## R7. Ranh giới gói

- `progress` (đã có dictation F4) thêm goals, lesson_progress, study_days. Port: `Timezones` (auth), `Roadmaps` (topic: lộ
  trình, tên, trình độ, danh sách chủ đề theo trình độ), `LessonTitles` (lesson), `Reviews` (vocab: số thẻ đến hạn trong quota,
  số thẻ đã ôn ở bước Ôn hôm nay). Không import `topic`, `lesson`, `vocab` (như F14, research R1).
