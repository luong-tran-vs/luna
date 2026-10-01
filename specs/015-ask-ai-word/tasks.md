---

description: "Task list for F9 – Hỏi AI về từ"
---

# Tasks: Hỏi AI về từ (F9)

**Input**: Design documents from `specs/015-ask-ai-word/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/ask-api.md, quickstart.md

**Tests**: REQUIRED by constitution principle VI and khối 2:
- a cache hit does not call the Provider;
- concurrent requests make one call;
- a new revision does not use the old result;
- input validation;
- AI errors;
- lookup returns the stored result;
- the Gemini `Explain` call;
- the popup button and states;
- saving uses the AI meaning.

Real-AI quality, 360px and keyboard are checked by hand (quickstart.md).

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Add `note?: string` to `LookupResult` and `AskResult {result: LookupResult; cached: boolean}` in `frontend/src/app/core/models/reading.ts`.

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: The provider method and the stored results are shared by every story.

- [X] T002 Add to `backend/internal/ai/ai.go`:
  - `ExplainRequest {Text, Sentence, Level}`;
  - `Explanation {Lemma, MeaningVi, NoteVi}` with JSON tags;
  - `Provider.Explain`;
  - `Disabled.Explain` returning `ErrNotConfigured`.
  - Add `Explain` to the fake AIs in `backend/internal/lesson/fake_test.go` and `backend/internal/writing/fake_test.go` (the writing fake only needs to compile). The lesson fake should be configurable (result, error, call count, an optional channel to block on).
- [X] T003 Write Gemini tests in `backend/internal/ai/gemini/gemini_test.go`, then implement `Explain` in `backend/internal/ai/gemini/gemini.go` (research R1).
  - The tests check:
    - one POST request;
    - `responseSchema` is an OBJECT where `lemma`, `meaningVi` and `noteVi` are all required;
    - temperature is 0.2;
    - the prompt contains the level, the sentence and the text;
    - the response decodes;
    - HTTP 429 gives `ErrQuota`;
    - an empty key gives `ErrNotConfigured` and makes no request.
  - The implementation logs with `op=explain` and `words`.
- [X] T004 Add the lesson types and port in `backend/internal/lesson/ask.go`:
  - `AskKey`, `AskResult`, `AskRepository` (`Get`, `Put`);
  - `ErrUnusableExplanation`;
  - `LookupResult.Note` in `backend/internal/lesson/reading.go`.

  Change `NewReader` to also take `asks AskRepository` and `ai ai.Provider`, and update every caller:
  - `backend/internal/lesson/*_test.go` (add `newFakeAsks()` and a fake AI);
  - `backend/cmd/api/main.go` (the temporary value for `asks` is filled in T009).
- [X] T005 Create `backend/internal/storage/mongo/ai_lookups.go`:
  - `NewAILookups(db)`;
  - `Get` by the four key fields;
  - `Put` by insert; on a duplicate key, read the existing document and return it;
  - doc conversion.

  Also:
  - add the unique index `(lessonId, revision, sentenceIndex, textLower)` in `backend/internal/storage/mongo/indexes.go`;
  - add a pure doc conversion test in `ai_lookups_test.go`;
  - wire `mongo.NewAILookups(database)` and `aiProvider` into `lesson.NewReader` in `backend/cmd/api/main.go`.

**Checkpoint**: `go build ./...` and `go test ./...` pass.

---

## Phase 3: User Story 1 - Hỏi AI nghĩa theo ngữ cảnh trong popup tra từ (Priority: P1) 🎯 MVP

**Goal**: The popup has a **Hỏi AI** button. It shows the meaning in context, the base form and the note. Saving uses the AI meaning.

**Independent Test**: quickstart.md bước 1.

### Tests for User Story 1 (REQUIRED) ⚠️

- [X] T006 [P] [US1] Write `backend/internal/lesson/ask_test.go` with a fake AI and fake asks.
  - `Reader.Ask` with a phrase that is in the sentence:
    - calls `Explain` once, passing the text, sentence and lesson level;
    - returns `LookupResult{Source: "ai", Text, Lemma, Meanings: [{Text: meaning}], Note}` with `cached: false`;
    - stores the result under `(lesson, revision, sentence, normalized text)`;
    - an empty lemma becomes the text;
    - takes the IPA from the dictionary by lemma when known.
  - Validation errors (`ValidationError` on `text` / `sentenceIndex`):
    - empty text;
    - more than 100 runes;
    - more than 6 words;
    - text not in the sentence;
    - sentence out of range.
  - An unknown lesson gives `ErrNotFound`.
- [X] T007 [P] [US1] Write `backend/internal/lesson/ask_handler_test.go` for `POST /api/lessons/{id}/ask`:
  - 200 with the shape `{result: {source, text, lemma, ipa, meanings, note}, cached}`;
  - 400 `validation_failed`;
  - 400 `invalid_body`;
  - 404;
  - 401;
  - the route goes through the guard (a deny guard gives 403).

### Implementation for User Story 1

- [X] T008 [US1] Implement `Reader.Ask` in `backend/internal/lesson/ask.go` so T006 passes (research R3, R4). Include:
  - the `singleflight.Group` field;
  - the key string;
  - the AI call through `context.WithoutCancel` with a 12 s timeout;
  - the check that the meaning is not empty, capping meaning at 200 and note at 300 runes.
- [X] T009 [US1] Implement `backend/internal/lesson/ask_handler.go`: register `POST /api/lessons/{id}/ask` in `ReadingHandler.Register`, and add `note` to `lookupJSON` in `reading_handler.go`, so T007 passes.
- [X] T010 [P] [US1] Add `ask(lessonId, text, sentence): Observable<AskResult>` (POST `/api/lessons/{id}/ask` with body `{text, sentenceIndex}`) to `frontend/src/app/features/lesson/reading-api.service.ts`, with a spec case.
- [X] T011 [US1] Extend `frontend/src/app/shared/components/word-popup/word-popup.spec.ts`, then implement the changes in `word-popup.ts/.html/.css`.
  - New inputs `asking` and `askError`; new output `ask`.
  - **Hỏi AI** shows for a dictionary result and for not-found, and is hidden for an `ai` result.
  - While `asking`: the button is disabled and reads "Đang hỏi AI…", plus a `role="status"`.
  - A `note` shows under the meanings.
  - `askError` shows under the button in `role="alert"`.
  - The button is at least 44px.
- [X] T012 [US1] Handle `ask` in `frontend/src/app/features/lesson/reading/reading.ts/.html`:
  - set `asking` and call the API;
  - on success, set `popupState` to the AI result;
  - ignore the answer if the selection changed;
  - clear `askError` when the selection changes or the popup closes.

  Extend `reading.spec.ts`: lookup gives a dictionary result → ask → the popup shows the AI meaning and note → **Lưu vào sổ từ** posts a card with the AI lemma and meaning and `source: 'ai'`.

**Checkpoint**: MVP — hỏi AI được và lưu thẻ bằng nghĩa AI.

---

## Phase 4: User Story 2 - Dùng lại kết quả đã hỏi (Priority: P1)

**Goal**: The stored result is shared by everyone, one AI call per key, and a new revision is not served old results.

**Independent Test**: quickstart.md bước 2.

### Tests for User Story 2 (REQUIRED) ⚠️

- [X] T013 [P] [US2] Add to `backend/internal/lesson/ask_test.go`:
  - a second `Ask` with the same key (another user, "Make  up FOR" spacing and case) returns `cached: true` without calling `Explain`;
  - 10 goroutines asking the same key while the fake AI blocks make exactly 1 call and all get the same result;
  - after the lesson's revision changes, `Ask` calls the AI again;
  - the same text in another sentence is a different key.
- [X] T014 [P] [US2] Add to `backend/internal/lesson/reading_test.go`: `Lookup` returns a stored ask result (`Source: "ai"`, `Note`) before the dictionary for that sentence, but not for another sentence or an old revision; annotations still come first.

### Implementation for User Story 2

- [X] T015 [US2] Make `Ask` read `asks.Get` before and inside the singleflight call, and make `Lookup` read `asks.Get` after annotations, in `backend/internal/lesson/ask.go` / `reading.go`, so T013–T014 pass.

**Checkpoint**: kết quả dùng chung, không gọi lặp.

---

## Phase 5: User Story 3 - AI lỗi không làm hỏng popup (Priority: P2)

**Goal**: Clear errors; the dictionary meaning or manual meaning stays usable.

**Independent Test**: quickstart.md bước 3.

- [X] T016 [P] [US3] Add tests to `backend/internal/lesson/ask_test.go` and `ask_handler_test.go`. Errors are returned and nothing is stored:
  - `ErrNotConfigured` and `ErrInvalidKey` give 503 `ai_not_configured`, with different messages;
  - `ErrQuota` gives 429 `ai_quota`;
  - another error, an empty meaning or a deadline gives 502 `ai_failed`;
  - a later ask after an error calls the AI again.
- [X] T017 [US3] Map the errors in `backend/internal/lesson/ask_handler.go` (research R4) so T016 passes.
- [X] T018 [US3] Extend `frontend/src/app/features/lesson/reading/reading.spec.ts`:
  - 503, 429 and 502 show the server message under the button;
  - the dictionary meaning stays shown and **Lưu vào sổ từ** still saves it with `source: 'dictionary'`;
  - a network error shows "Không hỏi được AI, vui lòng thử lại" and the button works again;
  - not-found then ask failure keeps the manual meaning field.

  Implement anything missing in `reading.ts`.

**Checkpoint**: tất cả user story xong.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T019 [P] Update `README.md`, section "Bước Đọc và từ điển": Hỏi AI (when the button shows, shared stored results, 1 request per word and sentence, errors).
- [X] T020 Run lint (frontend `npm run lint`, backend golangci-lint via Docker) with zero issues.
- [X] T021 Run all tests (`CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`, `npx ng test --watch=false`) with zero failures and no unhandled errors; `npx ng build` without warnings.
- [X] T022 Rebuild Docker and smoke test with curl, with AI off:
  - ask → 503 `ai_not_configured`, and nothing stored;
  - text not in the sentence → 400;
  - another user's locked lesson → 403;
  - insert an `ai_lookups` document by hand (mongosh) → `lookup` and `ask` return it with `cached: true` and no AI log line;
  - bump the lesson revision → the stored result is ignored.
- [ ] T023 Manual check (needs a Gemini key): quickstart.md bước 1–4 (real meanings in context, 1 log line per key under concurrent curl, 360px, keyboard, screen reader).
- [ ] T024 Tick the 2 F9 acceptance criteria in `docs/phases/giai-doan-2.md` once T021–T023 pass.

---

## Dependencies & Execution Order

- Setup (T001) ∥ Foundational (T002 → T003; T004 → T005).
- Then US1 → US2 → US3 → Polish.
- US2 and US3 change the same files as US1 (`ask.go`, `ask_handler.go`), so they run after it.
- Frontend tasks (T010–T012, T018) can run in parallel with backend tasks after T001.

### Parallel Opportunities

- T001 ∥ T002 ∥ T004.
- T006 ∥ T007 ∥ T010.
- T013 ∥ T014.
- T016 ∥ T018.

---

## Parallel Example: User Story 1

```bash
Task: "Ask tests in backend/internal/lesson/ask_test.go"
Task: "ask() in frontend/src/app/features/lesson/reading-api.service.ts"
Task: "word-popup Hỏi AI button in frontend/src/app/shared/components/word-popup/"
```

---

## Implementation Strategy

1. Setup + Foundational, then US1 (hỏi AI, lưu thẻ). **STOP and VALIDATE** with quickstart bước 1.
2. US2: cache dùng chung, 1 lần gọi mỗi khoá.
3. US3: lỗi AI.
4. Polish: README, lint, test, Docker, manual checks, tick the criteria.

---

## Notes

- `ai_lookups` has no `userId` and is not part of the learner export.
- Never store a failed or empty AI answer.
- Only commit when the user asks.

## Ghi chú triển khai

- T014: các ca `Lookup` đọc kết quả đã hỏi nằm trong `ask_test.go` (`TestLookupReturnsStoredAsk`, `TestLookupStoredAskRules`)
  thay vì `reading_test.go`, để dùng chung môi trường `newAskEnv`.
- Lỗi gọi AI được bọc bằng `errAskAI` để handler phân biệt với lỗi lưu trữ (lỗi lưu trữ → 500 `internal_error`).
- `Lookup` không có câu (`sentence < 0`) thì bỏ qua bước đọc `ai_lookups`, vì kết quả gắn với từng câu.
- T022 (Docker, AI tắt) đã chạy: 503 không lưu gì; 400 khi từ không có trong câu / câu sai / JSON hỏng; 401 khi chưa đăng
  nhập; 403 với bài bị khoá; chèn `ai_lookups` bằng mongosh → `lookup` và `ask` trả kết quả đó (`cached: true`), không
  có dòng log gọi AI; tăng revision → kết quả cũ bị bỏ qua. Dữ liệu thử đã được dọn.
- T023, T024 để người dùng làm (cần khoá Gemini).
