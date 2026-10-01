---

description: "Task list for F7 – AI sinh bài học"
---

# Tasks: AI sinh bài học (F7)

**Input**: Design documents from `specs/012-ai-lesson-generation/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/generate-api.md, quickstart.md

**Tests**: REQUIRED by constitution principle VI and khối 2:
- Gemini client: schema, prompt, 429.
- Draft filtering and input validation.
- AI error mapping.
- Create + `appendToRoadmap` with compensation.
- Handler: 200 / 400 / 403 / 404 / 503 / 429 / 502.
- Dialog validation, save one / save all, AI error keeps drafts, leave guard.

Real-AI behaviour (level, length, variety) and 360px / keyboard are checked by hand (quickstart.md).

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Add a location for the generate route to `frontend/nginx.conf` (research R2):
  - `location ~ ^/api/admin/topics/[^/]+/generate$` with the same `proxy_pass $backend`, proxy headers and `proxy_connect_timeout 3s` as `/api/`.
  - `proxy_read_timeout 75s`.
  - A comment explaining the 60 s AI call.
- [X] T002 [P] Create `frontend/src/app/core/models/generate.ts`:
  - `LessonKind = 'reading' | 'dialogue'`, `GenerateInput {count, words, kind, idea}`, `GeneratedDraft {title, content, words}`, `GenerateResult {drafts, requested, dropped}`.
  - Limits `MIN_COUNT 1`, `MAX_COUNT 5`, `MIN_WORDS 50`, `MAX_WORDS 800`, `MAX_IDEA 500`, `DEFAULT_COUNT 3`.
  - `DEFAULT_WORDS: Record<Level, number>` (A1 120, A2 160, B1 220, B2 300, C1 400, C2 400).
  - `AI_SOURCE = 'AI sinh'`, `AI_LICENSE = 'Nội dung do AI tạo'`.
  - `countWords(text)`: whitespace-separated tokens, 0 for blank.
  - Spec `generate.spec.ts`: `countWords` on blank, multiple spaces, newlines, dialogue lines; `DEFAULT_WORDS` covers every level.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: The `ai.Provider` change breaks every implementation until T003–T006 are done.

- [X] T003 Extend `backend/internal/ai/ai.go` per data-model.md:
  - `LessonKind` with `KindReading = "reading"`, `KindDialogue = "dialogue"` and `ValidKind`.
  - `GenerateRequest {Level, TopicName string; Count, Words int; Kind LessonKind; Idea string; ExistingTitles []string}`.
  - `LessonDraft {Title, Content string}` with JSON tags `title`, `content`.
  - `Provider.GenerateLessons(ctx, GenerateRequest) ([]LessonDraft, error)`.
  - `Disabled.GenerateLessons` returns `ErrNotConfigured`.
  - Update the package doc (annotation and lesson generation).
- [X] T004 Update the fakes in `backend/internal/lesson/fake_test.go`:
  - The fake provider gets `GenerateLessons`: configurable `drafts`, `err`, optional `delay`/blocking until ctx done; records the last `GenerateRequest` and the call count.
  - The fake `Topics` gets `AppendLesson`: appends to an in-memory roadmap per topic, with a `failAppend` flag.
  - `go vet ./internal/lesson/...` compiles.
- [X] T005 Write Gemini tests in `backend/internal/ai/gemini/gemini_test.go`:
  - `GenerateLessons` sends one POST to `/v1beta/models/gemini-test:generateContent` with the key header.
  - `generationConfig.responseMimeType` is `application/json`; `responseSchema` is an ARRAY of OBJECT with required `title` and `content`; `temperature` is 0.9.
  - The prompt contains the level, topic name, count, word target, the existing titles, the idea when set; for `dialogue` it contains the `Name: line` rule.
  - The response is decoded into `[]ai.LessonDraft`.
  - Errors: 429 → `ai.ErrQuota`; 401 → `ai.ErrInvalidKey`; malformed inner JSON → error; empty key → `ai.ErrNotConfigured` with no HTTP call.
  - Existing Annotate tests still pass.
- [X] T006 Implement `GenerateLessons` in `backend/internal/ai/gemini/gemini.go` so T005 passes (research R3):
  - Extract the shared request/response/log code of `Annotate` into `c.generate(ctx, op string, prompt string, schema map[string]any, temperature float64, attrs ...slog.Attr) (string, error)`, returning the first candidate's text.
  - Add `generateSchema` and `generatePrompt(req)`, with the English prompt per research R3: target 0.9–1.1× words, CEFR vocabulary only, distinct titles ≤ 8 words, avoid listed titles, dialogue with 2–3 named speakers one line each, no markdown.
  - The log line has `op=generate`, `count`, `words`, never text or key.

**Checkpoint**: `go build ./...` and `go test ./internal/ai/...` pass.

---

## Phase 3: User Story 1 - Sinh bản nháp cho lộ trình của một chủ đề (Priority: P1) 🎯 MVP

**Goal**: Quản trị viên mở dialog trên lộ trình một chủ đề, sinh 1–5 bản nháp đúng trình độ, độ dài, dạng bài, không trùng bài đã có.

**Independent Test**: quickstart.md bước 1.

### Tests for User Story 1 (REQUIRED) ⚠️

- [X] T007 [US1] Write `backend/internal/lesson/generate_test.go`.
  - `ValidateGenerate` field errors with the Vietnamese messages of data-model.md: count 0/6, words 49/801, kind "poem", idea of 501 runes. 1/5, 50/800 and 500 runes are accepted.
  - `Service.Generate` with the fake provider:
    - passes level/topic name/count/words/kind/trimmed idea and the titles of every lesson of the topic (not other topics) to the provider;
    - drops empty title/content, duplicate titles inside the batch and titles matching an existing lesson (case, extra spaces, trailing punctuation ignored), word counts outside ±20%, and titles > 200 / content > 10000 chars;
    - `Dropped` counts them; extra drafts beyond `count` are cut and not counted;
    - CRLF is normalised and content trimmed; `Words` is filled;
    - all dropped → `ErrUnusableDraft`;
    - unknown topic → `ErrTopicNotFound`;
    - AI errors (`ErrQuota`, `ErrNotConfigured`) are returned unwrapped-comparable with `errors.Is`;
    - a provider that blocks returns `context.DeadlineExceeded` once the generate timeout passes (timeout injectable via `Service`/`Deps` field for the test).
- [X] T008 [US1] Write `backend/internal/lesson/generate_handler_test.go` for `POST /api/admin/topics/{id}/generate` (contracts/generate-api.md):
  - 200 `{drafts:[{title,content,words}], requested, dropped}`.
  - 400 `validation_failed` with `fields.count` for count 0 and 6.
  - 400 `invalid_body` for broken JSON.
  - 403 for a learner session; 401 without a session.
  - 404 `not_found` for an unknown topic.
  - 503 `ai_not_configured` for `ErrNotConfigured` and `ErrInvalidKey` (different messages).
  - 429 `ai_quota`.
  - 502 `ai_unusable`.
  - 502 `ai_failed` for other errors and for a deadline.

### Implementation for User Story 1

- [X] T009 [US1] Implement `backend/internal/lesson/generate.go` and add `ErrUnusableDraft` to `backend/internal/lesson/model.go` so T007 passes (research R4, R5):
  - `GenerateInput`, `ValidateGenerate`, `Draft`, `GenerateResult`.
  - `const generateTimeout = 60 * time.Second` with an overridable `GenerateTimeout` field in `Deps`.
  - `Service.Generate(ctx, topicID, in)`: `Topics.Get` → `Lessons.List(Filter{TopicID})` titles → `AI.GenerateLessons` under `context.WithTimeout` → `filterDrafts`.
  - Title key helper.
- [X] T010 [US1] Implement `backend/internal/lesson/generate_handler.go` and register `POST /api/admin/topics/{id}/generate` (admin) in `Handler.Register` in `backend/internal/lesson/handler.go`, so T008 passes:
  - Call `http.NewResponseController(w).SetWriteDeadline(time.Now().Add(75*time.Second))` first; ignore `http.ErrNotSupported`.
  - Decode the body, call `svc.Generate`.
  - Map errors per research R7 in `writeGenerateError`; `ValidationError` → field errors; `ErrTopicNotFound` → 404 "Không tìm thấy chủ đề"; unknown errors are logged.
  - Check `backend/cmd/api/main.go` still compiles (no new wiring needed: `Deps.AI` exists).
- [X] T011 [P] [US1] Add `generateLessons(topicId, input): Observable<GenerateResult>` (POST `${BASE}/topics/${topicId}/generate`) to `frontend/src/app/features/admin/admin-api.service.ts`, with a test in `admin-api.service.spec.ts` (URL, body).
- [X] T012 [US1] Write `frontend/src/app/features/admin/generate-dialog/generate-dialog.spec.ts`. Create the component with `ng g c features/admin/generate-dialog`.
  - Inputs: `open`, `topic` (name + level), `options` (initial `GenerateInput`), `busy`, `error` (string | null).
  - Outputs: `generate` (GenerateInput), `closed`.
  - When opened it calls `showModal()` on a native `<dialog>`; stub `HTMLDialogElement.prototype.showModal/close` in jsdom.
  - The heading shows "Sinh bài bằng AI" and "A1 · Gia đình"; fields are prefilled from `options`.
  - Invalid count (0, 6, 2.5), words (49, 801) and idea > 500 show the Vietnamese field errors on submit and do not emit.
  - Valid submit emits trimmed values.
  - While `busy`: the button reads "Đang sinh…", has `aria-busy="true"`, inputs are disabled, and Escape (`cancel` event) does not close.
  - `error` is shown in `role="alert"` and the values stay.
  - Cancel emits `closed`.
- [X] T013 [US1] Implement `generate-dialog.ts/.html/.css` so T012 passes (research R8):
  - Reactive form; number inputs with `inputmode="numeric"`; radio group "Bài đọc" / "Hội thoại"; textarea for ý chính with a character counter; hint "Khoảng 96–144 từ được chấp nhận" (±20% of the current value).
  - One column, buttons ≥ 44px, `max-width: min(560px, 100vw - 32px)`, no horizontal scroll at 360px.
- [X] T014 [US1] Write `frontend/src/app/features/admin/draft-list/draft-list.spec.ts` for the display part. Create the component with `ng g c features/admin/draft-list`.
  - Input `drafts: DraftState[]` (export `DraftState` from `draft-list.ts` per data-model.md).
  - Each draft shows an editable title input and content textarea with visible labels (`Tiêu đề bản nháp 1`…) and the live word count "118 từ".
  - Editing emits `changed {key, title?, content?}`.
  - Empty list renders nothing.
- [X] T015 [US1] Implement the display/edit part of `draft-list.ts/.html/.css` so T014 passes (cards, one column, textarea auto height via `rows` from line count, `countWords` from `core/models/generate.ts`).
- [X] T016 [US1] Extend `frontend/src/app/features/admin/roadmap/roadmap.spec.ts`:
  - **Sinh bài bằng AI** is shown only when a topic is selected.
  - Opening passes the topic and options with `DEFAULT_WORDS` of the topic level.
  - On `generate` the API is called with the topic id, `busy` is true during the call, and on success the dialog closes and the drafts appear in `lu-draft-list`.
  - `dropped > 0` or fewer drafts than requested shows a `role="status"` note ("Đã loại 1 bản không đạt yêu cầu").
  - The lesson list and roadmap are not reloaded or changed by generating.
  - The last options are kept when reopening.
- [X] T017 [US1] Implement the generate part of `frontend/src/app/features/admin/roadmap/roadmap.ts/.html/.css` so T016 passes:
  - Signals `dialogOpen`, `generating`, `generateError`, `options` (reset `words` to the level default when the selected topic changes level), `drafts`, `generateNote`.
  - The button goes next to the roadmap heading.

**Checkpoint**: sinh được bản nháp và sửa được trên trang; chưa lưu.

---

## Phase 4: User Story 2 - Duyệt, sửa và lưu bản nháp (Priority: P1)

**Goal**: Lưu / Bỏ / Lưu tất cả. Bài lưu có nguồn "AI sinh", giấy phép "Nội dung do AI tạo", vào cuối lộ trình trong cùng request, chạy audio và chú thích.

**Independent Test**: quickstart.md bước 2.

### Tests for User Story 2 (REQUIRED) ⚠️

- [X] T018 [P] [US2] Add `topic.Service.AppendLesson(ctx, topicID, lessonID)` tests to `backend/internal/topic/service_test.go`:
  - appends at the end;
  - no duplicate when already there;
  - unknown topic → `ErrNotFound`.
- [X] T019 [US2] Add `Create` tests to `backend/internal/lesson/service_test.go`:
  - With `AppendToRoadmap: true` the lesson is created, appended at the end of its topic roadmap (fake Topics), and TTS + annotate jobs are enqueued.
  - Without the flag nothing is appended (F2 behaviour unchanged).
  - When `AppendLesson` fails: the created lesson is deleted, no job is enqueued, and the error is returned.
  - Validation errors still come before any write.
- [X] T020 [US2] Add to `backend/internal/lesson/handler_test.go`: `POST /api/admin/lessons` with `appendToRoadmap: true` → 201 with `inRoadmap: true` and `audioStatus`/`annotationStatus` `running`; without it → `inRoadmap: false`; `PUT` ignores the field.

### Implementation for User Story 2

- [X] T021 [US2] Implement `AppendLesson` in `backend/internal/topic/service.go`: `repo.Get` for `ErrNotFound`, then `repo.AppendLesson`. T018 passes.
- [X] T022 [US2] Implement the save path so T019–T020 pass (research R6):
  - `AppendToRoadmap bool` in `lesson.Input` (`backend/internal/lesson/validate.go`) and `appendToRoadmap` in `inputJSON` (`backend/internal/lesson/handler.go`); keep `Input(in)` conversion valid.
  - `AppendLesson` in the `Topics` port (`backend/internal/lesson/repository.go`).
  - `Service.Create` in `backend/internal/lesson/service.go`: create → append (on error `Lessons.Delete` + log if that fails too) → enqueue.
  - `lessonTopicsPort.AppendLesson` in `backend/cmd/api/main.go`, mapping `topic.ErrNotFound` → `lesson.ErrTopicNotFound`.
- [X] T023 [P] [US2] Add `appendToRoadmap?: boolean` to `LessonInput` in `frontend/src/app/core/models/lesson.ts`.
- [X] T024 [US2] Extend `frontend/src/app/features/admin/draft-list/draft-list.spec.ts`:
  - Each draft has **Lưu** and **Bỏ** buttons emitting `save(key)` / `discard(key)`.
  - **Lưu tất cả** is shown only with ≥ 2 drafts and emits `saveAll`.
  - A draft with `saving` shows "Đang lưu…", disables its buttons and fields.
  - `fields.title` / `fields.content` show under the matching field with `aria-invalid` and `aria-describedby`.
  - `error` shows in `role="alert"` inside the card.
  - Buttons have accessible names including the draft number.
- [X] T025 [US2] Implement the actions of `draft-list.ts/.html/.css` so T024 passes.
- [X] T026 [US2] Extend `frontend/src/app/features/admin/roadmap/roadmap.spec.ts`.
  - **Lưu** posts `{title, content, topicId, source: AI_SOURCE, license: AI_LICENSE, appendToRoadmap: true}`, removes the draft on success and reloads roadmap + topic lessons.
  - **Bỏ** removes without any request.
  - **Lưu tất cả** posts the drafts one after another in display order (the second starts only after the first finishes), keeps the failed ones with their error (400 field errors via `ApiError` with `errorInterceptor`; network error → "Không lưu được, vui lòng thử lại."), continues after a failure, and reloads once at the end.
  - Saving the same draft twice while saving sends one request.
  - The topic list (`remaining`/warning) is refreshed from the reloaded roadmap.
- [X] T027 [US2] Implement save, discard and save-all in `frontend/src/app/features/admin/roadmap/roadmap.ts` so T026 passes. Reuse `load(id)` and update `topics` from the new roadmap summary.

**Checkpoint**: luồng chính F7 chạy được: sinh → duyệt → lưu vào cuối lộ trình.

---

## Phase 5: User Story 3 - AI lỗi không làm mất công sức (Priority: P2)

**Goal**: Thông báo đúng loại lỗi AI, giữ bản nháp và lựa chọn; hỏi xác nhận khi rời trang hoặc đổi chủ đề còn bản nháp.

**Independent Test**: quickstart.md bước 3.

### Tests for User Story 3 (REQUIRED) ⚠️

- [X] T028 [P] [US3] Create `frontend/src/app/core/guards/unsaved-changes.guard.ts` (`ng g guard core/guards/unsaved-changes --functional`, type CanDeactivate).
  - Export `interface CanLeave { canLeave(): boolean | Promise<boolean> }` and `unsavedChangesGuard: CanDeactivateFn<CanLeave>` delegating to it.
  - Spec `unsaved-changes.guard.spec.ts`: true / false / promise results are passed through.
- [X] T029 [US3] Extend `frontend/src/app/features/admin/roadmap/roadmap.spec.ts`.
  - **AI errors**: for each error code (`ai_not_configured`, `ai_quota`, `ai_unusable`, `ai_failed`, network error with no body → "Sinh bài thất bại, vui lòng thử lại."):
    - the dialog stays open and shows the server `message`;
    - the options entered are kept;
    - existing (edited) drafts are unchanged.
  - A second successful batch appends drafts after the old ones.
  - **Leaving**: `canLeave()` returns true with no drafts. With drafts it opens `lu-confirm-dialog` ("Rời trang?", "Các bản nháp chưa lưu sẽ mất.", confirm "Rời trang"); confirm resolves true, cancel resolves false and keeps drafts.
  - **Changing topic**: changing the topic select with drafts asks the same; cancel restores the previous selection and drafts, confirm clears drafts and loads the new topic.
  - **`beforeunload`**: calls `preventDefault()` only when drafts exist.

### Implementation for User Story 3

- [X] T030 [US3] Implement in `frontend/src/app/features/admin/roadmap/roadmap.ts/.html` so T029 passes:
  - error message from `ApiError.body.message` with a fallback;
  - `canLeave()` with a pending promise resolved by the confirm dialog outputs;
  - topic change confirmation;
  - `host: {'(window:beforeunload)': 'onBeforeUnload($event)'}`.
- [X] T031 [US3] Add `canDeactivate: [unsavedChangesGuard]` to the `roadmap` route in `frontend/src/app/features/admin/admin.routes.ts`.

**Checkpoint**: tất cả user story xong.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T032 [P] Add a section "Sinh bài bằng AI" to `README.md`, next to the admin / roadmap sections:
  - where the button is, the options and defaults;
  - drafts are not saved until Lưu and are lost when leaving;
  - source / license of saved lessons;
  - AI errors and the free quota (1 request per batch);
  - the 75 s nginx route note.
- [X] T033 Run lint (frontend `npm run lint`, backend golangci-lint via Docker) with zero issues.
- [X] T034 Run all tests (`CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`, `npx ng test --watch=false`) with zero failures; `npx ng build` without warnings.
- [X] T035 Rebuild Docker and smoke test with curl (quickstart.md bước 1–2).
  - Learner → 403; count 0 → 400.
  - With AI disabled → 503 `ai_not_configured` in the UI and the rest of the roadmap page works.
  - If a Gemini key is configured:
    - generate 3 A1 drafts of 120 words: all 96–144 words, titles distinct and not in the topic;
    - save one with `appendToRoadmap` → last in the roadmap, source "AI sinh", jobs run to done;
    - the drafts never appear in `GET /api/admin/lessons`.
- [ ] T036 Manual check: quickstart.md bước 1–4 in the browser (real AI quality per level and dialogue format, 360px, keyboard only, leave confirmation, F5 prompt).
- [ ] T037 Tick the 4 F7 acceptance criteria in `docs/phases/giai-doan-2.md` once T034–T036 pass.

---

## Dependencies & Execution Order

- **Setup** (T001, T002) → **Foundational** (T003 → T004, T005 → T006) → **US1** → **US2** → **US3** → **Polish**.
- US2 backend (T018–T022) depends only on Foundational and can run in parallel with US1 backend/frontend.
- US2 frontend needs the US1 roadmap/draft-list components.
- US3 depends on US1 (dialog and drafts state); T028 (guard) can be done any time.
- Within each story: tests → implementation; backend ∥ frontend.

### Parallel Opportunities

- T001 ∥ T002 ∥ T003.
- T005/T006 (gemini) ∥ T004 (fakes).
- T007–T010 (backend generate) ∥ T011–T015 (frontend API, dialog, draft list).
- T018–T022 (backend save path) ∥ T012–T017.
- T023, T028 at any time.

---

## Parallel Example: User Story 1

```bash
Task: "generate service tests in backend/internal/lesson/generate_test.go"
Task: "generate-dialog component + spec in frontend/src/app/features/admin/generate-dialog/"
Task: "draft-list display + spec in frontend/src/app/features/admin/draft-list/"
```

---

## Implementation Strategy

1. Setup + Foundational. Then US1: sinh bản nháp. **STOP and VALIDATE** with curl and the dialog (quickstart bước 1).
2. US2: lưu, bỏ, lưu tất cả. Now the full F7 flow works (quickstart bước 2).
3. US3: lỗi AI giữ bản nháp, chặn rời trang (quickstart bước 3).
4. Polish: README, lint, tests, Docker, manual checks, tick acceptance criteria.

---

## Notes

- `topic` stays AI-free; the generate route lives in `lesson.Handler` under `/api/admin/topics/{id}/generate`.
- Drafts are never written to the DB; only `POST /api/admin/lessons` writes.
- Log lines never contain lesson text or the API key.
- Only commit when the user asks.

---

## Ghi chú triển khai

- Hộp sinh bài dùng `<dialog>` gốc với `showModal()`; jsdom không có `showModal`/`close` nên các spec gán tạm hai hàm này
  trên `HTMLDialogElement.prototype`.
- Guard đặt tên `core/guards/unsaved-changes.guard.ts` cho giống `auth.guard.ts` (lệnh `ng g guard` sinh ra
  `unsaved-changes-guard.ts`).
- `lesson.Service.Generate` bọc mọi lỗi của lời gọi AI bằng `errGenerateAI`; handler nhờ đó phân biệt 502 `ai_failed` với 500
  (lỗi DB). Lỗi topic không tồn tại khi nối lộ trình cũng được ánh xạ về `lesson.ErrTopicNotFound` trong `main.go`.
- Lưu bản nháp có trường lỗi khác `title`/`content` (ví dụ `topicId`) thì hiện thành thông báo chung trong thẻ bản nháp.
- Smoke test Docker (T035), máy chưa có `deploy/.env` nên AI tắt:
  - admin → 503 `ai_not_configured`; người học → 403; `count: 0` → 400 `fields.count`; chủ đề không tồn tại → 404;
  - lưu với `appendToRoadmap: true` → 201, `inRoadmap: true`, nguồn "AI sinh", bài nằm cuối lộ trình (đã xoá bài thử sau đó);
  - nginx đã nạp location riêng cho `/generate`.
  Phần sinh bài với AI thật chưa chạy được vì chưa có `GEMINI_API_KEY`; gộp vào T036 cho người dùng.
- Còn lại cho người dùng: T036 (sinh thật với khoá Gemini: độ dài, trình độ, hội thoại, khác nhau; 360px; bàn phím; hỏi khi rời
  trang và F5) và T037 (tick 4 tiêu chí F7 sau khi T036 đạt).
