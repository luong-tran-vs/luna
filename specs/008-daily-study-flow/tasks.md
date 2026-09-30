---

description: "Task list for L – Luồng một ngày học"
---

# Tasks: Luồng một ngày học (L)

**Input**: Design documents from `specs/008-daily-study-flow/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: pure rules (midnight, timezone change, several days off, end of roadmap,
topic change before/after starting today's lesson, returning to an old topic, streak, quota), service (step order, idempotent
completion, quota, effective date of a topic change, locked lessons), endpoints (success, invalid input, 401, 403 locked lesson,
other users), frontend pages and new component inputs; no network (fakes for repositories and ports, fake clock). Every acceptance
criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US5 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Create `frontend/src/app/core/models/study.ts` (`Step`, `StepState`, `GoalView`, `Today`, `MyLessons`, `SetGoalResult`) per data-model.md §6
- [X] T002 [P] Create `backend/internal/progress/study_model.go` (`Step`, `Steps`, `StepDone`, `GoalStatus`, `Goal`, `LessonProgress`, `StudyDay`, `TodayKind`, `TodayState`, `TopicInfo`, errors `ErrNoGoal`, `ErrTopicNotFound`, `ErrStepLocked`, `ErrListenIncomplete`, `ErrNoLesson`, `ErrNotCurrentStep`, `ErrLessonLocked`, `DefaultReviewLimit = 30`) and `backend/internal/progress/study_repository.go` (`GoalRepository`, `ProgressRepository`, `DayRepository`, ports `Roadmaps`, `LessonTitles`, `Reviews`, `Timezones`) per data-model.md §5

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Write `backend/internal/progress/rules_test.go` (table tests): `DayKey` at 23:59/00:01 in Asia/Ho_Chi_Minh and America/New_York, same instant different timezones; `NextDayKey` across month/year; `EffectiveGoal` picks the single active goal, none → false; `TodayLesson`: no goal → `noGoal`; today's study day fixes the lesson even if the roadmap changed; today completed → `doneToday`; otherwise first roadmap lesson not completed → `studying`; all completed or empty roadmap → `noNewLesson`; returning to an old topic resumes its first unfinished lesson; after several days off still exactly one lesson; `Streak`: consecutive days ending today, ending yesterday when today not done, 0 after a gap of one full day, 0 for none, ignores duplicates; `ReviewQuota` 30/0 → 30, 30/12 → 18, 30/40 → 0; `NextStep` review → read → listen → done; `CanStartNewLesson`
- [X] T004 Implement `backend/internal/progress/rules.go` (pure, no DB, no clock) so T003 passes (research R1)
- [X] T005 Extend `backend/internal/progress/fake_test.go` with in-memory `GoalRepository`, `ProgressRepository`, `DayRepository` and fake ports (`Roadmaps` with topics and roadmaps, `LessonTitles`, `Reviews` with due count and reviewed-today count, `Timezones`), and a settable clock; add `newStudyEnv()`
- [X] T006 [P] Create Mongo repositories `backend/internal/storage/mongo/goals.go` (list, activate: pause the other active goal and upsert the chosen one active with `effectiveFrom`, set completed), `backend/internal/storage/mongo/lessonprogress.go` (get, completed newest first, upsert, set position), `backend/internal/storage/mongo/studydays.go` (get, start without overwriting `lessonId`, update, completed keys since a day); indexes in `backend/internal/storage/mongo/indexes.go` (unique `goals (userId, topicId)`, `goals (userId, status)`, unique `lesson_progress (userId, lessonId)`, unique `study_days (userId, dayKey)`)
- [X] T007 [P] Create `frontend/src/app/core/services/study-api.service.ts` (`ng g s core/services/study-api --type=service`) with `goals()`, `setGoal(topicId)`, `today()`, `reviewCards()`, `completeStep(step)`, `savePosition(step, sentenceIndex)`, `myLessons()`, `topics(level)` (`GET /api/topics`) and spec cases in `study-api.service.spec.ts` (contracts/study-api.md)

**Checkpoint**: rules tested; repositories and API client ready

---

## Phase 3: User Story 1 - Đặt mục tiêu (Priority: P1) 🎯 MVP

**Goal**: Chọn trình độ → chủ đề; mục tiêu hiện tiến độ; chúc mừng khi hết lộ trình

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T008 [P] [US1] Write `backend/internal/progress/study_test.go` (goals): `SetGoal` unknown topic → `ErrTopicNotFound`; first goal active with `effectiveFrom` = today; `Goals` returns the active goal with topic name, level, `completedLessons` counted from completed lesson progress of lessons in the current roadmap, `totalLessons` = roadmap length, plus paused goals; deleted topics omitted; `Today` with no goal → `noGoal`, with a goal → `studying` on the first roadmap lesson; other users isolated
- [X] T009 [P] [US1] Write `backend/internal/progress/study_handler_test.go` (goals): `GET /api/goals` shape, `POST /api/goals` 200 / 400 missing `topicId` / 400 unknown field / 404 unknown topic, `GET /api/today` `noGoal`; 401 without session
- [X] T010 [P] [US1] Write `frontend/src/app/features/lesson/goal/goal.spec.ts`: level step (A1…C2 buttons, current goal level preselected), topic step lists `GET /api/topics?level=` with lesson count, "Chưa có bài" for empty topics, progress "2/12 bài" from `GET /api/goals`, choosing posts `setGoal` and navigates to `/today`; `?completed=1` shows the congratulations with links "Chọn chủ đề khác" (same level) and "Lên trình độ B1"

### Implementation for User Story 1

- [X] T011 [US1] Implement `StudyService` (`NewStudyService(StudyDeps{Goals, Progress, Days, Dictation, Roadmaps, Titles, Reviews, Timezones, ReviewLimit, Now})`, `Goals`, `SetGoal` (immediate effect only), `Today` minimal: `noGoal` / `studying` / `noNewLesson`) in `backend/internal/progress/study.go` so T008 passes
- [X] T012 [US1] Implement `StudyHandler` (`GET/POST /api/goals`, `GET /api/today`) in `backend/internal/progress/study_handler.go` so T009 passes
- [X] T013 [US1] Wire in `backend/cmd/api/main.go`: Mongo repos, ports (`Roadmaps` over `topic.Service`, `LessonTitles` over `mongo.Lessons.Summaries`, `Reviews` over vocab, `Timezones` reuse `userTimezones`), study service and handler
- [X] T014 [US1] Generate `ng g c features/lesson/goal`, implement it (two steps, radiogroup for level, list of topic buttons) and add route `goal` (title "Mục tiêu · Luna", `authGuard`) in `frontend/src/app/app.routes.ts` so T010 passes
- [ ] T015 [US1] Manual check: quickstart.md bước 1

**Checkpoint**: Người học có mục tiêu và biết bài hôm nay là bài nào

---

## Phase 4: User Story 2 - Học bài hôm nay: Ôn → Đọc → Nghe (Priority: P1)

**Goal**: Ba bước theo thứ tự, quota 30 thẻ, tự hoàn thành Ôn khi không có thẻ, lưu vị trí, hoàn thành bài

**Independent Test**: quickstart.md bước 2

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T016 [P] [US2] Add to `backend/internal/vocab/review_test.go` and `review_handler_test.go`: `Review` stores `context` (`daily`/`free`, default `free`, other → validation error); `ReviewedSince(userID, "daily", since)` counts only daily reviews of that user after `since`
- [X] T017 [P] [US2] Add to `backend/internal/progress/study_test.go` (steps): `Today` reports steps done/current/locked, `reviewCount = min(due, quota)`; opening with 0 review cards auto-completes review (study day started, lesson fixed); `ReviewCards` limit = quota (30 − reviewed today in daily context), error when the current step is not review; `CompleteStep` out of order → `ErrStepLocked`; listen before dictation summary completed → `ErrListenIncomplete`; completing twice changes nothing; last step → lesson progress `completedAt`, study day `completed`, streak +1, `goalCompleted` when it was the last roadmap lesson (goal marked completed); `SetPosition` saves step + sentence for the current step only (`ErrNotCurrentStep`), negative → validation error; `Today` returns the saved position
- [X] T018 [P] [US2] Add to `backend/internal/progress/study_handler_test.go` (steps): `GET /api/today/review-cards`, `POST /api/today/steps/{step}/complete` (200, 400 unknown step, 409 `step_locked`, 409 `listen_incomplete`, 409 `no_lesson`), `PUT /api/today/position` (204, 400, 409); 401 on all
- [X] T019 [P] [US2] Write `frontend/src/app/shared/components/step-indicator/step-indicator.spec.ts`: three steps Ôn · Đọc · Nghe with ✓ / ● / 🔒 and visually-hidden "đã xong" / "đang làm" / "chưa mở", `aria-current="step"` on the current step, progress text "1/3 bước"
- [X] T020 [P] [US2] Add to `frontend/src/app/shared/components/review-session/review-session.spec.ts`: input `context` is sent in the review body (default `free`)
- [X] T021 [P] [US2] Add to `frontend/src/app/features/lesson/reading/reading.spec.ts` and `listening/listening.spec.ts`: input `lessonId` overrides the route id; `startSentence` opens that sentence (Listening) / scrolls to it (Reading); `position` output emitted when the sentence changes (Listening) or the top visible sentence changes (Reading, stubbed IntersectionObserver); `mode="review"`: Listening does not POST results nor restore saved ones, Reading hides "Đã đọc xong"
- [X] T022 [P] [US2] Write `frontend/src/app/features/lesson/today/today.spec.ts`: `noGoal` → link to `/goal`; `studying` shows the step indicator, the lesson title, the review session with `context="daily"` and cards from `/api/today/review-cards`; review `finished` → POST complete review → reloads and shows Reading of the lesson at `sentenceIndex`; Reading `completed` → POST complete read → Listening at its position; Listening `completed` → POST complete listen → "Đã xong bài hôm nay, hẹn bạn ngày mai", streak and goal progress; `position` outputs → PUT position debounced 1 s (fake timers, only the last value sent); 409 on complete shows the message; `goalCompleted` → navigate to `/goal?completed=1`

### Implementation for User Story 2

- [X] T023 [US2] Implement `context` in `backend/internal/vocab` (model, `ReviewInput.Context`, validation, log) and `ReviewedSince` in the service; `context` field and `CountSince` in `backend/internal/storage/mongo/reviewlogs.go`; index `(userId, context, reviewedAt)` in `indexes.go`, so T016 passes
- [X] T024 [US2] Implement `Today` (full), `ReviewCards`, `CompleteStep`, `SetPosition` in `backend/internal/progress/study.go` so T017 passes; `Reviews` port adapter in `backend/cmd/api/main.go` (`DueCount`, `Due(limit)`, `ReviewedSince`)
- [X] T025 [US2] Add routes `GET /api/today/review-cards`, `POST /api/today/steps/{step}/complete`, `PUT /api/today/position` to `backend/internal/progress/study_handler.go` so T018 passes
- [X] T026 [US2] Generate `ng g c shared/components/step-indicator` and implement it so T019 passes; add input `context` to `review-session` and `ReviewInput`/`VocabApiService.review` so T020 passes
- [X] T027 [US2] Add inputs `lessonId`, `startSentence`, `mode` and output `position` to `frontend/src/app/features/lesson/reading/reading.ts` and `listening/listening.ts` so T021 passes (existing routes keep working unchanged)
- [X] T028 [US2] Generate `ng g c features/lesson/today`, implement it and add route `today` (title "Hôm nay · Luna", `authGuard`) in `app.routes.ts` so T022 passes
- [ ] T029 [US2] Manual check: quickstart.md bước 2

**Checkpoint**: Học trọn một bài trong ngày

---

## Phase 5: User Story 3 - Mỗi ngày một bài và chuỗi ngày học (Priority: P1)

**Goal**: Bài mới lúc 0 giờ, không dồn bài, streak, "Chưa có bài mới", thẻ dư sang hôm sau

**Independent Test**: quickstart.md bước 3

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T030 [P] [US3] Add to `backend/internal/progress/study_test.go` (days): completing today's lesson then moving the clock to 23:59 → `doneToday`; to 00:01 next day (user timezone) → next lesson `studying`; three days off → one lesson, streak 0, then 1 after completing; streak with today not done shows yesterday's count; empty or finished roadmap → `noNewLesson`; 45 due cards with 30 reviewed in daily context today → `reviewCount` 0 and review auto-completes, next day quota 30 again; changing the user's timezone mid-day never produces a second new lesson for a day that already has a completed lesson
- [X] T031 [P] [US3] Add to `today.spec.ts`: `doneToday` shows "Đã xong bài hôm nay", streak, links to free review (`/vocabulary/review`) and `/lessons`; `noNewLesson` shows "Chưa có bài mới" with links to free review and `/goal`

### Implementation for User Story 3

- [X] T032 [US3] Complete day handling in `backend/internal/progress/study.go` (streak from `DayRepository.CompletedKeys`, timezone via port, `doneToday`/`noNewLesson` states) so T030 passes
- [X] T033 [US3] Implement the `doneToday` and `noNewLesson` views in `frontend/src/app/features/lesson/today/` so T031 passes
- [ ] T034 [US3] Manual check: quickstart.md bước 3 (day change with the DB command, days off, quota)

**Checkpoint**: Nhịp một bài mỗi ngày đúng

---

## Phase 6: User Story 4 - Đổi chủ đề hoặc trình độ và quay lại (Priority: P2)

**Goal**: Đổi có hiệu lực ngay hoặc từ ngày mai, tiến độ từng lộ trình riêng, streak không đổi

**Independent Test**: quickstart.md bước 4

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- [X] T035 [P] [US4] Add to `backend/internal/progress/study_test.go` (switching): before any step today → new topic's lesson at once, `startsTomorrow` false; after the review step → `startsTomorrow` true, `effectiveFrom` tomorrow, today's lesson unchanged and still completable, tomorrow the new topic's first unfinished lesson; the old goal becomes paused and keeps its progress; returning resumes lesson 3 of 5; streak unchanged; a B1 topic while studying A1 → only B1 lessons afterwards
- [X] T036 [P] [US4] Add to `goal.spec.ts`: `startsTomorrow` response shows "Chủ đề mới bắt đầu từ ngày mai" before navigating; paused goals show their progress in the topic list

### Implementation for User Story 4

- [X] T037 [US4] Implement effective-date handling in `SetGoal` (`effectiveFrom`, `startsTomorrow`) in `backend/internal/progress/study.go` and the handler response so T035 passes
- [X] T038 [US4] Implement the message and paused progress in `frontend/src/app/features/lesson/goal/` so T036 passes
- [ ] T039 [US4] Manual check: quickstart.md bước 4

**Checkpoint**: Đổi chủ đề an toàn

---

## Phase 7: User Story 5 - Trang "Bài học" và khoá bài sắp tới (Priority: P2)

**Goal**: Hôm nay / Đã học / Sắp tới; xem lại không đổi tiến độ; bài chưa tới lượt bị chặn

**Independent Test**: quickstart.md bước 5

### Tests for User Story 5 (REQUIRED - constitution VI) ⚠️

- [X] T040 [P] [US5] Add to `backend/internal/progress/study_test.go` (access and list): `MyLessons` today, completed (all topics, newest first, with topic name), upcoming (rest of the active roadmap after today's lesson, not completed, in order, id + title only); `CanOpen`: admin always, learner today's lesson, lessons with progress, not upcoming ones
- [X] T041 [P] [US5] Add to `backend/internal/progress/study_handler_test.go` and `handler_test.go`: `GET /api/lessons/mine`; `Guard` middleware → 403 `lesson_locked` for a learner on an upcoming lesson, 200 for today's lesson and for an admin; dictation routes guarded; update `backend/internal/lesson/reading_handler_test.go` for the guard parameter (reading, lookup, vocabulary)
- [X] T042 [P] [US5] Write `frontend/src/app/features/lesson/my-lessons/my-lessons.spec.ts`: sections Hôm nay (link `/today`), Đã học (links `/lessons/:id/read?review=1` and `/listen?review=1`, topic name, date), Sắp tới (titles with 🔒 and visually-hidden "chưa mở", no links); empty states; update `reading.spec.ts`/`listening.spec.ts`: `?review=1` sets review mode; 403 `lesson_locked` shows "Bài này sẽ mở khi tới lượt"

### Implementation for User Story 5

- [X] T043 [US5] Implement `MyLessons`, `CanOpen` and `Guard(requireAuth)` in `backend/internal/progress/study.go`/`study_handler.go`; `GET /api/lessons/mine`; make `lesson.ReadingHandler.Register` and progress dictation `Register` take the guard; wire in `backend/cmd/api/main.go` so T040, T041 pass
- [X] T044 [US5] Generate `ng g c features/lesson/my-lessons`, implement it, set it as route `''` (title "Bài học · Luna") in `frontend/src/app/features/lesson/lesson.routes.ts`; read `?review=1` in Reading/Listening; show the locked message on 403 so T042 passes
- [ ] T045 [US5] Manual check: quickstart.md bước 5

**Checkpoint**: Mọi user story hoạt động độc lập

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T046 [P] Add "Học hôm nay" button (`/today`) to `frontend/src/app/features/home/home.html` and "Bài học" link (`/lessons`) to `frontend/src/app/shared/components/app-header/app-header.html`, with spec assertions
- [X] T047 Update `README.md`: section "Một ngày học" (mục tiêu, bài hôm nay, streak, 30 thẻ, trang Bài học, lệnh lùi ngày để thử)
- [X] T048 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T049 Run all tests (`go test -race ./...`, `npm test -- --watch=false`) with zero failures; `npx ng build` without warnings
- [X] T050 Rebuild and smoke test in Docker with `curl` and UTF-8 body files: goals, today (noGoal → studying), review cards quota, step order 409, listen incomplete 409, complete all steps → doneToday + streak, day change via DB, topic switch after starting, lessons/mine, locked lesson 403 for a learner and 200 for an admin, 401
- [X] T051 Compute WCAG contrast for new pairs (step indicator done/current/locked, streak badge, locked upcoming items) in both modes
- [ ] T052 Manual check at 360px width in light and dark mode (quickstart.md bước 6)
- [ ] T053 Tick the 9 L acceptance criteria in `docs/phases/giai-doan-1.md` once T015, T029, T034, T039, T045, T050, T052 pass

---

## Dependencies & Execution Order

- **Setup** → **Foundational** (rules first) → **US1** → **US2** → **US3** → **US4** → **US5** → **Polish**
- US3 and US4 extend `study.go` after US2; US5 needs today's lesson (US2) for `CanOpen`
- Backend and frontend of each story can proceed in parallel once the contract is fixed

### Parallel Opportunities

- T001 ∥ T002; T006 ∥ T007 (after T002, T004)
- T008 ∥ T009 ∥ T010; T016 ∥ T017 ∥ T018 ∥ T019 ∥ T020 ∥ T021 ∥ T022
- T030 ∥ T031; T035 ∥ T036; T040 ∥ T041 ∥ T042

---

## Parallel Example: User Story 2

```bash
Task: "vocab review context tests in backend/internal/vocab/"
Task: "Step tests in backend/internal/progress/study_test.go"
Task: "step-indicator spec in frontend/src/app/shared/components/step-indicator/"
Task: "Reading/Listening input specs in frontend/src/app/features/lesson/"
Task: "today page spec in frontend/src/app/features/lesson/today/"
```

---

## Implementation Strategy

1. Setup + Foundational (rules) → US1 (mục tiêu) → US2 (một bài trọn ba bước) → **STOP and VALIDATE** with quickstart bước 1–2
2. US3 → nhịp ngày, streak, hết bài
3. US4 → đổi chủ đề
4. US5 → trang Bài học, khoá bài
5. Polish → nút trang chủ, header, README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (`GOTOOLCHAIN=local`)
- `progress` không import `topic`, `lesson`, `vocab`; chỉ `main.go` nối qua port (research R7)
- Gửi dữ liệu tiếng Việt khi test bằng curl trên Windows: dùng `--data-binary @file.json`
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- **Không có `GET /api/today/review-cards` (lệch contract, T025):** `GET /api/today` trả `reviewCount` = min(thẻ đến hạn, 30 −
  thẻ đã ôn ở bước Ôn hôm nay); trang `/today` lấy thẻ bằng `GET /api/vocab/review/due?limit=<reviewCount>` (F5) và gửi đánh giá
  với `context: "daily"`. Số thẻ đã ôn đếm từ `review_logs` nên giới hạn vẫn đúng kể cả khi người học lấy nhiều hơn.
- **Trạng thái mục tiêu chỉ có `active` | `paused`** (data-model có `completed`): "đã học hết lộ trình" tính khi đọc
  (`goalCompleted`), nên quản trị viên thêm bài sau thì mục tiêu tự "chưa xong" lại, không cần cập nhật trạng thái.
- Bài hôm nay cố định khi bước đầu tiên xong. Vì bước Ôn tự xong khi không có thẻ đến hạn, **mở `/today` hôm không có thẻ đã tính
  là bắt đầu** (đổi chủ đề sau đó có hiệu lực từ mai) — đúng spec Assumptions, test `TestSwitchAfterStarting` phủ trường hợp này.
- Bước Ôn dùng kiểu "Xem từ đoán nghĩa"; chọn kiểu ôn cho bước Ôn để F12.
- Reading báo vị trí = câu đầu tiên đang thấy trên màn hình (IntersectionObserver); Listening báo câu vừa chuyển tới; `/today` lưu
  sau 1 giây kể từ lần đổi cuối (`PUT /api/today/position`).
- Reading/Listening: input `lessonId`, `startSentence`, `mode` (`study` | `review`, mặc định theo `?review=1`), output
  `position`; chế độ xem lại không có "Đã đọc xong", Listening không khôi phục và không lưu kết quả. Lỗi 403 hiện "Bài này sẽ
  mở khi tới lượt" (`features/lesson/load-error.ts`).
- Guard (`StudyHandler.Guard`) bọc `GET /api/lessons/{id}`, `/lookup`, `/vocabulary`, `/dictation`, `/dictation/summary`;
  `lesson.ReadingHandler.Register` và `progress.Handler.Register` nhận thêm tham số `guard`.
- Smoke test Docker (T050) với `hoc@example.com`: chưa có mục tiêu → `noGoal`; `GET /api/topics?level=A1` 200; chọn "A1 · Mua
  sắm" → bài hôm nay "F14 X", 2 thẻ đến hạn → `reviewCount` 2, ôn 1 thẻ `context: daily` → 1; hoàn thành Đọc trước Ôn → 409
  `step_locked`; vị trí 1 lưu và trả lại, vị trí 9 → 400; Nghe trước khi chép hết → 409 `listen_incomplete`; chép 2 câu rồi Nghe →
  `doneToday`, streak 1, mục tiêu 1/1, `goalCompleted`; gọi lại không cộng; `/api/lessons/mine` đúng; bài Z (chưa tới lượt) 403
  `lesson_locked` cả `dictation/summary`, quản trị viên 200; đổi sang "Gia đình" → `startsTomorrow`; lùi ngày trong DB → bài hôm
  nay "F14 Z", streak 1; 401 khi thiếu phiên.
- Dữ liệu `hoc@example.com` sau smoke test: mục tiêu "A1 · Gia đình", bài hôm nay "F14 Z" chưa bắt đầu, đã học "F14 X".
- Tương phản (T051): bước hiện tại primary-ink/primary-soft 7.96 / 7.86, viền primary/surface 7.37 / 6.22; dấu ✓ ok/surface
  4.52 / 7.97; bước khoá và bài sắp tới muted 6.33 / 6.60 (surface), 5.80 / 7.26 (bg); streak accent-ink/bg 5.78 / 12.45; thanh
  mục tiêu primary/track 5.85 / 5.45; trình độ đang chọn 7.37 / 6.87.
