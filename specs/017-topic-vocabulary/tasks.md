---

description: "Task list for F18 – Từ vựng theo chủ đề"
---

# Tasks: Từ vựng theo chủ đề (F18)

**Input**: Design documents from `specs/017-topic-vocabulary/`

**Prerequisites**: plan.md, spec.md, research.md (R1–R9), data-model.md, contracts/topic-words-api.md, quickstart.md

**Tests**: REQUIRED by constitution principle VI and khối 2:
- word cleaning and validation, one-time seeding (no overwrite, unmatched topics), seed file valid;
- coverage (phrases, case, plural, -ed/-ing, irregular, annotation lemma, no partial match);
- word plan (unused first, least used next, no overlap across lessons);
- handlers (400 per word, 404, 403);
- Gemini prompt contains TargetWords / FocusWords;
- annotate adds missed topic words from the dictionary;
- Vitest: edit word list (add many, duplicate, delete, errors), generate dialog (groups, remove/add, 0 words), draft missing words,
  keyboard.

Real-AI quality, first startup on the real DB, 360px, light/dark are checked by hand (quickstart.md).

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Copy `docs/spec-inputs/f18-topic-words.json` to `backend/internal/topic/seed/topic-words.json`; in both files replace
  "Intensive care unit (ICU)" with "Intensive care unit" (research R2).
- [X] T002 [P] Extend frontend models (data-model.md › Frontend): `wordCount`, `usedWordCount`, `TopicWord` in
  `frontend/src/app/core/models/topic.ts`; `targetWords` in `GenerateInput`, `targetWords`/`missingWords` in `GeneratedDraft`,
  `DEFAULT_TARGET_WORDS` (A1–A2 8, B1–B2 10, C1–C2 12), `MAX_TARGET_WORDS = 15` in `frontend/src/app/core/models/generate.ts`; update
  existing fixtures in admin specs so they compile.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: word matching and the topic word fields are used by every story.

- [X] T003 [P] Write `backend/internal/wordmatch/wordmatch_test.go` (research R3): single word, phrase ("take a shower"), case
  ("FAMILY"), plural ("grandmothers" → "Grandmother"), -es/-ies, -ed/-ing ("watched", "watching"), irregular through
  `dictionary.Candidates` ("took a shower"), annotation lemma map ("went" → "go"), no partial match ("son" vs "season"), "/" splits
  ("and/or"), apostrophe and hyphen ("o'clock", "T-shirt", ’), `Find` returns the original span text.
- [X] T004 Implement `backend/internal/wordmatch/wordmatch.go` (`New`, `Find`, `Contains`, package doc) so T003 passes.
- [X] T005 Add `Words []string`, `WordsSeeded bool` to `topic.Topic` and `WordCount`, `UsedWordCount` to `topic.Summary`;
  `WordUse`, `LessonText` types in `backend/internal/topic/model.go`; `Repository.SetWords` and `Lessons.Texts` in
  `backend/internal/topic/repository.go`; update the topic fakes in `backend/internal/topic/*_test.go`.
- [X] T006 Storage in `backend/internal/storage/mongo/topics.go`: `topicDoc` fields `words`, `wordsSeeded`, `toTopic`,
  `SetWords` (`$set words, wordsSeeded=true, updatedAt`, `ReturnDocument(After)`, 404 when missing); `Lessons.TopicTexts(ctx,
  topicIDs)` in `backend/internal/storage/mongo/lessons.go` (find by `topicId $in`, project `topicId, content, annotations.text,
  annotations.lemma`); pure doc conversion test in `backend/internal/storage/mongo/topics_test.go`.
- [X] T007 Wire in `backend/cmd/api/main.go`: `topicLessons.Texts` → `repo.TopicTexts` (lemma map from single-word annotations,
  lowercase); `lesson.TopicRef` gains `Words []string` (`backend/internal/lesson/model.go`) filled by `lessonTopicsPort.Get`/`Names`;
  update `fakeTopics` in `backend/internal/lesson/fake_test.go` (configurable words per topic).

**Checkpoint**: `go build ./...` and `go test ./...` pass.

---

## Phase 3: User Story 1 - Danh sách từ vựng của chủ đề và độ phủ (Priority: P1) 🎯 MVP

**Goal**: seeded word lists, coverage "đã dùng X/Y", admin page to view/add/delete words.

**Independent Test**: quickstart.md bước 1–2.

### Tests for User Story 1

- [X] T008 [P] [US1] Write `backend/internal/topic/words_test.go`: `CleanWords` (trim/collapse, blank lines dropped, 40-char limit,
  allowed characters, duplicates case-insensitive marked on the later index, > 100 words → `words`, nothing returned on error);
  `Coverage` (used/unused, `LessonCount` per word over several lessons, phrase, plural, annotation lemma, no lessons).
- [X] T009 [P] [US1] Write `backend/internal/topic/seed_test.go`: embedded file loads with 42 topics / 1.257 words and every topic passes
  `CleanWords`; `SeedKey` ("Sức khoẻ" = "sức  khỏe", "Thuý" = "thúy", case); `PlanSeed` (matched topic gets words, unmatched gets empty,
  already seeded skipped, same name on two levels both matched).
- [X] T010 [P] [US1] Write service/handler tests in `backend/internal/topic/service_test.go` and `handler_test.go`: `List` fills
  `wordCount`/`usedWordCount` with one `Texts` call; `GET /api/admin/topics/{id}/words` 200 shape, 404; `PUT …/words` 200 (normalized,
  `wordsSeeded=true`), 400 `fields.words.{i}` / `fields.words`, 400 `invalid_body`, 404, 403 learner, 401 anonymous.

### Implementation for User Story 1

- [X] T011 [US1] Implement `CleanWords`, `Coverage` in `backend/internal/topic/words.go` (uses `wordmatch`) so T008 passes.
- [X] T012 [US1] Implement `backend/internal/topic/seed.go` (`//go:embed seed/topic-words.json`, `LoadSeed`, `SeedKey` with the
  tone-placement table of research R2, `PlanSeed`) so T009 passes.
- [X] T013 [US1] Implement `SeedTopicWords(ctx, db, seed, log)` in `backend/internal/storage/mongo/seed_topic_words.go` (find
  `wordsSeeded != true`, `PlanSeed`, conditional `UpdateOne` per topic, log `topic words seeded` with matched/unmatched counts) and call
  it in `PrepareInBackground` after `MigrateTopics` (`backend/internal/storage/mongo/indexes.go`, loading the seed via
  `topic.LoadSeed`).
- [X] T014 [US1] Implement `Service.List` coverage, `Words`, `SetWords` in `backend/internal/topic/service.go`; routes
  `GET/PUT /api/admin/topics/{id}/words` and `topicJSON.wordCount/usedWordCount` in `backend/internal/topic/handler.go` so T010 passes.
- [X] T015 [P] [US1] Frontend API: `topicWords(id)`, `setTopicWords(id, words)` in
  `frontend/src/app/features/admin/admin-api.service.ts` (+ spec).
- [X] T016 [US1] Topics page `frontend/src/app/features/admin/topics/topics.html/.ts/.css` (+ spec): "Từ vựng: đã dùng X/Y" or "Chưa có
  từ vựng" per topic, link **Từ vựng** to `/admin/topics/{id}/words`.
- [X] T017 [US1] New page `frontend/src/app/features/admin/topic-words/` (ts/html/css/spec) and route `topics/:id/words` in
  `frontend/src/app/features/admin/admin.routes.ts` (research R8): header "Từ vựng · {chủ đề}" with back link, words with "Đã dùng · n
  bài"/"Chưa dùng" labels and ✕ (`aria-label="Xoá {từ}"`), textarea "Thêm từ" (lines or commas), **Lưu** (sends kept + new words),
  **Huỷ**, field errors `words.{i}` under the matching word (new words listed as pending until saved), `words` error on top, success
  message `role="status"`. Spec: add many, duplicate error under the word, delete then save, server 400 mapping, keyboard (Tab/Enter on
  ✕ and Lưu).

**Checkpoint**: MVP — lists seeded, coverage shown, admin edits words.

---

## Phase 4: User Story 2 - Sinh bài bằng AI theo từ mục tiêu (Priority: P1)

**Goal**: per-lesson target words planned, editable, sent to AI in the same single request; drafts report missing words.

**Independent Test**: quickstart.md bước 3.

### Tests for User Story 2

- [X] T018 [P] [US2] Add to `backend/internal/topic/words_test.go`: `PlanWords` (unused first, then fewer lessons, then list order; no
  overlap when `count*k ≤ n`; reuse least used when short; fewer words than `perLesson`; `perLesson=0`; empty list); handler test for
  `GET …/word-plan` (200 `groups`, 400 `count`/`perLesson`, 404) in `backend/internal/topic/handler_test.go`.
- [X] T019 [P] [US2] Add to `backend/internal/ai/gemini/gemini_test.go`: generate prompt contains "Lesson 1 must use" with every word of
  `TargetWords[0]`, skips empty groups, no target section when `TargetWords` is nil.
- [X] T020 [P] [US2] Add to `backend/internal/lesson/generate_test.go` and `generate_handler_test.go`: `targetWords` length ≠ count →
  `targetWords`; group > 15 → `targetWords.{i}`; unknown or duplicate word → `targetWords.{i}.{j}`; words rewritten to the topic
  spelling and passed to the AI request; drafts carry `targetWords` of their original index after filtering and `missingWords` (plural
  form counts as used); draft not dropped for missing words; no `targetWords` → drafts with empty arrays and AI request with nil.

### Implementation for User Story 2

- [X] T021 [US2] Implement `PlanWords` in `backend/internal/topic/words.go`, `Service.WordPlan` and route
  `GET /api/admin/topics/{id}/word-plan` in `backend/internal/topic/service.go`/`handler.go` so T018 passes.
- [X] T022 [US2] Add `TargetWords [][]string` to `ai.GenerateRequest` in `backend/internal/ai/ai.go` and the target section to
  `generatePrompt` in `backend/internal/ai/gemini/gemini.go` so T019 passes.
- [X] T023 [US2] Implement in `backend/internal/lesson/generate.go` and `generate_handler.go`: `GenerateInput.TargetWords`, validation
  against `TopicRef.Words`, `filterDrafts` keeping original indexes, `Draft.TargetWords/MissingWords` via `wordmatch`, JSON
  `targetWords`/`missingWords` (always arrays) so T020 passes.
- [X] T024 [P] [US2] Frontend API: `wordPlan(id, count, perLesson)` and `targetWords` in `generateLessons` in
  `frontend/src/app/features/admin/admin-api.service.ts` (+ spec).
- [X] T025 [US2] `frontend/src/app/features/admin/generate-dialog/` (+ spec) per research R8: inputs `topicId`, `topicWords`;
  field "Từ mục tiêu mỗi bài" (0–15, default by level) only when `topicWords` is not empty; plan loaded on open and on count/perLesson
  change (debounced, stale responses ignored); per-lesson groups with ✕ (`aria-label="Bỏ {từ} khỏi bài {i}"`) and an add input with a
  datalist of topic words not in the group ("Từ không có trong danh sách" for others, no duplicates, ≤ 15); emits `targetWords` (empty
  array when 0). Spec: groups shown, remove/add, unknown word rejected, 0 words → no groups and empty `targetWords`, no topic words → no
  field, re-plan on count change, state kept after an error, keyboard.
- [X] T026 [US2] `frontend/src/app/features/admin/roadmap/roadmap.ts/.html` (+ spec): load `topicWords` when opening the dialog, pass
  `topicId`/`topicWords`, keep `targetWords`/`missingWords` on draft state.
- [X] T027 [US2] `frontend/src/app/features/admin/draft-list/` (+ spec): when `targetWords.length > 0` show "Dùng a/b từ mục
  tiêu" and "Còn thiếu: …" (text, not color only); nothing when there are no target words.

**Checkpoint**: generation with target words works end to end.

---

## Phase 5: User Story 3 - Chú thích đưa từ của chủ đề vào từ vựng của bài (Priority: P2)

**Goal**: topic words present in a lesson are always in its annotations; still one AI request.

**Independent Test**: quickstart.md bước 4.

### Tests for User Story 3

- [X] T028 [P] [US3] Write `backend/internal/lesson/focus_test.go`: `focusWords` (only topic words found in the content, phrase, plural);
  `addMissedFocus` (word already annotated by text or lemma → unchanged; missed word → added with span text, sentence index, lemma =
  lowercase topic word, first dictionary meaning; dictionary miss → skipped); `ProcessAnnotate` sends `FocusWords` to the AI (1 call),
  saves annotations including added words, topic without words → empty `FocusWords` and unchanged behavior.
- [X] T029 [P] [US3] Add to `backend/internal/ai/gemini/gemini_test.go`: annotate prompt lists `FocusWords` and says they count toward
  the 25 items; no focus section when empty.

### Implementation for User Story 3

- [X] T030 [US3] Change `Provider.Annotate` to take `ai.AnnotateRequest{Sentences, Level, FocusWords}` in `backend/internal/ai/ai.go`
  (`Disabled` too) and `backend/internal/ai/gemini/gemini.go` (prompt section) so T029 passes; update fakes in
  `backend/internal/lesson/fake_test.go` (record the request) and `backend/internal/writing/fake_test.go`.
- [X] T031 [US3] Create `backend/internal/lesson/focus.go` (`focusWords`, `addMissedFocus`); add `Dict Dictionary` to `lesson.Deps`
  and use it in `ProcessAnnotate` (`backend/internal/lesson/service.go`); pass `dict` in `backend/cmd/api/main.go`; fake dictionary in
  tests, so T028 passes.

**Checkpoint**: all user stories done.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T032 [P] Update `README.md`: section "Từ vựng theo chủ đề" (seed once, coverage rule, word page, target words when generating,
  annotation adds topic words, 1 AI request each).
- [X] T033 Run lint (frontend `npm run lint`, backend golangci-lint via Docker) with zero issues.
- [X] T034 Run all tests (`CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`, `npx ng test --watch=false`) with zero failures and no
  unhandled errors; `npx ng build` without warnings.
- [X] T035 Rebuild Docker and smoke test with AI off: startup log `topic words seeded`, 42 topics get words, restart does not reseed;
  edit a list via API (400 per word, 200); coverage numbers on `GET /api/admin/topics`; `word-plan` groups without overlap; generate →
  AI error, nothing saved; insert a lesson with topic words and run annotate (AI off) → fails as before. Restore any test data.
- [ ] T036 Manual check (needs a Gemini key): quickstart.md bước 3–5 (drafts use target words, missing words, annotation contains topic
  words, 360px light/dark, keyboard).
- [ ] T037 Tick the 7 F18 acceptance criteria in `docs/phases/giai-doan-3.md` once T034–T036 pass.

---

## Dependencies & Execution Order

- Setup (T001, T002) → Foundational (T003 → T004; T005 → T006 → T007).
- US1 (T008–T017) after Foundational. US2 needs `Coverage` (T011) for `PlanWords` and `TopicRef.Words` (T007). US3 needs
  `wordmatch` (T004) and `TopicRef.Words` (T007) only, so it can run in parallel with US1/US2 on the backend.
- T030 changes `ai.Provider` (all fakes) — do not run it in parallel with T022 (same files).
- Frontend tasks (T015–T017, T024–T027) only need T002 and the contract.

### Parallel Opportunities

- T001 ∥ T002 ∥ T003.
- T008 ∥ T009 ∥ T010; T015 ∥ T011–T014.
- T018 ∥ T019 ∥ T020 ∥ T024.
- T028 ∥ T029.

---

## Parallel Example: User Story 1

```bash
Task: "CleanWords/Coverage tests in backend/internal/topic/words_test.go"
Task: "Seed tests in backend/internal/topic/seed_test.go"
Task: "topicWords/setTopicWords in frontend/src/app/features/admin/admin-api.service.ts"
```

---

## Implementation Strategy

1. Setup + Foundational (`wordmatch`, topic fields, storage, ports).
2. US1: seed, coverage, word page. **STOP and VALIDATE** (quickstart bước 1–2) — MVP.
3. US2: word plan, target words in generation, dialog and drafts.
4. US3: focus words in annotation + dictionary supplement.
5. Polish: README, lint, tests, Docker smoke, manual checks, tick the criteria.

---

## Notes

- The seed never overwrites a list once `wordsSeeded` is true; admin saves always set it.
- One AI request per generation batch and per annotation; target/focus words ride in the existing request.
- Learners never see topic word lists; no learner API changes.
- Only commit when the user asks.

## Ghi chú triển khai

- T035 (Docker, AI tắt) đã chạy: lần khởi động đầu log `topic words seeded` matched=42, unmatched=2 (hai chủ đề "Chung");
  "Sức khoẻ" nhận 29 từ; khởi động lại không nạp lại. `GET /api/admin/topics` có `wordCount`/`usedWordCount` (A1 Gia đình 1/30);
  PUT sai → 400 theo từ (`words.1` trùng, `words.2` ký tự), PUT hợp lệ chuẩn hoá khoảng trắng; `word-plan` 3×8 không trùng và bỏ
  qua từ đã dùng ("Father"); `count=9` → 400; sinh bài với `targetWords` → 503 `ai_not_configured`, từ ngoài danh sách → 400
  `targetWords.0.0`. Danh sách "Gia đình" đã khôi phục, tài khoản thử đã xoá. Chú thích khi AI tắt vẫn lỗi như cũ (test đơn vị).
- Dữ liệu ban đầu: "Intensive care unit (ICU)" sửa thành "Intensive care unit" ở cả `docs/spec-inputs/` và bản nhúng.
- Frontend: hộp thoại nhận `GenerateOptions` (thêm `perLesson`, roadmap nhớ giá trị và bỏ đi trước khi gọi API); trang Từ vựng
  đánh dấu "Sẽ xoá" (bấm lại "Giữ lại") thay vì xoá ngay; lỗi chia từ vẫn cho sinh bài không có từ mục tiêu.
- `generate-dialog/` và `draft-list/` nằm ở `features/admin/` (không phải `features/admin/roadmap/`).
- T036, T037 để người dùng làm (cần khoá Gemini, kiểm tra 360px sáng/tối).
