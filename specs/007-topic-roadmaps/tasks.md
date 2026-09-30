---

description: "Task list for F14 – Chủ đề và lộ trình theo trình độ"
---

# Tasks: Chủ đề và lộ trình theo trình độ (F14)

**Input**: Design documents from `specs/007-topic-roadmaps/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: topic service (duplicate name per level, delete blocked while
lessons remain, roadmap only holds the topic's lessons, warning per topic, level change follows lessons), lesson service (topic
required, level from topic, topic change moves roadmap), `PlanMigration` (lessons without topic, merged names, kept order,
deleted ids, re-run), endpoint tests (success, invalid input, 401/403); no network (fakes for repositories and ports). Every
acceptance criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US4 from spec.md). This feature changes F2 code
(`specs/003-lesson-admin`).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 Record the current F2 data for the migration check (quickstart.md bước 0.1): export lessons (`title, level, topic`) and the `roadmap` document to `specs/007-topic-roadmaps/migration-before.json` (not committed if it holds personal content)
- [X] T002 [P] Create `frontend/src/app/core/models/topic.ts` (`Topic`, `TopicInput`, `TopicRoadmap`) per data-model.md §6

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Create `backend/internal/topic/model.go` (`Topic`, `Input`, `Summary`, `Roadmap`, `LessonRef`, `ErrNotFound`, `ErrNameTaken`, `*InUseError`, `*ValidationError`, `MinRemaining = 3`, `NameKey(name)`) and `backend/internal/topic/repository.go` (`Repository`, port `Lessons`) per data-model.md §5
- [X] T004 Create `backend/internal/topic/fake_test.go`: in-memory `Repository` enforcing unique `(level, nameKey)`, fake `Lessons` (lesson → topic map, titles, statuses; `SetLevelByTopic` records calls)
- [X] T005 Change `backend/internal/lesson`: `Lesson.TopicID` replaces `Topic`; `Summary` gets `TopicID`, `TopicName`; `Input` drops `Level`/`Topic`, adds `TopicID`; `Filter{Level, TopicID}`; add `TopicRef`, port `Topics` and `ErrTopicNotFound` in `model.go`/`repository.go`; drop `RoadmapRepository`, `Roadmap`, `ErrInRoadmap` stays; update `validate.go` (`topicId` required, "Vui lòng chọn chủ đề") and `validate_test.go`; add fake `Topics` to `backend/internal/lesson/fake_test.go` and make the package compile (service/handler bodies adapted minimally, roadmap endpoints removed)
- [X] T006 [P] Create `backend/internal/storage/mongo/topics.go` (`topic.Repository`: create with `nameKey`, duplicate key → `ErrNameTaken`, list sorted by level then name, update, delete, `SetLessons`, `RemoveLesson` `$pull`, `AppendLesson` `$push` guarded by `$ne`) and update `backend/internal/storage/mongo/indexes.go` (unique `topics (level, nameKey)`, `lessons (topicId)`, drop the `lessons (topic)` index if present)
- [X] T007 Update `backend/internal/storage/mongo/lessons.go`: `topicId` field (ObjectID), filter by `level`/`topicId`, new methods `CountByTopic`, `TopicOf`, `Refs`, `SetLevelByTopic`; delete `backend/internal/storage/mongo/roadmap.go`
- [X] T008 [P] Update `frontend/src/app/core/models/lesson.ts` (`topicId`, `topicName`; `LessonInput` with `topicId`; `LessonFilter {level, topicId}`; remove `Roadmap`) and fix compile errors with minimal changes in `features/admin` (full UI changes come in the stories)

**Checkpoint**: backend and frontend compile; existing tests updated for the new fields pass

---

## Phase 3: User Story 1 - Quản lý danh mục chủ đề (Priority: P1) 🎯 MVP

**Goal**: Thêm, sửa, xoá chủ đề; danh sách nhóm theo trình độ

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T009 [P] [US1] Write `backend/internal/topic/service_test.go` (CRUD): create validates name 1–60 (trimmed), level A1–C2, description ≤ 200; duplicate name in the same level (case and extra spaces ignored) → `name` field error "Chủ đề này đã có ở trình độ A1", same name in another level allowed; list sorted A1→C2 then name with `LessonCount` from `Lessons.CountByTopic`, `Remaining = len(LessonIDs)`, `Warning` when < 3; update renames and, when the level changes, calls `SetLevelByTopic` (and rejects a duplicate in the new level); delete blocked with `*InUseError{Count}` while lessons remain, allowed when empty; unknown id → `ErrNotFound`
- [X] T010 [P] [US1] Write `backend/internal/topic/handler_test.go` (CRUD): `GET/POST /api/admin/topics`, `PUT/DELETE /api/admin/topics/{id}` success, 400 fields and unknown body field, 404, 409 `topic_in_use` with `count`; learner → 403, anonymous → 401; `GET /api/topics?level=A1` for a learner returns `{id, name, level, description, lessonCount}` only
- [X] T011 [P] [US1] Write `frontend/src/app/features/admin/topics/topics.spec.ts`: topics grouped under level headings A1…C2 with lesson count, roadmap count and description; add form (name, level select, description) with Vietnamese errors, 400 `name` error shown on the field; edit in place; delete opens `ConfirmDialog`, 409 shows "Chủ đề còn N bài…"; empty state

### Implementation for User Story 1

- [X] T012 [US1] Implement `topic.Service` CRUD (`NewService(repo, lessons, now)`, `List`, `Create`, `Update`, `Delete`, `Public(level)`) in `backend/internal/topic/service.go` so T009 passes
- [X] T013 [US1] Implement `Handler` (`Register(mux, requireAdmin, requireAuth)`) for topic CRUD and `GET /api/topics` in `backend/internal/topic/handler.go` so T010 passes
- [X] T014 [US1] Wire in `backend/cmd/api/main.go`: `mongo.NewTopics(database)`, `topic.Lessons` adapter over `mongo.NewLessons` (CountByTopic, TopicOf, Refs, SetLevelByTopic), topic service and handler
- [X] T015 [US1] Add topic methods to `frontend/src/app/features/admin/admin-api.service.ts` (`topics(level?)`, `createTopic`, `updateTopic`, `deleteTopic`) with cases in `admin-api.service.spec.ts`
- [X] T016 [US1] Generate `ng g c features/admin/topics`, implement it, add route `topics` (title "Chủ đề · Luna") in `frontend/src/app/features/admin/admin.routes.ts` and a "Chủ đề" link next to the existing admin navigation, so T011 passes
- [ ] T017 [US1] Manual check: quickstart.md bước 1

**Checkpoint**: Danh mục chủ đề hoạt động

---

## Phase 4: User Story 2 - Gán chủ đề cho bài và lọc danh sách bài (Priority: P1)

**Goal**: Bài bắt buộc có chủ đề, trình độ theo chủ đề, đổi chủ đề chuyển lộ trình, lọc theo trình độ + chủ đề

**Independent Test**: quickstart.md bước 2

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T018 [P] [US2] Update `backend/internal/lesson/service_test.go`: create/update without `topicId` → field error; unknown topic → "Chủ đề không tồn tại"; level stored = topic level; update to another topic while in the old roadmap → `Topics.MoveLesson(id, old, new)` called; same topic → not called; list filter by `Level` and `TopicID`, `TopicName` and `InRoadmap` from the `Topics` port; delete blocked when the lesson is in a topic roadmap (`ErrInRoadmap`)
- [X] T019 [P] [US2] Add to `backend/internal/topic/service_test.go`: `MoveLesson` removes from the old roadmap and appends to the end of the new one only when it was in the old roadmap; `Get`, `Names`, `RoadmapLessonIDs` (port `lesson.Topics`)
- [X] T020 [P] [US2] Update `backend/internal/lesson/handler_test.go`: lesson JSON has `topicId`, `topicName`, no `topic`; body with `level` or `topic` → 400 `invalid_body`; `GET /api/admin/lessons?level=A1&topicId=…` filters; `/api/admin/roadmap` → 404; update `reading_handler_test.go` so `topic` is the topic name
- [X] T021 [P] [US2] Update `frontend/src/app/features/admin/lesson-form/lesson-form.spec.ts`: one topic select with `<optgroup>` per level, no level/topic inputs, "Vui lòng chọn chủ đề", posts `topicId`, edit preselects the topic, no topics → link "Tạo chủ đề" to `/admin/topics`
- [X] T022 [P] [US2] Update `frontend/src/app/features/admin/lesson-list/lesson-list.spec.ts` (filters): level select and topic select; choosing a level limits topic options and clears a non-matching topic; requests send `level` and `topicId`; rows show level and topic name

### Implementation for User Story 2

- [X] T023 [US2] Implement `lesson.Service` changes (create/update from topic, `MoveLesson`, list names and `InRoadmap`, delete check via port) in `backend/internal/lesson/service.go` and `reading.go` (topic name via port) so T018 passes
- [X] T024 [US2] Implement `topic.Service` port methods (`Get`, `Names`, `RoadmapLessonIDs`, `MoveLesson`) in `backend/internal/topic/service.go` so T019 passes
- [X] T025 [US2] Update `backend/internal/lesson/handler.go` and `reading_handler.go` (JSON, filters, removed roadmap routes) so T020 passes; wire `lesson.Topics` = topic service in `backend/cmd/api/main.go` (lesson service and reader)
- [X] T026 [US2] Update `lesson-form` (`frontend/src/app/features/admin/lesson-form/`) and `lesson-list` filters (`frontend/src/app/features/admin/lesson-list/`) and `AdminApiService.list` so T021, T022 pass; show topic name in `lesson-detail` (+ spec assertion)
- [ ] T027 [US2] Manual check: quickstart.md bước 2

**Checkpoint**: Bài gắn với chủ đề, lọc được

---

## Phase 5: User Story 3 - Lộ trình riêng cho từng chủ đề (Priority: P1)

**Goal**: Chọn chủ đề, thêm/gỡ/kéo thả bài của chủ đề, cảnh báo dưới 3 bài theo từng chủ đề

**Independent Test**: quickstart.md bước 3

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T028 [P] [US3] Add to `backend/internal/topic/service_test.go` (roadmap): `Roadmap(id)` returns lessons in order skipping deleted ones, `Remaining`, `Warning` for 0, 1, 2 lessons and none for 3; `SetRoadmap` rejects duplicates, unknown lessons and lessons of another topic (`lessonIds` field error "Chỉ thêm được bài của chủ đề này"), saves the order
- [X] T029 [P] [US3] Add to `backend/internal/topic/handler_test.go`: `GET/PUT /api/admin/topics/{id}/roadmap` 200, 400 cases, 404, 403/401
- [X] T030 [P] [US3] Update `frontend/src/app/features/admin/roadmap/roadmap.spec.ts`: topic select (optgroups) driving `?topicId=`, loads `/api/admin/topics/{id}/roadmap`, lists the topic's lessons not in the roadmap to add, drag-drop and Up/Down save via PUT, remove, warning block listing every topic with `warning` ("A1 · Gia đình: còn 2 bài chưa học", empty roadmap "Lộ trình chưa có bài") where clicking selects that topic
- [X] T031 [P] [US3] Update `lesson-list.spec.ts` (roadmap): "Thêm vào lộ trình" adds to the lesson's topic roadmap (GET + PUT topic roadmap); the old global warning banner becomes a link "N chủ đề sắp hết bài" to `/admin/roadmap`

### Implementation for User Story 3

- [X] T032 [US3] Implement `topic.Service.Roadmap` and `SetRoadmap` in `backend/internal/topic/service.go` and roadmap routes in `handler.go` so T028, T029 pass
- [X] T033 [US3] Add `topicRoadmap(id)`, `setTopicRoadmap(id, ids)` to `AdminApiService` (remove `getRoadmap`/`saveRoadmap`) with spec cases; update `frontend/src/app/features/admin/roadmap/` so T030 passes and `lesson-list` so T031 passes
- [ ] T034 [US3] Manual check: quickstart.md bước 3

**Checkpoint**: Mỗi chủ đề một lộ trình

---

## Phase 6: User Story 4 - Chuyển dữ liệu cũ sang chủ đề (Priority: P1)

**Goal**: Dữ liệu F2 tự chuyển khi khởi động, không mất bài, giữ thứ tự, chạy lại không trùng

**Independent Test**: quickstart.md bước 0

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- [X] T035 [P] [US4] Write `backend/internal/topic/migrate_test.go` for `PlanMigration`: A1 "Gia đình" + A1 "gia đình " merge into one topic keeping the first-created spelling; lesson without topic → "Chung" of its level; B1 "Công việc" separate; roadmap X(A1), Y(B1), Z(A1) → A1 [X, Z], B1 [Y]; lessons outside the old roadmap assigned but not added; roadmap ids of deleted lessons skipped; lesson already having `TopicID` kept; existing topic with the same `(level, nameKey)` reused (no new topic); every lesson assigned exactly once; empty input → empty plan

### Implementation for User Story 4

- [X] T036 [US4] Implement `PlanMigration` in `backend/internal/topic/migrate.go` so T035 passes
- [X] T037 [US4] Implement `MigrateTopics(ctx, db, log)` in `backend/internal/storage/mongo/migrate.go` (marker `migrations {_id: "f14-topics"}`, read lessons/roadmap/topics, apply plan: upsert topics by `(level, nameKey)`, write `lessonIds` only when the roadmap document still exists, `$set topicId, level` + `$unset topic` per lesson, write marker, drop `roadmap`) and call it in `backend/cmd/api/main.go` after indexes and before serving (fail start-up on error)
- [X] T038 [US4] Manual check: quickstart.md bước 0 (compare with T001 export, restart twice, no duplicates, order kept)

**Checkpoint**: Mọi user story hoạt động độc lập trên dữ liệu thật

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T039 Update `README.md` (Soạn bài học: chủ đề, lộ trình theo chủ đề, chuyển dữ liệu tự động) and mark the replaced parts in `specs/003-lesson-admin/spec.md` (FR-024 and topic field: "thay bởi F14")
- [X] T040 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T041 Run all tests (`go test -race ./...`, `npm test -- --watch=false`) with zero failures; `npx ng build` without warnings
- [X] T042 Rebuild and smoke test in Docker with `curl` and UTF-8 body files: topic CRUD, duplicate name, delete in use, lesson create with `topicId`, change topic moves roadmap, roadmap with another topic's lesson → 400, filters, learner `/api/topics`, 401/403, old `/api/admin/roadmap` → 404
- [X] T043 Compute WCAG contrast for new UI pairs (level headings, warning list, optgroup labels) in both modes
- [ ] T044 Manual check at 360px width in light and dark mode (quickstart.md bước 5)
- [ ] T045 Tick the 4 F14 acceptance criteria in `docs/phases/giai-doan-1.md` once T017, T027, T034, T038, T042, T044 pass

---

## Dependencies & Execution Order

- **Setup (Phase 1)** → **Foundational (Phase 2)** → **US1** → **US2** (needs topics to assign) → **US3** (roadmap per topic)
  → **US4** (migration writes the final data shape)
- T001 must run **before** the new backend is started against the existing database (the migration then runs once)
- US4 logic (T035, T036) only needs Phase 2 and can be written in parallel with US1–US3; applying it (T037, T038) comes last
- **Polish** after all stories

### Parallel Opportunities

- T002 ∥ T003; T006 ∥ T007 ∥ T008 (after T003, T005)
- T009 ∥ T010 ∥ T011; T018 ∥ T019 ∥ T020 ∥ T021 ∥ T022; T028 ∥ T029 ∥ T030 ∥ T031
- T035 ∥ any US1–US3 task

---

## Parallel Example: User Story 2

```bash
Task: "Lesson service tests in backend/internal/lesson/service_test.go"
Task: "Topic port tests in backend/internal/topic/service_test.go"
Task: "Lesson handler tests in backend/internal/lesson/handler_test.go"
Task: "lesson-form spec in frontend/src/app/features/admin/lesson-form/"
Task: "lesson-list filter spec in frontend/src/app/features/admin/lesson-list/"
```

---

## Implementation Strategy

1. Setup + Foundational (export F2 data first) → US1 → **STOP and VALIDATE** with quickstart bước 1 (on a fresh DB)
2. US2 → bài gắn chủ đề, lọc
3. US3 → lộ trình theo chủ đề, cảnh báo
4. US4 → chuyển dữ liệu thật; chỉ triển khai lên DB đang có dữ liệu sau bước này
5. Polish → README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (`GOTOOLCHAIN=local`)
- `topic` và `lesson` không import nhau; chỉ `main.go` nối (research R1)
- Gửi dữ liệu tiếng Việt khi test bằng curl trên Windows: dùng `--data-binary @file.json`
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- **Thời điểm chuyển dữ liệu (lệch T037):** tasks ghi "chạy trước khi nhận request, lỗi thì dừng khởi động". Làm vậy sẽ phá
  hành vi F0 (backend vẫn khởi động và `/api/health` báo DB down khi Mongo chưa lên). `mongo.PrepareInBackground` (đổi tên
  từ `EnsureIndexesInBackground`) chạy `MigrateTopics` rồi `EnsureIndexes`, thử lại mỗi 10 giây; khi DB sẵn sàng, việc chuyển
  dữ liệu xong trong vài mili giây sau khi khởi động.
- Kiểm tra chuyển dữ liệu trên Docker (T038): dữ liệu F2 mẫu tạo bằng bản cũ qua API (`migration-before.json`): "Reading
  test" (B1, không chủ đề, trong lộ trình), X (A1 "Gia đình"), Y (B1 "Công việc"), Z (A1 "gia đình"), W (A2, không chủ đề,
  ngoài lộ trình); lộ trình cũ X, Reading test, Y, Z. Kết quả: A1 · Gia đình [X, Z], B1 · Chung [Reading test], B1 · Công việc
  [Y], A2 · Chung []; mọi bài có `topicId`, không còn `topic`; collection `roadmap` đã xoá; marker `f14-topics`. Khởi động lại
  backend: dữ liệu giống hệt.
- `PlanMigration` giữ cách viết tên của bài tạo sớm nhất và gộp khác hoa thường/khoảng trắng; ghi chủ đề bằng upsert theo
  `(level, nameKey)`; `lessonIds` chỉ ghi khi còn document `roadmap` (lần chạy lại sau khi đã xoá roadmap không làm rỗng lộ
  trình).
- `lesson` và `topic` không import nhau: `main.go` có `topicLessons` (topic.Lessons trên `mongo.Lessons`) và
  `lessonTopicsPort` (lesson.Topics trên `topic.Service`). `lesson.NewReader` nhận thêm `Topics` để trang Đọc giữ trường `topic`
  = tên chủ đề.
- Danh sách bài: banner cảnh báo chung của F2 thành "N chủ đề sắp hết bài chưa học" + link trang Lộ trình; nút "Thêm vào lộ
  trình" thêm vào lộ trình của chủ đề của bài. Trang Lộ trình nhớ chủ đề trong `?topicId=` và có phần "Bài của chủ đề chưa có
  trong lộ trình".
- Smoke test Docker (T042): tạo chủ đề 201, trùng "  mua  SẮM " → 400; xoá chủ đề còn 2 bài → 409 `count: 2`; `/api/admin/roadmap`
  → 404; `/api/topics?level=A1` cho người học 200, `/api/admin/topics` cho người học 403, ẩn danh 401; lọc `level` + `topicId`;
  đổi chủ đề X (đang ở lộ trình Gia đình) sang Mua sắm → Gia đình [Z], Mua sắm [X], trình độ theo chủ đề; PUT lộ trình với bài
  chủ đề khác → 400 "Chỉ thêm được bài của chủ đề này"; body cũ có `level`/`topic` → 400; đổi trình độ Công việc B1 → B2 kéo bài
  theo (đã đặt lại B1).
- Dữ liệu mẫu để thử tay còn trong DB: bài "F14 X/Y/Z/W …" và "F14 N mua sam", chủ đề "A1 · Mua sắm".
- Tương phản (T043): tiêu đề trình độ primary-ink/bg 9.10 (sáng) / 10.52 (tối); link trong khối cảnh báo 9.93 / 9.57; số bài
  (muted/surface) 6.33 / 6.60.
