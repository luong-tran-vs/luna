---

description: "Task list for F15 – Câu hỏi hiểu bài và ghi chú ngữ pháp"
---

# Tasks: Câu hỏi hiểu bài và ghi chú ngữ pháp (F15)

**Input**: Design documents from `specs/013-reading-comprehension/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/reading-quiz-api.md, quickstart.md

**Tests**: REQUIRED by constitution principle VI and khối 2:
- Backend: `CleanExtras`, `ProcessAnnotate` (1 request), `Retry`, `UpdateExtras`, `Reader.View`/`Answer` (no answer leak, right/wrong, 409, other user), `progress` read with and without questions, stats, export, Gemini httptest.
- Frontend (Vitest): quiz, grammar note, reading fallback, lesson-extras, lesson-detail, stats.
- Real AI quality, 360px and keyboard are checked by hand (quickstart.md).

**Organization**: Tasks are grouped by user story (US1–US5 from spec.md). US3 (AI fills the data) comes first among the P1 stories because US1 and US2 read what it stores. US1/US2 can still be tested alone with lessons whose extras are set by fakes or by T031 (admin edit).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Add frontend types per data-model.md:
  - `Question`, `GrammarNote`, `ExtrasInput`, and the new `Lesson` fields in `frontend/src/app/core/models/lesson.ts`.
  - `Quiz`, `QuizQuestion`, `QuizAnswer`, `AnswerResult` and `ReadingLesson.quiz` / `grammarNote` in `frontend/src/app/core/models/reading.ts`.
  - `Stats.reading {answered, correct, rate}` in `frontend/src/app/core/models/dashboard.ts`.
  - Update the existing spec fixtures that build these objects so `npx ng test` still compiles.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: The provider signature and the lesson fields are shared by every story.

- [X] T002 Extend `backend/internal/ai/ai.go`:
  - Add `Question {Prompt string; Options []string; AnswerIndex int; ExplanationVi string}` (JSON `prompt`, `options`, `answerIndex`, `explanationVi`).
  - Add `GrammarNote {Title, BodyVi string; Examples []string}` (JSON `title`, `bodyVi`, `examples`).
  - Add `LessonExtras {Annotations []Annotation; Questions []Question; GrammarNote *GrammarNote; WritingPrompt string}` (JSON `annotations`, `questions`, `grammarNote`, `writingPrompt`).
  - Change `Provider.Annotate` to return `(LessonExtras, error)`; `Disabled` returns `LessonExtras{}` with `ErrNotConfigured`.
- [X] T003 Extend `backend/internal/lesson/model.go`:
  - Types `Question`, `GrammarNote`, `Extras {Questions []Question; GrammarNote *GrammarNote; WritingPrompt string}`.
  - `Lesson` fields `Extras`, `ExtrasEditedByAdmin bool`, `QuizVersion int`.
  - Errors `ErrQuizChanged`, `ErrNoQuiz`, `*AlreadyAnsweredError{Answer}` (with `Is(ErrAlreadyAnswered)`).
  - Change the `lesson.Repository` methods per data-model.md: `SaveAnnotations(ctx, id, revision, anns, extras)` and new `ReplaceExtras(ctx, id, extras Extras, bumpQuiz bool) error`.
  - Update `backend/internal/lesson/fake_test.go`: lesson fake stores extras and bumps `QuizVersion` like Mongo; fake AI `Annotate` returns `ai.LessonExtras` with configurable extras.
  - Update `ProcessAnnotate` in `backend/internal/lesson/service.go` to read `.Annotations` (extras saved later in T008).
- [X] T004 Update `backend/internal/storage/mongo/lessons.go` per data-model.md:
  - `lessonDoc` gets `questions`, `grammarNote`, `writingPrompt`, `extrasEditedByAdmin`, `quizVersion` (missing → empty / 0) with conversion both ways.
  - `SaveAnnotations` sets extras, `extrasEditedByAdmin: false` and `$inc quizVersion` at the revision.
  - `ReplaceContent` clears extras, resets the flag and `$inc quizVersion`.
  - `ReplaceExtras` sets extras, `extrasEditedByAdmin: true` and `$inc quizVersion` only when `bumpQuiz`.
- [X] T005 Update the Gemini client `Annotate` in `backend/internal/ai/gemini/gemini.go` and its tests in `gemini_test.go` (research R1):
  - Tests: one request; `responseSchema` is an OBJECT with `annotations` (required) plus optional `questions` (ARRAY of OBJECT `prompt`, `options` ARRAY of STRING, `answerIndex` INTEGER, `explanationVi`), `grammarNote` (OBJECT `title`, `bodyVi`, `examples`) and `writingPrompt`; the prompt asks for 3–5 questions with 4 options, a Vietnamese grammar note with examples copied from the lesson, and a writing prompt; the response decodes into `ai.LessonExtras`; existing error cases still map.
  - Implementation: response size limit 2 MB.

**Checkpoint**: `go build ./...` and `go test ./...` pass with behaviour unchanged for learners.

---

## Phase 3: User Story 3 - AI sinh câu hỏi, ngữ pháp và đề viết khi chú thích bài (Priority: P1)

**Goal**: Một lần chú thích lưu thêm câu hỏi hợp lệ, ghi chú ngữ pháp có ví dụ trong bài, đề viết; bài cũ chạy lại được.

**Independent Test**: quickstart.md bước 1.

### Tests for User Story 3 (REQUIRED) ⚠️

- [X] T006 [US3] Write `backend/internal/lesson/extras_test.go` for `CleanExtras(x ai.LessonExtras, content string) Extras` (research R2).
  - Keeps valid questions (trimmed).
  - Drops a question when:
    - it has 3 or 5 options;
    - two options repeat (case or spaces only);
    - an option is empty;
    - `answerIndex` is -1 or 4;
    - the explanation is empty;
    - the prompt is empty or duplicated.
  - Keeps at most 5 questions.
  - Grammar examples:
    - found ignoring case, extra spaces and trailing punctuation;
    - examples not in the content are dropped;
    - at most 3 are kept;
    - a note with no example left, an empty title or an empty body becomes nil.
  - The writing prompt is trimmed and cut to 500 runes.
  - Empty input gives empty `Extras`.
- [X] T007 [US3] Add to `backend/internal/lesson/service_test.go`.
  - `ProcessAnnotate`:
    - calls the AI once;
    - saves annotations plus cleaned extras;
    - bumps `QuizVersion`;
    - resets `ExtrasEditedByAdmin`;
    - broken extras still save the annotations;
    - no valid annotation still fails the job as before.
  - `Retry(job.TypeAnnotate)`:
    - works when annotation is `done` or `failed`;
    - returns `ErrAnnotationRunning` when running.
  - `Retry(job.TypeTTS)` still requires `failed`.

### Implementation for User Story 3

- [X] T008 [US3] Implement `CleanExtras` in `backend/internal/lesson/extras.go`, then call it in `ProcessAnnotate` and pass extras to `SaveAnnotations` in `backend/internal/lesson/service.go`, so T006 and the `ProcessAnnotate` part of T007 pass.
- [X] T009 [US3] Change `Retry` in `backend/internal/lesson/service.go` (annotate: allowed unless running; tts unchanged) and map `ErrAnnotationRunning` for retry in `backend/internal/lesson/handler.go`, so the rest of T007 passes. Add a handler test in `handler_test.go`: retry annotate on a done lesson → 200 `annotationStatus: running`.
- [X] T010 [US3] Expose extras in the admin lesson JSON in `backend/internal/lesson/handler.go`:
  - fields `questions` (with `answerIndex`, `explanationVi`), `grammarNote`, `writingPrompt`, `extrasEditedByAdmin`, `quizVersion`, empty arrays not null;
  - test in `handler_test.go`.
- [X] T011 [US3] Update `frontend/src/app/features/admin/lesson-detail/lesson-detail.html/.ts` and its spec:
  - **Chạy lại chú thích** is shown when annotation is `done` or `failed`.
  - If any annotation has `editedByAdmin` or `extrasEditedByAdmin` is true, the button opens `lu-confirm-dialog` listing what will be replaced ("Chú thích từ đã sửa tay", "Câu hỏi, ngữ pháp, đề viết đã sửa tay") before calling `retry('annotate')`; otherwise it calls it at once.
  - A read-only summary shows the number of questions and the grammar title until T034 adds the editor.

**Checkpoint**: chú thích một bài lưu đủ 4 phần trong 1 request; bài cũ chạy lại được.

---

## Phase 4: User Story 1 - Trả lời câu hỏi hiểu bài ở bước Đọc (Priority: P1) 🎯 MVP

**Goal**: Người học trả lời từng câu, thấy ngay đúng/sai không cần màu, tiếp tục khi vào lại; trả lời hết thì bước Đọc xong (server kiểm tra).

**Independent Test**: quickstart.md bước 3 (mục 1–8).

### Tests for User Story 1 (REQUIRED) ⚠️

- [X] T012 [P] [US1] Write `backend/internal/lesson/quiz_test.go` with an in-memory `fakeAnswers` (in `fake_test.go`, rejecting duplicate keys with `*AlreadyAnsweredError`).
  - `Reader.View(ctx, userID, id)`:
    - `Quiz` is nil without questions;
    - questions have only prompt and options;
    - `Answers` holds only this user's answers at the current `QuizVersion`;
    - another user's and older-version answers are not shown;
    - `GrammarNote` is passed through;
    - there is no writing prompt.
  - `Reader.Answer(ctx, userID, lessonID, AnswerInput{Version, QuestionIndex, Choice})`:
    - correct and wrong answers with `answerIndex` and explanation;
    - counts `answered`, `total`, `correct`;
    - a second answer to the same question gives `*AlreadyAnsweredError` with the first answer;
    - wrong version → `ErrQuizChanged`;
    - index or choice out of range → `ValidationError` (`questionIndex` / `choice`);
    - no questions → `ErrNoQuiz`;
    - unknown lesson → `ErrNotFound`;
    - two users answer the same question independently.
  - `Reader.QuizStatus(ctx, userID, lessonID)` returns (questions, answered at the current version).
- [X] T013 [P] [US1] Add to `backend/internal/lesson/reading_handler_test.go`:
  - `GET /api/lessons/{id}` returns `quiz` with no `answerIndex`/`explanationVi` in `questions`; `quiz: null` without questions.
  - `POST /api/lessons/{id}/answers`: 200 shape, 400 `validation_failed`, 400 `invalid_body`, 404, 409 `quiz_changed`, 409 `already_answered` with `answer`, 409 `no_quiz`, 401 without a session.
- [X] T014 [P] [US1] Add to the progress tests (`backend/internal/progress/study_test.go` or the file holding `CompleteStep` tests) with a fake `ReadingQuiz`:
  - completing `read` with questions not all answered → `ErrReadIncomplete`;
  - all answered (even all wrong) → done;
  - lesson without questions → done as before;
  - a `ReadingQuiz` error → error;
  - a handler test in `study_handler_test.go` maps it to 409 `read_incomplete`.

### Implementation for User Story 1

- [X] T015 [US1] Implement `backend/internal/lesson/quiz.go` so T012 passes (research R4, R5):
  - `Answer`, `AnswerInput`, `AnswerRepository`, `QuizView`, `AnswerView`;
  - `Reader.Answer`, `Reader.QuizStatus`, `Reader.Totals`.
  - Change `NewReader` to take the answers repository, and `View` in `backend/internal/lesson/reading.go` to take `userID` and fill `Quiz` / `GrammarNote`.
- [X] T016 [US1] Update `backend/internal/lesson/reading_handler.go` so T013 passes:
  - `quiz` / `grammarNote` JSON;
  - `view` reads the principal;
  - route `POST /api/lessons/{id}/answers` behind `requireAuth` + guard;
  - error mapping per contracts/reading-quiz-api.md.
- [X] T017 [US1] Create `backend/internal/storage/mongo/reading_answers.go` (`NewReadingAnswers(db)`, collection `reading_answers`):
  - `Insert`: a duplicate key error → load the existing answer and return `*AlreadyAnsweredError`.
  - `Get`, `List` sorted by `questionIndex`, `Totals` (aggregate like `DictationResults.Totals`).
  - Add the unique index `(userId, lessonId, quizVersion, questionIndex)` in `backend/internal/storage/mongo/indexes.go`.
  - Add a test in `backend/internal/storage/mongo` if the package already has DB-backed tests; otherwise a pure test of doc conversion.
- [X] T018 [US1] Add the `ReadingQuiz` port (`Status`, `Totals`) and `ErrReadIncomplete` in `backend/internal/progress` (`study.go` / `study_repository.go`), check it in `completeStep` for `StepRead`, and map 409 `read_incomplete` in `backend/internal/progress/study_handler.go`, so T014 passes.
- [X] T019 [US1] Wire everything in `backend/cmd/api/main.go`:
  - `answers := mongo.NewReadingAnswers(database)`;
  - `lesson.NewReader(lessons, dict, lessonTopics, answers)`;
  - `StudyDeps.Quiz: readingQuiz{reader}` (adapter for `Status` / `Totals`).
  - Run the full backend tests.
- [X] T020 [P] [US1] Add `answer(lessonId, input): Observable<AnswerResult>` (POST `/api/lessons/{id}/answers`) to `frontend/src/app/features/lesson/reading-api.service.ts`, with a test in `reading-api.service.spec.ts`.
- [X] T021 [US1] Write `frontend/src/app/features/lesson/reading/comprehension-quiz/comprehension-quiz.spec.ts`. Create the component with `ng g c features/lesson/reading/comprehension-quiz`.
  - Inputs `lessonId`, `quiz` (Quiz), `review` (bool). Output `allAnswered` ({correct, total}).
  - Shows "Câu 1/4" with 4 option buttons (≥ 44px via class) for the first unanswered question; answered questions above show their result; later questions are hidden.
  - Clicking an option calls `answer()` with the version, and disables all options while sending.
  - The result shows:
    - "✓ Đúng" or "✗ Sai" as text;
    - the chosen option marked "(bạn chọn)";
    - the right option marked "Đáp án đúng";
    - the explanation, inside `aria-live="polite"`.
  - The next question then appears.
  - Resumes at the first unanswered question from `quiz.answers` (reload case).
  - A network error re-enables the options and shows `role="alert"`.
  - 409 `already_answered` shows the stored answer.
  - 409 `quiz_changed` shows "Câu hỏi vừa được cập nhật" with a **Tải lại** button that emits `reload`.
  - After the last answer it shows "Đúng x/y câu" and emits `allAnswered` once.
  - With every question answered on load it shows the summary and does not emit.
- [X] T022 [US1] Implement `comprehension-quiz.ts/.html/.css` so T021 passes (research R9; result colours only as extra, text first).
- [X] T023 [US1] Extend `frontend/src/app/features/lesson/reading/reading.spec.ts`.
  - **Lesson with quiz:** no "Đã đọc xong" button and `lu-comprehension-quiz` shown after the text. In study mode, `allAnswered` → `completed` emitted. With all questions answered on load in study mode, a **Tiếp tục** button emits `completed`. In review mode nothing is emitted.
  - **Lesson without quiz:** the "Đã đọc xong" flow is unchanged.
  - **Reload output:** the quiz `reload` output reloads the lesson.
- [X] T024 [US1] Implement the changes in `frontend/src/app/features/lesson/reading/reading.ts/.html/.css` so T023 passes.
- [X] T025 [US1] Show the server message when completing `read` fails with `read_incomplete` in `frontend/src/app/features/lesson/today/today.ts` (already shows `body.message`; add a spec case in `today.spec.ts`).

**Checkpoint**: MVP — người học trả lời câu hỏi, bước Đọc hoàn thành theo câu trả lời.

---

## Phase 5: User Story 2 - Ghi chú ngữ pháp trong bước Đọc (Priority: P1)

**Goal**: Mục Ngữ pháp thu gọn được trong bước Đọc.

**Independent Test**: quickstart.md bước 3 (mục 1).

- [X] T026 [P] [US2] Write `frontend/src/app/features/lesson/reading/grammar-note/grammar-note.spec.ts`. Create the component with `ng g c features/lesson/reading/grammar-note`.
  - Input `note`.
  - Renders a `<details open>` whose `<summary>` reads "Ngữ pháp: {title}", with the body paragraphs (split on blank lines) and the examples as a list.
  - Toggling `open` collapses it.
- [X] T027 [US2] Implement `grammar-note.ts/.html/.css` so T026 passes; summary ≥ 44px, focus outline.
- [X] T028 [US2] Add the grammar note to `frontend/src/app/features/lesson/reading/reading.html`, between the text and the quiz, only when `lesson.grammarNote` is set (also in review mode); add the spec case in `reading.spec.ts` (shown / absent).

**Checkpoint**: bước Đọc có ngữ pháp và câu hỏi.

---

## Phase 6: User Story 4 - Quản trị viên sửa câu hỏi, ngữ pháp và đề viết (Priority: P2)

**Goal**: Sửa và lưu extras có kiểm tra theo ô; đổi câu hỏi tạo phiên bản mới.

**Independent Test**: quickstart.md bước 2.

### Tests for User Story 4 (REQUIRED) ⚠️

- [X] T029 [US4] Add tests to `backend/internal/lesson/extras_test.go` for `ValidateExtras(in ExtrasInput, content string) (Extras, error)`.
  - Every field error of contracts/reading-quiz-api.md, with exact keys like `questions.1.options.2` and `grammarNote.examples.0`.
  - 0 questions allowed; 6 questions → `questions`.
  - `grammarNote: nil` allowed.
- [X] T030 [US4] Add tests to `backend/internal/lesson/service_test.go` and `handler_test.go`.
  - `UpdateExtras` saves, sets `ExtrasEditedByAdmin`, bumps `QuizVersion` only when questions changed (same questions with a new grammar note → same version), and returns `ErrAnnotationRunning` while running.
  - `PUT /api/admin/lessons/{id}/extras`: 200 with the new lesson, 400 field errors, 404, 409, 403 for a learner.

### Implementation for User Story 4

- [X] T031 [US4] Implement `ExtrasInput`, `ValidateExtras` and `sameQuestions` in `backend/internal/lesson/extras.go`; implement `Service.UpdateExtras` in `backend/internal/lesson/service.go`; add the route `PUT /api/admin/lessons/{id}/extras` in `backend/internal/lesson/handler.go`. T029–T030 pass.
- [X] T032 [P] [US4] Add `updateExtras(id, input): Observable<Lesson>` to `frontend/src/app/features/admin/admin-api.service.ts`, with a test in its spec.
- [X] T033 [US4] Write `frontend/src/app/features/admin/lesson-extras/lesson-extras.spec.ts`. Create the component with `ng g c features/admin/lesson-extras`.
  - Input `lesson`. Output `saved` (Lesson).
  - Form prefilled from the lesson.
  - Questions:
    - **Thêm câu hỏi** adds an empty question and is disabled at 5;
    - **Xoá** removes one;
    - the "Đáp án đúng" radio changes `answerIndex`.
  - Grammar note:
    - a checkbox "Có ghi chú ngữ pháp" toggles the group;
    - examples can be added (max 3) and removed.
  - Writing prompt field.
  - **Lưu** sends `ExtrasInput`.
  - 400 field errors show under the matching inputs with `aria-invalid`.
  - Success shows "Đã lưu" and the "Đã sửa tay" tag; `annotationStatus === 'running'` disables saving with a note.
- [X] T034 [US4] Implement `lesson-extras.ts/.html/.css` (reactive `FormArray`, one column at 360px, inputs ≥ 44px) so T033 passes. Replace the read-only summary from T011 with `<lu-lesson-extras>` in `frontend/src/app/features/admin/lesson-detail/lesson-detail.html`, updating the lesson on `saved`; update `lesson-detail.spec.ts`.

**Checkpoint**: quản trị viên sửa được mọi phần mới.

---

## Phase 7: User Story 5 - Xem lại và thống kê tỷ lệ đúng (Priority: P3)

**Goal**: Xem lại câu trả lời cũ ở trang Bài học; thống kê và file xuất có tỷ lệ hiểu bài.

**Independent Test**: quickstart.md bước 4.

### Tests for User Story 5 (REQUIRED) ⚠️

- [X] T035 [P] [US5] Add stats tests to `backend/internal/progress/dashboard_service_test.go` and `dashboard_handler_test.go`:
  - `Stats` includes `Reading {Answered, Correct}` from `ReadingQuiz.Totals`;
  - `ReadingRate` nil when 0 answered, 0.75 for 9/12;
  - `/api/stats` JSON has `reading: {answered, correct, rate}` with `rate: null` when empty.
- [X] T036 [P] [US5] Add to `backend/internal/export/service_test.go`:
  - `readingAnswers` contains only the user's answers without `userId`;
  - empty → `[]`;
  - the collection is allowed.

### Implementation for User Story 5

- [X] T037 [US5] Implement the reading totals in `backend/internal/progress/stats.go` and `dashboard_handler.go` so T035 passes.
- [X] T038 [US5] Add `CollReadingAnswers` and `Export.ReadingAnswers` (`readingAnswers`) in `backend/internal/export/model.go` and `service.go`, and allow the collection in `backend/internal/storage/mongo/export.go`, so T036 passes.
- [X] T039 [US5] Show "Hiểu bài" in `frontend/src/app/features/stats/stats.html/.ts`:
  - rate as a whole percentage with "(n câu)", or "—" with "Trả lời câu hỏi ở bước Đọc để xem tỷ lệ";
  - spec cases in `stats.spec.ts`.
- [X] T040 [US5] Add a spec case in `frontend/src/app/features/lesson/reading/reading.spec.ts`: in review mode (`?review=1`) a fully answered quiz shows every result and no option can be clicked.

**Checkpoint**: tất cả user story xong.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T041 [P] Update `README.md`:
  - "Soạn bài học": câu hỏi, ngữ pháp, đề viết sinh cùng chú thích (1 request); sửa ở chi tiết bài; Chạy lại chú thích cho bài cũ và cảnh báo sửa tay.
  - "Bước Đọc và từ điển": trả lời câu hỏi để xong bước Đọc; bài chưa có câu hỏi dùng "Đã đọc xong"; mục Ngữ pháp.
  - "Màn hình chính và thống kê": tỷ lệ hiểu bài.
  - Export list: `readingAnswers`.
- [X] T042 Run lint (frontend `npm run lint`, backend golangci-lint via Docker) with zero issues.
- [X] T043 Run all tests (`CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`, `npx ng test --watch=false`) with zero failures; `npx ng build` without warnings.
- [X] T044 Rebuild Docker and smoke test with curl (quickstart.md bước 2–4) using a lesson whose extras are set through `PUT /extras`, since AI may be off:
  - learner GET hides answers;
  - answer right/wrong;
  - repeated answer → 409;
  - complete read before all answered → 409, after → done;
  - stats `reading`;
  - export `readingAnswers`;
  - another user sees nothing.
- [ ] T045 Manual check (needs a Gemini key): quickstart.md bước 1 and 3–5 in the browser (AI quality, 1 request in logs, grayscale, 360px, keyboard).
- [ ] T046 Tick the 4 F15 acceptance criteria in `docs/phases/giai-doan-2.md` once T043–T045 pass.

---

## Dependencies & Execution Order

- **Setup** (T001) ∥ **Foundational** (T002 → T003 → T004, T005).
- Then **US3** → **US1** → **US2** → **US4** → **US5** → **Polish**.
- **US1** needs only Foundational for the backend. It can start in parallel with US3 (fake extras in tests).
- **US2** frontend needs T001 only and can run in parallel with US1 frontend.
- **US4** backend `ValidateExtras` builds on `CleanExtras` helpers (T008).
- **US5** needs US1 (answers repository, `ReadingQuiz` port).

### Parallel Opportunities

- T001 ∥ T002–T005.
- T012 ∥ T013 ∥ T014 (different test files).
- T020, T026, T032 (frontend services and components) ∥ backend tasks.
- T035 ∥ T036.

---

## Parallel Example: User Story 1

```bash
Task: "quiz tests in backend/internal/lesson/quiz_test.go"
Task: "progress read gate tests in backend/internal/progress/study_test.go"
Task: "comprehension-quiz component + spec in frontend/src/app/features/lesson/reading/comprehension-quiz/"
```

---

## Implementation Strategy

1. Setup + Foundational → US3 (dữ liệu từ AI, chạy lại chú thích).
2. US1 (quiz + hoàn thành bước Đọc). **STOP and VALIDATE** with quickstart bước 3, using `PUT /extras` if AI is off.
3. US2 (ngữ pháp) → US4 (quản trị sửa) → US5 (xem lại, thống kê, xuất dữ liệu).
4. Polish: README, lint, test, Docker, kiểm tra thủ công, tick tiêu chí.

---

## Notes

- `progress` never imports `lesson`; `main.go` adapts `lesson.Reader` to `progress.ReadingQuiz`.
- Never send `answerIndex` or `explanationVi` of unanswered questions to learners.
- Writing prompt is admin-only until F8.
- Only commit when the user asks.

---

## Ghi chú triển khai

- Test backend đặt trong file mới thay vì nối vào file có sẵn: `extras_test.go`, `extras_service_test.go`, `extras_handler_test.go`,
  `extras_validate_test.go`, `quiz_test.go`, `quiz_handler_test.go` (lesson), `reading_quiz_test.go` (progress).
- Hai test cũ được sửa vì quy tắc đổi có chủ ý:
  - `TestJobFailedAndRetry`: chạy lại chú thích khi đang chạy giờ trả `ErrAnnotationRunning`, không còn `ErrNotFailed`.
  - `TestStatsEndpoint`: JSON `/api/stats` có thêm `reading`.
- T011: bỏ phần "tóm tắt chỉ đọc" tạm thời vì T034 thay luôn bằng form sửa.
- Radio "Đáp án đúng" trong `lesson-extras` không dùng `formControlName`: Angular bắt `name` của radio phải trùng tên control
  (NG01202), mà mỗi câu hỏi cần một nhóm radio riêng.
- Trong quiz, chữ của lựa chọn và ghi chú ("(bạn chọn)", "— Đáp án đúng") có khoảng cách thật trong văn bản để trình đọc màn hình
  không đọc dính.
- `reading.spec.ts` và `listening.spec.ts` có thêm route giả `forbidden`: test "says a locked lesson opens later" (có từ trước) làm
  interceptor điều hướng tới `/forbidden` và sinh lỗi "Unhandled Rejection" trong Vitest.
- Smoke test Docker (T044), AI tắt nên đặt câu hỏi qua `PUT /extras` cho bài hôm nay của `hoc` ("F14 Z gia dinh"):
  - extras sai → 400 theo ô (`questions.0.options.1`, `grammarNote.examples.0`); người học PUT → 403; lưu → `quizVersion` 1;
  - GET của người học không có `answerIndex`, `explanationVi`, đề viết;
  - trả lời sai → 200 kèm đáp án; trả lời lại → 409 `already_answered` kèm câu trả lời cũ; phiên bản cũ → 409 `quiz_changed`;
    lựa chọn 7 → 400;
  - ôn 3 thẻ để qua bước Ôn; hoàn thành Đọc khi mới trả lời 1/2 → 409 `read_incomplete`; trả lời hết (sai cả 2) → bước Đọc xong;
  - thống kê `hoc` 2 câu / 0 đúng, admin 1/1 (riêng từng người); file xuất `hoc` có 2 `readingAnswers`, không có `userId`;
  - đổi đáp án → `quizVersion` 2, GET không còn hiện câu trả lời cũ; chạy lại chú thích (đang lỗi) → 202; log backend 0 lỗi.
  Dữ liệu thử để lại: bài "F14 Z gia dinh" có 2 câu hỏi và ghi chú ngữ pháp (`quizVersion` 3); bài hôm nay của `hoc` đã qua bước Ôn
  và Đọc.
- Còn lại cho người dùng: T045 (cần `GEMINI_API_KEY`: chất lượng câu hỏi thật, 1 request trong log, chế độ xám, 360px, bàn phím) và
  T046 (tick 4 tiêu chí F15 sau khi T045 đạt).
