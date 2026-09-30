# Implementation Plan: Khung dự án Luna

**Branch**: `001-project-skeleton` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-project-skeleton/spec.md` + khối 2 trong
`docs/spec-inputs/f0-khung-du-an.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Dựng bộ khung chạy từ đầu đến cuối cho Luna: frontend Angular 21 (thanh trên cùng "Luna", trang chủ
tạm hiển thị tình trạng kết nối, nút chọn giao diện Sáng / Tối (menu thả xuống; chưa chọn thì theo thiết bị), token và font theo
design-system), backend Go 1.25 dùng `net/http` thuần với một endpoint `GET /api/health` ping MongoDB
(timeout 2 giây), và Docker Compose chạy cả ba (mongo, backend, frontend qua nginx) bằng một lệnh.
Kèm test mẫu, lint, cấu hình qua biến môi trường, graceful shutdown và README.

## Technical Context

**Language/Version**: Go 1.25 (backend); TypeScript 5.9 strict + Angular 21 (frontend); Node 22 để build

**Primary Dependencies**:
- Backend: thư viện chuẩn (`net/http`, `log/slog`, `os/signal`), `go.mongodb.org/mongo-driver/v2`
- Frontend: Angular 21 (standalone, signals, zoneless), `@fontsource-variable/lexend`, `@fontsource/noto-sans`, `angular-eslint`

**Storage**: MongoDB 8 (image `mongo:8`, volume dữ liệu). F0 chỉ ping, chưa có collection nào.

**Testing**: Backend `testing` + `net/http/httptest`; frontend Vitest (runner mặc định của Angular 21)

**Target Platform**: Trình duyệt hiện đại trên điện thoại và máy tính; máy chủ Linux container, không GPU

**Project Type**: Web application (frontend + backend + deploy)

**Performance Goals**: Trang chủ hiện trạng thái kết nối trong ≤ 3 giây (SC-002); đổi giao diện < 1 giây (SC-003)

**Constraints**: Chi phí 0 đồng; test và lint chạy không cần mạng; font đóng gói kèm app; dùng tốt từ
360px; WCAG AA hai chế độ; tắt máy chủ chờ tối đa 10 giây

**Scale/Scope**: Một người dùng, một người phát triển; 1 trang, 1 endpoint

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: cấu trúc theo `docs/architecture.md`, token theo
  `docs/design-system.md`; F0 là tính năng đầu tiên của Giai đoạn 1; spec bao phủ đủ 7 tiêu chí
  nghiệm thu. Lựa chọn giao diện lưu localStorage thay vì tài khoản đúng như `giai-doan-1.md` cho
  phép ("cho tới khi có F12").
- [x] **II. Chi phí 0 đồng**: Go, Angular, MongoDB Community, nginx, distroless, golangci-lint đều
  miễn phí; font Lexend và Noto Sans giấy phép OFL.
- [x] **III. AI là phần bổ sung**: F0 không có AI. Interface AI sẽ thêm khi tính năng cần (F2).
- [x] **IV. Mobile-first**: layout một cột, kiểm tra 360px; `tokens.css` chứa đủ token hai chế độ;
  trạng thái kết nối có chữ + biểu tượng, không chỉ màu (research R6).
- [x] **V. Dữ liệu người học**: chưa có dữ liệu người dùng; `/api/health` không trả dữ liệu nhạy cảm
  (không lộ URI hay lỗi chi tiết). Truy cập Mongo qua interface `health.Pinger`; interface
  repository đầu tiên sẽ có ở F1. Bí mật chỉ trong biến môi trường, `.env` trong `.gitignore`.
- [x] **VI. Kiểm thử**: unit test config, health handler (up/down), middleware recover; Vitest cho
  ThemeService và trang chủ (3 trạng thái). Không test nào cần mạng hay Mongo thật. Tiêu chí không
  test tự động được (một lệnh, 360px, font offline, tắt an toàn) có bước kiểm tra thủ công.
- [x] **VII. Đơn giản trước**: không framework HTTP, không viper, không thư viện DI, không thư viện
  UI. Interface duy nhất là `Pinger` (phục vụ nguyên tắc V/VI).

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/001-project-skeleton/
├── plan.md              # File này
├── research.md          # Phase 0
├── data-model.md        # Phase 1
├── quickstart.md        # Phase 1: kịch bản kiểm tra thủ công
├── contracts/
│   ├── health-api.md    # GET /api/health
│   └── config.md        # Biến môi trường
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks (chưa tạo)
```

### Source Code (repository root)

```text
README.md                         # Hướng dẫn chạy (FR-016)
.gitignore                        # .env, node_modules, dist, bin...

frontend/                         # ng new luna --directory frontend --prefix lu
├── angular.json
├── package.json
├── eslint.config.js              # angular-eslint
├── proxy.conf.json               # /api → http://localhost:8080 khi ng serve
├── Dockerfile                    # node:22 build → nginx:alpine
├── nginx.conf                    # SPA fallback + proxy /api → backend:8080
└── src/
    ├── index.html                # script nhỏ gắn data-theme trước khi Angular chạy
    ├── main.ts
    ├── styles.css                # import tokens.css + font, reset tối thiểu
    ├── styles/
    │   └── tokens.css            # toàn bộ token design-system
    └── app/
        ├── app.ts / app.html     # header + <router-outlet>
        ├── app.config.ts         # provideZonelessChangeDetection, provideRouter, provideHttpClient(withInterceptors)
        ├── app.routes.ts         # '' → features/home (lazy)
        ├── core/
        │   ├── interceptors/error.interceptor.ts   (+ .spec.ts)
        │   ├── models/health.ts                    # HealthResponse, ConnectionStatus
        │   └── services/
        │       ├── api.service.ts                  # getHealth()
        │       ├── theme.service.ts                (+ .spec.ts)
        ├── shared/components/app-header/           # tên Luna + nút chọn giao diện
        └── features/home/                          # trang chủ tạm (+ .spec.ts)

backend/
├── go.mod                        # module github.com/luongtran/luna/backend
├── Makefile                      # run, test, lint, build
├── .golangci.yml
├── Dockerfile                    # golang:1.25 build → distroless/static:nonroot
├── cmd/api/main.go               # đọc config, nối phụ thuộc, chạy server, graceful shutdown
└── internal/
    ├── health/
    │   ├── handler.go            # Pinger interface + Handler
    │   └── handler_test.go
    ├── platform/
    │   ├── config/config.go (+ _test.go)
    │   ├── httpx/
    │   │   ├── middleware.go     # RequestID, Logger, Recover
    │   │   ├── middleware_test.go
    │   │   └── json.go           # WriteJSON
    │   └── logger/logger.go      # slog JSON handler theo LOG_LEVEL
    └── storage/mongo/client.go   # Connect, Ping, Disconnect (thoả health.Pinger)

deploy/
├── docker-compose.yml            # mongo, backend, frontend
└── .env.example
```

**Structure Decision**: Web application ba thư mục `frontend/`, `backend/`, `deploy/` theo
`docs/architecture.md`. F0 chỉ tạo những thư mục con được dùng ngay; các domain khác (auth, lesson...)
thêm ở tính năng tương ứng.

## Post-Design Constitution Check

Sau Phase 1, thiết kế không thêm phụ thuộc hay lớp trừu tượng nào ngoài danh sách trên. Các quyết định
ở research (R3 backend khởi động kể cả khi Mongo chưa sẵn sàng, R6 màu trạng thái chỉ tô biểu tượng,
R9 compose có giá trị mặc định) đều phục vụ trực tiếp tiêu chí nghiệm thu. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
