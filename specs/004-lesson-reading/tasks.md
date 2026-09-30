---

description: "Task list for F3 – Đọc"
---

# Tasks: Đọc (F3)

**Input**: Design documents from `specs/004-lesson-reading/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: unit tests for business logic (lemmatization, lookup order,
duplicate cards, tokenization, lesson lemma map, word audio cache), endpoint tests (success, invalid input,
unauthenticated, per-user isolation), no network (a tiny SQLite built inside the test with the real schema, fake
`Synthesizer`, fake repositories). Every acceptance criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US5 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US5)
- Paths: `backend/`, `frontend/`, `deploy/` at repo root (see plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Add `modernc.org/sqlite@v1.59.0` to `backend/go.mod` with `GOTOOLCHAIN=local`; confirm `go 1.25.1` stays and no dependency requires Go 1.26 (research R1)
- [X] T002 Add `DictionaryPath` (`DICTIONARY_PATH`, default `./data/dictionary/dictionary.db`) to `backend/internal/platform/config/config.go` with a case in `config_test.go` (contracts/config.md)
- [X] T003 [P] Create `deploy/fetch-dictionary.sh` and `deploy/fetch-dictionary.ps1`: download `https://github.com/minhqnd/dictionary/releases/download/v2.0.0/dictionary.db` into `deploy/data/dictionary/dictionary.db` via a temp file, verify SHA-256 `9259403f0675b2991a1bd0ef6d0dbc5933afdb135632af095a60662f09bbf1d3`, skip when an identical file exists, delete and fail on mismatch
- [X] T004 [P] Update `deploy/docker-compose.yml` (backend env `DICTIONARY_PATH: /data/dictionary/dictionary.db`, volume `./data/dictionary:/data/dictionary:ro`), `backend/.env.example` and local `backend/.env` (`DICTIONARY_PATH=../deploy/data/dictionary/dictionary.db`); confirm `deploy/data/` is in `.gitignore`
- [X] T005 [P] Import Noto Sans for IPA in `frontend/src/styles.css` (`@fontsource/noto-sans/latin-400.css`, `latin-ext-400.css`); check their `unicode-range` covers U+0250–02AF, U+02C8, U+02CC, U+02D0 (research R12)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Tách token, đưa về dạng gốc, đọc từ điển, bài cho người học, API client và trang đọc cơ bản

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Backend

- [X] T006 [P] Write `backend/internal/lesson/tokenize_test.go` and implement `Words(text string) []string` in `backend/internal/lesson/tokenize.go` with regex `[\p{L}]+(?:['’\-][\p{L}]+)*` (samples: "don't", "well-known", "U.S.", "9.30", "café", "rock'n'roll", quotes and dashes around words) (research R5)
- [X] T007 [P] Write `backend/internal/dictionary/lemma_test.go` (went/gone→go, saw→see, studies→study, watches→watch, parks→park, liked→like, stopped→stop, making→make, running→run, laughing→laugh, children→child, people→person, John's→john, uppercase and ’ normalized; first candidate is the word itself; no duplicates) and implement `Candidates(word string) []string` in `backend/internal/dictionary/lemma.go` with the irregular table in `backend/internal/dictionary/irregular.go` (research R2)
- [X] T008 Write `backend/internal/dictionary/sqlite_test.go` building a tiny SQLite file in `t.TempDir()` with the real schema (research R1) and entries `go`, `park`, `study`, `child` (with pos, vi and en definitions, two IPAs); test `Lookup` returns ≤ 3 Vietnamese meanings in order with POS, prefers `/…/` IPA, ignores English definitions, is case-insensitive via lowercasing, returns `ok=false` for unknown words; `Open` on a missing file returns an error; then implement `Open(path)` (read-only `file:…?mode=ro&immutable=1`), `(*SQLite).Lookup`, `Close`, and `None` (always not found) in `backend/internal/dictionary/sqlite.go`
- [X] T009 Write `backend/internal/lesson/reading_test.go` (foundation cases) and implement in `backend/internal/lesson/reading.go`: `Dictionary` port interface, `Reader` struct (`NewReader(lessons Repository, dict Dictionary)`), `ReadingView(ctx, id)` returning sentences, `paragraphs` (re-split content by blank lines, count sentences per paragraph with `SplitSentences`), `lemmas` (for each distinct lowercase word: annotation lemma if an annotation text or lemma matches, else first `dictionary.Candidates` found in the dictionary; omitted when nothing found) and `phrases` (annotations with a space); no annotations/revision/errors exposed (data-model.md §3)
- [X] T010 Implement `GET /api/lessons/{id}` in `backend/internal/lesson/reading_handler.go` (`RegisterReading(mux, requireAuth)`, JSON per contracts/reading-api.md) with tests in `backend/internal/lesson/reading_handler_test.go` (200 shape without `annotations`, 404, 401); wire in `backend/cmd/api/main.go`: open dictionary at `cfg.DictionaryPath` (missing → warn log `dictionary unavailable` and `dictionary.None{}`), close on shutdown, build `lesson.NewReader`, register routes

### Frontend

- [X] T011 [P] Create `frontend/src/app/core/models/reading.ts` and `frontend/src/app/core/models/vocab.ts` (data-model.md §6)
- [X] T012 [P] Write `frontend/src/app/shared/utils/tokenize.spec.ts` (same samples as T006) and implement `tokenize(sentence: string): Token[]` in `frontend/src/app/shared/utils/tokenize.ts` using `/[\p{L}]+(?:['’-][\p{L}]+)*/gu`; word tokens keep their order index; separators are non-word tokens
- [X] T013 [P] Write `frontend/src/app/features/lesson/reading-api.service.spec.ts` and implement `ReadingApiService` (`ng g s features/lesson/reading-api --type=service`) in `frontend/src/app/features/lesson/reading-api.service.ts`: `getLesson(id)`, `lookup(id, q, sentence)`, `wordAudioUrl(text)`, `saveCard(input)`, `words()` per contracts
- [X] T014 Create `frontend/src/app/features/lesson/lesson.routes.ts` with `':id/read'` (lazy `Reading`, title "Đọc · Luna") and add `{ path: 'lessons', canActivate: [authGuard], loadChildren: … }` to `frontend/src/app/app.routes.ts`

**Checkpoint**: `go test ./...` and `npm test -- --watch=false` pass; `GET /api/lessons/{id}` returns the reading view

---

## Phase 3: User Story 1 - Đọc bài và tra một từ (Priority: P1) 🎯 MVP

**Goal**: Trang đọc hiển thị bài theo đoạn/câu, chạm từ mở popup nghĩa với nhãn nguồn

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T015 [P] [US1] Add lookup cases to `backend/internal/lesson/reading_test.go` with a fake `Dictionary`: annotation text match wins over dictionary; annotation in sentence N preferred over another sentence; annotation lemma match (`go` finds *went → go* annotation); inflected word not annotated → dictionary via candidates (*studies* → *study*, `lemma: "study"`); IPA for AI source looked up by lemma; nothing → `ErrLookupNotFound`; `Reader` has no AI dependency (compile-time: constructor takes only repository and dictionary)
- [X] T016 [P] [US1] Add `GET /api/lessons/{id}/lookup` cases to `backend/internal/lesson/reading_handler_test.go`: 200 `source`/`lemma`/`meanings`; 404 `not_found` vs `lesson_not_found`; 400 empty `q`, > 100 chars, > 6 words, non-numeric `sentence`; 401
- [X] T017 [P] [US1] Write `frontend/src/app/shared/components/word-popup/word-popup.spec.ts`: shows text, `→ lemma` only when different, IPA with class using `--font-ipa`, source label "AI · theo ngữ cảnh"/"Từ điển", up to 3 meanings with Vietnamese POS names (N→danh từ, V→động từ, A→tính từ, unknown code shown as is), loading state, "Chưa có nghĩa" state; close button emits `closed`; `role="dialog"` labelled by the word
- [X] T018 [P] [US1] Write `frontend/src/app/features/lesson/reading/reading.spec.ts` (render + tap): title, paragraphs as separate blocks, words as focusable `.w` spans with `data-s`/`data-t`, punctuation not clickable; clicking a word calls `lookup(id, 'went', 0)` and shows the popup with the result; clicking another word switches the popup; Escape and outside click close it; 404 lesson → "Không tìm thấy bài học"

### Implementation for User Story 1

- [X] T019 [US1] Implement `Reader.Lookup(ctx, lessonID, q string, sentence int) (LookupResult, error)` in `backend/internal/lesson/reading.go` (research R3) so T015 passes
- [X] T020 [US1] Implement the lookup handler in `backend/internal/lesson/reading_handler.go` (validation per contract, error mapping) so T016 passes
- [X] T021 [US1] Generate `ng g c shared/components/word-popup` and implement `WordPopup` in `frontend/src/app/shared/components/word-popup/` (inputs `result`, `loading`, `notFound`, `text`; outputs `closed`, `save`, `play`; tokens only; width `min(320px, 100vw - 16px)`) so T017 passes
- [X] T022 [US1] Generate `ng g c features/lesson/reading` and implement `Reading` in `frontend/src/app/features/lesson/reading/`: load lesson, render paragraphs → sentences (`lang="en"`, `--text-reading`, max ~65ch) → tokens; roving tabindex over words (arrow keys, Enter opens popup); `cdkConnectedOverlay` anchored to the tapped word (positions below then above, `push`, offsetY 8, `overlayOutsideClick`/Escape close, focus return) (research R7) so T018 passes
- [X] T023 [US1] Add "Mở bước Đọc" link (`/lessons/{id}/read`) to `frontend/src/app/features/admin/lesson-detail/lesson-detail.html` and a spec assertion in `lesson-detail.spec.ts`
- [ ] T024 [US1] Manual check: quickstart.md bước 1 (AI and dictionary sources, inflections, < 300 ms in Network tab, no `ai request` logs, popup at 360px)

**Checkpoint**: Đọc và tra từ hoạt động độc lập

---

## Phase 4: User Story 2 - Lưu từ vào sổ từ (Priority: P1)

**Goal**: Lưu từ kèm câu ngữ cảnh, không trùng, tô màu ở mọi bài, tự nhập nghĩa khi không tra được

**Independent Test**: quickstart.md bước 2 và 6

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T025 [P] [US2] Write `backend/internal/vocab/service_test.go` with a fake repository and fake lesson checker: `Save` normalizes lemma (trim, lowercase, collapse spaces); duplicate lemma for same user → `*ExistsError` carrying the existing card; same lemma for another user → created; validation errors (text/lemma 1–100, meaning 1–200, sentence ≤ 1000, source enum, unknown lesson) as `*ValidationError`; `Words` returns only the user's words
- [X] T026 [P] [US2] Write `backend/internal/vocab/handler_test.go`: `POST /api/vocab/cards` 201 without `userId` in body, 409 `card_exists` with `card`, 400 fields, 401; `GET /api/vocab/words` per user; user id always from the session principal (a `userId` field in the body is rejected as unknown)
- [X] T027 [P] [US2] Extend `reading.spec.ts` and `word-popup.spec.ts`: words loaded from `/api/vocab/words`; tokens whose `lemmas[token] ?? token` is saved get class `saved` (underline + color); popup shows "✓ Đã có trong sổ" for saved lemma; "Lưu vào sổ từ" posts `{text, lemma, ipa, meaningVi (meanings joined by "; "), contextSentence, lessonId, source}` then marks saved and highlights; 409 treated as saved; not-found state shows meaning input (required, ≤ 200) and posts `source: "manual"`; save error shows `role="alert"` and keeps popup open

### Implementation for User Story 2

- [X] T028 [P] [US2] Create `backend/internal/vocab/model.go` (`Card`, `WordRef`, `Source`, errors) and `backend/internal/vocab/repository.go` (`Repository` per data-model.md §5)
- [X] T029 [US2] Implement `Service` (`NewService(repo, lessonExists func(ctx, id) (bool, error), now)`, `Save`, `Words`) in `backend/internal/vocab/service.go` so T025 passes
- [X] T030 [P] [US2] Implement `backend/internal/storage/mongo/cards.go` (`vocab.Repository`; duplicate key → `vocab.ErrExists`, then `FindByLemma` for the existing card) and add unique index `(userId, lemma)` and `(userId, createdAt)` in `backend/internal/storage/mongo/indexes.go`
- [X] T031 [US2] Implement `backend/internal/vocab/handler.go` (`Register(mux, requireAuth)`) and wire vocab service in `backend/cmd/api/main.go` (lesson existence via lesson repository) so T026 passes
- [X] T032 [US2] Implement saving in `word-popup` (save button, saved state, manual meaning form) and highlighting + saved-set updates in `reading` so T027 passes; add `.saved` style (`--color-primary-soft` background, `--color-primary-ink` text, dotted underline) in `reading.css`
- [ ] T033 [US2] Manual check: quickstart.md bước 2 and bước 6 (save, duplicate, highlight across lessons, manual meaning, another account sees nothing)

**Checkpoint**: US1 + US2 = bước Đọc có giá trị đầy đủ cho F5

---

## Phase 5: User Story 3 - Tra cả cụm từ (Priority: P2)

**Goal**: Chọn nhiều từ liền nhau để tra và lưu cả cụm; tô màu cụm đã lưu

**Independent Test**: quickstart.md bước 3

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T034 [P] [US3] Add phrase cases to `backend/internal/lesson/reading_test.go`: `gave up` matches annotation text; `give up` matches annotation lemma; phrase not annotated → dictionary exact phrase or not found (no per-word candidates for phrases); `phrases` in the reading view only multi-word annotations
- [X] T035 [P] [US3] Write `frontend/src/app/shared/utils/selection.spec.ts` for pure `selectWords(tokensTouched: {s: number, t: number}[]): {sentence, first, last} | {error: 'too-long'} | null` (sorted, clipped to first sentence, 1 word, 2–6 words, > 6 → too-long, empty → null) and extend `reading.spec.ts`: a simulated selection over "gave up" calls `lookup(id, 'gave up', 1)`; saved phrase lemma highlights the phrase tokens (via `phrases`)

### Implementation for User Story 3

- [X] T036 [US3] Support phrases in `Reader.Lookup` in `backend/internal/lesson/reading.go` so T034 passes
- [X] T037 [US3] Implement `selectWords` in `frontend/src/app/shared/utils/selection.ts`; in `reading` listen to `pointerup`/`keyup` + debounced `selectionchange`, map the DOM `Range` to touched word spans, anchor the overlay to the first span, clear the selection after opening, show "Chọn tối đa 6 từ" hint; highlight saved phrases so T035 passes
- [ ] T038 [US3] Manual check: quickstart.md bước 3 (desktop drag-select and phone long-press, partial word, cross-sentence, 7 words)

---

## Phase 6: User Story 4 - Nghe phát âm (Priority: P2)

**Goal**: Nút nghe trong popup, âm thanh từ tạo một lần và dùng chung

**Independent Test**: quickstart.md bước 4

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- [X] T039 [P] [US4] Write `backend/internal/tts/wordcache_test.go` with a counting fake `Synthesizer` and temp dir: `Normalize` (trim, collapse spaces, lowercase; reject digits/symbols/empty/> 60); first `Path` call synthesizes and writes `{dir}/words/{sha256}.mp3` atomically, second call does not synthesize; 10 concurrent calls synthesize once (singleflight); synth error returns error and leaves no file; handler `GET /api/tts/word` → 200 `audio/mpeg` + immutable cache, 400 invalid text, 503 `tts_unavailable`, 401 without session
- [X] T040 [P] [US4] Extend `word-popup.spec.ts` / `reading.spec.ts`: ▶ plays `wordAudioUrl(lemma or text)` through one shared `Audio` (spy on `HTMLMediaElement.prototype.play`), pressing again restarts (`currentTime = 0`), play rejection shows "Chưa phát được âm thanh" without closing the popup

### Implementation for User Story 4

- [X] T041 [US4] Implement `WordAudio` (`NewWordAudio(synth Synthesizer, dir string)`, `Normalize`, `Path(ctx, text)` with `singleflight`) in `backend/internal/tts/wordcache.go` and handler `GET /api/tts/word` (`RegisterWordAudio(mux, requireAuth)`), wire in `backend/cmd/api/main.go` so T039 passes
- [X] T042 [US4] Implement play in `word-popup`/`reading` with a shared `Audio` element so T040 passes
- [ ] T043 [US4] Manual check: quickstart.md bước 4 (play, replay, instant second play, Kokoro stopped → message)

---

## Phase 7: User Story 5 - Hoàn thành bước Đọc (Priority: P2)

**Goal**: Nút Đã đọc xong chỉ bật khi đã cuộn tới cuối bài

**Independent Test**: quickstart.md bước 5

### Tests for User Story 5 (REQUIRED - constitution VI) ⚠️

- [X] T044 [P] [US5] Extend `reading.spec.ts` with a mock `IntersectionObserver`: button disabled with hint "Đọc hết bài để hoàn thành" until the end sentinel intersects; stays enabled after scrolling away; without `IntersectionObserver` enabled immediately; click emits `completed` output once and shows "Đã hoàn thành bước Đọc"

### Implementation for User Story 5

- [X] T045 [US5] Implement end sentinel, observer (disconnect after first hit, cleanup on destroy), `completed = output<void>()` and confirmation banner in `frontend/src/app/features/lesson/reading/` so T044 passes (research R10)
- [ ] T046 [US5] Manual check: quickstart.md bước 5 (long lesson, short lesson)

**Checkpoint**: Mọi user story hoạt động độc lập

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T047 Update `README.md`: tải từ điển (`deploy/fetch-dictionary.*`), `DICTIONARY_PATH`, mở bước Đọc từ trang quản trị, backend local cần `DICTIONARY_PATH`
- [X] T048 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T049 Run all tests (`go test -race ./...`, `npm test -- --watch=false`) with zero failures
- [X] T050 Rebuild and smoke test in Docker with the real dictionary: `curl` lookup timings for *went/park/studies/running/children* (< 300 ms), duplicate save 409, `/api/vocab/words` per user, `/api/tts/word` first/second call timing, 401 for every new endpoint, no `ai request` log lines during lookups
- [X] T051 Compute WCAG contrast for new pairs (saved-word highlight text/background and underline, popup source label, IPA text, hint text, done button) in both modes
- [ ] T052 Manual check at 360px width in both light and dark mode (quickstart.md bước 7): reading page, popup placement near screen edges, IPA glyphs, no horizontal scroll
- [ ] T053 Tick the 4 F3 acceptance criteria in `docs/phases/giai-doan-1.md` once T024, T033, T046, T050, T052 pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 → T002; T003–T005 parallel
- **Foundational (Phase 2)**: backend T006, T007 parallel → T008 → T009 → T010; frontend T011–T013 parallel → T014
- **US1 (Phase 3)**: after Foundational
- **US2 (Phase 4)**: after US1 (popup and reading page exist)
- **US3 (Phase 5)**: after US1; highlight of saved phrases needs US2
- **US4 (Phase 6)**: after US1 (popup); backend part independent
- **US5 (Phase 7)**: after US1 (reading page); independent of US2–US4
- **Polish (Phase 8)**: after all stories

### Within Each User Story

- Tests written first and FAIL before implementation
- Backend: domain logic → handler → wiring in `main.go`
- Frontend: pure utils → components → page integration
- Manual check task last

### Parallel Opportunities

- Setup: T003 ∥ T004 ∥ T005
- Foundational: T006 ∥ T007 (backend) ∥ T011 ∥ T012 ∥ T013 (frontend)
- US1: tests T015–T018 parallel; backend T019–T020 ∥ frontend T021–T022
- US2: T025–T028, T030 parallel
- US4 backend (T039, T041) ∥ US3/US5 frontend work

---

## Parallel Example: User Story 1

```bash
Task: "Lookup cases in backend/internal/lesson/reading_test.go"
Task: "Lookup handler cases in backend/internal/lesson/reading_handler_test.go"
Task: "WordPopup spec in frontend/src/app/shared/components/word-popup/word-popup.spec.ts"
Task: "Reading page spec in frontend/src/app/features/lesson/reading/reading.spec.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup (incl. downloading the dictionary) → Phase 2 Foundational
2. Phase 3 US1 → **STOP and VALIDATE** with quickstart bước 1

### Incremental Delivery

1. US1 → đọc và tra từ
2. US2 → lưu từ, tô màu (đủ cho F5)
3. US3 → tra cụm
4. US4 → phát âm
5. US5 → Đã đọc xong (đủ cho L)
6. Polish → README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- [P] tasks = different files, no dependencies
- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (dùng `GOTOOLCHAIN=local`)
- Không gọi AI trong mọi đường tra cứu; không nhận `userId` từ client
- Code, API, commit message bằng tiếng Anh; chữ trên giao diện bằng tiếng Việt
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai (2026-09-29)

**Khác với plan/tasks** (có lý do):
- **Research R1 đã được đính chính**: từ điển thật *có* mục cho dạng biến đổi nhưng nghĩa đầu tiên chỉ là ghi chú ("động từ
  quá khứ của go."). Thêm `dictionary.Resolve` (`backend/internal/dictionary/resolve.go`): tra chính từ, nếu nghĩa đầu là
  ghi chú dạng biến đổi thì chuyển sang từ gốc, không có thì thử ứng viên. Port `lesson.Dictionary` dùng `Resolve`.
- `dictionary.Open(ctx, path)` nhận context (lint `noctx`); thêm `realdb_test.go` chạy với file thật khi đặt
  `LUNA_DICTIONARY`.
- Font IPA: file subset của `@fontsource/noto-sans` không có `unicode-range`, nên khai báo `@font-face` riêng ở
  `frontend/src/styles/fonts.css` (latin + latin-ext, latin-ext phủ IPA Extensions và ˈ ˌ ː).
- Output nghe của `WordPopup` tên `listen` (không dùng `play` vì trùng sự kiện DOM media); handler bàn phím/con trỏ của
  trang đọc gắn ở host component; từ dùng `role="button"` + roving `tabindex`.
- `ReadingHandler`, `WordAudio` và vocab `Handler` mỗi cái có `Register(mux, requireAuth)` riêng.

**Kết quả kiểm tra**
- Backend: `go test -race ./...` qua; golangci-lint 0 issues; `go.mod` vẫn `go 1.25.1` (`modernc.org/sqlite v1.59.0`).
- Frontend: 182 test Vitest qua; `ng lint` sạch; `ng build` thành công (trang đọc là chunk lazy 64 kB).
- Từ điển thật (test `realdb_test.go`): went→go, studies→study, stopped→stop, taken→take, goes→go, children→child;
  mỗi lần tra ~0,5 ms.
- Docker thật qua nginx: `dictionary loaded`; tra 9 từ mất 5–17 ms mỗi request; *went*/*gave up* ra nguồn AI (từ chú
  thích), *park/studies/children/yesterday* ra từ điển với IPA; từ/cụm không có → 404; log không có `ai request` nào.
  Lưu thẻ 201, lưu lại `" GO "` → 409 kèm thẻ cũ; `/api/vocab/words` chỉ thấy của chính mình; gửi `userId` → 400.
  Phát âm *give up*: lần 1 0,97 s, lần 2 0,005 s (cache), mp3 hợp lệ; chữ lạ → 400. Mọi endpoint mới → 401 khi chưa
  đăng nhập. Index `cards (userId, lemma)` unique.
- Tiếng Việt lưu và trả về đúng UTF-8 (kiểm bằng dữ liệu gửi từ file; `curl -d "…"` trên Git Bash Windows làm hỏng mã hoá
  tham số — không phải lỗi app).
- Độ tương phản các cặp màu mới (từ đã lưu, gạch chân, nhãn nguồn, IPA, focus) đạt AA ở cả hai chế độ.

**Còn lại cần kiểm tra bằng trình duyệt**: T024, T033, T038 (kéo chọn trên máy tính và chạm giữ trên điện thoại),
T043 (nghe, và khi dừng Kokoro), T046, T052 (360px), rồi T053. Bài "Reading test" (có chú thích *went*, *gave up*) đã tạo
sẵn để thử.
