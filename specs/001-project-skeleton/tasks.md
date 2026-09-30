---

description: "Task list for F0 – Khung dự án Luna"
---

# Tasks: Khung dự án Luna (F0)

**Input**: Design documents from `specs/001-project-skeleton/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED by constitution principle VI: unit tests for business logic, endpoint tests
(success, invalid input, forbidden), no network. F0 has no user data, so "forbidden" does not apply.
Every acceptance criterion maps to a test task or a manual check task (quickstart.md step numbers).

**Organization**: Tasks are grouped by user story (US1–US4 from spec.md).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Paths: `frontend/`, `backend/`, `deploy/` at repo root (see plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Tạo hai project và công cụ lint

- [X] T001 Create root `.gitignore` covering `.env`, `**/.env`, `node_modules/`, `dist/`, `.angular/`, `coverage/`, `backend/bin/` (keep `deploy/.env.example` tracked)
- [X] T002 Scaffold frontend with `npx ng new luna --directory frontend --prefix lu --style css --routing --ssr false --ai-config claude --interactive false` (Angular 21 defaults: standalone, zoneless, Vitest); confirm `npm test -- --watch=false` passes in `frontend/`
- [X] T003 [P] Add angular-eslint via `npx ng add angular-eslint` in `frontend/`, ensure `npm run lint` script exists in `frontend/package.json` and passes on the scaffold
- [X] T004 [P] Install fonts `npm i @fontsource-variable/lexend @fontsource/noto-sans` in `frontend/package.json` (research R10)
- [X] T005 [P] Initialize Go module `github.com/luongtran/luna/backend` (go 1.25) in `backend/go.mod` and add `go.mongodb.org/mongo-driver/v2`
- [X] T006 [P] Create `backend/.golangci.yml` (golangci-lint v2 format) with linters govet, staticcheck, errcheck, ineffassign, unused, gosec, revive, errorlint, bodyclose, noctx, contextcheck, misspell, gocritic and formatters gofumpt, goimports (research R2)
- [X] T007 [P] Create `backend/Makefile` with targets `run` (`go run ./cmd/api`), `test` (`go test ./...`), `lint` (docker run pinned `golangci/golangci-lint:v2.x` image mounting `backend/` and Go module/build caches), `build` (`CGO_ENABLED=0 go build -o bin/api ./cmd/api`) (research R1)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config, log, middleware ở backend; token, style, HTTP ở frontend

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Backend

- [X] T008 [P] Write tests in `backend/internal/platform/config/config_test.go` (table-driven, injected `getenv`): missing `MONGO_URI` → error naming `MONGO_URI`; defaults `HTTP_ADDR=:8080`, `LOG_LEVEL=info`; invalid `LOG_LEVEL` → error; both errors reported together; error text never contains the URI value
- [X] T009 Implement `Config` and `Load(getenv func(string) string) (Config, error)` using `errors.Join` in `backend/internal/platform/config/config.go` (contracts/config.md, data-model.md §4) so T008 passes
- [X] T010 [P] Implement `logger.New(level string) *slog.Logger` (JSON handler to stdout) in `backend/internal/platform/logger/logger.go`
- [X] T011 [P] Implement `WriteJSON(w, status, v)` helper in `backend/internal/platform/httpx/json.go`
- [X] T012 Write tests in `backend/internal/platform/httpx/middleware_test.go`: RequestID reuses valid incoming `X-Request-ID`, rejects invalid/too-long ones and generates 32-hex id, sets response header; Recover turns a panicking handler into 500 `{"error":"internal_error"}`; Logger records status code
- [X] T013 Implement `RequestID`, `Logger(*slog.Logger)`, `Recover(*slog.Logger)` middleware and `Chain` helper in `backend/internal/platform/httpx/middleware.go` (research R5) so T012 passes

### Frontend

- [X] T014 [P] Create `frontend/src/styles/tokens.css` with every token from `docs/design-system.md` (colors §3.1–3.3 light in `:root`, dark in `:root[data-theme="dark"]` and in `@media (prefers-color-scheme: dark) { :root:not([data-theme]) }`, `color-scheme` per mode, `--font-ui`, `--font-ipa`, `--text-*` size/line-height/weight, `--radius-*`, `--space-1..6` = 4/8/12/16/24/32px) (research R8)
- [X] T015 Update `frontend/src/styles.css`: import `styles/tokens.css` and only the `latin`, `latin-ext`, `vietnamese` subset CSS of `@fontsource-variable/lexend`; base styles `body { background: var(--color-bg); color: var(--color-text); font: var(--text-base) var(--font-ui); margin: 0 }`, `box-sizing: border-box`, `overflow-x` safe defaults (research R10)
- [X] T016 [P] Create `HealthResponse` and `ConnectionStatus` types in `frontend/src/app/core/models/health.ts` (data-model.md §1–2)
- [X] T017 [P] Write `frontend/src/app/core/interceptors/error.interceptor.spec.ts`: network error (status 0) → `ApiError{kind:'network'}`; HTTP 503 → `ApiError{kind:'http', status:503, body}` preserving body; success passes through
- [X] T018 Implement `errorInterceptor` (`HttpInterceptorFn`, via `ng generate interceptor core/interceptors/error --functional`) and `ApiError` type in `frontend/src/app/core/interceptors/error.interceptor.ts` (research R7) so T017 passes
- [X] T019 Configure `frontend/src/app/app.config.ts` with `provideZonelessChangeDetection()`, `provideRouter(routes)`, `provideHttpClient(withInterceptors([errorInterceptor]))`; set `<html lang="vi">` and `<title>Luna</title>` in `frontend/src/index.html`
- [X] T020 [P] Create `frontend/proxy.conf.json` (`/api` → `http://localhost:8080`) and wire it as `proxyConfig` of the `serve` target in `frontend/angular.json`

**Checkpoint**: `go test ./...` and `npm test -- --watch=false` pass; user stories can begin

---

## Phase 3: User Story 1 - Chạy toàn bộ app bằng một lệnh (Priority: P1) 🎯 MVP

**Goal**: `docker compose -f deploy/docker-compose.yml up --build` mở được app có header "Luna" và trang chủ tạm

**Independent Test**: quickstart.md bước 1 trên máy chỉ có Docker

### Tests for User Story 1 (REQUIRED - constitution VI) ⚠️

- [X] T021 [P] [US1] Write `frontend/src/app/shared/components/app-header/app-header.spec.ts`: renders a `<header>` containing text "Luna"
- [X] T022 [P] [US1] Write `frontend/src/app/app.spec.ts` (replace scaffold test): app renders `lu-app-header` and a `router-outlet`; route `''` lazy-loads the home component

### Implementation for User Story 1

- [X] T023 [P] [US1] Implement `mongo.Connect(ctx, uri)` returning a `*Client` with `ServerSelectionTimeout` 2s, `Ping(ctx) error` (readpref primary) and `Disconnect(ctx) error`; no ping at connect (research R3) in `backend/internal/storage/mongo/client.go`
- [X] T024 [US1] Implement `backend/cmd/api/main.go`: `run(ctx) error` pattern → load config (print each error to stderr, exit 1), create logger, connect Mongo, build `http.ServeMux`, wrap with `RequestID → Logger → Recover`, `http.Server` with ReadHeaderTimeout 5s / Read 15s / Write 15s / Idle 60s, `signal.NotifyContext` on SIGINT/SIGTERM, `Shutdown` with 10s timeout then `Disconnect` with 5s timeout, log start/stop (research R5)
- [X] T025 [P] [US1] Create `backend/Dockerfile`: stage `golang:1.25` (`go mod download`, `CGO_ENABLED=0 go build -trimpath -o /api ./cmd/api`) → `gcr.io/distroless/static-debian12:nonroot`, `EXPOSE 8080`, `ENTRYPOINT ["/api"]`; add `backend/.dockerignore`
- [X] T026 [P] [US1] Generate `AppHeader` component (`ng g c shared/components/app-header`) in `frontend/src/app/shared/components/app-header/` showing "Luna" in `--text-lg`, `--color-surface` background, bottom border `--color-line`, horizontal padding `--space-4`, fits 360px; so T021 passes
- [X] T027 [P] [US1] Generate `Home` component (`ng g c features/home`) in `frontend/src/app/features/home/` with heading "Trang chủ" and a placeholder area for connection status (filled in US2)
- [X] T028 [US1] Update `frontend/src/app/app.ts`, `frontend/src/app/app.html` and `frontend/src/app/app.routes.ts`: layout = header + `<main>` with `router-outlet`; `''` → `loadComponent` home; `**` → redirect `''`; so T022 passes
- [X] T029 [P] [US1] Create `frontend/nginx.conf`: listen 80, `root /usr/share/nginx/html`, `try_files $uri $uri/ /index.html`, `location /api/` proxy to `http://backend:8080` using Docker DNS `resolver 127.0.0.11` + variable upstream so nginx starts when backend is down, long cache for hashed assets (research R9)
- [X] T030 [P] [US1] Create `frontend/Dockerfile`: stage `node:22-alpine` (`npm ci`, `npx ng build`) → `nginx:alpine` copying `dist/luna/browser` and `nginx.conf`; add `frontend/.dockerignore`
- [X] T031 [US1] Create `deploy/docker-compose.yml`: `mongo` (`mongo:8`, volume `mongo-data`, port `127.0.0.1:27017:27017`); `backend` (build `../backend`, env `MONGO_URI: ${MONGO_URI:-mongodb://mongo:27017}`, `LOG_LEVEL: ${LOG_LEVEL:-info}`, `stop_grace_period: 15s`, no host port); `frontend` (build `../frontend`, port `${WEB_PORT:-8000}:80`, depends_on backend) (research R9)
- [X] T032 [P] [US1] Create `deploy/.env.example` listing `MONGO_URI`, `LOG_LEVEL`, `WEB_PORT` with comments (contracts/config.md)
- [X] T033 [US1] Create root `README.md` (Vietnamese) section "Chạy nhanh": prerequisites (Docker only), `docker compose -f deploy/docker-compose.yml up --build`, open `http://localhost:8000`, stop/cleanup commands, optional `deploy/.env`
- [X] T034 [US1] Manual check: quickstart.md bước 1 (fresh `docker compose ... up --build` → header "Luna", home page shows) and record result

**Checkpoint**: App chạy bằng một lệnh; US1 kiểm chứng độc lập

---

## Phase 4: User Story 2 - Xem tình trạng kết nối trên trang chủ (Priority: P1)

**Goal**: `GET /api/health` và trang chủ hiện 4 trạng thái kết nối

**Independent Test**: quickstart.md bước 2 và 3

### Tests for User Story 2 (REQUIRED - constitution VI) ⚠️

- [X] T035 [P] [US2] Write `backend/internal/health/handler_test.go` with fake `Pinger` and `httptest`: nil error → 200 `{"status":"ok","database":"up"}`; error → 503 `{"status":"degraded","database":"down"}`; pinger blocking past deadline → 503 within ~2s (use short injected timeout); `Content-Type: application/json`; POST → 405 via mux pattern `GET /api/health` (contracts/health-api.md)
- [X] T036 [P] [US2] Write `frontend/src/app/features/home/home.spec.ts` with `provideHttpClientTesting`: initial "Đang kiểm tra kết nối…"; 200+up → "Đã kết nối"; 503+down → "Mất kết nối cơ sở dữ liệu"; network error, 502, and 200 with malformed body → "Không kết nối được máy chủ"; status region has `role="status"` (research R7)

### Implementation for User Story 2

- [X] T037 [US2] Implement `Pinger` interface, `Response` type and `Handler` (constructor `NewHandler(p Pinger, timeout time.Duration, log *slog.Logger)`, ping with `context.WithTimeout`, log ping errors at warn without leaking details to response) in `backend/internal/health/handler.go` so T035 passes
- [X] T038 [US2] Register `mux.Handle("GET /api/health", health.NewHandler(mongoClient, 2*time.Second, log))` in `backend/cmd/api/main.go`
- [X] T039 [P] [US2] Implement `ApiService.getHealth(): Observable<HealthResponse>` calling `/api/health` with `timeout(5000)` in `frontend/src/app/core/services/api.service.ts` (`ng g s core/services/api`)
- [X] T040 [US2] Implement status logic in `frontend/src/app/features/home/home.ts`: signal `status: ConnectionStatus` starting `checking`, map responses/`ApiError` per research R7 table (503 counts as `database-down` only if body has `database: "down"`) so T036 passes
- [X] T041 [US2] Implement status card in `frontend/src/app/features/home/home.html` + `home.css`: `role="status" aria-live="polite"`, text in `--color-text`, icon (✓ / ! / ✕ / spinner, `aria-hidden="true"`) and left border colored `--color-ok` / `--color-warn` / `--color-bad` / `--color-text-muted`, `--radius-lg`, `--color-surface`, spinner respects `prefers-reduced-motion` (research R6, data-model.md §2)
- [X] T042 [US2] Manual check: quickstart.md bước 2 (stop/start mongo) and bước 3 (stop backend) in Docker; record results

**Checkpoint**: US1 + US2 hoạt động; tiêu chí nghiệm thu "trạng thái kết nối" đạt

---

## Phase 5: User Story 3 - Chọn giao diện Sáng / Tối / Theo hệ thống (Priority: P2)

**Goal**: Nút chọn giao diện trong header, đổi ngay, nhớ sau khi tải lại, theo hệ thống khi chọn "Theo hệ thống"

**Independent Test**: quickstart.md bước 4

### Tests for User Story 3 (REQUIRED - constitution VI) ⚠️

- [X] T043 [P] [US3] Write `frontend/src/app/core/services/theme.service.spec.ts` (mock `localStorage` and `matchMedia`): default `system`; `system` + dark media → `data-theme="dark"` on `<html>`; `set('light')` updates attribute immediately and stores `luna.theme`; stored `dark` restored on new instance; invalid stored value → `system`; `localStorage` throwing on get/set → no exception, preference still applied; media `change` event while `system` → attribute updates
- [X] T044 [P] [US3] Extend `frontend/src/app/shared/components/app-header/app-header.spec.ts`: renders `role="radiogroup"` with 3 `role="radio"` options labelled "Sáng", "Tối", "Theo hệ thống"; clicking one calls `ThemeService.set` and updates `aria-checked`

### Implementation for User Story 3

- [X] T045 [US3] Implement `ThemeService` (`ng g s core/services/theme`) in `frontend/src/app/core/services/theme.service.ts`: `preference` signal (`light|dark|system`), `systemDark` signal from `matchMedia` listener, computed `resolved`, `effect` setting `data-theme` and `style.colorScheme` on `document.documentElement` and writing `localStorage['luna.theme']` inside try/catch (research R8, data-model.md §3) so T043 passes
- [X] T046 [US3] Add inline script in `frontend/src/index.html` `<head>` that reads `luna.theme` (try/catch) and sets `data-theme` before Angular boots to avoid a light flash (research R8)
- [X] T047 [US3] Add theme radiogroup to `frontend/src/app/shared/components/app-header/app-header.html` + `.ts` + `.css`: 3 icon buttons (sun / moon / monitor inline SVG with visually-hidden labels), 44×44px targets, selected state uses `--color-primary-soft` + `--color-primary-ink` and a non-color indicator (border/weight), arrow-key navigation, fits 360px next to "Luna" so T044 passes
- [ ] T048 [US3] Manual check: quickstart.md bước 4 (default, switch, reload without flash, OS change, blocked storage); record results

**Checkpoint**: US1–US3 hoạt động

---

## Phase 6: User Story 4 - Công cụ cho người phát triển (Priority: P3)

**Goal**: Chạy riêng từng phần, cấu hình qua biến môi trường, test/lint offline, tắt an toàn

**Independent Test**: quickstart.md bước 7–10

### Tests for User Story 4 (REQUIRED - constitution VI) ⚠️

- Covered by T008 (thiếu `MONGO_URI`), T012 (middleware); các tiêu chí còn lại kiểm tra thủ công bên dưới vì phụ thuộc tín hiệu hệ điều hành, mạng và Docker.

### Implementation for User Story 4

- [X] T049 [US4] Extend `README.md` with sections "Phát triển" (mongo only via compose, backend `go run ./cmd/api` with `MONGO_URI` in PowerShell and bash, frontend `npm start` at `http://localhost:4200` with proxy), "Cấu hình" (table from contracts/config.md, `cp deploy/.env.example deploy/.env`), "Test và lint" (`make test|lint` plus direct PowerShell equivalents incl. the full `docker run golangci/golangci-lint` command, `npm test -- --watch=false`, `npm run lint`), "Cấu trúc thư mục"
- [ ] T050 [US4] Manual check: quickstart.md bước 10 (backend local + `npm start`, edit `features/home` → browser auto-reloads)
- [X] T051 [US4] Manual check: quickstart.md bước 8 (run backend without `MONGO_URI` → `config: MONGO_URI is required`, exit code 1) and `git ls-files | grep -i "\.env$"` returns nothing
- [X] T052 [US4] Manual check: quickstart.md bước 9 (Ctrl+C and `docker compose stop backend` during an in-flight `/api/health` with mongo stopped → full 503 response received, shutdown log after request log, exit 0)
- [X] T053 [US4] Manual check: quickstart.md bước 7 (after caches warm, disconnect network, run frontend test+lint and backend test+lint → all complete)

**Checkpoint**: Tất cả user story hoạt động độc lập

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T054 Run lint (frontend `npm run lint` + backend `make lint` or README docker equivalent) with zero errors; fix findings
- [X] T055 Run all tests (`go test -race ./...` in `backend/`, `npm test -- --watch=false` in `frontend/`) with zero failures
- [ ] T056 Manual check at 360px width in both light and dark mode (quickstart.md bước 5): no horizontal scroll, no contrast errors in Lighthouse Accessibility, status uses icon + text
- [ ] T057 Manual check offline font (quickstart.md bước 6): Vietnamese diacritics render in Lexend with network offline, no requests to external domains
- [X] T058 Grep `frontend/src/app/**/*.css` for hard-coded colors (`#`, `rgb(`, `hsl(`) and font sizes in `px` outside `tokens.css`; replace with tokens (constitution IV)
- [ ] T059 Tick the 7 F0 acceptance criteria in `docs/phases/giai-doan-1.md` once T034, T042, T048, T050–T057 pass

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001, T002, T005 first; T003/T004 after T002; T006/T007 after T005
- **Foundational (Phase 2)**: after Setup; backend (T008–T013) and frontend (T014–T020) tracks are independent
- **US1 (Phase 3)**: after Foundational
- **US2 (Phase 4)**: after US1 (needs `main.go` T024, home component T027, compose T031 for manual check)
- **US3 (Phase 5)**: after Foundational + T026 (header component); independent of US2
- **US4 (Phase 6)**: after US1–US3 (manual checks exercise the whole app)
- **Polish (Phase 7)**: after all stories

### Within Each User Story

- Tests written first and FAIL before implementation
- Backend: storage → handler → wiring in `main.go`
- Frontend: service → component logic → template/styles
- Manual check task last

### Parallel Opportunities

- Setup: T003 ∥ T004 (frontend), T006 ∥ T007 (backend), and the frontend/backend tracks in parallel
- Foundational: T008, T010, T011, T014, T016, T017, T020 all [P]
- US1: T021 ∥ T022 ∥ T023 ∥ T025 ∥ T026 ∥ T027 ∥ T029 ∥ T030 ∥ T032
- US2: T035 ∥ T036 ∥ T039; backend T037–T038 ∥ frontend T040–T041
- US3 can run in parallel with US2 once T026 exists

---

## Parallel Example: User Story 2

```bash
# Tests first, together:
Task: "Write backend/internal/health/handler_test.go (up/down/timeout/405)"
Task: "Write frontend/src/app/features/home/home.spec.ts (4 states)"

# Then implement backend and frontend tracks side by side:
Task: "Implement backend/internal/health/handler.go + register in cmd/api/main.go"
Task: "Implement ApiService.getHealth + home status logic and card"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 Setup → Phase 2 Foundational
2. Phase 3 US1 → **STOP and VALIDATE** with quickstart bước 1

### Incremental Delivery

1. US1 → app chạy bằng một lệnh
2. US2 → trạng thái kết nối (hoàn tất phần "từ đầu đến cuối")
3. US3 → giao diện sáng/tối
4. US4 → hoàn thiện README và kiểm tra công cụ phát triển
5. Polish → lint, test, 360px, font offline, tick tiêu chí nghiệm thu

---

## Notes

- [P] tasks = different files, no dependencies
- Tạo file Angular bằng `ng generate` (docs/architecture.md); component selector prefix `lu`
- Code, API, commit message bằng tiếng Anh; chữ trên giao diện và README bằng tiếng Việt
- Chỉ commit khi người dùng yêu cầu

---

## Ghi chú triển khai (2026-09-29)

**Khác với mô tả task** (theo mặc định của công cụ, không đổi hành vi):
- Angular CLI 21 đặt tên interceptor là `core/interceptors/error-interceptor.ts` (không phải `error.interceptor.ts`); service tạo bằng `--type=service` nên giữ `*.service.ts`.
- Driver MongoDB v2 không nhận context khi connect: `mongo.Connect(uri)`. `logger.New(w, level)` nhận thêm `io.Writer` để dễ test.
- Không cần `provideZonelessChangeDetection()`: Angular 21 mặc định zoneless.
- npm 10.9 lỗi `edgesOut` khi cài trên máy này; đã dùng `npx -y npm@11 install` (ghi trong README). `npm ci` trong Docker vẫn chạy được.

**Kết quả kiểm tra tự động / dòng lệnh**
- Backend: `go test -race ./...` qua; golangci-lint v2.14.0: 0 issues; chạy lại được với `GOPROXY=off` và container `--network none`.
- Frontend: 28 test Vitest qua; `ng lint` sạch; `ng build` đóng gói 3 file font Lexend (latin, latin-ext, vietnamese).
- Docker (qua nginx `localhost:8000`): đủ cả → 200; tắt mongo → 503 sau ~2 s; bật lại → 200 không cần khởi động lại backend; tắt backend → 502 (giao diện hiểu là "Không kết nối được máy chủ"), trang vẫn 200.
- Tắt an toàn: `docker compose stop backend` giữa request → request nhận đủ 503, log "shutting down" trước "server stopped", mã thoát 0.
- Thiếu `MONGO_URI` / `LOG_LEVEL` sai → in đúng tên biến, mã thoát 1. Không có file `.env` nào được git theo dõi.
- Độ tương phản (tính theo WCAG): mọi cặp chữ/nền ≥ 4.5:1, biểu tượng và viền ≥ 3:1 ở cả hai chế độ.

**Còn lại cần kiểm tra bằng trình duyệt**: T048, T050, T056, T057, rồi T059.
