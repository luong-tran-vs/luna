---

description: "Task list for F17 – Trang chi tiết bài và luyện tập từ vựng"
---

# Tasks: Trang chi tiết bài và luyện tập từ vựng (F17)

**Input**: Design documents from `specs/016-vocab-practice/`

**Prerequisites**: plan.md, spec.md, research.md (R1–R13), data-model.md, contracts/practice-api.md, quickstart.md

**Tests**: REQUIRED by constitution principle VI and khối 2:
- `CleanPractice`, `BuildFill`, tile splitting;
- the practice job: success, AI error does not touch annotations, stale revision ignored, regenerate;
- handler guard (403 locked lesson, 404), admin regenerate, practice audio route;
- Gemini `Practice` schema/prompt with a fake server;
- Vitest: step navigation and progress, fill (select, remove, check right/wrong), tiles (order right/wrong, Làm lại), summary, no-practice
  state, keyboard.

Real-AI quality, 360px, light/dark and screen reader are checked by hand (quickstart.md).

**Organization**: Tasks are grouped by user story (US1–US5 from spec.md). US2 (generation) comes before US1 although both are P1, because
the page needs content; US1 can still be validated alone by inserting a `practice` document with mongosh.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Create `frontend/src/app/core/models/practice.ts` (data-model.md › Frontend): `PracticeStatus`, `PracticeExample`,
  `PracticeTurn`, `PracticeDialogue`, `FillPart` (`{text: string} | {blank: number}`), `PracticeFill`, `PracticeTranslation`,
  `PracticeView`, `AdminPractice`.
- [X] T002 [P] Add icons `play`, `pause`, `volume` to `IconName` and the SVG map in `frontend/src/app/shared/components/icon/icon.ts`
  (same stroke style as the existing set).

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: types, storage and job types are shared by every story.

- [X] T003 Add to `backend/internal/ai/ai.go` (research R1): `PracticeWord`, `PracticeRequest`, `Practice`, `Example`, `Dialogue`, `Turn`,
  `Translation` with camelCase JSON tags; `Provider.Practice`; `Disabled.Practice` returning `ErrNotConfigured`. Add `Practice` to the
  fakes: `backend/internal/lesson/fake_test.go` (configurable result, error, call counter) and `backend/internal/writing/fake_test.go`
  (compile only).
- [X] T004 [P] Write tests in `backend/internal/ai/gemini/gemini_test.go` (`TestPractice`, `TestPracticeErrors`,
  `TestPracticeWithoutKeyMakesNoRequest`), then implement `practiceSchema`, `practicePrompt` and `Practice` in
  `backend/internal/ai/gemini/gemini.go`:
  - one POST, temperature 0.5, `responseMimeType` JSON, schema OBJECT with all top-level fields required, `speaker` INTEGER;
  - prompt contains the level, the title, numbered sentences and each `lemma | text | meaningVi`;
  - 429 → `ErrQuota`, 401/403 → `ErrInvalidKey`; empty key → `ErrNotConfigured` with 0 requests;
  - log `op=practice` with `words` and `sentences` counts only.
- [X] T005 Add `TypePractice = "practice"` and `TypePracticeAudio = "practice_audio"` in `backend/internal/job/model.go`.
- [X] T006 Extend `backend/internal/lesson/model.go` (research R2):
  - `Practice` (same shape as `ai.Practice` but `Dialogue *Dialogue`), `Lesson.Practice *Practice`, `PracticeStatus Status`,
    `PracticeError string`, `PracticeVersion int`; `StatusNone Status = ""`;
  - `StatusOf` becomes an explicit `switch` (TTS → audio, Annotate → annotation, Practice → practice);
  - errors `ErrNoValidPractice`, `ErrPracticeRunning`, `ErrAnnotationNotDone`;
  - `Topics` port gains `Position(ctx, topicID, lessonID string) (int, error)` (1-based, 0 when not in the roadmap).
  - `Repository` gains `SavePractice(ctx, id string, revision, prevVersion int, p Practice) (bool, error)`.
  - Update fakes in `backend/internal/lesson/fake_test.go`: `fakeTopics.Position`, fake `SavePractice` (respects revision and version),
    fake `SetStatus` for `TypePractice`, `SaveAnnotations` clearing practice (T008 semantics), `ReplaceContent` clearing practice.
- [X] T007 Implement storage in `backend/internal/storage/mongo/lessons.go`:
  - `practiceDoc`, `exampleDoc`, `dialogueDoc`, `turnDoc`, `translationDoc` with direct struct conversion; `lessonDoc` fields `practice`,
    `practiceStatus`, `practiceError`, `practiceVersion`; `toLesson`/`fromLesson`;
  - `SavePractice`: filter `{_id, revision, practiceVersion: prev}`, `$set practice, practiceStatus=done, practiceError=""`,
    `$inc practiceVersion`, return matched;
  - `SaveAnnotations` also sets `practice=nil, practiceStatus=running, practiceError=""`;
  - `ReplaceContent` also sets `practice=nil, practiceStatus="", practiceError=""`;
  - `SetStatus` explicit `switch` with a `TypePractice` branch; unknown type → error.
  - Add a pure conversion test in `backend/internal/storage/mongo/lessons_test.go` (practice round-trip, nil dialogue).
- [X] T008 Implement `lessonTopicsPort.Position` in `backend/cmd/api/main.go` via `topicSvc.Get` → index in `LessonIDs` + 1.

**Checkpoint**: `go build ./...` and `go test ./...` pass.

---

## Phase 3: User Story 2 - Phần luyện tập được sinh tự động (Priority: P1)

**Goal**: after annotation, one background AI request produces cleaned practice content plus audio; AI failure leaves everything else intact;
editing the lesson regenerates.

**Independent Test**: quickstart.md bước 1–2 (with the fake AI in tests; with a real key by hand).

### Tests for User Story 2

- [X] T009 [P] [US2] Write `backend/internal/lesson/practice_test.go` (research R4–R6):
  - `containsWord`: case-insensitive, whole word ("meet" not in "meeting"), matches `text` or `lemma`, multi-word phrases;
  - `CleanPractice`: example without its word dropped; unknown lemma dropped; duplicate lemma keeps first; dialogue missing a speaker
    name dropped; bad turns dropped (speaker ∉ {0,1}, empty text/meaning, > 200 chars); < 4 turns → `nil` dialogue; > 10 → first 10;
    translation with empty `en`, < 2 or > 15 tiles, or no vocabulary word dropped; distractors trimmed, deduped, without answer tiles,
    max 4; max 5 translations; long `objectiveVi`/`grammarTipVi` dropped; nothing usable → `ErrNoValidPractice`;
  - `answerTiles("Nice to meet you, Anna.")` → `["Nice","to","meet","you,","Anna."]`;
  - `BuildFill`: blanks match `text` or `lemma` as whole words; phrase blanks; at most 5; spread over turns round-robin, preferring unused
    words; parts keep surrounding text and punctuation; `answer` without surrounding punctuation; `wordBank` = answers + ≤ 2 other
    vocabulary words; no candidate → `nil`.
- [X] T010 [P] [US2] Write `backend/internal/lesson/practice_job_test.go` (research R3, R7):
  - annotate success → practice cleared, status running, one `practice` job enqueued with the same revision;
  - `ProcessPractice` success → exactly 1 AI call with the lesson words (deduped by lemma, ≤ 30), practice saved, status done, version +1,
    one `practice_audio` job (no TargetID: it always uses the current version);
  - AI `ErrNotConfigured` → permanent; other AI error → retryable; `JobFailed` → status failed with a Vietnamese message while
    annotations, questions, grammar note, writing prompt stay unchanged;
  - all content invalid → permanent `ErrNoValidPractice` (no retry);
  - stale revision → no AI call, nothing saved; version changed meanwhile → not saved;
  - lesson content update → practice cleared, status none;
  - `ProcessPracticeAudio`: writes `example-i`, `turn-i`, `answer-i` under `{dir}/{id}/{rev}/practice/{ver}/`, skips existing non-empty
    files, removes other version dirs, stale revision/version → no TTS call, TTS error → error returned and `JobFailed` leaves
    `practiceStatus` unchanged.

### Implementation for User Story 2

- [X] T011 [US2] Create `backend/internal/lesson/practice.go`: `containsWord`, `CleanPractice`, `answerTiles`, `BuildFill` and the `Fill`,
  `FillTurn`, `Part`, `Blank` types so T009 passes.
- [X] T012 [US2] Create `backend/internal/lesson/practice_job.go`: `practiceWords(l)`, `ProcessPractice`, `ProcessPracticeAudio`
  (reuse `current`, `writeFileAtomic`); in `backend/internal/lesson/service.go` make `ProcessAnnotate` enqueue `TypePractice` after a
  successful `SaveAnnotations`, and `JobFailed` handle `TypePractice` (set failed + message) and `TypePracticeAudio` (log only) so T010
  passes.
- [X] T013 [US2] Wire the worker in `backend/cmd/api/main.go`: handlers `job.TypePractice: lessonSvc.ProcessPractice`,
  `job.TypePracticeAudio: lessonSvc.ProcessPracticeAudio`; `onFailed` routes both to `lessonSvc.JobFailed`.

**Checkpoint**: creating a lesson with the fake/real AI produces practice and audio files; AI off keeps the lesson usable.

---

## Phase 4: User Story 1 - Học từ vựng qua câu ví dụ và hội thoại mẫu (Priority: P1) 🎯 MVP

**Goal**: learner API + new lesson detail page with top bar, header, tabs, step 1 (vocabulary) and step 2 (dialogue), Next button.

**Independent Test**: quickstart.md bước 3 (and the no-practice part of bước 2).

### Tests for User Story 1

- [X] T014 [P] [US1] Write `backend/internal/lesson/practice_handler_test.go`:
  - `GET /api/lessons/{id}/practice` 200 with practice (examples with/without `audioUrl` depending on a file in a temp dir, dialogue,
    `fill`, translations with `answer`/`tiles`, `lessonNumber` from `fakeTopics.Position`);
  - 200 without practice (`status` none/running, empty arrays, `dialogue`/`fill` null);
  - 403 `lesson_locked` through the guard, 404 unknown lesson, 401 without auth;
  - `GET /api/audio/{id}/{rev}/practice/{ver}/{kind}/{i}`: 200 `audio/mpeg` with the immutable cache header; bad id, negative/non-number
    rev/ver/index, unknown kind, missing file → 404.
- [X] T015 [P] [US1] Write `frontend/src/app/features/lesson/lesson-detail/practice-logic.spec.ts`: seeded shuffle is deterministic and
  a permutation; `levelLabel` (A1–A2 Cơ bản, B1–B2 Trung cấp, C1–C2 Nâng cao); `hasPractice*` helpers.
- [X] T016 [P] [US1] Write `frontend/src/app/features/lesson/lesson-detail/lesson-detail.spec.ts` (replace the old spec):
  - loads lesson, vocabulary, practice, my lessons and dashboard; shows close link to `/lessons`, "1/4", streak "3 ngày", "Bài 3", title,
    "Mục tiêu: …", "Mức độ: Cơ bản"; hides "Bài N" when `lessonNumber` is 0 and streak when the dashboard fails;
  - Next moves 1→2→3→4 and updates the progress bar (`aria-valuenow`) and text;
  - tabs: Bài đọc shows paragraphs and grammar note; back to Bài học keeps the step; arrow keys switch tabs;
  - no practice → step 1 lists words without "Ví dụ", steps 2–4 show "Bài này chưa có phần luyện tập";
  - 403 `lesson_locked` → "Bài này sẽ mở khi tới lượt"; 404 → not-found message.
- [X] T017 [P] [US1] Write `vocab-step.spec.ts` (word, ipa, "— nghĩa", word audio uses `/api/tts/word?text=`, example plays its
  `audioUrl`, collapse toggles `aria-expanded`) and `dialogue-step.spec.ts` (play all plays turns in order and skips `null` audio, rate
  0.75/1/1.25 passed to the player, stop, single-turn play, meaning toggle hides/shows all meanings, "Lượt x/y") in
  `frontend/src/app/features/lesson/lesson-detail/`.
- [X] T018 [P] [US1] Extend `frontend/src/app/shared/components/audio-player/audio-player.spec.ts` for the new `timeupdate` output, and
  `frontend/src/app/features/lesson/reading-api.service.spec.ts` for `practice(id)` → `GET /api/lessons/{id}/practice`.

### Implementation for User Story 1

- [X] T019 [US1] Backend learner API (research R7, R8):
  - `Reader.Practice(ctx, id) (PracticeView, error)` in `backend/internal/lesson/reading.go` (or a new `practice_view.go`): status,
    `lessonNumber` via `Topics.Position`, examples/turns/answers with `audioUrl` from `os.Stat`, `BuildFill` on the stored dialogue,
    translations with `answer`/`tiles`;
  - `NewReader` takes `audioDir`; update callers in tests and `backend/cmd/api/main.go`;
  - register `GET /api/lessons/{id}/practice` in `backend/internal/lesson/reading_handler.go` with `requireAuth(guard(...))`;
  - `PracticeAudioHandler(dir)` in `backend/internal/lesson/practice_handler.go`, registered next to the existing audio route in
    `backend/internal/lesson/handler.go`;
  - JSON per `contracts/practice-api.md` so T014 passes.
- [X] T020 [P] [US1] Add `timeupdate` output (current time in seconds) to
  `frontend/src/app/shared/components/audio-player/audio-player.ts`.
- [X] T021 [P] [US1] Add `practice(id)` to `frontend/src/app/features/lesson/reading-api.service.ts`.
- [X] T022 [US1] Create `frontend/src/app/features/lesson/lesson-detail/practice-logic.ts`: `shuffle(items, seed)` (small seeded PRNG),
  `levelLabel`, `hasDialogue/hasFill/hasTranslations` so T015 passes.
- [X] T023 [US1] Create `vocab-step/` and `dialogue-step/` components in `frontend/src/app/features/lesson/lesson-detail/` (research R9,
  R10): inputs from the page, emit `play(url)` to the page-level `lu-audio-player`; speed radiogroup copied from
  `listening.html:64-79`; elapsed `mm:ss` from `timeupdate`; current turn `aria-current`; meaning toggle button with `aria-pressed`.
- [X] T024 [US1] Rewrite `frontend/src/app/features/lesson/lesson-detail/lesson-detail.ts/.html/.css`:
  - `forkJoin` of `getLesson` (403/404 handling), `vocabulary`, `practice`, `myLessons`, `dashboard` (each optional one with
    `catchError`);
  - top bar (close `routerLink="/lessons"`, 4-segment progress bar `role="progressbar"` + "n/4", streak), header, `role="tablist"` tabs,
    Bài đọc tab (paragraphs + grammar note), step host with placeholders for steps 3–4;
  - fixed Next button `bottom: var(--tabbar-h, 0px)`, page `padding-bottom` so content is not hidden; tokens only;
  - one `lu-audio-player` for all audio, so T016–T017 pass.

**Checkpoint**: MVP — learner sees steps 1–2 with real or inserted practice; locked lessons blocked.

---

## Phase 5: User Story 3 - Điền từ vào ô trống trong hội thoại (Priority: P2)

**Goal**: step 3 with blanks, word bank, check with text + icon results, grammar tip.

**Independent Test**: quickstart.md bước 4.

### Tests for User Story 3

- [X] T025 [P] [US3] Add to `practice-logic.spec.ts`: `checkFill(answers, blanks)` case-insensitive and trimmed; unfilled counts wrong;
  `nextEmptyBlank`.
- [X] T026 [P] [US3] Write `frontend/src/app/features/lesson/lesson-detail/fill-step/fill-step.spec.ts`:
  - first blank selected (`aria-pressed`); tapping a bank word fills it, disables that bank button and selects the next empty blank;
  - ✕ removes the word (bank button enabled again, blank selected); selecting a filled blank then a word replaces it;
  - Kiểm tra disabled until all blanks are filled;
  - all correct → "Chính xác!" in `role="status"`; a wrong blank shows "Sai" + "Đáp án: …", correct ones "Đúng";
  - grammar tip card shown after checking; duplicate words in the bank are interchangeable;
  - keyboard: Tab/Enter/Space operate blanks, bank and ✕;
  - "Nghe câu" emits the turn audio.

### Implementation for User Story 3

- [X] T027 [US3] Add `checkFill`, `nextEmptyBlank` to `practice-logic.ts` so T025 passes.
- [X] T028 [US3] Create `fill-step/` (ts/html/css) per research R11, emitting results (`correct`, `total`, `checked`) to the page; wire it
  as step 3 in `lesson-detail.html`, with the shuffled `wordBank` from the page seed, so T026 passes.

**Checkpoint**: step 3 works with practice data.

---

## Phase 6: User Story 4 - Ghép câu dịch Việt → Anh và tổng kết (Priority: P2)

**Goal**: step 4 tile translation, one sentence at a time, then summary with retry and lesson buttons.

**Independent Test**: quickstart.md bước 5.

### Tests for User Story 4

- [X] T029 [P] [US4] Add to `practice-logic.spec.ts`: `checkTranslation(chosen, answer)` exact order, tiles with identical text
  interchangeable; `summary(fillResult, translateResults)` with unchecked items counted wrong.
- [X] T030 [P] [US4] Write `frontend/src/app/features/lesson/lesson-detail/translate-step/translate-step.spec.ts`: "Câu 1/4", tapping
  tiles appends in order and disables them, ✕ removes one, Làm lại clears, Kiểm tra disabled when empty, right order → "Chính xác!",
  wrong → "Chưa đúng" + "Câu đúng: …", speaker button enabled only after checking, keyboard operation.
- [X] T031 [P] [US4] Extend `lesson-detail.spec.ts`: Next inside step 4 moves to the next sentence; last sentence shows "Hoàn thành";
  summary "Điền đúng x/y ô", "Dịch đúng a/b câu"; today's lesson → "Học bài này" link `/today`; completed lesson → Đọc lại / Nghe lại /
  Bài viết with `review=1`; Làm lại returns to step 1 with cleared state and a new seed; no request is sent to save results.

### Implementation for User Story 4

- [X] T032 [US4] Add `checkTranslation`, `summary` to `practice-logic.ts` so T029 passes.
- [X] T033 [US4] Create `translate-step/` (ts/html/css) per research R11 so T030 passes.
- [X] T034 [US4] Create `practice-summary/` and wire step 4, "Hoàn thành", summary, Làm lại and lesson buttons in
  `lesson-detail.ts/.html` so T031 passes.

**Checkpoint**: full 4-step flow with summary.

---

## Phase 7: User Story 5 - Quản trị viên xem và tạo lại phần luyện tập (Priority: P3)

**Goal**: admin sees status/content and can regenerate.

**Independent Test**: quickstart.md bước 1.6 and bước 2.1.

### Tests for User Story 5

- [X] T035 [P] [US5] Add to `practice_job_test.go` / `handler_test.go` in `backend/internal/lesson/`: `RegeneratePractice` sets running,
  keeps old content and enqueues 1 job; running → `ErrPracticeRunning` (409 `practice_running`); annotation not done →
  `ErrAnnotationNotDone` (409 `annotation_not_done`); 404 unknown; 403 for a learner; admin `GET` lesson JSON includes `practice`,
  `practiceStatus` (`"none"` when empty), `practiceError`.
- [X] T036 [P] [US5] Write `frontend/src/app/features/admin/lesson-detail/practice-section/practice-section.spec.ts` (status chip,
  error text, content rendering, button disabled with reason while running or annotation not done, click calls the API and emits the
  lesson) and extend `frontend/src/app/features/admin/admin-api.service.spec.ts` for `regeneratePractice`.

### Implementation for User Story 5

- [X] T037 [US5] Implement `Service.RegeneratePractice` in `backend/internal/lesson/practice_job.go`; route
  `POST /api/admin/lessons/{id}/practice/regenerate` (202 + lesson), `lessonJSON` practice fields and 409 mappings in
  `backend/internal/lesson/handler.go` so T035 passes.
- [X] T038 [US5] Frontend admin: `Lesson` gains `practice`, `practiceStatus`, `practiceError` and `isRunning()` includes practice in
  `frontend/src/app/core/models/lesson.ts`; `regeneratePractice(id)` in `frontend/src/app/features/admin/admin-api.service.ts`;
  `practice-section/` component inserted after `<lu-lesson-extras>` in
  `frontend/src/app/features/admin/lesson-detail/lesson-detail.html` so T036 passes (update admin lesson fixtures in existing specs).

**Checkpoint**: all user stories done.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T039 [P] Update `README.md`: lesson detail page (4 steps, optional, no XP, results not saved) and practice generation (1 AI request
  after annotation, audio job, regenerate button, edit lesson regenerates).
- [X] T040 Run lint (frontend `npm run lint`, backend `make lint`) with zero issues.
- [X] T041 Run all tests (`CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`, `npx ng test --watch=false`) with zero failures; `npx ng
  build` without warnings.
- [X] T042 Rebuild Docker and smoke test with AI off: lesson created → practice failed with message, annotations unaffected; `GET
  /practice` → empty 200; locked lesson → 403; insert a `practice` document with mongosh → page shows all 4 steps; edit content →
  practice cleared; progress, streak and stats unchanged after doing the practice (SC-006).
- [ ] T043 Manual check (needs a Gemini key): quickstart.md bước 1–6 (real content quality, 1 `op=practice` log line per generation,
  audio, 360px light/dark, keyboard, screen reader).
- [ ] T044 Tick the 9 F17 acceptance criteria in `docs/phases/giai-doan-3.md` once T041–T043 pass.

---

## Dependencies & Execution Order

- Setup (T001, T002) ∥ Foundational (T003 → T004; T005 → T006 → T007 → T008).
- US2 (T009–T013) after Foundational.
- US1 backend (T014, T019) after T011 (`BuildFill`); US1 frontend (T015–T018, T020–T024) only needs T001–T002 and the contract.
- US3 and US4 after T024 (page shell); US4 summary (T034) after US3 (fill results feed the summary).
- US5 backend (T035, T037) after T012; US5 frontend (T036, T038) after T001.
- Polish after all stories.

### Parallel Opportunities

- T001 ∥ T002 ∥ T003.
- T009 ∥ T010 ∥ T015 ∥ T016 ∥ T017 ∥ T018.
- T020 ∥ T021 ∥ T022.
- T025 ∥ T026; T029 ∥ T030 ∥ T031; T035 ∥ T036.
- Backend US2/US5 work and frontend US1/US3/US4 work can proceed side by side.

---

## Parallel Example: User Story 1

```bash
Task: "Practice handler tests in backend/internal/lesson/practice_handler_test.go"
Task: "practice-logic spec in frontend/src/app/features/lesson/lesson-detail/practice-logic.spec.ts"
Task: "lesson-detail spec in frontend/src/app/features/lesson/lesson-detail/lesson-detail.spec.ts"
Task: "timeupdate output in frontend/src/app/shared/components/audio-player/audio-player.ts"
```

---

## Implementation Strategy

1. Setup + Foundational.
2. US2: generation pipeline (backend only). Validate with fake AI tests.
3. US1: API + page with steps 1–2. **STOP and VALIDATE** (quickstart bước 3) — MVP.
4. US3: fill-in-the-blank. US4: tile translation + summary.
5. US5: admin section and regenerate.
6. Polish: README, lint, tests, Docker smoke, manual checks, tick the criteria.

---

## Notes

- Practice content has no `userId`; learner results never reach the server.
- One AI request per generation: invalid content fails permanently; audio retries never call the AI.
- Audio URLs include the practice version (immutable cache).
- Only commit when the user asks.

## Ghi chú triển khai

- T042 (Docker, AI tắt) phát hiện lỗi: `jobs.targetId` chỉ nhận ObjectID, nên job `practice_audio` mang version dạng chuỗi
  không xếp hàng được, và job `practice` bị retry (gọi lại AI). Đã sửa: job audio không mang version, luôn dùng version hiện
  tại; đã lưu phần luyện tập thì lỗi xếp hàng chỉ ghi log. Có test `TestProcessPracticeKeepsSavedPracticeWhenAudioCannotBeQueued`
  và `TestJobDocPracticeAudio`.
- T042 đã chạy: tạo bài → chú thích lỗi "AI chưa được cấu hình", không sinh job luyện tập; Tạo lại trước khi có chú thích → 409
  `annotation_not_done`; sau khi sửa tay chú thích → 202, rồi failed "AI chưa được cấu hình", chú thích giữ nguyên; người học
  mở bài chưa tới lượt → 403 `lesson_locked`; chưa đăng nhập → 401; chèn phần luyện tập bằng mongosh + job `practice_audio` →
  Kokoro tạo đủ mp3, `GET /practice` trả đủ ví dụ, hội thoại, ô trống, câu dịch kèm `audioUrl`; route audio 200 `audio/mpeg`,
  sai kind/index → 404; sửa nội dung bài → phần luyện tập bị xoá (`none`). Dữ liệu thử (bài, 2 tài khoản) đã dọn.
- Frontend lệch nhẹ so với research: output `timeChange` (lint cấm tên trùng sự kiện DOM `timeupdate`), các bước phát audio qua
  output `playAudio`; bước 2 nhận chính `lu-audio-player` của trang để phát lần lượt. Interceptor không chuyển sang
  `/forbidden` khi lỗi là `lesson_locked`, để các trang bài hiện "Bài này sẽ mở khi tới lượt".
- Đáp án điền/ghép nằm trong component của từng bước (`linkedSignal`); tab Bài đọc chỉ ẩn tab Bài học nên đổi tab không mất bài
  đang làm.
- Lint backend: đã chuyển các file `.go` trong working tree về LF (nội dung git không đổi) vì gofumpt báo lỗi với CRLF.
- T043, T044 để người dùng làm (cần khoá Gemini, kiểm tra 360px sáng/tối, trình đọc màn hình).
