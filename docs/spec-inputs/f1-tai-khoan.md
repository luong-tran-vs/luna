# Đầu vào cho F1: Tài khoản

Nguồn: [giai-doan-1.md › F1](../phases/giai-doan-1.md).

## Khối 1: `/speckit-specify`

```text
F1 – Tài khoản (Giai đoạn 1) cho Luna.

Mục tiêu: mỗi người có tài khoản riêng để dữ liệu học tập tách biệt; có vai trò quản trị viên để soạn bài học.

Kịch bản:
- Người mới mở app, chưa đăng nhập → được đưa tới trang đăng nhập, có liên kết sang đăng ký.
- Đăng ký bằng email và mật khẩu → đăng nhập luôn và vào trang chủ.
- Tài khoản đăng ký đầu tiên trong hệ thống là quản trị viên; các tài khoản sau là người học.
- Đăng nhập, đăng xuất. Đăng nhập được giữ trên thiết bị đó 30 ngày.
- Người học cố vào trang quản trị → bị chặn và thấy thông báo không có quyền.
- Phiên đăng nhập hết hạn giữa chừng → được đưa về trang đăng nhập, đăng nhập lại thì quay về trang đang xem.

Quy tắc:
- Email không phân biệt hoa thường và không trùng.
- Mật khẩu tối thiểu 8 ký tự.
- Sai email hoặc mật khẩu: một thông báo chung, không nói sai phần nào.
- Đăng nhập sai 5 lần liên tiếp thì tạm khoá đăng nhập tài khoản đó 15 phút.
- Khi đăng ký, lưu múi giờ của trình duyệt vào tài khoản (dùng để tính ngày học ở các tính năng sau).

Ngoài phạm vi: quên mật khẩu, đổi mật khẩu, xác minh email, đăng nhập bằng Google.

Tiêu chí nghiệm thu: theo mục F1 trong docs/phases/giai-doan-1.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/auth, internal/storage/mongo)
- Collection users: _id, email (lưu chữ thường, unique index), passwordHash, role ("admin" | "learner"), timezone (IANA), createdAt.
- Băm mật khẩu argon2id (golang.org/x/crypto/argon2), tham số theo khuyến nghị OWASP.
- Phiên đăng nhập lưu phía server: collection sessions (tokenHash, userId, expiresAt, TTL index trên expiresAt). Token ngẫu nhiên 32 byte, chỉ lưu hash.
- Cookie: HttpOnly, SameSite=Strict, Secure khi chạy HTTPS, Max-Age 30 ngày.
- Tài khoản đầu tiên là admin: trong service, nếu số user = 0 thì role = admin (dự án một người, không cần xử lý tranh chấp).
- Khoá đăng nhập: đếm lần sai theo email trong bộ nhớ (map có mutex), 5 lần/15 phút.
- Endpoint: POST /api/auth/register, POST /api/auth/login, POST /api/auth/logout, GET /api/auth/me.
- Middleware trong platform/httpx: RequireAuth (gắn user vào context), RequireAdmin.
- Test: service (đăng ký, trùng email, user đầu là admin, khoá sau 5 lần sai) với repository giả; handler với httptest (thành công, lỗi đầu vào, sai quyền).

Frontend (features/auth, core/)
- Trang /login và /register (reactive forms, thông báo lỗi tiếng Việt).
- core/services/auth.service.ts: signal currentUser, gọi /api/auth/me khi khởi động app.
- core/guards: authGuard, adminGuard (dạng CanActivateFn).
- core/interceptors/auth.interceptor.ts: gửi cookie (withCredentials); error.interceptor: 401 → /login?returnUrl=..., 403 → thông báo không có quyền.
- Gửi Intl.DateTimeFormat().resolvedOptions().timeZone khi đăng ký.
- Test Vitest: guard, auth.service, form validation.
```
