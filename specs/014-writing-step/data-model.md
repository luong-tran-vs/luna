# Data Model: Bước Viết (F8)

## `writings` (MỚI)

| Trường | Kiểu | Ghi chú |
| --- | --- | --- |
| _id | ObjectID | |
| userId | ObjectID | người viết |
| lessonId | ObjectID | bài học |
| lessonRevision | int | revision của bài lúc nộp |
| lessonTitle | string | tên bài lúc nộp (danh sách, bài đã xoá) |
| prompt | string | đề lúc nộp (đề của bài hoặc `Tóm tắt bài bằng 3–5 câu.`); nháp: rỗng |
| text | string | bài viết; nháp có thể rỗng; khi nộp 5–400 từ |
| status | string | `draft` \| `submitted` |
| grade | object \| null | null khi còn nháp |
| grade.status | string | `pending` \| `done` \| `failed` |
| grade.error | string | lý do tiếng Việt khi failed |
| grade.criteria | array | 4 phần tử theo thứ tự `task`, `grammar`, `vocabulary`, `coherence`: `{name, score 1–5, commentVi}` |
| grade.overallVi | string | nhận xét chung |
| grade.correctedText | string | bản đã sửa |
| grade.gradedAt | date | khi done/failed |
| grade.seen | bool | false khi vừa có kết quả; true sau khi xem |
| createdAt, updatedAt | date | |
| submittedAt | date | khi nộp |

Index:
- unique `(userId, lessonId)`.
- `(userId, submittedAt -1)` cho danh sách.
- `(userId, grade.status, grade.seen)` cho `unseen-count`.

Chuyển trạng thái:

```text
(chưa có) → draft (PUT nháp, chỉ khi CanWrite) → draft …
draft → submitted + grade.pending (POST submit, 5–400 từ, chỉ khi CanWrite)
grade.pending → done (job grade OK, seen=false) | failed (JobFailed, seen=false)
grade.failed → pending (POST regrade, seen=true)
done/failed: seen false → true (POST seen)
submitted: không đổi text/prompt nữa
```

## `jobs` (mở rộng)

- Thêm `targetId` (ObjectID, chỉ có ở job `grade`): id bài viết.
- `lessonId` chỉ ghi khi job thuộc về bài học (tts/annotate). Job `grade` không có `lessonId`, nên không bị xoá theo bài học.
- `job.Job` có thêm trường `TargetID string`; `job.TypeGrade = "grade"`.

## `lesson_progress` (không đổi cấu trúc)

- `steps.write: bool` được ghi tự động khi `progress.Steps` có `StepWrite`.
- Bài có `completedAt` từ trước F8 vẫn là bài đã xong.

## Kiểu Go mới

`ai`:
- `GradeRequest {Level, LessonText, Prompt, Text string}`.
- `Grade {Task, Grammar, Vocabulary, Coherence Criterion; OverallVi, CorrectedText string}`, với `Criterion {Score int; CommentVi string}`.
- `Provider.GradeWriting(ctx, GradeRequest) (Grade, error)`.

`writing`:
- `Writing` và `Grade {Status, Error, Criteria []Criterion, OverallVi, CorrectedText, GradedAt, Seen}`, với `Criterion {Name, Score, CommentVi}`.
- `Status`, `GradeStatus`.
- `Summary`: dòng của danh sách, gồm id, lessonId, lessonTitle, submittedAt, grade status, average.
- `UnseenCount {Unseen, Pending int; Latest *LatestResult{ID, Status}}`.

`writing` ports:
- `Repository`: `Get`, `GetByLesson`, `SaveDraft`, `Submit` (có điều kiện), `SetGrade`, `SetPending`, `MarkSeen`, `List`, `Unseen`, `Stats`.
- `Lessons.Info(ctx, id) (LessonInfo{Title, Level, Content, WritingPrompt, Revision}, error)`.
- `Steps.CanWrite(ctx, userID, lessonID) (bool, error)`.
- `job.Repository` để enqueue.
- `ai.Provider`.

Lỗi:
- `ErrNotFound`: không có, hoặc của người khác.
- `ErrSubmitted`.
- `ErrLocked`: không phải lúc viết.
- `ErrNotFailed`.
- `ErrUnusableGrade`.
- `*ValidationError{Fields}`: `text`.

Hằng số:
- `DefaultPrompt = "Tóm tắt bài bằng 3–5 câu."`
- `MinWords = 5`, `MaxWords = 400`.

`progress`:
- `StepWrite`, `ErrWriteIncomplete`.
- Port `Writings {Submitted(ctx, userID, lessonID) (bool, error); Stats(ctx, userID) (submitted int, average *float64, err error)}`.
- `SkillCounts.Write`, `StepCounts.Write`.
- `StatsView.Writing {Submitted int; AverageScore *float64}`.
- `StudyService.CanWrite`.

`export`: `CollWritings`; `Export.Writings []Doc` (JSON `writings`).

## Frontend

- `core/models/study.ts`: `Step` có thêm `'write'`, `STEPS` có 4 bước.
- `core/models/writing.ts` (MỚI):
  - `WritingStatus`, `GradeStatus`;
  - `Criterion {name, score, commentVi}`;
  - `Grade {status, error, criteria, overallVi, correctedText, average: number | null, gradedAt}`;
  - `Writing {id, lessonId, lessonTitle, prompt, text, status, submittedAt, grade}`;
  - `LessonWriting {prompt, level, canWrite, writing: Writing | null}`;
  - `WritingSummary`;
  - `UnseenCount {unseen, pending, latest}`;
  - `MIN_WRITING_WORDS`, `MAX_WRITING_WORDS`;
  - `SUGGESTED_WORDS: Record<Level, {min, max}>`;
  - `CRITERIA_LABELS`.
- `core/models/dashboard.ts`: `SkillCounts.write`; `Stats.writing {submitted, averageScore}`; `Stats.lessons.write`.
