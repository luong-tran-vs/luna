# Luna: Cấu trúc mã nguồn

> Phiên bản: v1 (2026-09-29). Dùng làm đầu vào cho `/speckit.plan`.

## 1. Tổng quan repo

```
Luna/
├── frontend/        ← Angular
├── backend/         ← Go
├── deploy/          ← docker-compose, sao lưu (F13)
└── docs/
```

## 2. Frontend (Angular)

**Quy ước**
- Standalone component + signals, không dùng NgModule. TypeScript strict.
- Chia theo **tính năng**; mỗi tính năng lazy load theo route.
- Interceptor và guard viết dạng hàm (`HttpInterceptorFn`, `CanActivateFn`).
- Tạo file bằng `ng generate` (component, service, guard, interceptor, pipe...).
- Chỉ có **một** thư mục hàm tiện ích: `shared/utils/` (không tách `helpers/`).
- Màu, font, cỡ chữ chỉ dùng token trong `styles/tokens.css` ([design-system.md](design-system.md)).

```
frontend/src/
├── app/
│   ├── core/                 ← khởi tạo một lần, dùng toàn app
│   │   ├── interceptors/     ← auth (gắn token), error (thông báo lỗi, 401 → đăng nhập)
│   │   ├── guards/           ← auth, admin
│   │   ├── services/         ← api, auth, theme
│   │   └── models/           ← Lesson, Sentence, Card, Progress, Settings...
│   ├── shared/               ← dùng lại giữa các tính năng
│   │   ├── components/       ← progress-bar, step-indicator, word-popup, waveform...
│   │   ├── pipes/
│   │   ├── directives/
│   │   └── utils/            ← hàm thuần, có unit test (so sánh chép chính tả, tách từ...)
│   ├── features/
│   │   ├── auth/             ← F1
│   │   ├── admin/            ← F2
│   │   ├── lesson/           ← L: review/, reading/ (F3), listening/ (F4)
│   │   ├── vocabulary/       ← F5
│   │   ├── home/             ← F6
│   │   └── settings/         ← F12, F13
│   ├── app.routes.ts
│   └── app.config.ts
└── styles/
    └── tokens.css
```

## 3. Backend (Go)

**Quy ước**
- Theo bộ skill `samber/cc-skills-golang` trong `.claude/skills/` (layout, naming, error handling, testing, lint...).
- Kiến trúc: **chia theo domain + ports**. Mỗi domain là một package gồm handler, service, và interface repository.
- Chỉ tách interface ở hai chỗ constitution bắt buộc: **cơ sở dữ liệu** (repository) và **AI** (provider).
- Dependency injection: **tự viết constructor** `NewXxx(...)`, nối mọi thứ trong `cmd/api/main.go`. Không dùng thư viện DI.
- 12-factor: cấu hình đọc từ biến môi trường, log JSON ra stdout (`slog`), tắt server an toàn (graceful shutdown).
- Test đặt cạnh file code (`xxx_test.go`), dữ liệu mẫu trong `testdata/`.

```
backend/
├── cmd/
│   └── api/main.go           ← đọc config, nối phụ thuộc, chạy server
├── internal/
│   ├── auth/                 ← F1
│   ├── lesson/               ← F2: bài học, lộ trình, tách câu
│   ├── vocab/                ← F5: thẻ, lịch ôn FSRS
│   ├── progress/             ← L, F6: luồng một ngày học, streak, thống kê
│   ├── settings/             ← F12
│   ├── dictionary/           ← F3: tra từ điển SQLite
│   ├── ai/                   ← interface Provider + gemini/, openrouter/, ollama/
│   ├── wordmatch/            ← F18: khớp từ, cụm từ trong văn bản
│   ├── job/                  ← chạy nền: chú thích AI, phần luyện tập, chấm bài viết (trạng thái đang chạy/xong/lỗi)
│   ├── storage/
│   │   └── mongo/            ← hiện thực các interface repository
│   └── platform/
│       ├── config/
│       ├── httpx/            ← router, middleware (auth, admin, log, recover)
│       └── logger/
├── Makefile
├── .golangci.yml
└── go.mod
```

**Mỗi domain gồm**

| File | Vai trò |
|---|---|
| `handler.go` | Nhận HTTP, kiểm tra đầu vào, gọi service, trả JSON |
| `service.go` | Logic nghiệp vụ; chỉ phụ thuộc interface |
| `repository.go` | Interface truy cập dữ liệu (hiện thực nằm ở `storage/mongo/`) |
| `model.go` | Kiểu dữ liệu của domain |
| `*_test.go` | Test service với repository giả lập |
