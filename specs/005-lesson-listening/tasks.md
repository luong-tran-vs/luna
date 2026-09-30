---

description: "Task list for F4 – Nghe"
---

# Tasks: Nghe (F4)

**Input**: Design documents from `specs/005-lesson-listening/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: unit tests for business logic (dictation comparison with the
full sample set of research R1, summary/revision rules), endpoint tests (success, invalid input, other users cannot read
results, unauthenticated), no network (fake repositories, stubbed `HTMLMediaElement`). Every acceptance criterion maps to a
test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Add `interactive-widget=resizes-content` to the viewport meta in `frontend/src/index.html` (research R6)
- [X] T002 [P] Create `frontend/src/app/core/models/dictation.ts` (`DictationResult`, `DictationSummary`, `DictationInput`) per data-model.md §5

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 [P] Write `frontend/src/app/features/lesson/listening-api.service.spec.ts` and implement `ListeningApiService` (`ng g s features/lesson/listening-api --type=service`) in `frontend/src/app/features/lesson/listening-api.service.ts`: `summary(id)` → GET `/api/lessons/{id}/dictation/summary`, `record(id, input)` → POST `/api/lessons/{id}/dictation`, both unwrapping `summary` (contracts/dictation-api.md)
- [X] T004 Add route `':id/listen'` (lazy `Listening`, title "Nghe · Luna") to `frontend/src/app/features/lesson/lesson.routes.ts` and generate `ng g c features/lesson/listening` as an empty page

**Checkpoint**: `npm test -- --watch=false` passes; `/lessons/{id}/listen` opens

---

## Phase 3: User Story 1 - Nghe từng câu (Priority: P1) 🎯 MVP

**Goal**: Phát từng câu, trước/sau/lặp lại, 4 tốc độ, ẩn/hiện chữ

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T005 [P] [US1] Write `frontend/src/app/shared/components/audio-player/audio-player.spec.ts` (spy `HTMLMediaElement.prototype.play/pause`): `play()` sets `src` and `playbackRate`; changing `src` pauses the previous audio; `replay()` restarts from 0 and plays; `rate` input change applies immediately and survives a `src` change; `play()` rejection emits `failed`; `ended` emitted from the media event
- [X] T006 [P] [US1] Write `frontend/src/app/features/lesson/listening/listening.spec.ts` (navigation): loads `/api/lessons/{id}` and dictation summary; shows "Câu 1/N"; previous disabled on first, next disabled on last; next/previous change sentence and play it; repeat replays; speed radiogroup (0.5x/0.75x/1x/1.25x, default 1x) keeps the rate across sentences; transcript hidden by default, "Hiện chữ"/"Ẩn chữ" toggles; no sentence has audio → "Audio của bài chưa sẵn sàng" and no answer input; a sentence without audio shows "Chưa có audio" but keeps the input; 404 lesson → "Không tìm thấy bài học"

### Implementation for User Story 1

- [X] T007 [US1] Generate `ng g c shared/components/audio-player` and implement `AudioPlayer` in `frontend/src/app/shared/components/audio-player/` (hidden `<audio>`, inputs `src`, `rate`, methods `play()`, `replay()`, `stop()`, outputs `playing`, `ended`, `failed`) (research R5) so T005 passes
- [X] T008 [US1] Implement sentence navigation, speed radiogroup (arrow keys, 44px targets), transcript toggle and missing-audio states in `frontend/src/app/features/lesson/listening/listening.{ts,html,css}` so T006 passes
- [X] T009 [P] [US1] Add "Mở bước Nghe" link (`/lessons/{id}/listen`) to `frontend/src/app/features/admin/lesson-detail/lesson-detail.html` with an assertion in `lesson-detail.spec.ts`
- [ ] T010 [US1] Manual check: quickstart.md bước 1 (play, navigation, repeat < 1 s, speed kept, transcript, no overlapping audio)

**Checkpoint**: Nghe từng câu hoạt động độc lập

---

## Phase 4: User Story 2 - Chép chính tả và xem lỗi (Priority: P1)

**Goal**: Gõ lại câu, Kiểm tra, xem đúng / sai (kèm từ đúng) / thiếu từng từ

**Independent Test**: quickstart.md bước 2 và 4

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T011 [P] [US2] Write `frontend/src/app/shared/utils/dictation-compare.spec.ts` with the full sample set of research R1: `normalizeWords` (case, `’`, hyphen/dashes, punctuation, `don't`/`it's`/`rock'n'roll`, `9.30`, `1,000`, extra spaces/newlines); `compareDictation` exact match; missing at start/middle/end; extra at start/middle/end (wrong without `expected`); one misspelling (`dont` vs `don't`); empty typed (all missing, 0 correct); single-word sentence; totally wrong input; repeated word ("the the"); counts `correctWords` and `totalWords` (= expected + extras); a 40-word sentence completes in < 100 ms
- [X] T012 [P] [US2] Extend `listening.spec.ts` (dictation): input has `autocapitalize="off"`, `spellcheck="false"`, `enterkeyhint="done"`; empty check shows "Hãy gõ câu bạn nghe được" and records nothing; Enter checks; result renders ok words plain, wrong words with class `wrong` + "→ expected", extras with class `extra`, missing with class `missing`, each wrong/missing with visually-hidden label ("sai, đúng là …", "thiếu", "thừa"); full correct sentence and "4/5 từ đúng" shown; editing and checking again replaces the result; audio still replayable after checking

### Implementation for User Story 2

- [X] T013 [US2] Implement `normalizeWords` and `compareDictation` (Levenshtein alignment with tie-break ok → wrong → missing → extra) in `frontend/src/app/shared/utils/dictation-compare.ts` so T011 passes
- [X] T014 [US2] Implement the answer bar (sticky bottom, `scrollIntoView` on focus) and the result view with styles (`text-decoration: line-through` + `--color-bad`, `underline dotted` + `--color-warn`, text in `--color-text`) in `frontend/src/app/features/lesson/listening/` (research R4, R6) so T012 passes
- [ ] T015 [US2] Manual check: quickstart.md bước 2 (table of inputs, grayscale emulation) and bước 4 (360px phone keyboard keeps the Check button visible)

**Checkpoint**: US1 + US2 = luyện nghe và chép chính tả dùng được (chưa lưu)

---

## Phase 5: User Story 3 - Hoàn thành bước Nghe và lưu tỷ lệ đúng (Priority: P2)

**Goal**: Lưu kết quả mới nhất mỗi câu, tổng kết tỷ lệ đúng, hoàn thành khi kiểm tra hết, mở lại tiếp tục

**Independent Test**: quickstart.md bước 3 và 5

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T016 [P] [US3] Write `backend/internal/progress/service_test.go` with fake repository and fake `Lessons`: `Record` validates (sentence index in range, typed 1–1000 runes, total 1–1000, 0 ≤ correct ≤ total) as `*ValidationError`; unknown lesson → `ErrLessonNotFound`; stores current lesson revision; recording again replaces the sentence result; `Summary` counts only current-revision results with index < sentence count, sums words, `rate` (0 when no words), `completed` when every sentence checked (wrong ones included), results sorted by index; users are isolated
- [X] T017 [P] [US3] Write `backend/internal/progress/handler_test.go`: POST 200 returns summary; re-POST same sentence keeps `checkedCount`; 400 each invalid field and unknown `userId` field; 404 unknown lesson; GET summary 200 per user (user B sees nothing of A); 401 without session on both
- [X] T018 [P] [US3] Extend `listening.spec.ts` (persistence): summary results restored on load (checked sentences show their comparison from `typed`, progress "Đã kiểm tra X/N", starts at first unchecked sentence); each check POSTs `{sentenceIndex, typed, correctWords, totalWords}`; last sentence checked (with errors) → "Đã hoàn thành bước Nghe" with rate % and `completed` output emitted once; failed POST shows "Chưa lưu được, sẽ thử lại" and is resent before the next POST; per-sentence progress chips show checked state with ✓ and text

### Implementation for User Story 3

- [X] T019 [US3] Create `backend/internal/progress/model.go` (`Result`, `StoredResult`, `Summary`, errors, `ValidationError`) and `backend/internal/progress/repository.go` (`DictationRepository`, `Lessons` port) per data-model.md §3
- [X] T020 [US3] Implement `Service` (`NewService(repo, lessons, now)`, `Record(ctx, userID, lessonID, input) (Summary, error)`, `Summary(ctx, userID, lessonID)`) in `backend/internal/progress/service.go` so T016 passes
- [X] T021 [US3] Implement `Handler` (`Register(mux, requireAuth)`, POST `/api/lessons/{id}/dictation`, GET `/api/lessons/{id}/dictation/summary`) in `backend/internal/progress/handler.go` so T017 passes
- [X] T022 [P] [US3] Implement `backend/internal/storage/mongo/dictation.go` (`progress.DictationRepository`, upsert by `(userId, lessonId, sentenceIndex)`) and add the unique index in `backend/internal/storage/mongo/indexes.go`
- [X] T023 [US3] Wire in `backend/cmd/api/main.go`: `progress.Lessons` adapter over the lesson repository (revision + sentence count, `lesson.ErrNotFound` → `progress.ErrLessonNotFound`), service, handler
- [X] T024 [US3] Implement persistence in `frontend/src/app/features/lesson/listening/` (restore, record after each check, retry queue, completion banner, `completed = output<void>()`) so T018 passes
- [ ] T025 [US3] Manual check: quickstart.md bước 3 (complete with wrong sentences, reload keeps results, other account sees 0, content edit resets) and bước 5 (offline save then retry)

**Checkpoint**: Mọi user story hoạt động độc lập

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T026 Update `README.md`: mở bước Nghe từ trang quản trị, chép chính tả và tỷ lệ đúng
- [X] T027 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T028 Run all tests (`go test -race ./...`, `npm test -- --watch=false`) with zero failures
- [X] T029 Rebuild and smoke test in Docker: dictation POST/GET via `curl` with a UTF-8 body file (200, re-POST, 400 cases, per-user isolation, 401), revision reset after editing content
- [X] T030 Compute WCAG contrast for new pairs (wrong/missing decoration colors on surface and bg, speed radio selected state, progress chips) in both modes
- [ ] T031 Manual check at 360px width in both light and dark mode (quickstart.md bước 4): listening page, answer bar with keyboard open, result view, no horizontal scroll
- [ ] T032 Tick the 5 F4 acceptance criteria in `docs/phases/giai-doan-1.md` once T010, T015, T025, T029, T031 pass

---

## Dependencies & Execution Order

- **Setup (Phase 1)** → **Foundational (Phase 2)** → **US1** → **US2** (needs the listening page) → **US3** (needs checking)
- Backend of US3 (T016, T017, T019–T023) has no frontend dependency and can start right after Setup
- **Polish** after all stories

### Parallel Opportunities

- T002 ∥ T003; T005 ∥ T006 ∥ T009; T011 ∥ T012; T016 ∥ T017 ∥ T018 ∥ T022
- Whole backend track of US3 ∥ frontend US1/US2

---

## Parallel Example: User Story 3

```bash
Task: "Service tests in backend/internal/progress/service_test.go"
Task: "Handler tests in backend/internal/progress/handler_test.go"
Task: "Mongo repository in backend/internal/storage/mongo/dictation.go"
Task: "Persistence cases in frontend/src/app/features/lesson/listening/listening.spec.ts"
```

---

## Implementation Strategy

1. Setup + Foundational → US1 (nghe từng câu) → **STOP and VALIDATE** with quickstart bước 1
2. US2 → chép chính tả (giá trị chính của bước Nghe)
3. US3 → lưu kết quả, tỷ lệ đúng, hoàn thành (đủ cho L và F6)
4. Polish → README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (`GOTOOLCHAIN=local`)
- So sánh chỉ ở frontend; backend không nhận `userId` từ client
- Gửi dữ liệu tiếng Việt khi test bằng curl trên Windows: dùng `--data-binary @file.json`
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- `AudioPlayer` có output `started` / `finished` / `failed` (không dùng `playing` / `ended` vì trùng tên sự kiện DOM, lint
  `no-output-native`). `play(src?)` / `replay(src?)` nhận nguồn trực tiếp để trang phát câu mới ngay trong thao tác bấm
  (tránh trình duyệt chặn phát tự động); đổi input `src` chỉ dừng audio khi nguồn thật sự khác.
- Trang Nghe dùng một nút **Nghe câu** luôn phát lại từ đầu (gộp "phát" và "lặp lại"); các ô số câu (✓ khi đã kiểm tra)
  cho phép nhảy tới câu bất kỳ.
- Tỷ lệ đúng hiển thị ở trang được tính lại từ `typed` đã lưu (cùng thuật toán), nên vẫn đúng khi có lần lưu đang chờ gửi
  lại. Backend là nguồn số liệu cho thống kê (L, F6).
- Xoá bài không xoá `dictation_results` của bài đó (giống thẻ trong sổ từ); tổng kết của bài không còn tồn tại trả 404.
- Smoke test Docker (T029): POST 200, POST lại cùng câu giữ `checkedCount`, 400 cho từng lỗi và trường `userId`, người
  dùng khác thấy 0, 401 không cookie, 404 bài lạ / id sai; tạo bài tạm → lưu → sửa nội dung → summary về 0 → xoá bài tạm.
- Tương phản (T030): đường gạch `--color-bad` 5.17 / 4.74 (sáng, surface / bg), 6.08 / 6.69 (tối); `--color-warn`
  3.59 / 3.29 (sáng), 9.25 / 10.18 (tối); nút tốc độ đang chọn 7.37 (sáng) / 6.87 (tối); ô câu đã kiểm tra 7.96 / 7.86;
  chữ kết quả luôn `--color-text` ≥ 13:1.
