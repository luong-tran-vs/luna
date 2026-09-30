---

description: "Task list for F2 – Trang quản trị bài học"
---

# Tasks: Trang quản trị bài học (F2)

**Input**: Design documents from `specs/003-lesson-admin/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: unit tests for business logic (sentence splitting,
AI result cleaning, worker retries, revision guard), endpoint tests (success, invalid input, forbidden), no network
(fake repositories, fake `Synthesizer`/`Provider`, `httptest.Server` for Kokoro/Gemini clients). Every acceptance
criterion maps to a test task or a manual check task (quickstart.md step numbers).

**Organization**: Tasks are grouped by user story (US1–US5 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)
- Paths: `backend/`, `frontend/`, `deploy/` at repo root (see plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Cấu hình, phụ thuộc, dịch vụ TTS

- [X] T001 Extend `backend/internal/platform/config/config.go` and `config_test.go` with `TTSURL` (`TTS_URL`, default `http://localhost:8880`, must parse as http/https URL), `TTSVoice` (`TTS_VOICE`, default `af_heart`), `AudioDir` (`AUDIO_DIR`, default `./data/audio`), `AIProvider` (`AI_PROVIDER`, `gemini|none`, default `gemini`), `GeminiAPIKey` (`GEMINI_API_KEY`, optional), `GeminiModel` (`GEMINI_MODEL`, default `gemini-3.5-flash-lite`); error messages per contracts/config.md; key never appears in errors
- [X] T002 [P] Install `@angular/cdk@^21` in `frontend/package.json` (use `npx -y npm@11 install` if npm 10 fails, see README)
- [X] T003 [P] Update `deploy/docker-compose.yml`: add service `kokoro` (`ghcr.io/remsky/kokoro-fastapi-cpu` pinned to a concrete release tag, `restart: unless-stopped`, port `127.0.0.1:8880:8880`); backend gets volume `audio-data:/data/audio`, env `TTS_URL: http://kokoro:8880`, `AUDIO_DIR: /data/audio`, `AI_PROVIDER: ${AI_PROVIDER:-gemini}`, `GEMINI_API_KEY: ${GEMINI_API_KEY:-}`, `GEMINI_MODEL: ${GEMINI_MODEL:-gemini-3.5-flash-lite}`, `depends_on: kokoro`; declare volume `audio-data`
- [X] T004 [P] Update `backend/Dockerfile` so `/data/audio` exists owned by uid 65532 in the final distroless image (create in build stage, `COPY --chown=65532:65532`) (research R11)
- [X] T005 [P] Add new variables with comments to `backend/.env.example` (local: `TTS_URL=http://localhost:8880`, `AUDIO_DIR=./data/audio`, `AI_PROVIDER`, `GEMINI_API_KEY=`, `GEMINI_MODEL`) and `deploy/.env.example` (`AI_PROVIDER`, `GEMINI_API_KEY=`, `GEMINI_MODEL`); add `backend/data/` to root `.gitignore`
- [X] T006 Remove F1 placeholder API: delete `backend/internal/admin/` and the `GET /api/admin/ping` route in `backend/cmd/api/main.go` (replaced by lesson API)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Kiểu dữ liệu, tách câu, kiểm tra đầu vào, repository, hàng đợi job; kiểu và API client frontend

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Backend

- [X] T007 Create `backend/internal/lesson/model.go`: `Level` (A1…C2 + `ValidLevel`), `Status` (`running|done|failed`), `JobKind` (`tts|annotate`), `Sentence`, `Annotation`, `Lesson` (incl. `Revision`, statuses, errors), `Summary`, `Filter{Level, Topic}`, `Roadmap{Lessons []Summary; Remaining int; Warning bool}`, errors `ErrNotFound`, `ErrInRoadmap`, `ErrNotFailed`, `ErrAnnotationRunning`, `*ValidationError{Fields map[string]string}` (data-model.md)
- [X] T008 Create `backend/internal/lesson/repository.go` with `Repository` and `RoadmapRepository` exactly as data-model.md §5
- [X] T009 [P] Write `backend/internal/lesson/split_test.go` (table-driven, research R6): spec example → 3 sentences; titles `Mr./Mrs./Dr./Prof.` never split; initials `J. K. Rowling`; decimals `9.30`, `$3.50`; `a.m.`/`U.S.` split only before a capital; `e.g.`/`i.e.`/`vs.` never split; `...` and `…` split before capital only; `"Stop!" she said.` stays one sentence; closing quote/paren after `.`; blank line forces split; newlines inside a paragraph and extra spaces normalized; empty/whitespace → no sentences
- [X] T010 Implement `SplitSentences(text string) []string` in `backend/internal/lesson/split.go` so T009 passes
- [X] T011 [P] Write `backend/internal/lesson/validate_test.go`: `ValidateInput` trims fields; errors (Vietnamese) for empty/too long title (200), content (10 000 runes, and 0 sentences after split, >200 sentences), invalid level, topic > 60, empty/too long source (200), license (100); all field errors returned together
- [X] T012 Implement `Input` struct and `ValidateInput(in Input) (Input, []string, error)` (returns trimmed input and split sentences) in `backend/internal/lesson/validate.go` (research R8) so T011 passes
- [X] T013 Create `backend/internal/job/model.go` (`Job{ID, Type, LessonID, Revision, Status, Attempts, Error, RunAt, CreatedAt, UpdatedAt}`, types `tts|annotate`, statuses `pending|running|done|failed`, `MaxAttempts = 3`, `ErrPermanent` wrapper helper `Permanent(err)`) and `backend/internal/job/repository.go` (`Repository` as data-model.md §5)
- [X] T014 [P] Create in-memory fakes: `backend/internal/lesson/fake_test.go` (lessons, roadmap, job enqueuer, fake `tts.Synthesizer`/`ai.Provider` placeholders filled in US2) and `backend/internal/job/fake_test.go` (job repository with `ClaimNext` by `runAt`, fake clock)
- [X] T015 [P] Implement MongoDB repositories: `backend/internal/storage/mongo/lessons.go` (`lesson.Repository`; `SaveAudio`/`SaveAnnotations` use filter `{_id, revision}` and return false when no match; `List` projects summary fields, sorted `createdAt` desc, filters level/topic, computes `inRoadmap` from roadmap ids passed by service), `backend/internal/storage/mongo/roadmap.go` (single doc `_id:"main"`), `backend/internal/storage/mongo/jobs.go` (`job.Repository`; `ClaimNext` via `FindOneAndUpdate` `{status:pending, runAt≤now}` sort `runAt`, `$set status:running`, `$inc attempts`, return after); extend `backend/internal/storage/mongo/indexes.go` with `lessons.createdAt`, `lessons.level`, `lessons.topic`, `jobs{status,runAt}`, `jobs.lessonId`

### Frontend

- [X] T016 [P] Create lesson types in `frontend/src/app/core/models/lesson.ts` (data-model.md §6)
- [X] T017 [P] Write `frontend/src/app/features/admin/admin-api.service.spec.ts` and implement `AdminApiService` (`ng g s features/admin/admin-api --type=service`) in `frontend/src/app/features/admin/admin-api.service.ts`: `list(filter)`, `get(id)`, `create(input)`, `update(id, input)`, `remove(id)`, `retry(id, job)`, `saveAnnotations(id, items)`, `getRoadmap()`, `saveRoadmap(ids)` — URLs, methods and bodies per contracts/admin-lessons-api.md
- [X] T018 [P] Create `StatusChip` component (`ng g c features/admin/status-chip`) in `frontend/src/app/features/admin/status-chip/` with input `status: JobStatus` and `label` ("Audio"/"Chú thích"): text "Đang chạy"/"Xong"/"Lỗi" + icon (spinner/✓/!, `aria-hidden`), color only on icon/border via `--color-text-muted`/`--color-ok`/`--color-bad`, spinner respects `prefers-reduced-motion`; spec in `status-chip.spec.ts`
- [X] T019 Create `frontend/src/app/features/admin/admin.routes.ts` (empty `Routes` for now) and change `admin` in `frontend/src/app/app.routes.ts` to `loadChildren` with `canActivate: [authGuard, adminGuard]`; delete F1 placeholder `frontend/src/app/features/admin/admin.{ts,html,css,spec.ts}`; update `frontend/src/app/app.spec.ts` admin test accordingly

**Checkpoint**: `go test ./...` and `npm test -- --watch=false` pass

---

## Phase 3: User Story 1 - Tạo bài học và xem trước (Priority: P1) 🎯 MVP

**Goal**: Tạo bài, danh sách có lọc, xem trước các câu đã tách

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T020 [P] [US1] Write `backend/internal/lesson/service_test.go` create/read cases: `Create` validates, stores trimmed fields, `Revision=1`, sentences from splitter, both statuses `running`, enqueues exactly one `tts` and one `annotate` job with revision 1 and calls notifier; invalid input → `*ValidationError` and nothing stored; `Get` unknown → `ErrNotFound`; `List` filters by level/topic, newest first, `inRoadmap` set from roadmap
- [X] T021 [P] [US1] Write `backend/internal/lesson/handler_test.go` (mux with fake resolver like F1 admin test): `POST /api/admin/lessons` 201 body per contract, 400 with `fields`; `GET /api/admin/lessons?level=B1` filters, `level=Z9` → 400; `GET /api/admin/lessons/{id}` 200 / 404 (unknown and malformed id); learner → 403; anonymous → 401
- [X] T022 [P] [US1] Write `frontend/src/app/features/admin/lesson-list/lesson-list.spec.ts`: renders rows with title, level, topic, two status chips, link to detail; level/topic filters call `list` with params; empty state "Chưa có bài học"; "Thêm bài" link to `/admin/lessons/new`
- [X] T023 [P] [US1] Write `frontend/src/app/features/admin/lesson-form/lesson-form.spec.ts` (create mode): Vietnamese errors for required title/content/level/source/license and max lengths; posts trimmed values; server `fields` shown under inputs; success navigates to `/admin/lessons/{id}`; submit disabled while pending
- [X] T024 [P] [US1] Write `frontend/src/app/features/admin/lesson-detail/lesson-detail.spec.ts` (preview): shows title, level, topic, source, license, numbered sentences in order, both status chips; 404 → "Không tìm thấy bài học"

### Implementation for User Story 1

- [X] T025 [US1] Implement `Service` in `backend/internal/lesson/service.go`: `NewService(lessons Repository, roadmap RoadmapRepository, jobs job.Repository, notify func(), now func() time.Time)`, `Create`, `Get`, `List` (research R1, R2) so T020 passes
- [X] T026 [US1] Implement `Handler` in `backend/internal/lesson/handler.go` (`List`, `Create`, `Get`; JSON per contract with `audioUrl` null until set; error mapping `ValidationError`→400 `validation_failed`, `ErrNotFound`→404 `not_found`) and register routes behind `requireAdmin` in `backend/cmd/api/main.go` (build lesson service with Mongo repos) so T021 passes
- [X] T027 [US1] Generate `ng g c features/admin/lesson-list` and implement in `frontend/src/app/features/admin/lesson-list/`: header with "Thêm bài" and "Lộ trình" links, filters (level select, topic input with debounce), card list (one column at 360px) with title link, level/topic tags, `StatusChip` ×2, tokens only, so T022 passes
- [X] T028 [US1] Generate `ng g c features/admin/lesson-form` and implement create mode in `frontend/src/app/features/admin/lesson-form/` (reactive form reusing `src/styles/forms.css` classes; `textarea` for content with character counter; level `select`; license/source inputs with hints) so T023 passes
- [X] T029 [US1] Generate `ng g c features/admin/lesson-detail` and implement preview in `frontend/src/app/features/admin/lesson-detail/` (info block, ordered sentence list, status chips, "Sửa" link placeholder hidden until US5) so T024 passes
- [X] T030 [US1] Add routes `''` (lesson-list), `lessons/new` (lesson-form), `lessons/:id` (lesson-detail) with titles to `frontend/src/app/features/admin/admin.routes.ts`
- [X] T031 [US1] Manual check: quickstart.md bước 1 (6 sentences split correctly, missing source rejected, filters)

**Checkpoint**: Tạo và xem trước bài hoạt động; job được xếp hàng (chưa có worker)

---

## Phase 4: User Story 2 - Audio và chú thích tự động chạy nền (Priority: P1)

**Goal**: Worker chạy TTS + AI, trạng thái, Chạy lại, phục vụ audio, polling

**Independent Test**: quickstart.md bước 2

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T032 [P] [US2] Write `backend/internal/tts/kokoro_test.go` with `httptest.Server`: request path `/v1/audio/speech`, JSON body `{model:"kokoro", input, voice, response_format:"mp3", speed:1}`; 200 returns bytes; 500, empty body, timeout (short client timeout) → error
- [X] T033 [P] [US2] Write `backend/internal/ai/gemini/gemini_test.go` with `httptest.Server`: URL `/v1beta/models/{model}:generateContent`, header `x-goog-api-key`, `generationConfig.responseMimeType` and `responseSchema` present, prompt contains level and numbered sentences; parses `candidates[0].content.parts[0].text`; 429 → error mentioning quota; 403 → `job.Permanent`-wrapped error; malformed JSON / no candidates → error; empty key → `ai.ErrNotConfigured` without any HTTP call
- [X] T034 [P] [US2] Write `backend/internal/lesson/annotate_test.go` (research R5): trims; drops items missing fields; finds sentence case-insensitively, falls back to first sentence containing text; drops text not in lesson; dedupes; all invalid → error
- [X] T035 [P] [US2] Write `backend/internal/job/worker_test.go` (fake repo + clock + handler map): success → `done`; error → back to `pending` with `runAt` +30 s then +2 m; third failure → `failed` and `OnFailed(job, err)` called; `job.Permanent` error → `failed` immediately; unknown type → failed; `Start` calls `ResetRunning`; `Notify` wakes an idle worker; cancelling ctx stops the loop
- [X] T036 [P] [US2] Add processing cases to `backend/internal/lesson/service_test.go` (temp `AudioDir`): `ProcessTTS` writes `{dir}/{id}/{rev}/{i}.mp3` via temp file + rename, skips existing files (synth not called again), sets `audioPath` and `done`, deletes older revision dirs, stale revision → nothing saved; `ProcessAnnotate` calls provider exactly once, saves cleaned annotations and `done`, stale revision discarded, `ai.ErrNotConfigured` → permanent error; `JobFailed` sets `failed` + message (Vietnamese: "AI chưa được cấu hình", "AI hết hạn mức", "Không tạo được audio"); `Retry` on `failed` enqueues new job and sets `running`, on `done`/`running` → `ErrNotFailed`
- [X] T037 [P] [US2] Write `backend/internal/lesson/audio_test.go`: behind `RequireAuth` fake → 401 without cookie; valid path serves bytes with `audio/mpeg` and immutable cache header; malformed id, negative/non-numeric revision or index, `..` → 404; missing file → 404
- [X] T038 [P] [US2] Add retry cases to `backend/internal/lesson/handler_test.go`: `POST /api/admin/lessons/{id}/retry?job=annotate` 202 when failed, 409 `not_failed`, 400 bad `job`, 404
- [X] T039 [P] [US2] Write `frontend/src/app/shared/utils/poll-while.spec.ts` (fake timers): re-fetches every 5 s while predicate true, stops when false, stops on unsubscribe
- [X] T040 [P] [US2] Extend `lesson-list.spec.ts` and `lesson-detail.spec.ts`: "Chạy lại" button only on failed status and calls `retry(id, 'tts'|'annotate')`; polling refreshes while any status running; detail shows ▶ button per sentence when `audioUrl` set (plays via shared `<audio>`), error reason text, read-only annotation table (text, lemma, meaning)

### Implementation for User Story 2

- [X] T041 [P] [US2] Create `backend/internal/tts/tts.go` (`Synthesizer` interface) and implement `Kokoro` client (`NewKokoro(baseURL, voice string, client *http.Client)`) in `backend/internal/tts/kokoro.go` so T032 passes
- [X] T042 [P] [US2] Create `backend/internal/ai/ai.go` (`Annotation`, `Provider`, `ErrNotConfigured`), `backend/internal/ai/disabled.go` (`Disabled` provider), and implement `backend/internal/ai/gemini/gemini.go` (`New(apiKey, model string, client *http.Client, log)`, baseURL overridable for tests, prompt per research R4, one `ai request` log line per call without key/content) so T033 passes
- [X] T043 [US2] Implement `CleanAnnotations(items []ai.Annotation, sentences []string) ([]Annotation, error)` and shared `findSentence` in `backend/internal/lesson/annotate.go` so T034 passes
- [X] T044 [US2] Implement `Worker` in `backend/internal/job/worker.go` (`NewWorker(repo, handlers map[Type]func(ctx, Job) error, onFailed func(ctx, Job, error), now, log)`, `Start(ctx)`, `Notify()`, poll 2 s, backoff 30 s / 2 m, `Permanent` handling) so T035 passes
- [X] T045 [US2] Implement `ProcessTTS`, `ProcessAnnotate`, `JobFailed`, `Retry` in `backend/internal/lesson/service.go` (inject `tts.Synthesizer`, `ai.Provider`, `audioDir`) so T036 passes
- [X] T046 [US2] Implement audio handler in `backend/internal/lesson/audio.go` (`GET /api/audio/{lessonId}/{revision}/{index}`, strict parsing, `http.ServeFile`) so T037 passes; add `Retry` handler to `backend/internal/lesson/handler.go` so T038 passes
- [X] T047 [US2] Wire in `backend/cmd/api/main.go`: `tts.NewKokoro(cfg.TTSURL, cfg.TTSVoice, …)`, provider by `cfg.AIProvider` (`gemini` with key → Gemini, else `ai.Disabled`), `job.NewWorker` with handlers `tts`→`svc.ProcessTTS`, `annotate`→`svc.ProcessAnnotate`, `onFailed`→`svc.JobFailed`, start worker (stops on ctx), pass `worker.Notify` to lesson service; routes `POST /api/admin/lessons/{id}/retry` (admin) and `GET /api/audio/{lessonId}/{revision}/{index}` (`requireAuth`); `os.MkdirAll(cfg.AudioDir)` at startup
- [X] T048 [US2] Implement `pollWhile<T>(load: () => Observable<T>, keepPolling: (value: T) => boolean, intervalMs = 5000)` in `frontend/src/app/shared/utils/poll-while.ts` so T039 passes
- [X] T049 [US2] Add retry buttons, polling (via `pollWhile` while any status `running`) to `lesson-list` and `lesson-detail`; add per-sentence play button with one shared `<audio>` element, error reasons and read-only annotation table to `lesson-detail` so T040 passes
- [ ] T050 [US2] Manual check: quickstart.md bước 2 (auto status update ≤ 10 s, play each sentence, no audio regeneration, AI not configured → failed then retry with key → done and exactly 1 `ai request` log, Kokoro stopped → failed then retry, restart mid-job resumes)

**Checkpoint**: US1 + US2 = bài học đầy đủ dữ liệu cho F3/F4

---

## Phase 5: User Story 3 - Xếp lộ trình (Priority: P2)

**Goal**: Thêm, gỡ, đổi thứ tự lộ trình; cảnh báo dưới 3 bài

**Independent Test**: quickstart.md bước 3

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T051 [P] [US3] Add roadmap cases to `backend/internal/lesson/service_test.go`: empty roadmap → `Remaining 0, Warning true`; `GetRoadmap` returns summaries in stored order (skipping ids of deleted lessons); `SetRoadmap` rejects duplicates and unknown ids with `*ValidationError`; 3 lessons → no warning, 2 → warning
- [X] T052 [P] [US3] Add to `backend/internal/lesson/handler_test.go`: `GET /api/admin/roadmap` 200 shape; `PUT` 200 / 400 duplicate / 400 unknown id; learner 403
- [X] T053 [P] [US3] Write `frontend/src/app/features/admin/roadmap/roadmap.spec.ts`: renders lessons in order with position numbers and status chips; warning text "Lộ trình chỉ còn N bài chưa học" (and "Lộ trình chưa có bài" when empty); Up/Down buttons reorder and call `saveRoadmap` with new order (first Up / last Down disabled); drop event (`CdkDragDrop`) reorders and saves; remove calls save without the id; save error reverts order and shows `role="alert"`
- [X] T054 [P] [US3] Extend `lesson-list.spec.ts`: "Thêm vào lộ trình" shown only when `!inRoadmap`, appends id and saves; "Trong lộ trình" badge otherwise; roadmap warning banner on the list page

### Implementation for User Story 3

- [X] T055 [US3] Implement `GetRoadmap` and `SetRoadmap` in `backend/internal/lesson/service.go` (research R7) so T051 passes
- [X] T056 [US3] Implement roadmap handlers in `backend/internal/lesson/handler.go` and routes `GET/PUT /api/admin/roadmap` in `backend/cmd/api/main.go` so T052 passes
- [X] T057 [US3] Generate `ng g c features/admin/roadmap` and implement in `frontend/src/app/features/admin/roadmap/` with `CdkDropList`/`CdkDrag`/`CdkDragHandle` (44px handle), Up/Down/Gỡ buttons with `aria-label` incl. lesson title, optimistic update + revert on error, warning banner, tokens only, so T053 passes
- [X] T058 [US3] Add "Thêm vào lộ trình" action, badge and warning banner to `frontend/src/app/features/admin/lesson-list/` (loads roadmap alongside list) so T054 passes; add route `roadmap` to `frontend/src/app/features/admin/admin.routes.ts`
- [ ] T059 [US3] Manual check: quickstart.md bước 3 (drag with handle and persist after reload, keyboard Up/Down, warning under 3, no duplicates)

**Checkpoint**: Lộ trình dùng được cho L

---

## Phase 6: User Story 4 - Xem và sửa chú thích (Priority: P2)

**Goal**: Sửa, thêm, xoá chú thích; đánh dấu "đã sửa tay"

**Independent Test**: quickstart.md bước 4

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- [X] T060 [P] [US4] Add `UpdateAnnotations` cases to `backend/internal/lesson/service_test.go`: new item → `editedByAdmin=true` and correct `sentenceIndex`; unchanged item keeps its flag; changed `meaningVi` or `lemma` → `true`; removed item gone; text not in lesson → `*ValidationError` with key `annotations.N.text`; empty lemma/meaning → validation; > 100 items → validation; status `running` → `ErrAnnotationRunning`
- [X] T061 [P] [US4] Add `PUT /api/admin/lessons/{id}/annotations` cases to `backend/internal/lesson/handler_test.go`: 200, 400 fields, 409 `annotation_running`, 404, learner 403
- [X] T062 [P] [US4] Extend `lesson-detail.spec.ts`: "Sửa chú thích" switches the table to editable rows (text read-only for existing rows, lemma/meaning inputs, delete button), "Thêm chú thích" adds an empty row, save sends full list, server field error shown on the right row, "Đã sửa tay" label shown for `editedByAdmin`, cancel restores; editing disabled while annotation status is running

### Implementation for User Story 4

- [X] T063 [US4] Implement `UpdateAnnotations` in `backend/internal/lesson/service.go` (reuse `findSentence`) so T060 passes
- [X] T064 [US4] Implement annotations handler in `backend/internal/lesson/handler.go` and route in `backend/cmd/api/main.go` so T061 passes
- [X] T065 [US4] Implement annotation editor in `frontend/src/app/features/admin/lesson-detail/` (reactive `FormArray`; stacked card layout below 640px instead of a wide table) so T062 passes
- [ ] T066 [US4] Manual check: quickstart.md bước 4

---

## Phase 7: User Story 5 - Sửa và xoá bài (Priority: P3)

**Goal**: Sửa thông tin/nội dung (làm lại khi nội dung đổi, cảnh báo chú thích sửa tay); xoá bài ngoài lộ trình

**Independent Test**: quickstart.md bước 5

### Tests for User Story 5 (REQUIRED - constitution VI) ⚠️

- [X] T067 [P] [US5] Add `Update`/`Delete` cases to `backend/internal/lesson/service_test.go`: info-only change keeps revision, sentences, annotations, statuses and enqueues nothing; content change → `revision+1`, new sentences, annotations cleared, both `running`, pending jobs of the lesson deleted, two new jobs with new revision; `Delete` in roadmap → `ErrInRoadmap`; otherwise removes lesson, its jobs and `{AudioDir}/{id}`; unknown id → `ErrNotFound`
- [X] T068 [P] [US5] Add `PUT` and `DELETE /api/admin/lessons/{id}` cases to `backend/internal/lesson/handler_test.go`: 200 / 400 / 404; delete 204, 409 `lesson_in_roadmap`, 404; learner 403
- [X] T069 [P] [US5] Extend `lesson-form.spec.ts` (edit mode at `lessons/:id/edit`): prefilled values; saving with changed content and any `editedByAdmin` annotation opens in-page confirm dialog "Các chú thích đã sửa tay sẽ bị thay mới" (cancel → no request, confirm → PUT); no dialog when content unchanged or no manual edits
- [X] T070 [P] [US5] Extend `lesson-detail.spec.ts`: "Sửa" link to edit page; "Xoá" opens confirm dialog; 409 shows "Gỡ bài khỏi lộ trình trước khi xoá"; success navigates to `/admin`

### Implementation for User Story 5

- [X] T071 [US5] Implement `Update` and `Delete` in `backend/internal/lesson/service.go` (jobs `DeletePending`/`DeleteForLesson`, `os.RemoveAll` audio dir, notify worker) so T067 passes
- [X] T072 [US5] Implement update/delete handlers in `backend/internal/lesson/handler.go` and routes in `backend/cmd/api/main.go` so T068 passes
- [X] T073 [US5] Add edit mode and confirm dialog (`<dialog>`-based, focus trapped, Esc cancels) to `frontend/src/app/features/admin/lesson-form/`; add route `lessons/:id/edit` to `frontend/src/app/features/admin/admin.routes.ts` so T069 passes
- [X] T074 [US5] Add "Sửa" and "Xoá" actions with confirm dialog to `frontend/src/app/features/admin/lesson-detail/` so T070 passes
- [X] T075 [US5] Manual check: quickstart.md bước 5 (title edit keeps audio files; content edit warns, rebuilds, new revision dir, old dir removed; delete blocked in roadmap, allowed after removal and audio dir gone)

**Checkpoint**: Mọi user story hoạt động độc lập

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T076 Update `README.md`: Kokoro service (first pull size, CPU only), how to get a free Gemini key and set `GEMINI_API_KEY` in `deploy/.env`/`backend/.env`, `AI_PROVIDER=none`, new config rows, audio volume, running Kokoro alone for local dev (`up -d mongo kokoro`)
- [X] T077 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T078 Run all tests (`go test -race ./...`, `npm test -- --watch=false`) with zero failures
- [X] T079 Rebuild and smoke test in Docker (`up --build -d`): create lesson via API, audio `done` with files on the volume, annotation `failed` without key, retry flows, learner 403 / anonymous 401 on admin API and audio (quickstart.md bước 2, 6)
- [X] T080 Compute WCAG contrast for new color pairs (chips, drag handle, dialog, warning banner) in both modes; all text ≥ 4.5:1, icons/borders ≥ 3:1
- [ ] T081 Manual check at 360px width in both light and dark mode (quickstart.md bước 7): list, form, detail with annotation editor, roadmap, dialogs; no horizontal scroll; touch targets ≥ 44px
- [ ] T082 Tick the 5 F2 acceptance criteria in `docs/phases/giai-doan-1.md` once T031, T050, T059, T066, T075, T079, T081 pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001, T006 first; T002–T005 parallel
- **Foundational (Phase 2)**: after Setup. Backend T007 → T008 → (T009–T012 parallel pairs) ; T013 → T014/T015. Frontend T016–T018 parallel → T019
- **US1 (Phase 3)**: after Foundational
- **US2 (Phase 4)**: after US1 (processes lessons created by US1 service; detail/list pages extended)
- **US3 (Phase 5)**: after US1 (needs lessons + list page); independent of US2
- **US4 (Phase 6)**: after US2 (annotations come from the AI job; editor extends detail page)
- **US5 (Phase 7)**: after US2 + US3 (re-enqueues jobs; delete checks roadmap)
- **Polish (Phase 8)**: after all stories

### Within Each User Story

- Tests written first and FAIL before implementation
- Backend: clients/helpers → service → handler → wiring in `main.go`
- Frontend: service method → component → route
- Manual check task last

### Parallel Opportunities

- Setup: T002 ∥ T003 ∥ T004 ∥ T005
- Foundational: T009 ∥ T011 ∥ T014 ∥ T015 (backend) ∥ T016 ∥ T017 ∥ T018 (frontend)
- US1: all 5 test tasks T020–T024 parallel; backend T025–T026 ∥ frontend T027–T030
- US2: T032–T040 parallel tests; T041 ∥ T042 (TTS and AI clients)
- US3 can run in parallel with US2 once US1 is done

---

## Parallel Example: User Story 2

```bash
# Independent clients and helpers, tests first:
Task: "Kokoro client test + implementation (backend/internal/tts/)"
Task: "Gemini client test + implementation (backend/internal/ai/gemini/)"
Task: "CleanAnnotations test + implementation (backend/internal/lesson/annotate.go)"
Task: "Worker test + implementation (backend/internal/job/worker.go)"
Task: "pollWhile test + implementation (frontend/src/app/shared/utils/poll-while.ts)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup → Phase 2 Foundational
2. Phase 3 US1 → **STOP and VALIDATE** with quickstart bước 1

### Incremental Delivery

1. US1 → tạo và xem bài
2. US2 → audio + chú thích nền (bài đủ dữ liệu cho F3/F4)
3. US3 → lộ trình (đủ cho L)
4. US4 → sửa chú thích
5. US5 → sửa, xoá bài
6. Polish → README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- [P] tasks = different files, no dependencies
- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` trong `go.mod` (không `go get` bản yêu cầu Go 1.26)
- Không log `GEMINI_API_KEY` hay toàn bộ nội dung bài
- Code, API, commit message bằng tiếng Anh; chữ trên giao diện bằng tiếng Việt
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai (2026-09-29)

**Khác với mô tả task** (không đổi hành vi):
- Kokoro ghim `ghcr.io/remsky/kokoro-fastapi-cpu:v0.9.0` (bản mới nhất lúc triển khai, image 4,95 GB).
- `ai.Disabled` nằm trong `backend/internal/ai/ai.go` (không tách `disabled.go`). Lỗi AI dùng sentinel
  `ai.ErrNotConfigured`, `ai.ErrInvalidKey`, `ai.ErrQuota`; `lesson.ProcessAnnotate` đổi hai lỗi đầu thành
  `job.Permanent` để package `ai` không phụ thuộc `job`.
- Repository bài học tách `UpdateInfo` và `ReplaceContent` (thay cho một `Update`) để sửa thông tin không ghi đè
  kết quả audio/chú thích mà worker vừa lưu; `Summaries(ids)` thay cho `Exists`.
- `Handler.Register(mux, requireAuth, audioDir)` đăng ký cả route quản trị và `/api/audio/...`.
- Lưu chú thích sửa tay khi chú thích đang "lỗi" sẽ chuyển trạng thái sang "xong" (quản trị viên đã tự soạn).
- Hộp thoại xác nhận là component dùng chung `shared/components/confirm-dialog` (không dùng `<dialog>` vì jsdom
  chưa hỗ trợ `showModal`); style chung trang quản trị ở `src/styles/admin.css`.
- Nhãn nút lộ trình dạng "Lên: <tên bài>", "Xuống: …", "Gỡ khỏi lộ trình: …"; sau khi đổi chỗ bằng bàn phím,
  focus giữ trên bài vừa di chuyển.
- Thứ tự TDD: `lesson/service.go` được viết trước `service_test.go` (test viết ngay sau, bao phủ đủ các task).

**Kết quả kiểm tra**
- Backend: `go test -race ./...` qua; golangci-lint 0 issues; `go.mod` vẫn `go 1.25.1`.
- Frontend: 134 test Vitest qua; `ng lint` sạch; `ng build` thành công (lộ trình là chunk lazy riêng vì CDK).
- Docker thật (Kokoro CPU, không khoá Gemini), qua nginx:
  - Tạo bài mẫu quickstart → đúng 6 câu như mong đợi; audio 6 câu "xong" sau ~7 giây; file mp3 hợp lệ (`ID3`),
    `Content-Type: audio/mpeg`, cache immutable; không cookie → 401.
  - Chú thích "lỗi: AI chưa được cấu hình"; chạy lại → lại "lỗi" (không thử lại vô ích); log không có request AI
    nào (không có khoá thì không gọi mạng). Chạy lại audio khi đang "xong" → 409.
  - Sửa tiêu đề → revision 1, audio giữ; sửa nội dung → revision 2, audio mới trong thư mục `2/`, thư mục `1/` bị xoá.
  - Lộ trình 1 bài → cảnh báo; xoá bài trong lộ trình → 409; gỡ rồi xoá → 204 và thư mục audio biến mất.
  - Khởi động lại backend khi bài 20 câu đang tạo audio → log "resumed interrupted jobs", xong đủ 20 file, không
    còn file tạm; bài 20 câu có audio sau ~45 giây (mục tiêu ≤ 5 phút).
  - Learner → 403, ẩn danh → 401 trên `/api/admin/lessons`.
- Độ tương phản các cặp màu mới (chip, nút xoá, tag, nút phụ, cảnh báo, tay nắm) đạt AA ở cả hai chế độ.

**Chưa kiểm tra được / còn lại**
- Gọi Gemini thật: chưa có `GEMINI_API_KEY` (client đã test với server giả theo đúng định dạng API).
- Tình huống dừng container Kokoro rồi chờ hết 3 lần thử (~2,5 phút): chỉ kiểm tra bằng unit test worker.
- Kiểm tra bằng trình duyệt: T050 (tự cập nhật, nghe câu), T059 (kéo thả), T066 (sửa chú thích), T081 (360px), rồi T082.
