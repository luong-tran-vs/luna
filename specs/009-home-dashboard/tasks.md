---

description: "Task list for F6 – Màn hình chính và tiến độ"
---

# Tasks: Màn hình chính và tiến độ (F6)

**Input**: Design documents from `specs/009-home-dashboard/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: pure helpers (`TomorrowCutoff` across timezones and DST, `Action`),
dashboard service in the 4 states (no goal, studying, done today, no new lesson) with no writes, skill counts restricted to the
current roadmap, stats for a new learner and the rate, endpoints (200, 401), new components and pages; no network (fakes for
repositories and ports, fake clock). Every acceptance criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US4 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Create `frontend/src/app/core/models/dashboard.ts` (`DashboardKind`, `Dashboard` with `goal`, `goalCompleted`, `skills`, `lesson {id, title, topicName, level}`, `steps`, `currentStep`, `action {kind: 'start' | 'continue', step}`, `streak`, `tomorrowCards`; `Stats` with `cards`, `dictation {sentences, correctWords, totalWords, rate: number | null}`, `lessons {read, listen, completed}`), reusing `Step`, `StepState`, `GoalView` from `core/models/study.ts` (contracts/dashboard-api.md)
- [X] T002 [P] Add types `StepCounts{Read, Listen, Completed int}` and `DictationTotals{Sentences, CorrectWords, TotalWords int}` to `backend/internal/progress/study_model.go` and `backend/internal/progress/model.go`; extend interfaces: `ProgressRepository.StepCounts(ctx, userID, lessonIDs []string)` (nil = every lesson) in `backend/internal/progress/study_repository.go`, `Reviews.DueBefore(ctx, userID, before, createdBefore time.Time) (int, error)` and `Reviews.CardCount(ctx, userID) (int, error)` in the same file, `DictationRepository.Totals(ctx, userID) (DictationTotals, error)` in `backend/internal/progress/repository.go` (data-model.md "Phương thức repository mới")

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 [P] Add `CountDue(ctx, userID, before, createdBefore time.Time) (int, error)` and `Count(ctx, userID) (int, error)` to `vocab.Repository` in `backend/internal/vocab/repository.go`; implement in `backend/internal/storage/mongo/cards.go` with `CountDocuments` (`due < before` OR no `due` and `createdAt < createdBefore`; `{userId}`); add `Service.DueBefore(ctx, userID, before, createdBefore)` in `backend/internal/vocab/review.go` and `Service.Count(ctx, userID)` in `backend/internal/vocab/notebook.go`; extend the vocab fake repository and add service tests (due before cutoff counted, at cutoff not, legacy card created before `createdBefore` counted, other users isolated) in `backend/internal/vocab/`
- [X] T004 [P] Implement `LessonProgress.StepCounts` in `backend/internal/storage/mongo/study.go` (aggregate `$match {userId, lessonId: {$in}}` only when lessonIDs != nil, empty non-nil slice returns zeros without querying; `$group` summing `$cond` on `steps.read`, `steps.listen`, `completedAt` existing) and `DictationResults.Totals` in `backend/internal/storage/mongo/dictation.go` (`$match {userId}`, `$group` count + sum `correctWords` + sum `totalWords`); no documents → zeros (research R3, R5)
- [X] T005 Extend `backend/internal/progress/study_fake_test.go` and `backend/internal/progress/fake_test.go`: fake `StepCounts` computed from stored lesson progress, fake `Totals` from stored dictation results, fake `Reviews.DueBefore` (list of due times + legacy created times) and `CardCount`; wire into `newStudyEnv()`
- [X] T006 Update `dailyReviews` in `backend/cmd/api/main.go` with `DueBefore` and `CardCount` (call `vocab.Service.DueBefore`, `vocab.Service.Count`); add `progress.Service.Totals(ctx, userID)` in `backend/internal/progress/service.go`; `go build ./...` passes
- [X] T007 Write `backend/internal/progress/dashboard_test.go` (pure part): `TomorrowCutoff` = 00:00 of the day after tomorrow in Asia/Ho_Chi_Minh at 00:01 and 23:59, in America/New_York across the DST change, same instant different timezones; `Action`: no step done → `start` with current step, review done → `continue` `read`, current `review` with 0 cards → `start` `read`, current `done` → nil
- [X] T008 Implement `TomorrowCutoff(now, loc)` and `Action(steps, currentStep, reviewCount)` in `backend/internal/progress/dashboard.go` (pure) so T007 passes (research R2, R4)
- [X] T009 [P] Create `frontend/src/app/core/services/dashboard-api.service.ts` (`ng g s core/services/dashboard-api --type=service`) with `dashboard()` (`GET /api/dashboard`) and `stats()` (`GET /api/stats`) and spec cases in `dashboard-api.service.spec.ts`
- [X] T010 [P] Create `frontend/src/app/shared/components/progress-bar/` (`ng g c shared/components/progress-bar`): inputs `label`, `value`, `max`, `unit` (e.g. "bài", "bước"), `tone: 'primary' | 'read' | 'listen'` (fill `--color-primary` / `--color-skill-read` / `--color-skill-listen`, track `--color-track`); `role="progressbar"`, `aria-valuenow/min/max`, `aria-label`, visible text "value/max unit"; `max = 0` → empty bar, no NaN; spec in `progress-bar.spec.ts` (aria values, width %, max 0, tone attribute) (research R10)

**Checkpoint**: helpers tested; repositories, ports, API client and progress bar ready

---

## Phase 3: User Story 1 - Mở app là biết hôm nay làm gì (Priority: P1) 🎯 MVP

**Goal**: Màn hình chính hiện mục tiêu, bài hôm nay + thanh bước, nút Tiếp tục/Bắt đầu, streak, thẻ ngày mai

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T011 [P] [US1] Extend `backend/internal/progress/dashboard_test.go` (service, studying): goal name/level/`completedLessons`/`totalLessons`; lesson title, topic name and level (from the lesson progress topic when started, else the goal roadmap); steps and `currentStep` as `/api/today`; `action` start/continue; `streak`; `tomorrowCards` uses the cutoff in the learner's timezone and includes overdue cards; deleted lesson → empty title; calling `Dashboard` twice with no due cards creates no study day and no lesson progress (read-only, research R1); other users isolated
- [X] T012 [P] [US1] Write `backend/internal/progress/dashboard_handler_test.go`: `GET /api/dashboard` 200 JSON shape per contracts/dashboard-api.md (all keys present, `steps` has 3 keys, `null` fields), 401 without session, 500 on repository error
- [X] T013 [P] [US1] Update `frontend/src/app/shared/components/step-indicator/step-indicator.spec.ts`: the "N/3 bước" line is a `lu-progress-bar` with `aria-valuenow` = done count, `aria-valuemax` = 3
- [X] T014 [P] [US1] Rewrite `frontend/src/app/features/home/home.spec.ts` (studying): shows "A1 · Gia đình", goal bar "2/12 bài", "Đổi chủ đề" link to `/goal`, lesson title + "A1 · Gia đình", step indicator, streak "🔥 4 ngày", "Ngày mai: 17 thẻ cần ôn", "Ngày mai: không có thẻ cần ôn" for 0; button text "Tiếp tục: Đọc" / "Bắt đầu: Ôn" / "Bắt đầu: Đọc" linking `/today`; empty title → "Bài học"; loading placeholder; load error message with retry

### Implementation for User Story 1

- [X] T015 [US1] Implement `StudyService.Dashboard(ctx, userID)` and `DashboardView` in `backend/internal/progress/dashboard.go`: `load` + `view` (not `Today`), lesson topic via `Roadmaps.Roadmap(progress.TopicID)` or the goal topic, `Action`, `Reviews.DueBefore(TomorrowCutoff, start of tomorrow)` so T011 passes
- [X] T016 [US1] Create `backend/internal/progress/dashboard_handler.go`: `GET /api/dashboard` (auth required) with JSON per contracts/dashboard-api.md; register from `StudyHandler.Register` in `backend/internal/progress/study_handler.go` so T012 passes
- [X] T017 [US1] Use `lu-progress-bar` (tone primary, unit "bước") for the "N/3 bước" line in `frontend/src/app/shared/components/step-indicator/step-indicator.ts` so T013 passes; `/today` keeps working (`today.spec.ts` green)
- [X] T018 [US1] Rewrite `frontend/src/app/features/home/home.ts`, `home.html`, `home.css` as the dashboard (studying state): top row "Hôm nay" + streak, goal card (name, `lu-progress-bar` tone primary unit "bài", "Đổi chủ đề"), lesson card (title max 2 lines, "level · topic", `lu-step-indicator`), full-width primary button ≥ 48px "Tiếp tục: <bước>" / "Bắt đầu: <bước>" → `/today`, then "Ngày mai: N thẻ cần ôn"; fetch `dashboard()` on init (no caching), placeholder of the same height while loading; remove the F0 connection block and L's "Học hôm nay" button; layout per research R11 so T014 passes

**Checkpoint**: MVP — quickstart bước 1 ở 360px

---

## Phase 4: User Story 2 - Trạng thái đặc biệt (Priority: P1)

**Goal**: Chưa có mục tiêu, xong hôm nay, chưa có bài mới có thông báo và nút đúng; trạng thái kết nối chỉ hiện khi lỗi, ở chân trang

**Independent Test**: quickstart.md bước 3

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T019 [P] [US2] Extend `backend/internal/progress/dashboard_test.go` with states: no goal → `noGoal`, `goal`/`skills`/`lesson`/`action` nil, streak and tomorrow cards still set; done today → `doneToday`, lesson present, all steps done, `action` nil; roadmap finished or empty → `noNewLesson`, `goalCompleted` true when all done, `action` nil; empty roadmap → `totalLessons` 0
- [X] T020 [P] [US2] Extend `frontend/src/app/features/home/home.spec.ts` with states: `noGoal` → invitation + "Chọn chủ đề" (`/goal`), no goal bar/lesson card; `doneToday` → "Đã xong bài hôm nay, hẹn bạn ngày mai", full step indicator, "Ôn tự do" (`/vocabulary/review`), tomorrow cards; `noNewLesson` → "Chưa có bài mới", "Ôn tự do", "Chọn chủ đề khác" (`/goal`); `goalCompleted` → congratulations text
- [X] T021 [P] [US2] Write `frontend/src/app/shared/components/connection-status/connection-status.spec.ts`: renders nothing when `connected` or `checking`; shows the F0 messages for `database-down` and `server-unreachable` with `role="status"`; re-checks after `NavigationEnd`; hides again once healthy

### Implementation for User Story 2

- [X] T022 [US2] Make T019 pass in `backend/internal/progress/dashboard.go` (fix any gaps in `Dashboard` for the three states)
- [X] T023 [US2] Add the `noGoal`, `doneToday`, `noNewLesson` (+ `goalCompleted`) views to `frontend/src/app/features/home/home.html` / `home.ts` so T020 passes
- [X] T024 [US2] Create `frontend/src/app/shared/components/connection-status/` (`ng g c shared/components/connection-status`) by moving the F0 health check and messages out of the old home (`ApiService.health()`, check on init and after every `NavigationEnd`, render only on error); add `<footer>` with `<lu-connection-status />` in `frontend/src/app/app.html` (+ `app.css`, `app.ts` imports) so T021 passes; update `frontend/src/app/app.spec.ts` if it asserted the old home status (research R9)

**Checkpoint**: 4 trạng thái của màn hình chính đều đúng

---

## Phase 5: User Story 3 - Tiến độ theo kỹ năng và số liệu cập nhật ngay (Priority: P2)

**Goal**: Thanh Đọc và Nghe trong lộ trình hiện tại; số liệu mới mỗi lần quay về

**Independent Test**: quickstart.md bước 2

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T025 [P] [US3] Extend `backend/internal/progress/dashboard_test.go`: `skills` counts read/listen done only for lessons in the current roadmap (a lesson of an old topic is not counted), `total` = roadmap length; after `CompleteStep(read)` the next `Dashboard` shows read +1 and `action` `continue` `listen`; after `CompleteStep(listen)` listen +1, goal +1, streak +1, `doneToday`
- [X] T026 [P] [US3] Extend `frontend/src/app/features/home/home.spec.ts`: two skill bars "Đọc 3/12" (tone read) and "Nghe 2/12" (tone listen), no Viết/Nói; no skill bars without a goal; creating the component again (navigating back) calls `dashboard()` again and shows the new numbers

### Implementation for User Story 3

- [X] T027 [US3] Add `Skills` to `DashboardView` in `backend/internal/progress/dashboard.go` via `Progress.StepCounts(ctx, userID, topic.LessonIDs)` and to the JSON in `backend/internal/progress/dashboard_handler.go` so T025 passes
- [X] T028 [US3] Add the two small skill bars (`lu-progress-bar` tone read/listen, unit "bài") under the goal bar in `frontend/src/app/features/home/home.html` / `home.css` so T026 passes

**Checkpoint**: số liệu cập nhật sau mỗi bước

---

## Phase 6: User Story 4 - Trang thống kê (Priority: P3)

**Goal**: `/stats` hiện số từ, số câu đã chép, tỷ lệ đúng, số bài theo kỹ năng

**Independent Test**: quickstart.md bước 5

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- [X] T029 [P] [US4] Write `backend/internal/progress/stats_test.go`: new learner → zeros and nil rate; 25 cards, 40 sentences 328/400 → rate 0.82; read/listen/completed over every lesson (including old topics); other users isolated
- [X] T030 [P] [US4] Extend `backend/internal/progress/dashboard_handler_test.go`: `GET /api/stats` 200 JSON shape (`rate` null for new learner), 401
- [X] T031 [P] [US4] Write `frontend/src/app/features/stats/stats.spec.ts`: "25 từ đã học", "40 câu đã chép chính tả", "Tỷ lệ đúng 82%" (rounded), rate null → "—", lessons by skill and completed, load error with retry, link back to `/`

### Implementation for User Story 4

- [X] T032 [US4] Implement `StudyService.Stats(ctx, userID)` and `StatsView` in `backend/internal/progress/stats.go` (`Reviews.CardCount`, `Dictation.Totals`, `Progress.StepCounts(nil)`) so T029 passes
- [X] T033 [US4] Add `GET /api/stats` to `backend/internal/progress/dashboard_handler.go` so T030 passes
- [X] T034 [US4] Create `frontend/src/app/features/stats/stats` (`ng g c features/stats/stats`): heading "Thống kê", list of figures per contracts/dashboard-api.md; add route `stats` (`authGuard`, title "Thống kê · Luna") in `frontend/src/app/app.routes.ts`; add "Xem thống kê" link to `frontend/src/app/features/home/home.html` so T031 passes

**Checkpoint**: tất cả user story xong

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T035 Update `README.md`: section "Màn hình chính và thống kê" (các trạng thái, thẻ ngày mai, trang `/stats`, trạng thái kết nối ở chân trang)
- [X] T036 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T037 Run all tests (`go test -race ./...`, `npx ng test --watch=false`) with zero failures; `npx ng build` without warnings
- [X] T038 Rebuild and smoke test in Docker with `curl` (quickstart.md bước 6): `/api/dashboard` for `hoc@example.com` (studying), read-only check (no study day created), `/api/stats`, 401; index check with `explain` (bước 7)
- [X] T039 Compute WCAG contrast for new pairs (goal/skill bar fills on track, streak, footer error text) in both modes
- [ ] T040 Manual check at 360 × 640 in light and dark mode: goal bar, today's lesson and the button visible without scrolling; long titles wrap (quickstart.md bước 1, 3)
- [ ] T041 Tick the 3 F6 acceptance criteria in `docs/phases/giai-doan-1.md` once T018, T027/T028, T038, T040 pass

---

## Dependencies & Execution Order

- **Setup** → **Foundational** → **US1** → **US2** → **US3** → **US4** → **Polish**
- US2 and US3 extend `dashboard.go` and the home page after US1; US4 only needs Foundational (can run in parallel with US2/US3)
- Backend and frontend of each story can proceed in parallel once the contract is fixed

### Parallel Opportunities

- T001 ∥ T002; T003 ∥ T004 ∥ T009 ∥ T010 (after T002)
- T011 ∥ T012 ∥ T013 ∥ T014; T019 ∥ T020 ∥ T021; T025 ∥ T026; T029 ∥ T030 ∥ T031
- T024 (connection status) ∥ T022/T023

---

## Parallel Example: User Story 1

```bash
Task: "Dashboard service tests in backend/internal/progress/dashboard_test.go"
Task: "Dashboard handler tests in backend/internal/progress/dashboard_handler_test.go"
Task: "step-indicator spec in frontend/src/app/shared/components/step-indicator/"
Task: "home spec in frontend/src/app/features/home/"
```

---

## Implementation Strategy

1. Setup + Foundational → US1 (màn hình chính khi đang học) → **STOP and VALIDATE** với quickstart bước 1 ở 360px
2. US2 → các trạng thái đặc biệt, trạng thái kết nối ở chân trang
3. US3 → thanh kỹ năng, số liệu cập nhật
4. US4 → trang thống kê
5. Polish → README, lint, test, Docker, tương phản, 360px, tick tiêu chí nghiệm thu

---

## Notes

- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (`GOTOOLCHAIN=local`)
- `progress` không import `vocab`; chỉ `main.go` nối qua port `Reviews`
- Dashboard chỉ đọc: không gọi `Today`, `completeStep`, `Days.Start`
- Không thêm index (research R7)
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- Hàm thuần tên `NextAction` (không phải `Action`) vì `Action` là tên kiểu; `Dashboard` và `Stats` là phương thức của
  `StudyService` (`dashboard.go`, `stats.go`), route đăng ký trong `StudyHandler.Register`.
- Test service của dashboard/stats nằm trong `backend/internal/progress/dashboard_service_test.go` (hàm thuần ở
  `dashboard_test.go`). Handler test phủ 200 (hình dạng JSON) và 401; lỗi 500 dùng chung `StudyHandler.fail` đã có của L nên không
  thêm test riêng.
- Lỗi máy chủ trả `{"error":"internal_error"}` như các API L (contract ghi `internal`).
- `goal` trong `/api/dashboard` dùng lại JSON mục tiêu của L nên có thêm `status`, `effectiveFrom`.
- `connection-status`: lớp `ConnectionStatusBar` (tránh trùng kiểu `ConnectionStatus`); kiểm tra `/api/health` sau mỗi
  `NavigationEnd`, vùng `role="status"` luôn có trong DOM nhưng chỉ có chữ khi lỗi; chân trang `position: sticky; bottom: 0`.
- Trang `/stats` ở `frontend/src/app/features/stats/stats.ts`.
- Smoke test Docker (T038) với `hoc@example.com`: `/api/dashboard` → `studying`, "A1 · Gia đình" 0/1 bài, bài "F14 Z gia
  dinh", Ôn hiện tại, `action` start review, streak 1, 3 thẻ tới hết ngày mai; `/api/stats` → 3 thẻ, 4 câu, 12/17 (0.71), Đọc 1,
  Nghe 1, hoàn thành 1 — khớp đếm tay trong MongoDB; gọi dashboard không tạo `study_days` cho hôm nay; 401 khi thiếu phiên;
  truy vấn thẻ đến hạn dùng `IXSCAN userId_1_due_1`.
- Tương phản (T039): chữ báo lỗi ở chân trang là `--color-text` trên surface 16.38 / 13.77; chấm màu lỗi bad 5.17 / 6.08, warn
  3.59 / 9.25. Thanh kỹ năng trên nền track: Nghe 3.24 / 6.13, **Đọc 2.73** (sáng) / 6.99 — dưới 3:1 ở chế độ sáng, nhưng giá trị
  luôn có chữ "x/y" và `aria-valuetext`, nên thanh không phải cách duy nhất để biết số liệu; nếu muốn đạt 3:1 thì cần đậm
  `--color-skill-read` (ảnh hưởng design-system, chưa đổi).
- Còn lại cho người dùng: T040 kiểm tra 360 × 640 sáng/tối, rồi T041 tick tiêu chí F6.
