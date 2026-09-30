# Data Model: Màn hình chính và tiến độ (F6)

F6 không thêm collection. Hai "entity" dưới đây là view tính khi đọc (không lưu).

## Dashboard (view)

| Trường | Kiểu | Nguồn |
| --- | --- | --- |
| `Kind` | `noGoal` \| `studying` \| `doneToday` \| `noNewLesson` | `TodayState.Kind` (L) |
| `Goal` | `*GoalView` (topicId, topicName, level, completedLessons, totalLessons) | `goalView` (L) |
| `GoalCompleted` | bool | `completedLessons == totalLessons > 0` (L) |
| `Skills` | `{Read, Listen, Total int}` | `StepCounts(user, roadmap)`; `Total = len(roadmap)`; không có mục tiêu → không có |
| `Lesson` | `*{ID, Title, TopicName, Level}` | bài hôm nay (L); `Title` rỗng khi bài bị xoá (frontend hiện "Bài học") |
| `Steps` | `map[Step]StepState` | `view` (L) |
| `CurrentStep` | `review` \| `read` \| `listen` \| `done` | `NextStep` (L) |
| `Action` | `*{Kind: start\|continue, Step}` | `Action(...)` (research R2); nil khi không phải `studying` |
| `Streak` | int | `Streak` (L) |
| `TomorrowCards` | int | `CountDue(before = TomorrowCutoff, createdBefore = 0 giờ ngày mai)` |

Chủ đề của bài hôm nay: lấy từ `LessonProgress.TopicID` nếu bài đã bắt đầu (bài đang dở có thể thuộc chủ đề cũ khi vừa đổi mục tiêu),
ngược lại từ lộ trình của mục tiêu hiện hành.

### Quy tắc

- `Action`: `Kind = start` khi không bước nào `done`; `continue` khi có ≥ 1. `Step = CurrentStep`, trừ `CurrentStep = review` và
  `ReviewCount = 0` → `read`.
- `doneToday`: `Action = nil`; frontend hiện "Đã xong bài hôm nay, hẹn bạn ngày mai" + "Ôn tự do" (`/vocabulary/review`).
- `noNewLesson`: frontend hiện "Chưa có bài mới" + "Ôn tự do" + "Chọn chủ đề khác" (`/goal`). Nếu `GoalCompleted` thì lời chúc mừng
  như `/goal?completed=1`.
- `noGoal`: chỉ lời mời + "Chọn chủ đề" (`/goal`), vẫn có streak và số thẻ ngày mai.
- `TomorrowCutoff(now, loc)` = `time.Date(y, m, d+2, 0, 0, 0, 0, loc)` với `y, m, d` là ngày của `now.In(loc)`.
- Dashboard không ghi gì (không `Days.Start`, không `Upsert`).

## Stats (view)

| Trường | Kiểu | Nguồn |
| --- | --- | --- |
| `Cards` | int | `vocab.Repository.Count(user)` |
| `DictationSentences` | int | `DictationRepository.Totals` → `sentences` |
| `CorrectWords`, `TotalWords` | int | `Totals` |
| `Rate` | `*float64` | `correct / total`; nil khi `total = 0` |
| `ReadLessons`, `ListenLessons`, `CompletedLessons` | int | `StepCounts(user, nil)` |

## Phương thức repository mới

| Interface | Phương thức | Truy vấn MongoDB | Index |
| --- | --- | --- | --- |
| `progress.ProgressRepository` | `StepCounts(ctx, userID, lessonIDs []string) (StepCounts, error)`; `lessonIDs == nil` = mọi bài, rỗng (không nil) = 0 | `aggregate [$match {userId, lessonId $in}, $group {_id: null, read: $sum $cond steps.read, listen: …, completed: $sum $cond completedAt}]` | unique `(userId, lessonId)` |
| `progress.DictationRepository` | `Totals(ctx, userID) (DictationTotals, error)` | `aggregate [$match {userId}, $group {sentences: $sum 1, correct: $sum $correctWords, total: $sum $totalWords}]` | unique `(userId, lessonId, sentenceIndex)` |
| `vocab.Repository` | `CountDue(ctx, userID, before, createdBefore time.Time) (int, error)` | `countDocuments {userId, $or: [{due: {$lt: before}}, {due: {$exists: false}, createdAt: {$lt: createdBefore}}]}` | `(userId, due)`, `(userId, createdAt)` |
| `vocab.Repository` | `Count(ctx, userID) (int, error)` | `countDocuments {userId}` | `(userId, lemma)` |

Port `progress.Reviews` thêm `DueBefore(ctx, userID, before, createdBefore)` và `CardCount(ctx, userID)` (adapter `dailyReviews` gọi
`vocab.Service.DueBefore`, `vocab.Service.Count`). `progress.Service` thêm `Totals(ctx, userID)`.

`StepCounts` và `DictationTotals` là struct trong `progress`:

```go
type StepCounts struct{ Read, Listen, Completed int }
type DictationTotals struct{ Sentences, CorrectWords, TotalWords int }
```

Người học chưa có dữ liệu: aggregation trả không document → tất cả 0 (không lỗi).
