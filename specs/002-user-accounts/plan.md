# Implementation Plan: Tài khoản

**Branch**: `002-user-accounts` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-user-accounts/spec.md` + khối 2 trong
`docs/spec-inputs/f1-tai-khoan.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Thêm tài khoản email + mật khẩu vào khung F0. Backend: domain `internal/auth` (handler, service,
interface repository) với mật khẩu băm argon2id, phiên lưu phía server (token ngẫu nhiên 32 byte, chỉ
lưu SHA-256, cookie HttpOnly SameSite=Strict, 30 ngày, TTL index), khoá đăng nhập 5 lần/15 phút trong bộ
nhớ, tài khoản đầu tiên là admin; middleware `RequireAuth` / `RequireAdmin`; hiện thực repository trên
MongoDB. Frontend: trang `/login`, `/register`, trang quản trị tạm `/admin`, trang `/forbidden`;
`AuthService` (signal `currentUser`, gọi `/api/auth/me` khi khởi động), guard dạng hàm, interceptor xử lý
401/403; header hiện email, Đăng xuất, liên kết Quản trị.

## Technical Context

**Language/Version**: Go 1.25 (backend); TypeScript 5.9 strict + Angular 21 (frontend)

**Primary Dependencies**:
- Backend: thư viện chuẩn, `go.mongodb.org/mongo-driver/v2`, **mới**: `golang.org/x/crypto/argon2`
- Frontend: Angular 21 (`@angular/forms` reactive forms, router guards), không thêm thư viện

**Storage**: MongoDB 8. Database `luna` (mới: biến `MONGO_DATABASE`), collection `users`, `sessions`.

**Testing**: Go `testing` + `httptest` với repository giả; Vitest + `HttpTestingController`

**Target Platform**: như F0 (trình duyệt, container Linux không GPU)

**Project Type**: Web application (frontend + backend)

**Performance Goals**: Đăng nhập/đăng ký phản hồi < 1 giây (argon2id ~50–100 ms với tham số OWASP)

**Constraints**: Không lộ email tồn tại qua thông báo hoặc thời gian phản hồi ở đăng nhập; không log mật
khẩu hay token; quyền kiểm tra ở API; backend vẫn khởi động khi Mongo tắt (giữ quyết định F0 R3)

**Scale/Scope**: 1–vài tài khoản; 4 endpoint auth + 1 endpoint admin tạm; 4 trang giao diện

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Đối chiếu `.specify/memory/constitution.md` v1.0.0.

- [x] **I. docs/ là nguồn sự thật**: F1 là tính năng kế tiếp theo `docs/phases/`; cấu trúc `internal/auth`,
  `features/auth`, `core/guards` đúng `docs/architecture.md`; spec bao phủ 6 tiêu chí nghiệm thu F1.
- [x] **II. Chi phí 0 đồng**: `golang.org/x/crypto` (BSD); không dịch vụ ngoài.
- [x] **III. AI là phần bổ sung**: không có AI.
- [x] **IV. Mobile-first**: form một cột, header xuống hai dòng ở màn hẹp (research R11); chỉ dùng token;
  lỗi form có chữ, không chỉ viền đỏ.
- [x] **V. Dữ liệu người học**: quyền kiểm tra ở API (`RequireAuth`, `RequireAdmin`); truy cập DB qua
  `auth.UserRepository`, `auth.SessionRepository`; mật khẩu argon2id; token chỉ lưu hash; `/me` chỉ trả dữ
  liệu của chính người gọi; không bí mật mới nào cần commit.
- [x] **VI. Kiểm thử**: service (đăng ký, trùng email, user đầu là admin, khoá 5 lần, phiên hết hạn) với
  repository giả; handler với `httptest` cho thành công, lỗi đầu vào, sai quyền; middleware quyền; frontend
  test guard, AuthService, form, interceptor. Không test nào cần Mongo thật hay mạng.
- [x] **VII. Đơn giản trước**: không thư viện auth/JWT/session; bộ đếm khoá trong bộ nhớ; **bỏ**
  `auth.interceptor.ts` trong khối 2 vì request cùng origin đã tự gửi cookie (research R9).

**Kết quả**: ĐẠT, không có ngoại lệ.

## Project Structure

### Documentation (this feature)

```text
specs/002-user-accounts/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── auth-api.md      # /api/auth/*, /api/admin/ping, mã lỗi
│   └── config.md        # biến môi trường mới
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/api/main.go                     # + time/tzdata, nối auth, route, EnsureIndexes nền
└── internal/
    ├── auth/
    │   ├── model.go                    # User, Role, Session, lỗi nghiệp vụ
    │   ├── repository.go               # UserRepository, SessionRepository
    │   ├── password.go (+ _test)       # argon2id hash/verify (PHC string)
    │   ├── lockout.go (+ _test)        # đếm lần sai theo email, mutex, clock tiêm vào
    │   ├── service.go (+ _test)        # Register, Login, Logout, Authenticate
    │   ├── handler.go (+ _test)        # register, login, logout, me; cookie
    │   ├── validate.go (+ _test)       # email, mật khẩu, múi giờ
    │   └── fake_test.go                # repository giả trong bộ nhớ
    ├── admin/handler.go (+ _test)      # GET /api/admin/ping (trang quản trị tạm)
    ├── platform/
    │   ├── config/config.go            # + MONGO_DATABASE, COOKIE_SECURE
    │   └── httpx/
    │       ├── auth.go (+ _test)       # Principal, RequireAuth, RequireAdmin, PrincipalFrom
    │       ├── errors.go               # WriteError(code, message)
    │       └── json.go                 # + DecodeJSON (giới hạn kích thước, bắt Content-Type)
    └── storage/mongo/
        ├── client.go                   # + Database()
        ├── users.go                    # hiện thực auth.UserRepository
        ├── sessions.go                 # hiện thực auth.SessionRepository
        └── indexes.go                  # unique email, unique tokenHash, TTL expiresAt

frontend/src/app/
├── app.routes.ts                       # guard cho mọi route
├── app.config.ts                       # provideAppInitializer → AuthService.load()
├── core/
│   ├── models/user.ts
│   ├── services/auth.service.ts (+ spec)
│   ├── guards/auth.guard.ts (+ spec)   # authGuard, guestGuard, adminGuard
│   └── interceptors/error-interceptor.ts (+ spec)  # + 401 → /login?returnUrl, 403 → /forbidden
├── shared/utils/safe-return-url.ts (+ spec)        # chỉ chấp nhận đường dẫn nội bộ
├── features/
│   ├── auth/login/ (+ spec)
│   ├── auth/register/ (+ spec)
│   ├── admin/ (trang tạm)
│   └── forbidden/
└── shared/components/app-header/       # + email, Đăng xuất, Quản trị
```

**Structure Decision**: Giữ cấu trúc F0; thêm domain `auth` và `admin` ở backend, `features/auth`,
`features/admin`, `features/forbidden`, `core/guards`, `shared/utils` ở frontend theo
`docs/architecture.md` (hàm thuần chỉ nằm trong `shared/utils/`).

## Post-Design Constitution Check

Thiết kế Phase 1 thêm 2 interface repository (bắt buộc bởi nguyên tắc V), 1 phụ thuộc
(`golang.org/x/crypto`), 2 biến môi trường có mặc định. Không thêm lớp trừu tượng nào khác. **Vẫn ĐẠT.**

## Complexity Tracking

Không có vi phạm cần giải trình.
