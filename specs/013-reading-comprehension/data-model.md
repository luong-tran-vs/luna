# Data Model: Câu hỏi hiểu bài và ghi chú ngữ pháp (F15)

## `lessons` (mở rộng, không migration)

| Trường | Kiểu | Ghi chú |
| --- | --- | --- |
| questions | array<Question> | 0–5; thiếu = rỗng |
| grammarNote | GrammarNote \| null | thiếu = null |
| writingPrompt | string | thiếu = "" |
| extrasEditedByAdmin | bool | true sau khi quản trị viên lưu extras; AI lưu lại → false |
| quizVersion | int | thiếu = 0; `$inc` khi bộ câu hỏi bị thay (AI lưu, admin đổi câu hỏi, đổi nội dung) |

**Question** `{prompt, options[4], answerIndex, explanationVi}`:

- prompt: 1–300 ký tự.
- options: đúng 4 lựa chọn, mỗi lựa chọn 1–150 ký tự, khác nhau (so không phân biệt hoa thường, gộp khoảng trắng).
- answerIndex: 0–3.
- explanationVi: 1–500 ký tự.
- Nội dung các câu hỏi không trùng nhau.

**GrammarNote** `{title, bodyVi, examples[]}`:

- title: 1–100 ký tự.
- bodyVi: 1–1500 ký tự.
- examples: 1–3 ví dụ, mỗi ví dụ 1–300 ký tự, phải có trong `content`. Khi so, chuẩn hoá chữ thường, gộp khoảng trắng và bỏ
  dấu câu cuối của ví dụ.

**writingPrompt**: tối đa 500 ký tự.

Tác động lên bài khi đổi trạng thái:

| Sự kiện | Kết quả |
| --- | --- |
| Tạo bài | extras rỗng, `quizVersion` 0 |
| AI chú thích xong (`SaveAnnotations` ở đúng revision) | ghi annotations + extras đã làm sạch, `extrasEditedByAdmin = false`, `quizVersion += 1` |
| `ReplaceContent` (sửa nội dung) | xoá questions/grammarNote/writingPrompt, `extrasEditedByAdmin = false`, `quizVersion += 1` |
| Admin `PUT extras` | ghi extras, `extrasEditedByAdmin = true`, `quizVersion += 1` nếu questions khác bộ cũ |
| `Retry(annotate)` | cho chạy khi trạng thái là done hoặc failed; extras cũ giữ nguyên tới khi AI lưu kết quả mới |

## `reading_answers` (MỚI)

| Trường | Kiểu | Ghi chú |
| --- | --- | --- |
| _id | ObjectID | |
| userId | ObjectID | người trả lời |
| lessonId | ObjectID | |
| quizVersion | int | phiên bản bộ câu hỏi lúc trả lời |
| questionIndex | int | 0..len(questions)-1 |
| choice | int | 0–3 |
| correct | bool | |
| answeredAt | date | |

Index:

- **unique** `(userId, lessonId, quizVersion, questionIndex)`: mỗi câu chỉ ghi một lần cho mỗi phiên bản; chống đua giữa hai tab.
- Index này cũng phục vụ `List` và `Totals` (match `userId` theo tiền tố).

Câu trả lời của phiên bản cũ không bị xoá; chúng vẫn được tính vào thống kê.

## Kiểu Go mới / đổi

- `ai.LessonExtras {Annotations []Annotation; Questions []Question; GrammarNote *GrammarNote; WritingPrompt string}`.
  `Provider.Annotate` trả `(LessonExtras, error)`.
- `lesson.Question`, `lesson.GrammarNote`, `lesson.Extras {Questions; GrammarNote *GrammarNote; WritingPrompt}`.
  `Lesson` có thêm `Extras`, `ExtrasEditedByAdmin`, `QuizVersion`.
- `lesson.Repository`:
  - `SaveAnnotations(ctx, id, revision, anns, extras)` có thêm tham số extras.
  - Thêm `ReplaceExtras(ctx, id, extras, bumpQuiz bool)`.
- `lesson.AnswerRepository`:
  - `Insert(ctx, Answer) error`: trùng khoá trả `ErrAlreadyAnswered`.
  - `Get(ctx, userID, lessonID, version, index)`.
  - `List(ctx, userID, lessonID, version)`.
  - `Totals(ctx, userID) (answered, correct int, err)`.
- `lesson.Answer {UserID, LessonID string; QuizVersion, QuestionIndex, Choice int; Correct bool; AnsweredAt time.Time}`.
- Lỗi mới: `lesson.ErrQuizChanged`, `lesson.ErrAlreadyAnswered` (dạng `*AlreadyAnsweredError` kèm câu trả lời đã ghi).
- `progress.ReadingQuiz`:
  - `Status(ctx, userID, lessonID) (questions, answered int, err)`.
  - `Totals(ctx, userID) (answered, correct int, err)`.
- `progress.ErrReadIncomplete`.
- `progress.ReadingTotals {Answered, Correct int}`; `StatsView` có thêm `Reading`, `ReadingRate *float64`.
- `export.CollReadingAnswers`; `Export.ReadingAnswers []Doc` (JSON `readingAnswers`).

## Frontend

- `core/models/lesson.ts`:
  - `Lesson` có thêm `questions: Question[]`, `grammarNote: GrammarNote | null`, `writingPrompt: string`,
    `extrasEditedByAdmin: boolean`, `quizVersion: number`.
  - Thêm `ExtrasInput`.
- `core/models/reading.ts`:
  - `ReadingLesson` có thêm `quiz: Quiz | null` và `grammarNote: GrammarNote | null`.
  - `Quiz {version; questions: {prompt, options}[]; answers: QuizAnswer[]}`.
  - `QuizAnswer {questionIndex, choice, correct, answerIndex, explanationVi}`.
  - `AnswerResult {answer: QuizAnswer; answered; total; correct}`.
- `core/models/dashboard.ts`: `Stats` có thêm `reading: {answered, correct, rate: number | null}`.
