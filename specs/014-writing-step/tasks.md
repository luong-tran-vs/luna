---

description: "Task list for F8 – Bước Viết"
---

# Tasks: Bước Viết (F8)

**Input**: Design documents from `specs/014-writing-step/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/writing-api.md, quickstart.md

**Tests**: REQUIRED by constitution principle VI and khối 2:
- Writing service: 5–400 words, submit twice → 409, default prompt, `CanWrite`, owner-only.
- Grade job with a fake Provider: success, unusable result, quota, not configured.
- Regrade only when failed; unseen/seen.
- Progress: 4 steps, write gate, old completed lessons stay done.
- Gemini `GradeWriting`.
- Mongo jobs `targetId`.
- Export.
- Vitest: word-diff, draft autosave, submit, grade states, notifier/toast.

Real AI quality, 360px and keyboard are checked by hand (quickstart.md).

**Organization**: Tasks are grouped by user story (US1–US5 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Create `frontend/src/app/core/models/writing.ts` per data-model.md:
  - `WritingStatus`, `GradeStatus`, `Criterion`, `Grade`, `Writing`, `LessonWriting`, `WritingSummary`, `UnseenCount`.
  - `MIN_WRITING_WORDS = 5`, `MAX_WRITING_WORDS = 400`.
  - `SUGGESTED_WORDS: Record<Level, {min, max}>`: A1 30–60, A2 50–80, B1 80–120, B2 120–180, C1/C2 150–250.
  - `CRITERIA_LABELS`: task "Hoàn thành yêu cầu", grammar "Ngữ pháp", vocabulary "Từ vựng", coherence "Mạch lạc".
  - Spec `writing.spec.ts`: every level has a suggestion; labels cover the 4 names.
- [X] T002 [P] Create `frontend/src/app/shared/utils/word-diff.ts` with `wordDiff(original, corrected): DiffPart[]` (`{kind: 'same'|'add'|'del', text}`), per research R7:
  - Tokens are words with their trailing whitespace.
  - LCS dynamic programming; adjacent parts of the same kind are merged.
  - Spec `word-diff.spec.ts` covers: identical text (one `same`), an added word, a removed word, a replaced word (`del` then `add`), punctuation change, empty original, empty corrected; joining all `same`+`del` gives the original and all `same`+`add` gives the corrected.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: The 4th step, the job target and the writing package skeleton are shared by every story.

- [X] T003 Add `StepWrite = "write"` at the end of `progress.Steps` in `backend/internal/progress/study_model.go`.
  - Add `ErrWriteIncomplete`.
  - Add the `Writings` port (`Submitted`, `Stats`) in `backend/internal/progress/study_repository.go`.
  - Add the `Writings` field to `StudyDeps` (nil means writing is never required, so existing tests compile).
  - Fix every test that assumes 3 steps (search `StepListen`, `"listen"`, `currentStep":"done"` in `backend/internal/progress/*_test.go`).
  - `go test ./internal/progress/` passes.
- [X] T004 Add `job.TypeGrade = "grade"` and `Job.TargetID string` in `backend/internal/job/model.go`.
  - In `backend/internal/storage/mongo/jobs.go`:
    - the doc has `targetId` (ObjectID, omitempty);
    - `lessonId` is omitempty;
    - `Enqueue` only parses `LessonID` / `TargetID` when set;
    - conversion works both ways.
  - Add a pure conversion test in `backend/internal/storage/mongo/jobs_test.go` (grade job without lesson id round-trips its target id).
- [X] T005 Add to `backend/internal/ai/ai.go`:
  - `GradeRequest`, `Criterion`, `Grade` per data-model.md;
  - `Provider.GradeWriting`;
  - `Disabled.GradeWriting` returning `ErrNotConfigured`.
  - Add `GradeWriting` to the fake AI in `backend/internal/lesson/fake_test.go` (returns zero value) so the lesson tests compile.
- [X] T006 Write Gemini tests in `backend/internal/ai/gemini/gemini_test.go`, then implement `GradeWriting` in `backend/internal/ai/gemini/gemini.go` (research R5).
  - Tests:
    - one POST request;
    - `responseSchema` is an OBJECT with required `task`, `grammar`, `vocabulary`, `coherence` (each OBJECT `{score INTEGER, commentVi STRING}`), `overallVi`, `correctedText`;
    - temperature 0.3;
    - the prompt contains the level, the writing prompt, the lesson text, the learner text and "1 to 5";
    - the response decodes into `ai.Grade`;
    - 429 → `ErrQuota`;
    - empty key → `ErrNotConfigured` without a request.
  - Implementation: reuse `c.generate` with op `grade` and log attribute `words` (count only).
- [X] T007 Create the skeleton of `backend/internal/writing/`:
  - `model.go`: types, statuses, `DefaultPrompt = "Tóm tắt bài bằng 3–5 câu."`, `MinWords`, `MaxWords`, errors `ErrNotFound`, `ErrSubmitted`, `ErrLocked`, `ErrNotFailed`, `ErrUnusableGrade`, `*ValidationError`, `Average(grade) *float64` rounded to 1 decimal.
  - `repository.go`: `Repository`, `Lessons`, `Steps` ports per data-model.md.
  - `fake_test.go`: in-memory repository enforcing the unique (user, lesson) and the conditional submit; fake lessons, steps, jobs and AI.

**Checkpoint**: `go build ./...` and `go test ./...` pass; the app still works with 4 steps defined (writing not yet reachable).

---

## Phase 3: User Story 1 - Viết và nộp bài ở bước Viết (Priority: P1) 🎯 MVP

**Goal**: Bước Viết sau Nghe; nháp tự lưu; nộp 5–400 từ hoàn thành bài ngay.

**Independent Test**: quickstart.md bước 1.

### Tests for User Story 1 (REQUIRED) ⚠️

- [X] T008 [P] [US1] Write `backend/internal/writing/service_test.go`.
  - `Get` returns the lesson prompt (or `DefaultPrompt` when the lesson has none), the level, `canWrite` from `Steps`, and the user's writing or nil.
  - `SaveDraft`:
    - stores and updates text;
    - empty text allowed;
    - over 4000 chars → `ValidationError{text}`;
    - `CanWrite` false → `ErrLocked`;
    - after submit → `ErrSubmitted`.
  - `Submit`:
    - 4 words → "Bài viết cần ít nhất 5 từ";
    - 401 words → "Bài viết tối đa 400 từ";
    - 5 and 400 accepted;
    - stores prompt, lesson title and revision at submit time;
    - grade pending and not seen;
    - one `grade` job enqueued with `TargetID`;
    - a second submit → `ErrSubmitted` and no second job;
    - `CanWrite` false → `ErrLocked`.
  - Another user's writing is never returned.
- [X] T009 [P] [US1] Add to the progress tests (`backend/internal/progress/writing_step_test.go`) with a fake `Writings`.
  - After listen the current step is write and the day is not completed.
  - Completing write before submit → `ErrWriteIncomplete`; after submit → done, goal +1, streak counted.
  - A lesson completed before F8 (progress with `CompletedAt`, no `write` in Done) stays done and the day stays done.
  - `CanWrite` is true only for today's lesson at the write step.
  - Handler test: `write_incomplete` → 409.
- [X] T010 [P] [US1] Write `backend/internal/writing/handler_test.go` for `GET/PUT /api/lessons/{id}/writing` and `POST /api/lessons/{id}/writing/submit` per contracts/writing-api.md:
  - 200 shapes;
  - 400 `validation_failed`;
  - 409 `already_submitted`;
  - 409 `write_locked`;
  - 404;
  - 401;
  - routes go through the given guard.

### Implementation for User Story 1

- [X] T011 [US1] Implement `Get`, `SaveDraft`, `Submit` in `backend/internal/writing/service.go` (word count with `strings.Fields`; research R2–R3) so T008 passes.
- [X] T012 [US1] Implement the write gate in `completeStep` and `CanWrite` in `backend/internal/progress/study.go`; map 409 `write_incomplete` in `backend/internal/progress/study_handler.go`, so T009 passes.
- [X] T013 [US1] Implement the lesson routes in `backend/internal/writing/handler.go` (`NewHandler(svc, log)`, `Register(mux, requireAuth, guard)`), so T010 passes.
- [X] T014 [US1] Create `backend/internal/storage/mongo/writings.go`:
  - `NewWritings(db)`;
  - `GetByLesson`, `Get`, `SaveDraft` (upsert with `status != submitted`, duplicate-key safe), `Submit` (conditional update), `SetGrade`, `SetPending`, `MarkSeen`, `List`, `Unseen`, `Stats`;
  - doc conversion.
  - Add the 3 indexes in `backend/internal/storage/mongo/indexes.go`.
  - Add a pure doc conversion test in `writings_test.go`.
- [X] T015 [US1] Wire in `backend/cmd/api/main.go`:
  - `writingSvc := writing.NewService(...)` with adapters `writingLessons{lessons}` (Info: title, level, content, writing prompt, revision) and `writingSteps{studySvc}`;
  - `StudyDeps.Writings: writingProgress{writingSvc}`;
  - `writing.NewHandler(...).Register(mux, requireAuth, studyHandler.Guard)`.
  - Mind the construction order (study service and writing service reference each other only through adapters with a late-bound pointer).
  - Full backend tests pass.
- [X] T016 [P] [US1] Create `frontend/src/app/core/services/writing-api.service.ts` (`ng g s core/services/writing-api --type=service`) with all calls of the contract (`lessonWriting`, `saveDraft`, `submit`, `list`, `get`, `seen`, `regrade`, `unseenCount`) and a spec with URLs and bodies.
- [X] T017 [US1] Add `'write'` to `Step` / `STEPS` in `frontend/src/app/core/models/study.ts`.
  - Update `frontend/src/app/shared/components/step-indicator/` (label "Viết", 4 columns) and its spec.
  - Fix fixtures in specs that list steps (`steps: {review, read, listen}` → add `write`).
- [X] T018 [US1] Write `frontend/src/app/features/lesson/writing/writing.spec.ts`. Create the component with `ng g c features/lesson/writing`.
  - Inputs `lessonId`, `mode` ('study' | 'review'). Output `completed`.
  - Loads `GET /writing`; shows the prompt, the level suggestion ("Gợi ý: 30–60 từ") and the live word count.
  - Typing saves a draft once 1 s after the last input (fake timers: two quick inputs → one PUT with the last text); shows "Đang lưu…" then "Đã lưu nháp"; a failed save shows "Chưa lưu được nháp".
  - **Nộp** is disabled below 5 or above 400 words, with a hint.
  - Submit:
    - sends the text, then emits `completed` once;
    - shows "Đã nộp, AI đang chấm" with a link to `/writings/{id}`;
    - 400 shows the field message.
  - An already submitted writing or `canWrite: false` shows the text read-only and no Nộp button; in study mode with a submitted writing and the step not done yet, a **Tiếp tục** button emits `completed`.
- [X] T019 [US1] Implement `writing.ts/.html/.css` so T018 passes (textarea ≥ 8 rows, one column, buttons ≥ 44px, status in `aria-live="polite"`).
- [X] T020 [US1] Add the write step to `frontend/src/app/features/lesson/today/today.html/.ts` (`@case ('write')` with `<lu-writing [lessonId]="l.id" mode="study" (completed)="complete('write')" />`) and a spec case in `today.spec.ts` (stub component; completing calls `complete('write')`; `write_incomplete` message shown).
- [X] T021 [US1] Add the route `lessons/:id/write` (Writing, review) in `frontend/src/app/features/lesson/lesson.routes.ts`.

**Checkpoint**: MVP — bài hôm nay có 4 bước, nộp bài viết hoàn thành bài.

---

## Phase 4: User Story 2 - Nhận kết quả chấm và xem nhận xét (Priority: P1)

**Goal**: Chấm nền, thông báo trong app, trang chi tiết có 4 tiêu chí và phần so sánh.

**Independent Test**: quickstart.md bước 2.

### Tests for User Story 2 (REQUIRED) ⚠️

- [X] T022 [P] [US2] Write `backend/internal/writing/grade_test.go`.
  - `ProcessGrade` with a fake AI:
    - success saves 4 criteria in order with names, comments, overall and corrected text, `done`, `seen: false`, `gradedAt`;
    - passes level, lesson text, prompt and text to the AI;
    - a score of 0 or 6, an empty comment or empty corrected text → `ErrUnusableGrade` (not permanent);
    - `ErrNotConfigured` / `ErrInvalidKey` → permanent;
    - a missing writing → permanent;
    - a writing not pending → nil, no AI call.
  - `CheckGrade` trims and caps long texts.
  - `Average` of 4,3,4,4 → 3.8.
- [X] T023 [P] [US2] Add to `backend/internal/writing/service_test.go`:
  - `List` (newest first, submitted only, own only, average);
  - `Detail` (own only, else `ErrNotFound`);
  - `MarkSeen`;
  - `Unseen` (counts done/failed with `seen: false`, pending count, latest).
- [X] T024 [P] [US2] Add to `backend/internal/writing/handler_test.go`: `GET /api/writings`, `GET /api/writings/{id}` (404 for another user), `POST /api/writings/{id}/seen` (204), `GET /api/writings/unseen-count`.

### Implementation for User Story 2

- [X] T025 [US2] Implement `ProcessGrade`, `CheckGrade` in `backend/internal/writing/grade.go` and `List`, `Detail`, `MarkSeen`, `Unseen` in `service.go`, so T022–T023 pass.
- [X] T026 [US2] Implement the `/api/writings*` read routes in `backend/internal/writing/handler.go` so T024 passes.
- [X] T027 [US2] Wire the grade handler in `backend/cmd/api/main.go`:
  - `job.TypeGrade: writingSvc.ProcessGrade` in the worker map;
  - an `onFailed` function that sends grade jobs to `writingSvc.JobFailed` and others to `lessonSvc.JobFailed`.
- [X] T028 [US2] Write `frontend/src/app/core/services/writing-notifier.service.spec.ts`, then implement `writing-notifier.service.ts` (research R6).
  - `unseen` and `latest` signals.
  - `refresh()` loads `unseen-count`.
  - `submitted()` starts polling every 30 s while `pending > 0` (fake timers: stops when pending reaches 0).
  - Emits a toast message when `unseen` grows, with "Bài viết đã có kết quả" or "Chấm bài viết bị lỗi" and the writing id.
  - `markSeen(id)` calls `seen` and refreshes.
  - Refreshes on login and stops polling on logout (inject `AuthService`).
- [X] T029 [US2] Create `frontend/src/app/shared/components/toast/` (`ng g c shared/components/toast`) with a spec.
  - Shows the notifier's current toast in a `role="status"` `aria-live="polite"` region, with the message, a **Xem** link to `/writings/{id}` and a close button.
  - Hides after 8 s (fake timers) or on close.
  - Fixed at the bottom with `max-width: calc(100vw - 32px)`.
  - Add `<lu-toast />` to `frontend/src/app/app.html`.
- [X] T030 [US2] Update `frontend/src/app/shared/components/app-header/` and its spec:
  - link "Bài viết" (`/writings`);
  - a badge with the unseen count when > 0;
  - `aria-label="Bài viết, 2 kết quả mới"`;
  - the badge disappears at 0.
- [X] T031 [US2] Write `frontend/src/app/features/writings/writing-detail/writing-detail.spec.ts`. Create the component with `ng g c features/writings/writing-detail`.
  - Pending shows "Đang chấm" and reloads when the notifier's latest result is this writing.
  - Done:
    - shows 4 criteria with labels and "4/5";
    - average "3,8";
    - overall comment, original text, corrected text;
    - the diff with `<ins>` / `<del>` containing hidden "thêm: " / "bớt: " labels;
    - the legend "Gạch chân: thêm · Gạch ngang: bớt";
    - opening a done or failed unseen writing calls `seen`.
  - 404 shows "Không tìm thấy bài viết".
- [X] T032 [US2] Implement `writing-detail.ts/.html/.css` using `wordDiff`, so T031 passes. Add the routes `writings/:id` in `frontend/src/app/app.routes.ts` (auth guard, lazy).
- [X] T033 [US2] In `frontend/src/app/features/lesson/writing/writing.ts`, call `notifier.submitted()` after a successful submit, and show the grade summary (average or status) with a link for a submitted writing; extend `writing.spec.ts`.

**Checkpoint**: nộp → chấm nền → thông báo → xem nhận xét.

---

## Phase 5: User Story 3 - Chấm lỗi thì chấm lại (Priority: P2)

**Goal**: Lỗi chấm giữ bài viết, có lý do và Chấm lại.

**Independent Test**: quickstart.md bước 3.

- [X] T034 [P] [US3] Add to `backend/internal/writing/grade_test.go` and `service_test.go`:
  - `JobFailed` stores `failed` with the right Vietnamese reason for `ErrQuota` ("AI hết lượt, vui lòng chấm lại sau"), `ErrNotConfigured`, `ErrInvalidKey`, `ErrUnusableGrade` and other errors, with `seen: false`.
  - `Regrade`:
    - only when failed (`ErrNotFailed` for pending/done);
    - sets pending and seen;
    - enqueues one job;
    - other user → `ErrNotFound`.
  - Handler: `POST /api/writings/{id}/regrade` → 202 or 409 `not_failed`.
- [X] T035 [US3] Implement `JobFailed`, `failureMessage`, `Regrade` in `backend/internal/writing/grade.go` / `service.go` and the regrade route in `handler.go`, so T034 passes.
- [X] T036 [US3] Extend `writing-detail.spec.ts` and implement:
  - a failed writing shows "Chấm lỗi: {error}" and **Chấm lại**;
  - clicking calls `regrade`, shows "Đang chấm" and calls `notifier.submitted()`;
  - a 409 shows the message.

**Checkpoint**: chấm lỗi không mất bài, chấm lại được.

---

## Phase 6: User Story 4 - Trang Bài viết (Priority: P2)

**Goal**: Danh sách bài viết; xem lại từ trang Bài học.

**Independent Test**: quickstart.md bước 4.

- [X] T037 [P] [US4] Write `frontend/src/app/features/writings/writing-list/writing-list.spec.ts`. Create the component with `ng g c features/writings/writing-list`.
  - Rows newest first with date (vi-VN), lesson title, and average ("3,8 / 5"), "Đang chấm" or "Chấm lỗi".
  - Unseen results are marked "Mới".
  - Each row links to `/writings/{id}`.
  - The empty state reads "Chưa có bài viết. Học bài hôm nay để viết bài đầu tiên." with a link to `/today`.
- [X] T038 [US4] Implement `writing-list.ts/.html/.css` and the route `writings` in `frontend/src/app/app.routes.ts`, so T037 passes.
- [X] T039 [US4] Add a **Bài viết** link (`/lessons/{id}/write`) for completed lessons in `frontend/src/app/features/lesson/my-lessons/my-lessons.html`, and a spec case; extend `writing.spec.ts`: review mode shows the submitted writing and result summary, or "Bài này chưa có bài viết." when there is none, and never an editor.

**Checkpoint**: tất cả nhánh xem lại có.

---

## Phase 7: User Story 5 - Màn hình chính, thống kê và xuất dữ liệu (Priority: P3)

**Goal**: Thanh Viết, thống kê bài viết, xuất dữ liệu.

**Independent Test**: quickstart.md bước 5.

- [X] T040 [P] [US5] Add tests to the progress package (`dashboard_service_test.go`, `dashboard_handler_test.go`, `writing_step_test.go`):
  - `Skills.Write` counts goal lessons with the write step done;
  - `StatsView.Writing {Submitted, AverageScore}` from the `Writings` port (nil average → `null`);
  - `lessons.write` in stats JSON;
  - update exact-JSON expectations.
- [X] T041 [P] [US5] Add to `backend/internal/export/service_test.go`: `writings` has only the user's writings (drafts included) without `userId`; empty → `[]`.
- [X] T042 [US5] Implement the counts:
  - `Write` in `SkillCounts` / `StepCounts` (`backend/internal/progress/dashboard.go`, `study_model.go`; Mongo `$steps.write` in `backend/internal/storage/mongo/study.go`);
  - the stats writing totals (`stats.go`, `dashboard_handler.go`);
  - `writing.Service.Stats` (aggregate in Mongo `writings.go`) and the `writingProgress` adapter in `main.go`.
  - T040 passes.
- [X] T043 [US5] Add `CollWritings` / `Export.Writings` in `backend/internal/export/model.go` and `service.go`, so T041 passes.
- [X] T044 [US5] Update the frontend:
  - `core/models/dashboard.ts` (`skills.write`, `stats.lessons.write`, `stats.writing`);
  - `shared/components/progress-bar` tone `write` (`--color-skill-write`);
  - `features/home/home.html` Viết bar;
  - `features/stats` "Bài viết" card ("n bài viết", "Điểm trung bình 3,8" or "—") and "Viết: n bài" in Bài học.
  - Update the specs of home, stats and progress-bar.

**Checkpoint**: tất cả user story xong.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T045 [P] Update `README.md`:
  - "Một ngày học" (4 bước);
  - new section "Bước Viết và bài viết": đề, nháp, nộp, chấm nền, thông báo, Chấm lại, trang Bài viết;
  - "Màn hình chính và thống kê" (thanh Viết, bài viết);
  - export list (`writings`).
- [X] T046 Run lint (frontend `npm run lint`, backend golangci-lint via Docker) with zero issues.
- [X] T047 Run all tests (`CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`, `npx ng test --watch=false`) with zero failures and no unhandled errors; `npx ng build` without warnings.
- [X] T048 Rebuild Docker and smoke test with curl (AI off):
  - reach the write step;
  - draft save / reload;
  - 4 words → 400;
  - submit → `write/complete` done and day completed;
  - second submit → 409;
  - grade fails with "AI chưa được cấu hình";
  - `unseen-count` = 1;
  - `seen` → 0;
  - regrade → pending;
  - another user → 404;
  - stats `writing`;
  - export `writings`.
- [ ] T049 Manual check (needs a Gemini key): quickstart.md bước 2 and 6 (real grading, toast within a minute, grayscale diff, 360px, keyboard).
- [ ] T050 Tick the 5 F8 acceptance criteria in `docs/phases/giai-doan-2.md` once T047–T049 pass.

---

## Dependencies & Execution Order

- **Setup** (T001, T002) ∥ **Foundational** (T003 → T012; T004; T005 → T006; T007 → US1).
- Then **US1** → **US2** → **US3** → **US4** → **US5** → **Polish**.
- **US2** needs the US1 submit path (pending writings).
- **US3** needs US2 (`ProcessGrade`, detail page).
- **US4** needs the API service (T016) and the detail route.
- **US5** needs only Foundational and the writing repository (T014).
- Within each story: tests → implementation; backend ∥ frontend.

### Parallel Opportunities

- T001 ∥ T002 ∥ T003 ∥ T004 ∥ T005.
- T008 ∥ T009 ∥ T010.
- T016 ∥ T017 ∥ T018 (frontend) with T011–T015 (backend).
- T022 ∥ T023 ∥ T024.
- T028–T030 (notifier, toast, header) ∥ T025–T027.
- T037 ∥ T034.
- T040 ∥ T041.

---

## Parallel Example: User Story 1

```bash
Task: "writing service tests in backend/internal/writing/service_test.go"
Task: "progress write step tests in backend/internal/progress/writing_step_test.go"
Task: "lu-writing component + spec in frontend/src/app/features/lesson/writing/"
```

---

## Implementation Strategy

1. Setup + Foundational. Then US1 (bước Viết, nháp, nộp). **STOP and VALIDATE**: bài hôm nay xong khi nộp, quickstart bước 1.
2. US2: chấm nền, thông báo, chi tiết.
3. US3: Chấm lại.
4. US4: danh sách bài viết, xem lại.
5. US5: số liệu.
6. Polish: README, lint, test, Docker, kiểm tra thủ công, tick tiêu chí.

---

## Notes

- `writing` never imports `progress` or `lesson`; `main.go` adapts them.
- Grade jobs carry `TargetID` and no `LessonID`; `lessonSvc.JobFailed` must never receive them.
- Never return a writing to anyone but its author.
- Only commit when the user asks.

---

## Ghi chú triển khai

- Test progress cũ viết cho 3 bước:
  - Helper `complete(...)` trong `study_test.go`: khi bước cuối là Nghe thì tự nộp bài viết và hoàn thành Viết, nên các test "học xong bài" vẫn đúng nghĩa.
  - Các test dùng `"write"` làm "bước không hợp lệ" đổi sang `"speak"`.
  - Các kỳ vọng JSON hoặc số bước (3 → 4, `skills`, `lessons`, `steps`) cập nhật theo bước mới.
  - Test step-indicator / home / today ở frontend cập nhật tương tự.
- `job.Job.TargetID`: job `grade` không có `lessonId` (Mongo `omitempty`), nên `DeleteForLesson` không xoá nó. `onFailed` trong `main.go` chia theo loại job.
- `writing.Service.canWrite` kiểm "đã nộp" trước "đang ở bước Viết", để nộp lại sau khi bài đã xong vẫn báo 409 `already_submitted`, không phải `write_locked`.
- Notifier được làm cùng US1 (T028), vì component Viết gọi `notifier.submitted()` sau khi nộp. T033 vì vậy nằm sẵn trong T018–T019.
- Trang chi tiết: tiêu chí có dấu ":" ẩn và chữ "điểm" sau "4/5" để trình đọc màn hình không đọc dính chữ.
- Vài file bị git checkout ra CRLF (`core.autocrlf`) nên lệnh perl nhiều dòng không khớp. Các file sửa trong F8 đã chuyển về LF (`dashboard.ts`, `stats.*`, `home.html`); git chuẩn hoá khi commit.
- Smoke test Docker (T048), AI tắt, tài khoản `hoc`, bài hôm nay "F14 Z gia dinh":
  - lưu nháp trước khi xong Nghe → 409 `write_locked`; xong Nghe thì bước hiện tại là `write`, `GET /writing` có đề của bài (F15);
  - nháp lưu và tải lại; 4 từ → 400; hoàn thành Viết trước khi nộp → 409 `write_incomplete`;
  - nộp → `pending`, rồi `write/complete` → `doneToday`, streak 1, mục tiêu 1; nộp lại → 409 `already_submitted`;
  - chấm lỗi "AI chưa được cấu hình"; `unseen-count` 1 (latest failed); admin mở bài viết → 404;
  - `seen` → 0; Chấm lại → 202 pending, rồi lỗi lại → unseen 1;
  - thống kê `lessons.write` 1, `writing.submitted` 1, `averageScore` null; dashboard `skills.write` 1;
  - file xuất có 1 bài viết, không có `userId`; log backend 0 lỗi.
  Dữ liệu thử để lại: bài hôm nay của `hoc` đã xong (có 1 bài viết chấm lỗi).
- Còn lại cho người dùng: T049 (cần `GEMINI_API_KEY`: chấm thật, toast trong 1 phút, phần so sánh ở chế độ xám, 360px, bàn phím) và T050 (tick 5 tiêu chí F8 sau khi T049 đạt).
