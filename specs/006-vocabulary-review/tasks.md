---

description: "Task list for F5 – Sổ từ và ôn tập"
---

# Tasks: Sổ từ và ôn tập (F5)

**Input**: Design documents from `specs/006-vocabulary-review/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: FSRS schedule after each rating, first due next day in the learner's
timezone (near midnight, several timezones), editing keeps the schedule, deleting removes review logs, bulk save never
duplicates, day grouping near midnight, lesson without annotations, double-review protection; endpoint tests (success, invalid
input, other users, unauthenticated); no network (fake repositories and ports, fake clock, stubbed `HTMLMediaElement`). Every
acceptance criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Add `github.com/open-spaced-repetition/go-fsrs/v3@v3.3.1` to `backend/go.mod` (`GOTOOLCHAIN=local go get`, then check the `go 1.25.1` directive is unchanged and run `go mod tidy`) (research R1)
- [X] T002 [P] Extend `frontend/src/app/core/models/vocab.ts` per data-model.md §5 (`ReviewMode`, `Rating`, `Intervals`, schedule fields on `Card` with `lessonId: string | null`, `DayCard`, `CardPage`, `DueCard`, `DueList`, `VocabItem`, `ReviewSummary`, `LessonCount`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 [P] Write `backend/internal/vocab/schedule_test.go`: `FirstDue` (Asia/Ho_Chi_Minh 23:59 → next day 00:00 local; 00:01 → next day 00:00; America/New_York; Europe/Paris on 2026-03-28 → 2026-03-29 00:00 across the DST change); `toFSRS`/`fromFSRS` round trip; a card with a zero `Schedule` (F3 legacy) becomes `fsrs.NewCard()` with `Due = FirstDue(createdAt)`; `intervalsFor(new card, now)` = 1 min / 5 min / 10 min / Easy ≥ 1 day; `next` for each rating gives New→Learning (Again/Hard/Good), New→Review (Easy), Review+Again → Relearning with `Lapses+1`, and matches `fsrs.NewFSRS(fsrs.DefaultParam()).Next` for the same input
- [X] T004 Implement `backend/internal/vocab/schedule.go` (`FirstDue(t time.Time, loc *time.Location) time.Time`, `startOfDay`, `toFSRS`, `fromFSRS`, `effective(c Card, loc)` for legacy cards, `intervalsFor`, `next`) wrapping go-fsrs with `DefaultParam()` (research R1, R2) so T003 passes
- [X] T005 Extend `backend/internal/vocab/model.go` (`Schedule` on `Card`, `Rating`, `Mode`, `ReviewLog`, `Intervals`, `DueCard`, `ListQuery`, `DayCard`, `Page`, `Details`, `VocabItem`, `LessonCount`, `ErrLessonNotFound`, `*ConflictError{Card}`) and `backend/internal/vocab/repository.go` (new `Repository` methods, `ReviewLogRepository`, ports `Timezones`, `LessonVocabulary`, `LessonTitles`) per data-model.md §3; extend `backend/internal/vocab/fake_test.go` with in-memory implementations (conditional update on `reps` treating missing as 0, due query including legacy cards with `createdAt < startOfToday`, fake ports)
- [X] T006 Add to `backend/internal/vocab/service_test.go` (tests first): `NewService(Deps{…})` constructor used by all existing tests; `Save` stores `Schedule{Due: FirstDue(createdAt in user tz), State: New}` (23:59 and 00:01 cases, two timezones); `lessonId` empty allowed only with `source: manual` (→ `lessonId` field error otherwise); non-empty `lessonId` still must exist
- [X] T007 Refactor `backend/internal/vocab/service.go` to `NewService(d Deps)` (`Repo`, `Logs`, `LessonExists`, `Timezones`, `Vocabulary`, `Titles`, `Now func() time.Time`) and update `Save` so T006 and existing F3 tests pass; update `backend/internal/vocab/handler_test.go` helpers for the new constructor
- [X] T008 [P] Update `backend/internal/storage/mongo/cards.go`: schedule fields on `cardDoc` (decode missing fields as zero), `lessonId` omitted for manual cards, `toCard`/create mapping; add indexes `(userId, due)`, `(userId, lessonId, createdAt)` for `cards` and `(userId, cardId)`, `(userId, reviewedAt)` for `review_logs` in `backend/internal/storage/mongo/indexes.go`
- [X] T009 Wire `backend/cmd/api/main.go`: `vocab.Timezones` adapter over `auth` (`UserByID` → `time.LoadLocation`, fallback `Asia/Ho_Chi_Minh`), `vocab.NewService(vocab.Deps{…})`
- [X] T010 [P] Generate `ng g s core/services/vocab-api --type=service`; move `saveCard` and `words` from `features/lesson/reading-api.service.ts` into `frontend/src/app/core/services/vocab-api.service.ts` (+ `vocab-api.service.spec.ts`), update `reading.ts`, `reading-api.service.spec.ts` and `reading.spec.ts`

**Checkpoint**: `go test ./...` and `npm test -- --watch=false` pass; saving a word in Reading still works and new cards have `due`

---

## Phase 3: User Story 1 - Ôn thẻ đến hạn (Priority: P1) 🎯 MVP

**Goal**: Ôn tự do các thẻ đến hạn bằng 2 kiểu, 4 nút đánh giá, lịch FSRS cập nhật

**Independent Test**: quickstart.md bước 3

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T011 [P] [US1] Add to `backend/internal/vocab/service_test.go`: `Review` validates rating 1–4 and mode `flip`/`listen` (`*ValidationError`); unknown card or another user's card → `ErrNotFound`; each rating on a new card stores the go-fsrs result and returns new intervals; review card + Again → Relearning, `Lapses+1`; writes one log with `rating`, `mode`, `reviewedAt = now`, `Before` = schedule before; stale `reps` → `*ConflictError` with the current card and no log; legacy card (zero schedule) reviewable. `Due`: legacy card saved yesterday (local) is due and one saved today is not; ordered by due ascending; `limit`; `total`; `NextDue` (none → false); other users' cards excluded
- [X] T012 [P] [US1] Add to `backend/internal/vocab/handler_test.go`: `GET /api/vocab/review/due` (200 with `intervals` in seconds, `total`, `nextDue`; `limit` 0 or 201 → 400); `POST /api/vocab/cards/{id}/review` (200 with new schedule, 400 bad rating/mode, 400 unknown field `userId`, 404 other user, 409 `review_conflict` + `card` on re-send with same `reps`); 401 on both
- [X] T013 [P] [US1] Write `frontend/src/app/shared/utils/interval-label.spec.ts` (60 → "1 phút", 600 → "10 phút", 3 h → "3 giờ", 2 d → "2 ngày", 45 d → "2 tháng", 400 d → "1 năm") and `frontend/src/app/shared/utils/answer-match.spec.ts` (case, surrounding and repeated spaces, `’` vs `'`, phrase "give  up", wrong word)
- [X] T014 [P] [US1] Write `frontend/src/app/shared/components/review-session/review-session.spec.ts` (stub `HTMLMediaElement.prototype.play`): flip mode shows only word + listen button, flip by click / Enter / Space shows IPA, meaning, example and 4 buttons labelled with interval (`Good · 10 phút`); keys 1–4 rate after flip; listen mode auto-plays `/api/tts/word?text=went`, has "Nghe lại", input with `autocapitalize="off"` `spellcheck="false"`, empty check → "Hãy gõ từ bạn nghe được", "WENT" → "✓ Đúng", "want" → "✗ Sai" + correct word, play failure → "Chưa phát được âm thanh" + "Hiện từ"; rating POSTs `{rating, mode, reps}`; Again re-queues the card once at the end with the returned intervals; 409 counts as done and moves on; failed POST shows "Chưa lưu được đánh giá" + "Thử lại" and stays on the card; after the last card emits `finished` once with `{reviewed, counts}` and shows "Đã ôn N thẻ"
- [X] T015 [P] [US1] Write `frontend/src/app/features/vocabulary/review/review.spec.ts`: loads `/api/vocab/review/due`, shows "N thẻ đến hạn", mode radiogroup (Xem từ đoán nghĩa / Nghe rồi gõ), "Bắt đầu" starts the session; no due cards → "Không có thẻ nào đến hạn" + formatted `nextDue`; after `finished` shows the summary and "Ôn tiếp" reloads due cards

### Implementation for User Story 1

- [X] T016 [US1] Implement `Get`, `UpdateSchedule` (filter `{_id, userId, reps}` with `$or` missing `reps` when expected 0), `Due` (legacy `$or`, sort `due` asc), `NextDue` in `backend/internal/storage/mongo/cards.go` and create `backend/internal/storage/mongo/reviewlogs.go` (`Add`)
- [X] T017 [US1] Implement `Service.Due(ctx, userID, limit)` and `Service.Review(ctx, userID, cardID, rating, mode, reps)` in `backend/internal/vocab/service.go` so T011 passes
- [X] T018 [US1] Add routes `GET /api/vocab/review/due`, `POST /api/vocab/cards/{id}/review` and card JSON with schedule and `intervals` in `backend/internal/vocab/handler.go` so T012 passes
- [X] T019 [US1] Wire `mongo.NewReviewLogs(database)` into `vocab.Deps` in `backend/cmd/api/main.go`
- [X] T020 [P] [US1] Implement `frontend/src/app/shared/utils/interval-label.ts` and `frontend/src/app/shared/utils/answer-match.ts` so T013 passes
- [X] T021 [US1] Add `due(limit)` and `review(id, body)` to `VocabApiService` with spec cases in `frontend/src/app/core/services/vocab-api.service.spec.ts`
- [X] T022 [US1] Generate `ng g c shared/components/review-session` and implement it (inputs `cards`, `mode`; uses `AudioPlayer`; output `finished`; answer bar sticky like F4; state not shown by colour alone) so T014 passes
- [X] T023 [US1] Create `frontend/src/app/features/vocabulary/vocabulary.routes.ts` (`'review'` → `Review`, title "Ôn tập · Luna"), add lazy `vocabulary` route with `authGuard` in `frontend/src/app/app.routes.ts`, generate `ng g c features/vocabulary/review` and implement it so T015 passes
- [ ] T024 [US1] Manual check: quickstart.md bước 3 (both modes, keyboard, Again re-shown, empty state, offline rating then retry counted once)

**Checkpoint**: Ôn tập hoạt động độc lập với thẻ đã có từ F3

---

## Phase 4: User Story 2 - Quản lý sổ từ (Priority: P1)

**Goal**: Danh sách thẻ nhóm theo ngày (múi giờ), tìm, lọc bài, tải dần; thêm, sửa, xoá thẻ

**Independent Test**: quickstart.md bước 2 và 4

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T025 [P] [US2] Add to `backend/internal/vocab/service_test.go`: `List` gives `Day` in the user's timezone (cards at 23:30 and 00:30 local, UTC and Asia/Ho_Chi_Minh users), `Today`/`Yesterday`, newest first, `q` case-insensitive on text or lemma with regex characters escaped (`a.b` does not match `axb`), `lessonId` filter and `manual`, 30 per page with `HasMore`, page < 1 or `q` > 100 runes → `*ValidationError`; `Lessons` returns titles via `LessonTitles` (deleted lessons omitted) and the manual count; `Update` changes only meaning / IPA / example, keeps `Schedule` and logs, validates (meaning 1–200, IPA ≤ 100, example ≤ 1000), other user → `ErrNotFound`; `Delete` removes the card and its logs, is idempotent for leftover logs, `ErrNotFound` when nothing existed
- [X] T026 [P] [US2] Add to `backend/internal/vocab/handler_test.go`: `GET /api/vocab/cards` (200 shape with `day`, `today`, `yesterday`, `hasMore`; 400 bad page); `GET /api/vocab/lessons`; `POST /api/vocab/cards` without `lessonId` + `source: manual` → 201 with `due`, duplicate → 409; `PATCH /api/vocab/cards/{id}` (200, `text` field → 400 `invalid_body`, 404 other user); `DELETE` (204, 404); 401 on all
- [X] T027 [P] [US2] Write `frontend/src/app/features/vocabulary/notebook/notebook.spec.ts`: groups labelled "Hôm nay", "Hôm qua", `20/09/2026` from `today`/`yesterday`; "Xem thêm" loads page 2 and merges a group split across pages; search input sends `q` (debounced) and shows "Không tìm thấy từ nào"; lesson filter lists lessons + "Thẻ tự thêm" and sends `lessonId`; empty notebook shows guidance; each card shows word, IPA, meaning, example, listen button; "Sửa" opens the form; "Xoá" asks confirmation then removes the card; "Ôn tập (N)" links to `/vocabulary/review`
- [X] T028 [P] [US2] Write `frontend/src/app/features/vocabulary/card-form/card-form.spec.ts`: add mode requires word and meaning (Vietnamese errors), IPA and example optional, posts `source: 'manual'` without `lessonId`, 409 → "Từ này đã có trong sổ"; edit mode shows the word read-only and PATCHes only changed fields; 400 field errors shown on fields; emits `saved`/`cancelled`

### Implementation for User Story 2

- [X] T029 [US2] Implement `List`, `LessonCounts`, `UpdateDetails` (`$set` only the three fields), `Delete` in `backend/internal/storage/mongo/cards.go` and `DeleteByCard` in `backend/internal/storage/mongo/reviewlogs.go`
- [X] T030 [US2] Implement `Service.List`, `Service.Lessons`, `Service.Update`, `Service.Delete` in `backend/internal/vocab/service.go` so T025 passes
- [X] T031 [US2] Add routes `GET /api/vocab/cards`, `GET /api/vocab/lessons`, `PATCH /api/vocab/cards/{id}`, `DELETE /api/vocab/cards/{id}` in `backend/internal/vocab/handler.go` so T026 passes
- [X] T032 [US2] Wire `vocab.LessonTitles` adapter over `lesson.Repository.Summaries` in `backend/cmd/api/main.go`
- [X] T033 [US2] Add `list`, `lessons`, `create`, `update`, `remove` to `VocabApiService` with spec cases
- [X] T034 [US2] Generate `ng g c features/vocabulary/card-form` and implement it (reactive form) so T028 passes
- [X] T035 [US2] Generate `ng g c features/vocabulary/notebook`, implement it (reuse `ConfirmDialog`, `AudioPlayer`) and add route `''` (title "Sổ từ · Luna") in `vocabulary.routes.ts` so T027 passes
- [X] T036 [P] [US2] Add "Sổ từ" link (`/vocabulary`) for signed-in users to `frontend/src/app/shared/components/app-header/app-header.html` with an assertion in `app-header.spec.ts`
- [ ] T037 [US2] Manual check: quickstart.md bước 2 (groups, search, filter, add, duplicate, edit keeps `due`/`reps`, delete removes logs) and bước 4 (timezone change)

**Checkpoint**: US1 + US2 = sổ từ và ôn tập đầy đủ

---

## Phase 5: User Story 3 - Mục Từ vựng của bài ở bước Đọc (Priority: P2)

**Goal**: Liệt kê từ đã chú thích của bài, Lưu / Lưu tất cả không trùng, ✓ cho từ đã lưu, không gọi AI

**Independent Test**: quickstart.md bước 1

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T038 [P] [US3] Add to `backend/internal/lesson/reading_test.go`: `Reader.Vocabulary` groups annotations by lemma (went/goes → one "go"), keeps the first by sentence order with its sentence text, IPA from the fake dictionary (empty when unknown), `available=false` when `AnnotationStatus != done` or there are no annotations, `ErrNotFound` for an unknown lesson
- [X] T039 [P] [US3] Add to `backend/internal/lesson/reading_handler_test.go`: `GET /api/lessons/{id}/vocabulary` 200 (`available`, `items`), 404, 401
- [X] T040 [P] [US3] Add to `backend/internal/vocab/service_test.go` and `handler_test.go`: `SaveBulk` creates cards from `LessonVocabulary` items (`source: ai`, `lessonId`, `contextSentence` = item sentence, `due` = first due), skips lemmas already saved and lemmas not in the lesson, a second call adds 0, 0 or > 200 lemmas → validation error, unknown lesson → 404; `POST /api/vocab/cards/bulk` returns `{added, cards}`
- [X] T041 [P] [US3] Write `frontend/src/app/features/lesson/reading/lesson-vocabulary/lesson-vocabulary.spec.ts`: renders items (lemma, IPA, meaning, listen); items in the `saved` input show "✓ Đã lưu"; "Lưu" posts bulk with one lemma and emits `saved`; "Lưu tất cả" posts only unsaved lemmas and shows "Đã lưu N từ", hidden when all saved; `available: false` → "Chưa có danh sách từ vựng" and no "Lưu tất cả"; extend `reading.spec.ts`: saving from the popup marks the item ✓ in the section

### Implementation for User Story 3

- [X] T042 [US3] Implement `Reader.Vocabulary` in `backend/internal/lesson/reading.go` and route `GET /api/lessons/{id}/vocabulary` in `backend/internal/lesson/reading_handler.go` so T038, T039 pass
- [X] T043 [US3] Implement `Service.SaveBulk` and route `POST /api/vocab/cards/bulk` in `backend/internal/vocab/`, wire the `vocab.LessonVocabulary` adapter over `lesson.Reader` in `backend/cmd/api/main.go` so T040 passes
- [X] T044 [US3] Add `vocabulary(id)` to `ReadingApiService` and `bulk(lessonId, lemmas)` to `VocabApiService`; generate `ng g c features/lesson/reading/lesson-vocabulary`, implement it and place it in `reading.html` as a collapsible section (disclosure button with `aria-expanded`) sharing the reading page's saved set so T041 passes
- [ ] T045 [US3] Manual check: quickstart.md bước 1 (list, save one, save all twice, popup save updates ✓, lesson without annotations)

**Checkpoint**: Mọi user story hoạt động độc lập

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T046 Update `README.md`: section "Sổ từ và ôn tập" (sổ từ, ôn tự do, FSRS, mục Từ vựng) and `vocab` in the folder structure
- [X] T047 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T048 Run all tests (`go test -race ./...`, `npm test -- --watch=false`) with zero failures; `npx ng build` without warnings
- [X] T049 Rebuild and smoke test in Docker with `curl` and UTF-8 body files: list/search/filter, manual add + duplicate, PATCH keeps schedule, due list, review each rating + re-send → 409, DELETE removes logs, bulk twice, lesson vocabulary, other user 404, 401
- [X] T050 Compute WCAG contrast for new pairs (correct/incorrect answer states, rating buttons, "✓ Đã lưu", day group headings, card-flip surface) in both modes
- [ ] T051 Manual check at 360px width in light and dark mode (quickstart.md bước 5): notebook, card form, review session in both modes with keyboard open, lesson vocabulary section, no horizontal scroll
- [ ] T052 Tick the 7 F5 acceptance criteria in `docs/phases/giai-doan-1.md` once T024, T037, T045, T049, T051 pass

---

## Dependencies & Execution Order

- **Setup (Phase 1)** → **Foundational (Phase 2)** → **US1** ∥ **US2** → **US3**
- US1 and US2 share `service.go`, `handler.go`, `cards.go`: backend tasks of the two stories run one after another; their
  frontend tasks are independent
- US3 needs `VocabApiService` (T010) and the reading page; backend of US3 (T038–T040, T042–T043) only needs Phase 2
- **Polish** after all stories

### Parallel Opportunities

- T002 ∥ T003; T008 ∥ T010 (after T005)
- T011 ∥ T012 ∥ T013 ∥ T014 ∥ T015; T020 ∥ T016
- T025 ∥ T026 ∥ T027 ∥ T028; T036 ∥ T034
- T038 ∥ T039 ∥ T040 ∥ T041
- Frontend of each story ∥ backend of the same story once contracts are fixed

---

## Parallel Example: User Story 1

```bash
Task: "Service tests for Review and Due in backend/internal/vocab/service_test.go"
Task: "Handler tests for due and review in backend/internal/vocab/handler_test.go"
Task: "interval-label and answer-match specs in frontend/src/app/shared/utils/"
Task: "review-session spec in frontend/src/app/shared/components/review-session/"
Task: "Review page spec in frontend/src/app/features/vocabulary/review/"
```

---

## Implementation Strategy

1. Setup + Foundational → US1 (ôn thẻ đến hạn với thẻ F3 đã có) → **STOP and VALIDATE** with quickstart bước 3
2. US2 → sổ từ: xem, tìm, lọc, thêm, sửa, xoá
3. US3 → mục Từ vựng trong bước Đọc
4. Polish → README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (`GOTOOLCHAIN=local`)
- Đồng hồ giả: `Now func() time.Time` trong `vocab.Deps`; múi giờ lấy qua port, không nhận từ client
- Gửi dữ liệu tiếng Việt khi test bằng curl trên Windows: dùng `--data-binary @file.json`
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- **go-fsrs không an toàn khi dùng chung**: `Repeat`/`Next` ghi `seed` vào `Parameters` của bộ lập lịch, nên một biến toàn
  cục bị `go test -race` báo tranh chấp (và sẽ tranh chấp thật khi nhiều request cùng lúc). `schedule.go` tạo bộ lập lịch mới
  cho mỗi lần tính (`newScheduler()`), chi phí không đáng kể.
- Mặc định FSRS: thẻ mới Again/Hard/Good → 1/5/10 phút (Learning), Easy → 16 ngày (Review).
- Service tách theo file: `service.go` (lưu thẻ), `review.go` (đến hạn, đánh giá), `notebook.go` (danh sách, bài, sửa, xoá),
  `bulk.go` (Lưu tất cả); handler tương ứng `*_handler.go`. Test: `schedule_test.go`, `review_test.go`, `notebook_test.go`,
  `bulk_test.go` và các `*_handler_test.go`; fake chung ở `fake_test.go` (`newEnv()` có đồng hồ đặt được).
- JSON thẻ luôn có `due` (UTC), `reps`, `state`; thẻ F3 cũ hiện lịch suy ra (`effective`) chứ không ghi vào DB cho tới lần
  đánh giá đầu tiên. Trường lịch lưu `omitempty`, nên thẻ mới không có `reps`/`state` trong DB (thiếu = 0).
- `review-session`: nút **Lật thẻ** riêng (không bọc từ trong nút) để trình đọc màn hình đọc từ rồi mới tới hành động;
  Space/Enter trên vùng thẻ cũng lật; `aria-keyshortcuts` 1–4 trên nút đánh giá; "Đã ôn N thẻ" đếm số thẻ khác nhau,
  số lần từng mức đếm cả lần hiện lại.
- Mục **Từ vựng** chỉ tải khi mở (không thêm request cho trang Đọc); nút Lưu một từ cũng dùng `POST /api/vocab/cards/bulk`.
- Header thêm link **Sổ từ** sau email (email tự rút gọn ở 360px).
- Smoke test Docker (T049): bulk 2 lần → `added` 2 rồi 0; thêm thẻ tự do → 201, `lessonId: null`, `due` = 17:00Z (0 giờ
  giờ Việt Nam hôm sau), trùng → 409; thẻ F3 cũ (tạo bằng mongosh, 2 ngày trước) có trong danh sách đến hạn; Good → Learning
  sau 10 phút, gửi lại cùng `reps` → 409, Easy → Review sau 7 ngày; PATCH nghĩa giữ `due`/`reps`, PATCH `text` → 400; xoá →
  204, log còn 0, lần hai 404, tài khoản khác 404; mọi endpoint 401 khi không có phiên.
- Tương phản (T050): "✓ Đúng" `--color-ok`/surface 4.52 (sáng) / 7.97 (tối); "✗ Sai" 5.17 / 6.08; viền nút đánh giá
  Again 5.17/6.08, Hard 3.59/9.25, Good 4.52/7.97, Easy 7.37/6.22; "✓ Đã lưu" 9.93/9.57; tiêu đề ngày muted/bg 5.80/7.26; kiểu
  ôn đang chọn 7.37/6.87.
