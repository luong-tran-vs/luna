---

description: "Task list for F12 – Cài đặt"
---

# Tasks: Cài đặt (F12)

**Input**: Design documents from `specs/010-user-settings/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: input validation (theme, limit 5–200, timezone valid / invalid / empty /
"Local"), partial updates, defaults for old accounts, endpoints (200, 400 per field, 401, own settings only), the "day never goes
back" rule and timezone changes in `progress` (same day, next day, previous day), limit changes mid-day, frontend settings page,
`SettingsApiService`, `ThemeService` sync, header link; no network (fakes for repositories and ports, fake clock). Every
acceptance criterion maps to a test task or a manual check task.

**Organization**: Tasks are grouped by user story (US1–US3 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US3)

---

## Phase 1: Setup (Shared Infrastructure)

- [X] T001 [P] Create `backend/internal/settings/model.go` (`Theme` with `ThemeLight`, `ThemeDark`, `ThemeSystem`; `Settings{Theme, DailyReviewLimit, Timezone}`; `Patch` with pointer fields; `DefaultTheme = system`, `DefaultReviewLimit = 30`, `MinReviewLimit = 5`, `MaxReviewLimit = 200`; `ErrNotFound`; `ValidationError{Fields}`) and `backend/internal/settings/repository.go` (`Repository` with `Get`, `Update(patch)`) per data-model.md
- [X] T002 [P] Create `frontend/src/app/core/models/settings.ts` (`Settings`, `SettingsPatch`, reusing `ThemePreference` from `core/services/theme.service.ts`) per data-model.md "Frontend"

---

## Phase 2: Foundational (Blocking Prerequisites)

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Write `backend/internal/settings/validate_test.go`: theme light/dark/system ok, "blue" and "" rejected with "Chế độ giao diện không hợp lệ"; limit 5 and 200 ok, 4, 201, 0, -1 rejected with "Số thẻ từ 5 đến 200"; timezone Asia/Ho_Chi_Minh, Europe/London, UTC ok, "", "Local", "Mars/Olympus", "../etc" rejected with "Múi giờ không hợp lệ"; several invalid fields reported together; nil fields skipped
- [X] T004 Implement `backend/internal/settings/validate.go` (`ValidatePatch(p Patch) error`) so T003 passes (research R4)
- [X] T005 Write `backend/internal/settings/fake_test.go` (in-memory `Repository` with per-user settings and legacy timezone) and `backend/internal/settings/service_test.go`: `Get` fills defaults for an account with no settings and uses the legacy/registration timezone; `Update` applies only present fields and returns full settings; invalid patch writes nothing; `{}` returns current settings; unknown user → `ErrNotFound`; `Location` returns the stored zone (default Asia/Ho_Chi_Minh when unloadable); `ReviewLimit` returns the stored limit or 30; users isolated
- [X] T006 Implement `backend/internal/settings/service.go` (`NewService(repo)`, `Get`, `Update`, `Location`, `ReviewLimit`) so T005 passes
- [X] T007 Write `backend/internal/settings/handler_test.go`: `GET /api/settings` 200 shape; `PUT /api/settings` partial body 200 with full settings, 400 `validation_error` with field messages, 400 unknown field / bad JSON / wrong type (`"dailyReviewLimit":"10"`, `10.5`), `{}` 200; 401 without session; a user only changes their own settings
- [X] T008 Implement `backend/internal/settings/handler.go` (`NewHandler(svc, log)`, `Register(mux, requireAuth)`, JSON per contracts/settings-api.md, errors via `httpx`) so T007 passes
- [X] T009 Store settings in Mongo: add `Settings settingsDoc` (`theme`, `dailyReviewLimit`, `timezone`, omitempty) to `userDoc` in `backend/internal/storage/mongo/users.go`, make `toUser` read `settings.timezone` with fallback to the legacy `timezone`, make `Create` write `settings.timezone` (no top-level `timezone`); create `backend/internal/storage/mongo/settings.go` (`Settings` repository on `users`: `Get` with legacy fallback, `Update` with `$set` of `settings.<field>` per non-nil patch field and `ReturnDocument(After)`, unknown/invalid id → `settings.ErrNotFound`)
- [X] T010 Add migration `users-settings-timezone` in `backend/internal/storage/mongo/migrate.go` (same marker pattern as `MigrateTopics`: skip when marked; `updateMany` with pipeline `$set settings.timezone = $timezone`, `$unset timezone` for users that have `timezone` and no `settings.timezone`; mark done) and call it from `PrepareInBackground` before indexes
- [X] T011 Wire in `backend/cmd/api/main.go`: `settingsSvc := settings.NewService(mongo.NewSettings(db))`, register `settings.NewHandler(...)`; `userTimezones` uses `settingsSvc.Location`; `StudyDeps.ReviewLimit` uses `settingsSvc.ReviewLimit` (on error log and return 30); `go build ./...` and existing tests pass
- [X] T012 [P] Create `frontend/src/app/core/services/settings-api.service.ts` (`ng g s core/services/settings-api --type=service`) with `get()` (`GET /api/settings`) and `update(patch)` (`PUT /api/settings`) and spec cases in `settings-api.service.spec.ts`
- [X] T013 Add route `settings` (`authGuard`, title "Cài đặt · Luna", lazy `features/settings/settings`) in `frontend/src/app/app.routes.ts` and a "Cài đặt" link to `frontend/src/app/shared/components/app-header/app-header.html` with an assertion in `app-header.spec.ts`; create the page shell `frontend/src/app/features/settings/settings.ts` (`ng g c features/settings`, flatten to `features/settings/`) that loads `get()` with a load error + "Thử lại" state and headings Giao diện, Học tập, Múi giờ

**Checkpoint**: settings stored, validated and served; page shell reachable from the header

---

## Phase 3: User Story 1 - Chọn giao diện Sáng, Tối hoặc Theo hệ thống (Priority: P1) 🎯 MVP

**Goal**: Đổi giao diện có hiệu lực ngay, lưu theo tài khoản, dùng lại trên thiết bị khác

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T014 [P] [US1] Extend `frontend/src/app/core/services/theme.service.spec.ts`: logged out → `set()` writes localStorage and sends no request; when `AuthService.currentUser` becomes a user → `GET /api/settings`, theme applied to `<html data-theme>` and written to localStorage without a PUT; logged in `set('dark')` → applied immediately, localStorage written, `PUT /api/settings {"theme":"dark"}`; a failed PUT keeps the new theme; logging out keeps localStorage; `system` follows `prefers-color-scheme` changes (existing cases stay green)
- [X] T015 [P] [US1] Write the Giao diện cases in `frontend/src/app/features/settings/settings.spec.ts`: radiogroup "Giao diện" with Sáng / Tối / Theo hệ thống, current preference checked, choosing one calls `ThemeService.set` and shows "Đã lưu" after the PUT, a failed save shows an error; the group still works when loading settings failed; keyboard arrow selection

### Implementation for User Story 1

- [X] T016 [US1] Update `frontend/src/app/core/services/theme.service.ts`: inject `AuthService` and `SettingsApiService`; effect on the current user id → `get()` and apply the account theme (write localStorage, no PUT); `set(pref)` applies, writes localStorage and, when logged in, returns/starts `update({theme})`; expose a `saveState` signal (`idle` | `saving` | `saved` | `error`) for the page; keep the inline script in `frontend/src/index.html` unchanged; update the doc comment (F12) so T014 passes (research R8)
- [X] T017 [US1] Implement the Giao diện group in `frontend/src/app/features/settings/settings.html` / `settings.ts` / `settings.css` (radio inputs with labels, ≥ 44px targets, status line `role="status"`) so T015 passes; header quick switch (`app-header`) goes through the same `ThemeService.set`

**Checkpoint**: MVP — đổi giao diện ngay, đăng nhập thiết bị khác thấy đúng

---

## Phase 4: User Story 2 - Đặt số thẻ ôn tối đa mỗi ngày (Priority: P2)

**Goal**: Giới hạn 5–200 áp dụng cho bước Ôn từ lần mở Bài hôm nay tiếp theo

**Independent Test**: quickstart.md bước 2

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T018 [P] [US2] Extend `backend/internal/progress/study_test.go`: with `ReviewLimit` returning 10 and 25 due cards, `Today` gives `reviewCount` 10; after 20 daily reviews, limit 10 → review step has 0 cards (auto-completes when opened), limit 50 after 30 reviews → 20 (research R7)
- [X] T019 [P] [US2] Write the Học tập cases in `frontend/src/app/features/settings/settings.spec.ts`: number input "Số thẻ ôn mỗi ngày" (min 5, max 200) shows the stored value; **Lưu** sends `PUT {dailyReviewLimit, timezone}`; server 400 shows "Số thẻ từ 5 đến 200" under the input with `aria-invalid` and `aria-describedby`; empty or out-of-range input shows the same message without a request; success shows "Đã lưu"

### Implementation for User Story 2

- [X] T020 [US2] Make T018 pass in `backend/internal/progress/study.go` if needed (the limit is read on every `Today` call; no caching)
- [X] T021 [US2] Implement the Học tập part of the settings form in `frontend/src/app/features/settings/settings.html` / `settings.ts` (form with the limit input, client check 5–200 integer, **Lưu**, server field errors, "Đã lưu") so T019 passes

**Checkpoint**: giới hạn thẻ theo tài khoản

---

## Phase 5: User Story 3 - Đổi múi giờ tính ngày học (Priority: P2)

**Goal**: Chọn múi giờ từ danh sách; đổi giữa ngày không mất tiến độ, không sinh thêm bài

**Independent Test**: quickstart.md bước 3

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T022 [P] [US3] Extend `backend/internal/progress/rules_test.go`: `EffectiveDayKey` returns the later of the two keys, the local key when latest is "", across month/year
- [X] T023 [P] [US3] Extend `backend/internal/progress/study_flow_test.go` (timezone changes): learner in Asia/Ho_Chi_Minh finishes review + read position 2, switches to Europe/London the same day → same lesson, review done, current read at sentence 2; switches to a zone already on the next day → the unfinished lesson is still today's lesson with its steps done; after finishing today's lesson at 00:30 Viet Nam time, switching to America/Los_Angeles (previous day) → `doneToday`, no new lesson, streak unchanged; `LatestKey` fake returns the max day key
- [X] T024 [P] [US3] Write the Múi giờ cases in `frontend/src/app/features/settings/settings.spec.ts`: select "Múi giờ" lists zones from `Intl.supportedValuesOf` with offsets ("Europe/London (GMT+1)"), stored zone selected; typing "london" in the search box filters case-insensitively and keeps the selected zone; saving sends the chosen zone; server 400 shows "Múi giờ không hợp lệ"; fallback list when `Intl.supportedValuesOf` is missing

### Implementation for User Story 3

- [X] T025 [US3] Add `EffectiveDayKey(localKey, latestKey string) string` to `backend/internal/progress/rules.go` so T022 passes
- [X] T026 [US3] Add `LatestKey(ctx, userID) (string, error)` to `DayRepository` in `backend/internal/progress/study_repository.go`, the fake in `backend/internal/progress/study_fake_test.go` and `StudyDays` in `backend/internal/storage/mongo/study.go` (find one sorted by `dayKey` −1, projection `dayKey`, none → ""); use `EffectiveDayKey` in `StudyService.load` in `backend/internal/progress/study.go` so T023 passes (research R6)
- [X] T027 [US3] Implement the Múi giờ part in `frontend/src/app/features/settings/settings.html` / `settings.ts` / `settings.css` (search input + `<select size>` with labels, offsets via `Intl.DateTimeFormat` `shortOffset`, saved together with the limit) so T024 passes (research R9)

**Checkpoint**: tất cả user story xong

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T028 Update `README.md`: section "Cài đặt" (giao diện theo tài khoản, số thẻ 5–200, múi giờ, đổi múi giờ không mất tiến độ) and mention `settings` in the data notes if any
- [X] T029 Run lint (frontend `npm run lint` + backend golangci-lint via Docker) with zero errors
- [X] T030 Run all tests (`go test -race ./...`, `npx ng test --watch=false`) with zero failures; `npx ng build` without warnings
- [X] T031 Rebuild and smoke test in Docker with `curl` and UTF-8 body files (quickstart.md bước 4–5): GET defaults, PUT partial, 400 limit/timezone/theme, 401, `/api/auth/me` timezone from settings, `/api/today` `reviewCount` follows the limit, migration moved every `timezone` into `settings`
- [X] T032 Compute WCAG contrast for new pairs (selected radio, input borders, error text, "Đã lưu") in both modes
- [ ] T033 Manual check (quickstart.md bước 1–3): theme instant in both modes, another browser after login, 360px layout, keyboard only, design-system colours and fonts
- [ ] T034 Tick the 3 F12 acceptance criteria in `docs/phases/giai-doan-1.md` once T017, T031, T033 pass

---

## Dependencies & Execution Order

- **Setup** → **Foundational** (validation → service → handler → storage → migration → wiring; API client and page shell) → **US1**
  → **US2** → **US3** → **Polish**
- US2 and US3 share the settings form; do US2 then US3 in `features/settings`. Backend parts of US2/US3 are independent of US1
- US3 backend (`EffectiveDayKey`) is independent of the settings package and can start after Phase 1

### Parallel Opportunities

- T001 ∥ T002; T012 ∥ T003–T011 (frontend vs backend)
- T014 ∥ T015; T018 ∥ T019; T022 ∥ T023 ∥ T024
- T025/T026 (backend) ∥ T021/T027 (frontend)

---

## Parallel Example: User Story 3

```bash
Task: "EffectiveDayKey tests in backend/internal/progress/rules_test.go"
Task: "Timezone change flow tests in backend/internal/progress/study_flow_test.go"
Task: "Múi giờ cases in frontend/src/app/features/settings/settings.spec.ts"
```

---

## Implementation Strategy

1. Setup + Foundational → US1 (giao diện theo tài khoản) → **STOP and VALIDATE** với quickstart bước 1
2. US2 → giới hạn thẻ
3. US3 → múi giờ, quy tắc ngày học không lùi
4. Polish → README, lint, test, Docker, tương phản, kiểm tra thủ công, tick tiêu chí nghiệm thu

---

## Notes

- Angular: `ng generate` (component không hậu tố; service `--type=service`); Go: giữ `go 1.25.1` (`GOTOOLCHAIN=local`)
- `progress` và `vocab` không import `settings`; chỉ `main.go` nối qua `Timezones` và `ReviewLimit`
- `auth.User.Timezone` giữ nguyên (đọc từ `settings.timezone`) để API F1 không đổi
- Gửi dữ liệu tiếng Việt khi test bằng curl trên Windows: dùng `--data-binary @file.json`
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai

- **Đổi hành vi L (có chủ đích):** trước F12, đổi sang múi giờ còn "hôm qua" sau khi xong bài hôm nay cho thêm một bài mới
  (`TestTimezoneChangeKeepsOneLessonPerDay` cũ mong `f2`). Theo spec F12 ("không tạo thêm bài mới trong cùng một ngày thực"), ngày
  học giờ là `EffectiveDayKey(ngày theo múi giờ, ngày học gần nhất)` nên không bao giờ lùi; test cũ đã sửa thành `doneToday` với
  `f1`.
- `settings.Service` có `Location(ctx, userID)` đúng chữ ký port `Timezones` của `vocab`/`progress`, nên `main.go` truyền thẳng
  `settingsSvc` và bỏ adapter `userTimezones`; `ReviewLimit` bọc `settingsSvc.ReviewLimit` (lỗi → log + 30).
- Lỗi kiểm tra trả `{"error":"validation_failed", "message", "fields"}` như mọi API khác (`httpx.WriteFieldErrors`); contract ghi
  `validation_error`. Người dùng của phiên không còn tồn tại → 401 `unauthenticated`.
- `users.timezone` (F1) chuyển sang `users.settings.timezone` bằng migration `f12-users-settings-timezone` trong
  `storage/mongo/migrate_settings.go` (chạy trong `PrepareInBackground` sau migration F14); repository đọc dự phòng trường cũ.
  `Register` ghi `settings.timezone`; `auth.User.Timezone` và `/api/auth/me` giữ nguyên.
- `ThemeService` inject `AuthService` + `SettingsApiService`: khi `currentUser` có id thì `GET /api/settings` và áp theme của tài
  khoản (không PUT lại); `set()` áp ngay, ghi localStorage, và khi đã đăng nhập thì `PUT {theme}`, trạng thái ở `saveState`.
- Danh sách múi giờ: trình duyệt (và Node) liệt kê tên chuẩn ICU, ví dụ `Asia/Saigon` chứ không có `Asia/Ho_Chi_Minh`; múi giờ đang
  lưu luôn được thêm vào danh sách kèm giờ lệch nên vẫn chọn đúng. Tìm "ho chi minh" chỉ thấy mục đang lưu; server chấp nhận cả hai
  tên.
- Nhóm Giao diện dùng radio gốc (phím mũi tên của trình duyệt), lưu ngay; Học tập + Múi giờ chung một form **Lưu**.
- Smoke test Docker (T031): migration chuyển `timezone` của 2 tài khoản vào `settings` và ghi marker; `GET /api/settings` mặc định
  `system/30/Asia/Ho_Chi_Minh`; PUT từng phần 200; PUT sai cả 3 trường → 400 với 3 thông báo; `/api/auth/me` trả múi giờ mới; 401
  khi thiếu phiên; đã khôi phục cài đặt của `hoc@example.com` về 30 / Asia/Ho_Chi_Minh.
- Tương phản (T032): lựa chọn giao diện đang chọn primary-ink/primary-soft 7.96 / 7.86, viền primary 7.37 / 6.22 (như thanh bước
  L); chữ lỗi bad/surface 5.17 / 6.08; trạng thái "Đã lưu" muted/surface 6.33 / 6.60.
- Còn lại cho người dùng: T033 kiểm tra thủ công (giao diện ở hai chế độ, trình duyệt khác sau đăng nhập, 360px, bàn phím), rồi T034
  tick tiêu chí F12.
