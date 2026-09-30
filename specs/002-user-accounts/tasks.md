---

description: "Task list for F1 – Tài khoản"
---

# Tasks: Tài khoản (F1)

**Input**: Design documents from `specs/002-user-accounts/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: unit tests for business logic, endpoint tests
(success, invalid input, forbidden), no network (Mongo replaced by in-memory fake repositories). Every
acceptance criterion maps to a test task or a manual check task (quickstart.md step numbers).

**Organization**: Tasks are grouped by user story (US1–US4 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Paths: `backend/`, `frontend/`, `deploy/` at repo root (see plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Phụ thuộc và cấu hình mới

- [X] T001 Add `golang.org/x/crypto` (argon2) to `backend/go.mod` via `go get golang.org/x/crypto@latest`
- [X] T002 Extend `backend/internal/platform/config/config.go` and `config_test.go`: `MongoDatabase` (`MONGO_DATABASE`, default `luna`), `CookieSecure bool` (`COOKIE_SECURE`, `true|false` case-insensitive, default `false`, other value → error `config: COOKIE_SECURE must be true or false`) (contracts/config.md)
- [X] T003 [P] Add `MONGO_DATABASE: ${MONGO_DATABASE:-luna}` and `COOKIE_SECURE: ${COOKIE_SECURE:-false}` to backend service in `deploy/docker-compose.yml` and both variables with comments to `deploy/.env.example`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Kiểu dữ liệu, băm mật khẩu, kiểm tra đầu vào, phiên, middleware quyền, repository Mongo,
AuthService/guard/interceptor ở frontend

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Backend

- [X] T004 Create `backend/internal/auth/model.go`: `Role` (`RoleAdmin="admin"`, `RoleLearner="learner"`), `User`, `Session{TokenHash, UserID string; ExpiresAt, CreatedAt time.Time}`, sentinel errors `ErrNotFound`, `ErrEmailTaken`, `ErrInvalidCredentials`, `ErrUnauthenticated`, type `LockedError{RetryAfter time.Duration}`, type `ValidationError{Fields map[string]string}` with Vietnamese messages (data-model.md)
- [X] T005 Create `backend/internal/auth/repository.go` with `UserRepository` and `SessionRepository` interfaces exactly as data-model.md §4
- [X] T006 [P] Write `backend/internal/auth/password_test.go`: hash starts with `$argon2id$v=19$m=19456,t=2,p=1$`; same password hashes differ (salt); verify correct → true, wrong → false; 128-char Vietnamese password with spaces verifies (no truncation); malformed/other-algorithm hash → error
- [X] T007 Implement `HashPassword(pw string) (string, error)` and `VerifyPassword(pw, encoded string) (bool, error)` with argon2id IDKey, 16-byte salt, 32-byte key, PHC encoding, `subtle.ConstantTimeCompare` in `backend/internal/auth/password.go` (research R1) so T006 passes
- [X] T008 [P] Write `backend/internal/auth/validate_test.go` (table-driven): `NormalizeEmail` trims + lowercases; `ValidateCredentials` rejects empty, `a@b`, `Tên <a@b.com>`, >254 chars, password 7 runes, 129 runes, accepts 8 and 128 runes (Vietnamese chars count as 1); `NormalizeTimezone` keeps `Asia/Ho_Chi_Minh`, `Europe/Paris`, maps `""`, `Local`, `Mars/Base` to `Asia/Ho_Chi_Minh`
- [X] T009 Implement `NormalizeEmail`, `ValidateCredentials(email, pw) error` (returns `*ValidationError` with fields `email`, `password` and Vietnamese messages), `NormalizeTimezone` in `backend/internal/auth/validate.go` (research R7) so T008 passes
- [X] T010 Create in-memory fakes `fakeUsers` (unique email, `Count`) and `fakeSessions` guarded by mutex in `backend/internal/auth/fake_test.go` (package `auth`) for service and handler tests
- [X] T011 [P] Add `WriteError(w, status, code, message string)` (and variant with `fields`) in `backend/internal/platform/httpx/errors.go`; add `DecodeJSON(w, r, dst) error` in `backend/internal/platform/httpx/json.go` enforcing `Content-Type: application/json` (→ 415 `unsupported_media_type`), `MaxBytesReader` 16 KiB, `DisallowUnknownFields` (→ 400 `invalid_body`); tests in `backend/internal/platform/httpx/json_test.go`
- [X] T012 [P] Write `backend/internal/platform/httpx/auth_test.go`: `RequireAuth` with fake resolver → no cookie 401 `unauthenticated`, resolver error 401, valid → next sees `PrincipalFrom(ctx)`; `RequireAdmin` → learner 403 `forbidden`, admin passes, missing principal 401
- [X] T013 Implement `SessionCookieName = "luna_session"`, `Principal{UserID, Email, Role string}`, `RequireAuth(resolve func(ctx context.Context, token string) (Principal, error)) Middleware`, `RequireAdmin(next http.Handler) http.Handler`, `PrincipalFrom(ctx)` in `backend/internal/platform/httpx/auth.go` (research R8) so T012 passes
- [X] T014 Write `backend/internal/auth/service_test.go` foundation cases using fakes and a fixed clock: `Authenticate` valid token → user; unknown token → `ErrUnauthenticated`; expired session (clock past `ExpiresAt`) → `ErrUnauthenticated`; stored hash is SHA-256 of token, never the token itself
- [X] T015 Implement `Service` in `backend/internal/auth/service.go`: `NewService(users UserRepository, sessions SessionRepository, now func() time.Time)`, private `startSession(ctx, userID) (token string, err error)` (32 random bytes, base64url raw token, SHA-256 hex stored, 30-day `ExpiresAt`), `Authenticate(ctx, token) (User, error)` checking `ExpiresAt > now` (research R3) so T014 passes
- [X] T016 Implement `Handler` in `backend/internal/auth/handler.go`: `NewHandler(svc *Service, cookieSecure bool, log *slog.Logger)`, cookie helpers `setSession`/`clearSession` (HttpOnly, SameSite=Strict, Path=/, Max-Age 2592000, Secure per config), `userJSON` (id, email, role, timezone only), `Me` handler reading `httpx.PrincipalFrom` and returning `{"user":…}`; `ResolvePrincipal` adapter from `Service.Authenticate` to `httpx.Principal`; test `Me` 200 and 401 in `backend/internal/auth/handler_test.go` (contracts/auth-api.md)
- [X] T017 [P] Implement MongoDB repositories: add `Database(name string)` to `backend/internal/storage/mongo/client.go`; `backend/internal/storage/mongo/users.go` (`auth.UserRepository`, ObjectID ↔ hex, duplicate key → `auth.ErrEmailTaken`, no documents → `auth.ErrNotFound`); `backend/internal/storage/mongo/sessions.go` (`auth.SessionRepository`, `Delete` ignores missing); `backend/internal/storage/mongo/indexes.go` `EnsureIndexes(ctx)` for `users.email` unique, `sessions.tokenHash` unique, `sessions.expiresAt` TTL 0 (data-model.md)
- [X] T018 Wire in `backend/cmd/api/main.go`: `import _ "time/tzdata"`; build repositories on `cfg.MongoDatabase`, `auth.NewService`, `auth.NewHandler`; `requireAuth := httpx.RequireAuth(authHandler.ResolvePrincipal)`; route `GET /api/auth/me` behind `requireAuth`; start goroutine retrying `EnsureIndexes` every 10 s until success or ctx done (research R12)

### Frontend

- [X] T019 [P] Create `User` and `Role` types in `frontend/src/app/core/models/user.ts` (data-model.md §5)
- [X] T020 [P] Write `frontend/src/app/core/services/auth.service.spec.ts`: `load()` sets `currentUser` from `/api/auth/me` 200; 401 → `null`; network error → `null`; `isAdmin` computed; `clear()` sets `null`
- [X] T021 Implement `AuthService` (`ng g s core/services/auth --type=service`) in `frontend/src/app/core/services/auth.service.ts`: signals `currentUser: User | null`, `isAdmin`, `isLoggedIn`; `load(): Promise<void>` never rejects; `clear()` (research R10) so T020 passes
- [X] T022 Register `provideAppInitializer(() => inject(AuthService).load())` in `frontend/src/app/app.config.ts`
- [X] T023 [P] Write `frontend/src/app/shared/utils/safe-return-url.spec.ts` and implement `safeReturnUrl(url: unknown): string` in `frontend/src/app/shared/utils/safe-return-url.ts`: accepts `/admin`, `/?a=1`; returns `/` for `null`, `''`, `https://evil.example`, `//evil.example`, `/\\evil`, `javascript:alert(1)`, `/login`, `/register?x=1` (research R10)
- [X] T024 [P] Write `frontend/src/app/core/guards/auth.guard.spec.ts`: `authGuard` returns `/login?returnUrl=<url>` UrlTree when logged out, `true` when logged in; `guestGuard` returns `/` UrlTree when logged in; `adminGuard` returns `/forbidden` for learner, `true` for admin
- [X] T025 Implement `authGuard`, `guestGuard`, `adminGuard` (`CanActivateFn`) in `frontend/src/app/core/guards/auth.guard.ts` so T024 passes
- [X] T026 Extend `frontend/src/app/core/interceptors/error-interceptor.spec.ts`: 401 from `/api/admin/ping` → `AuthService.clear()` called and router navigates to `/login?returnUrl=<current url>`; 401 from `/api/auth/login`, `/api/auth/register`, `/api/auth/me` → no navigation; 403 → navigates `/forbidden`; all still throw `ApiError`
- [X] T027 Update `frontend/src/app/core/interceptors/error-interceptor.ts` to handle 401/403 per research R10 (skip auth endpoints; do not redirect when already on `/login`) so T026 passes

**Checkpoint**: `go test ./...` and `npm test -- --watch=false` pass; backend serves `/api/auth/me`

---

## Phase 3: User Story 1 - Đăng ký tài khoản (Priority: P1) 🎯 MVP

**Goal**: Đăng ký bằng email + mật khẩu, tài khoản đầu là admin, lưu múi giờ, đăng nhập luôn

**Independent Test**: quickstart.md bước 1

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T028 [P] [US1] Add `Register` cases to `backend/internal/auth/service_test.go`: first user → `admin`, second → `learner`; email normalized; duplicate (different case) → `ErrEmailTaken`; invalid input → `*ValidationError`; invalid timezone stored as `Asia/Ho_Chi_Minh`; password stored as argon2id hash (not plaintext); returns a token that `Authenticate` accepts
- [X] T029 [P] [US1] Add register cases to `backend/internal/auth/handler_test.go`: 201 `{"user":…}` without `passwordHash` + `Set-Cookie luna_session` HttpOnly SameSite=Strict Max-Age=2592000; 400 `validation_failed` with `fields.email` / `fields.password`; 409 `email_taken`; 415 for `text/plain`; 400 for unknown field
- [X] T030 [P] [US1] Write `frontend/src/app/features/auth/register/register.spec.ts`: shows Vietnamese errors for empty/invalid email and password < 8 after touch; submit disabled while pending; posts `{email, password, timezone}` with `Intl` timezone; 201 → `currentUser` set and navigates `/`; 409 → "Email này đã được dùng" in `role="alert"`; has link to `/login`

### Implementation for User Story 1

- [X] T031 [US1] Implement `Service.Register(ctx, email, password, timezone string) (User, string, error)` in `backend/internal/auth/service.go` (validate, normalize, `Count()==0` → admin, hash, create, `startSession`) (research R5) so T028 passes
- [X] T032 [US1] Implement `Handler.Register` in `backend/internal/auth/handler.go` (DecodeJSON, map `ValidationError` → 400 with fields, `ErrEmailTaken` → 409, set cookie, 201) and route `POST /api/auth/register` in `backend/cmd/api/main.go` so T029 passes
- [X] T033 [US1] Add `register(email, password)` to `frontend/src/app/core/services/auth.service.ts` (sends `Intl.DateTimeFormat().resolvedOptions().timeZone`, sets `currentUser`)
- [X] T034 [US1] Generate `ng g c features/auth/register` and implement reactive form in `frontend/src/app/features/auth/register/register.{ts,html,css}`: labels, `aria-invalid`, `aria-describedby` error text, server error `role="alert"`, keep email on error, submit button disabled while pending, link "Đã có tài khoản? Đăng nhập", single column fits 360px, tokens only (research R10) so T030 passes
- [X] T035 [US1] Add route `register` (lazy, `canActivate: [guestGuard]`) to `frontend/src/app/app.routes.ts` (home stays public until US2 adds the login page)
- [X] T036 [US1] Manual check: quickstart.md bước 1 (first account admin, second learner, duplicate email, short password, timezone and `$argon2id$` in DB)

**Checkpoint**: Đăng ký hoạt động độc lập; tài khoản đầu là admin

---

## Phase 4: User Story 2 - Đăng nhập, giữ đăng nhập và đăng xuất (Priority: P1)

**Goal**: Đăng nhập, phiên 30 ngày, đăng xuất, bảo vệ mọi trang, quay lại trang cũ khi hết phiên

**Independent Test**: quickstart.md bước 2 và 3

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T037 [P] [US2] Add `Login`/`Logout` cases to `backend/internal/auth/service_test.go`: correct (case-insensitive email) → user + token; wrong password and unknown email both → `ErrInvalidCredentials`; unknown email still calls password verify (dummy hash); `Logout` deletes session so `Authenticate` fails; `Logout` with unknown token → no error
- [X] T038 [P] [US2] Add login/logout cases to `backend/internal/auth/handler_test.go`: login 200 + cookie; 401 bodies for unknown email and wrong password are byte-identical; 400 missing fields; logout 204 + cookie `Max-Age=0`; `/me` with old cookie after logout → 401; logout without cookie → 204
- [X] T039 [P] [US2] Write `frontend/src/app/features/auth/login/login.spec.ts`: validation messages; 401 → "Email hoặc mật khẩu không đúng" in `role="alert"`; success navigates to `safeReturnUrl(returnUrl)` (`/admin` kept, `https://evil.example` → `/`); link to `/register`; submit disabled while pending
- [X] T040 [P] [US2] Extend `frontend/src/app/shared/components/app-header/app-header.spec.ts`: logged out → no email/logout; logged in → shows email and "Đăng xuất"; clicking it calls `AuthService.logout()` and navigates `/login`

### Implementation for User Story 2

- [X] T041 [US2] Implement `Service.Login(ctx, email, password) (User, string, error)` with dummy-hash timing equalization (dummy hash computed once in `NewService`) and `Service.Logout(ctx, token) error` in `backend/internal/auth/service.go` (research R2) so T037 passes
- [X] T042 [US2] Implement `Handler.Login` and `Handler.Logout` in `backend/internal/auth/handler.go`; routes `POST /api/auth/login`, `POST /api/auth/logout` in `backend/cmd/api/main.go` so T038 passes
- [X] T043 [US2] Add `login(email, password)` and `logout()` (always clears user even if API fails) to `frontend/src/app/core/services/auth.service.ts`
- [X] T044 [US2] Generate `ng g c features/auth/login` and implement form in `frontend/src/app/features/auth/login/login.{ts,html,css}` reading `returnUrl` query param, sharing styles/patterns with register, link "Chưa có tài khoản? Đăng ký" so T039 passes
- [X] T045 [US2] Update `frontend/src/app/app.routes.ts`: `login` (lazy, `guestGuard`), `''` home with `canActivate: [authGuard]`, `**` → `''`
- [X] T046 [US2] Update `frontend/src/app/shared/components/app-header/app-header.{ts,html,css}`: when logged in, second row (wraps below brand + theme at < 640px, same row ≥ 640px) with email (ellipsis + `title`) and "Đăng xuất" button (44px target) (research R11) so T040 passes
- [ ] T047 [US2] Manual check: quickstart.md bước 2 (persist after browser restart, cookie flags, logout invalidates other tab) and bước 3 (expired session → login → back to `/admin` after US4, or `/` now; unsafe returnUrl → `/`)

**Checkpoint**: US1 + US2 = luồng tài khoản tối thiểu hoàn chỉnh

---

## Phase 5: User Story 3 - Chống đoán mật khẩu (Priority: P2)

**Goal**: Lỗi chung + tạm khoá 15 phút sau 5 lần sai liên tiếp

**Independent Test**: quickstart.md bước 4

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T048 [P] [US3] Write `backend/internal/auth/lockout_test.go` with controllable clock: 4 fails → not locked; 5th fail → locked with 15 min remaining; locked → `Check` returns remaining; after 15 min → unlocked and counter reset; `Success` clears counter; concurrent `Fail` calls safe (`-race`)
- [X] T049 [P] [US3] Add lockout cases to `backend/internal/auth/service_test.go`: 5th wrong password → `*LockedError`; correct password while locked → `*LockedError` without verifying; after 15 min correct → success; 3 fails then success resets; unknown email also locks
- [X] T050 [P] [US3] Add to `backend/internal/auth/handler_test.go`: 5th failed login → 429 `account_locked`, `Retry-After: 900`, `retryAfterSeconds: 900`, message mentions minutes
- [X] T051 [P] [US3] Extend `frontend/src/app/features/auth/login/login.spec.ts`: 429 with `retryAfterSeconds: 900` → "Đăng nhập tạm khoá. Thử lại sau 15 phút." in `role="alert"`

### Implementation for User Story 3

- [X] T052 [US3] Implement `Lockout` (`NewLockout(now func() time.Time)`, `Check`, `Fail`, `Success`, 5 attempts / 15 min constants, `sync.Mutex`) in `backend/internal/auth/lockout.go` (research R6) so T048 passes
- [X] T053 [US3] Inject `*Lockout` into `NewService` and apply it in `Service.Login` in `backend/internal/auth/service.go`; update `backend/cmd/api/main.go` wiring so T049 passes
- [X] T054 [US3] Map `*LockedError` → 429 with `Retry-After` header and `retryAfterSeconds` in `backend/internal/auth/handler.go` so T050 passes
- [X] T055 [US3] Show lock message with minutes rounded up in `frontend/src/app/features/auth/login/login.ts` so T051 passes
- [X] T056 [US3] Manual check: quickstart.md bước 4 (identical 401 bodies, 429 from 5th attempt, lock message in UI)

**Checkpoint**: Tiêu chí "lỗi chung" và "khoá 5 lần/15 phút" đạt

---

## Phase 6: User Story 4 - Chỉ quản trị viên vào trang quản trị (Priority: P2)

**Goal**: Trang quản trị tạm + endpoint admin; người học bị chặn ở giao diện và API

**Independent Test**: quickstart.md bước 5

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- [X] T057 [P] [US4] Write `backend/internal/admin/handler_test.go`: route `GET /api/admin/ping` wrapped with `httpx.RequireAuth` (fake resolver) + `httpx.RequireAdmin` → admin 200 `{"ok":true}`, learner 403 `forbidden`, anonymous 401 `unauthenticated`
- [X] T058 [P] [US4] Write `frontend/src/app/features/admin/admin.spec.ts` (calls `/api/admin/ping`, shows "Trang quản trị (đang xây dựng)") and `frontend/src/app/features/forbidden/forbidden.spec.ts` (shows "Bạn không có quyền truy cập trang này" and link to `/`)
- [X] T059 [P] [US4] Extend `frontend/src/app/shared/components/app-header/app-header.spec.ts`: "Quản trị" link to `/admin` visible for admin only

### Implementation for User Story 4

- [X] T060 [US4] Implement `backend/internal/admin/handler.go` (`Ping` → 200 `{"ok":true}`) and register `GET /api/admin/ping` behind `requireAuth` + `httpx.RequireAdmin` in `backend/cmd/api/main.go` (research R13) so T057 passes
- [X] T061 [US4] Generate `ng g c features/admin` and `ng g c features/forbidden`; implement pages in `frontend/src/app/features/admin/` and `frontend/src/app/features/forbidden/` (tokens only, 360px) so T058 passes
- [X] T062 [US4] Add routes `admin` (lazy, `canActivate: [authGuard, adminGuard]`) and `forbidden` (lazy, `authGuard`) to `frontend/src/app/app.routes.ts`
- [X] T063 [US4] Add "Quản trị" link (admin only, `routerLink="/admin"`) to `frontend/src/app/shared/components/app-header/app-header.html` so T059 passes
- [X] T064 [US4] Manual check: quickstart.md bước 5 (learner blocked at `/admin` in UI, `curl` 403/401) and redo bước 3 with `/admin` as return URL

**Checkpoint**: Mọi user story hoạt động độc lập

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T065 Update `README.md`: tài khoản đầu tiên là quản trị viên, cách tạo lại từ đầu (`down -v`), biến `MONGO_DATABASE`, `COOKIE_SECURE` trong bảng cấu hình, cách xem users trong mongosh
- [X] T066 Run lint (frontend `npm run lint` + backend golangci-lint via Docker per README) with zero errors
- [X] T067 Run all tests (`go test -race ./...` in `backend/`, `npm test -- --watch=false` in `frontend/`) with zero failures
- [X] T068 Rebuild and smoke test in Docker: `docker compose -f deploy/docker-compose.yml up --build -d`, run quickstart.md bước 4 curl script and bước 6 (no password/token in logs, `/me` has no `passwordHash`)
- [ ] T069 Manual check at 360px width in both light and dark mode (quickstart.md bước 7): `/login`, `/register`, `/`, `/admin`, `/forbidden`, header two rows, no horizontal scroll, form errors readable
- [ ] T070 Tick the 6 F1 acceptance criteria in `docs/phases/giai-doan-1.md` once T036, T047, T056, T064, T068, T069 pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 → T002; T003 independent
- **Foundational (Phase 2)**: after Setup. Backend: T004 → T005 → (T006–T011 parallel) → T012 → T013 → T014 → T015 → T016; T017 after T005; T018 after T013, T016, T017. Frontend: T019 → T020 → T021 → T022; T023, T024 → T025, T026 → T027 (T025/T027 need T021)
- **US1 (Phase 3)**: after Foundational
- **US2 (Phase 4)**: after US1 (login page and home guard rely on registered accounts and `AuthService.register` patterns)
- **US3 (Phase 5)**: after US2 (extends `Service.Login` and login page)
- **US4 (Phase 6)**: after Foundational + US2 (needs login to test; independent of US3)
- **Polish (Phase 7)**: after all stories

### Within Each User Story

- Tests written first and FAIL before implementation
- Backend: service → handler → route in `main.go`
- Frontend: service method → component → route
- Manual check task last

### Parallel Opportunities

- Foundational: T006, T008, T011, T012, T017 (backend) ∥ T019, T020, T023, T024 (frontend)
- US1: T028 ∥ T029 ∥ T030; backend T031–T032 ∥ frontend T033–T035
- US2: T037 ∥ T038 ∥ T039 ∥ T040
- US3: T048–T051 all [P]
- US3 and US4 can proceed in parallel after US2

---

## Parallel Example: User Story 1

```bash
# Tests first, together:
Task: "Add Register cases to backend/internal/auth/service_test.go"
Task: "Add register cases to backend/internal/auth/handler_test.go"
Task: "Write frontend/src/app/features/auth/register/register.spec.ts"

# Then backend and frontend tracks side by side:
Task: "Service.Register + Handler.Register + route (backend)"
Task: "AuthService.register + register page + route (frontend)"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup → Phase 2 Foundational
2. Phase 3 US1 → **STOP and VALIDATE** with quickstart bước 1

### Incremental Delivery

1. US1 → đăng ký, tài khoản đầu là admin
2. US2 → đăng nhập/đăng xuất, bảo vệ trang (luồng tài khoản đầy đủ)
3. US3 → chống đoán mật khẩu
4. US4 → trang quản trị tạm, chặn người học
5. Polish → README, lint, test, Docker, 360px, tick tiêu chí nghiệm thu

---

## Notes

- [P] tasks = different files, no dependencies
- Tạo file Angular bằng `ng generate` (component không hậu tố; service dùng `--type=service`)
- Không log mật khẩu, token hay cookie; không trả `passwordHash` ra API
- Code, API, commit message bằng tiếng Anh; chữ trên giao diện bằng tiếng Việt
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai (2026-09-29)

**Khác với mô tả task** (không đổi hành vi):
- `golang.org/x/crypto` ghim ở **v0.55.0**: bản mới nhất (v0.57.0) yêu cầu Go 1.26 và tự nâng `go.mod` lên 1.26, trái ràng buộc Go 1.25 trong constitution. `x/sync`, `x/text` cũng giữ ở bản hỗ trợ Go 1.25.
- `import _ "time/tzdata"` nằm ở `backend/cmd/api/tzdata.go` (goimports không cho chèn chú thích giữa nhóm import).
- `NewService` nhận thêm `*Lockout` ngay từ đầu (thay vì thêm ở US3) và trả `error` vì phải tạo hash giả khi khởi động.
- `Handler.Me` đọc lại user qua `Service.UserByID` để trả cả `timezone` (Principal chỉ có id, email, role).
- Guard nằm ở `core/guards/auth.guard.ts`; lỗi form dùng chung ở `features/auth/auth-messages.ts`; style form dùng chung ở `src/styles/forms.css`.
- `.golangci.yml`: loại trừ `gosec` cho `_test.go` (mật khẩu và cookie giả trong test).

**Kết quả kiểm tra**
- Backend: `go test -race ./...` qua; golangci-lint 0 issues.
- Frontend: 85 test Vitest qua; `ng lint` sạch; `ng build` thành công.
- Docker (database trống, qua nginx `localhost:8000`):
  - Đăng ký: tài khoản đầu `admin`, thứ hai `learner`; múi giờ lưu đúng; `passwordHash` là `$argon2id$v=19$m=19456,t=2,p=1$…`; trùng email khác hoa thường → 409; mật khẩu 7 ký tự → 400 kèm lỗi tiếng Việt.
  - Cookie: `luna_session; Path=/; Max-Age=2592000; HttpOnly; SameSite=Strict`.
  - `/api/admin/ping`: admin 200, learner 403, ẩn danh 401. `/api/auth/me` chỉ trả id, email, role, timezone.
  - Đăng xuất → 204, cookie cũ gọi `/me` → 401.
  - Sai mật khẩu lần 1–4 → 401; lần 5, 6 → 429 `retryAfterSeconds: 900`; email lạ → 401 với body giống hệt; đúng mật khẩu khi đang khoá → 429.
  - Log backend không chứa mật khẩu, token hay hash.
  - Index: `users.email` unique, `sessions.tokenHash` unique, `sessions.expiresAt` TTL 0.
- Độ tương phản các cặp màu mới (nút chính, liên kết, viền ô nhập, lỗi) đạt AA ở cả hai chế độ.

**Còn lại cần kiểm tra bằng trình duyệt**: T047 (giữ đăng nhập sau khi đóng trình duyệt, đăng xuất ở tab khác, hết phiên → quay về `/admin`), T069 (360px sáng/tối), rồi T070.
