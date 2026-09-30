# Research: Tài khoản (F1)

Khối 2 trong `docs/spec-inputs/f1-tai-khoan.md` đã chốt phần lớn kỹ thuật. File này ghi các điểm khối 2
chưa nói rõ hoặc cần chốt chi tiết. Không còn mục NEEDS CLARIFICATION.

## R1. Băm mật khẩu argon2id

- **Decision**: `golang.org/x/crypto/argon2.IDKey` với tham số tối thiểu OWASP: m = 19 MiB (19456 KiB),
  t = 2, p = 1; salt 16 byte `crypto/rand`; khoá 32 byte. Lưu dạng PHC
  `$argon2id$v=19$m=19456,t=2,p=1$<salt b64>$<hash b64>` để đổi tham số sau này mà không vỡ hash cũ. So sánh
  bằng `subtle.ConstantTimeCompare`.
- **Rationale**: Đúng khối 2 và OWASP Password Storage Cheat Sheet; ~50–100 ms/lần trên máy thường, bộ nhớ
  19 MiB/lần chấp nhận được với 1 người dùng.
- **Alternatives considered**: bcrypt (giới hạn 72 byte, sẽ cắt mật khẩu dài — trái edge case "không bị
  cắt bớt"); scrypt (OWASP xếp sau argon2id).

## R2. Không lộ email tồn tại khi đăng nhập

- **Decision**: Email không tồn tại vẫn chạy `Verify` với một hash giả tạo sẵn lúc khởi động, để thời gian
  phản hồi tương đương. Cả hai trường hợp trả 401 `invalid_credentials`. Bộ đếm khoá tính theo email đã
  chuẩn hoá, áp dụng cả với email chưa đăng ký.
- **Rationale**: FR-009 (không chênh lệch thời gian), edge case tạm khoá.

## R3. Phiên đăng nhập

- **Decision**:
  - Token 32 byte `crypto/rand`, mã hoá base64url (không padding) trong cookie `luna_session`.
  - DB chỉ lưu `tokenHash = SHA-256(token)` (hex). Không cần muối/argon2 vì token đủ ngẫu nhiên 256 bit.
  - `expiresAt = now + 30 ngày`, cố định (không gia hạn). TTL index `expireAfterSeconds: 0` xoá bản ghi hết
    hạn; vì TTL monitor chạy mỗi ~60 giây nên `Authenticate` vẫn kiểm tra `expiresAt > now`.
  - Cookie: `HttpOnly`, `SameSite=Strict`, `Path=/`, `Max-Age=2592000`, `Secure` khi `COOKIE_SECURE=true`.
  - Đăng xuất: xoá bản ghi theo hash, trả cookie `Max-Age=0`. Idempotent (không có cookie vẫn 204).
- **Rationale**: Đúng khối 2; huỷ phiên phía server được ngay (FR-012), điều JWT không làm được.
- **`Secure` qua biến môi trường**: app chạy HTTP ở `localhost:8000` sau nginx; tự nhận HTTPS qua
  `X-Forwarded-Proto` phải tin proxy — biến `COOKIE_SECURE` (mặc định `false`) đơn giản và rõ ràng hơn.

## R4. Chống CSRF

- **Decision**: `SameSite=Strict` + endpoint POST chỉ nhận `Content-Type: application/json` (khác → 415).
  Không dùng token CSRF.
- **Rationale**: Form HTML từ site khác không gửi được JSON; SameSite=Strict chặn cookie đi kèm request
  từ site khác. Đủ cho app một origin.

## R5. Tài khoản đầu tiên là quản trị viên

- **Decision**: Trong `Service.Register`: `Count()` = 0 → `admin`, ngược lại `learner`. Không xử lý tranh
  chấp (khối 2 chấp nhận). Trùng email do đăng ký đồng thời được unique index chặn → `ErrEmailTaken`.

## R6. Khoá đăng nhập

- **Decision**: `Lockout` giữ `map[email]{failures int, lockedUntil time.Time}` bảo vệ bằng `sync.Mutex`,
  clock tiêm vào (`func() time.Time`) để test. Quy tắc:
  - `Check(email)`: nếu `now < lockedUntil` → trả thời gian còn lại.
  - `Fail(email)`: `failures++`; khi `failures >= 5` → `lockedUntil = now + 15m`, `failures = 0`, và lần sai
    thứ 5 trả luôn 429 để người dùng thấy thông báo tạm khoá ngay (spec US3 kịch bản 2).
  - `Success(email)`: xoá mục.
  - Mục đã hết khoá và không còn lần sai được dọn khi truy cập (map nhỏ, không cần goroutine dọn).
- **Phản hồi khi khoá**: 429 `account_locked`, header `Retry-After` (giây), body `retryAfterSeconds`.
  Đăng nhập khi đang khoá không kiểm tra mật khẩu và không tăng bộ đếm.

## R7. Kiểm tra đầu vào

- **Email**: `strings.TrimSpace` + `strings.ToLower`; `net/mail.ParseAddress` phải trả đúng chuỗi đó (loại
  "Tên <a@b>"), ≤ 254 ký tự, phần domain có dấu chấm.
- **Mật khẩu**: 8–128 **ký tự** (`utf8.RuneCountInString`), không trim.
- **Múi giờ**: `time.LoadLocation` thành công và không rỗng/`Local`; lỗi → `Asia/Ho_Chi_Minh` (không báo
  lỗi cho người dùng). Image distroless không có tzdata nên `main.go` import `_ "time/tzdata"`.
- **Thân request**: `http.MaxBytesReader` 16 KiB, `DisallowUnknownFields`.
- **Lỗi trả về**: 400 `{"error":"validation_failed","message":"…","fields":{"email":"…","password":"…"}}`,
  message tiếng Việt (FR-003).

## R8. Middleware quyền

- **Decision**: `platform/httpx` định nghĩa `Principal{UserID, Email, Role}` và
  `RequireAuth(resolve func(ctx, token string) (Principal, error))` đọc cookie `luna_session`, gắn
  Principal vào context; thiếu/không hợp lệ → 401 `unauthenticated`. `RequireAdmin` (đặt sau RequireAuth)
  → 403 `forbidden` nếu `Role != "admin"`. `auth.Service.Authenticate` được bọc thành hàm `resolve` trong
  `main.go`, nên `platform` không phụ thuộc `auth`.
- **Rationale**: Đúng khối 2 (middleware ở platform/httpx) mà không tạo phụ thuộc vòng.

## R9. Frontend: không cần `auth.interceptor.ts`

- **Decision**: Bỏ interceptor gắn `withCredentials` trong khối 2.
- **Rationale**: Frontend gọi `/api/...` cùng origin (dev proxy và nginx), trình duyệt tự gửi cookie
  `SameSite=Strict`. `withCredentials` chỉ cần cho request khác origin. Nguyên tắc VII.

## R10. Frontend: trạng thái đăng nhập và điều hướng

- `AuthService`: signal `currentUser: User | null`, `isAdmin = computed(...)`; `load()` gọi
  `/api/auth/me` qua `provideAppInitializer` (401 hoặc lỗi mạng → `null`, app vẫn khởi động).
  `login`, `register` cập nhật signal; `logout` gọi API rồi đặt `null` (kể cả khi API lỗi) và về `/login`.
- Guard (`CanActivateFn`):
  - `authGuard`: chưa đăng nhập → `UrlTree` `/login?returnUrl=<url>`.
  - `guestGuard` (cho `/login`, `/register`): đã đăng nhập → `/`.
  - `adminGuard`: không phải admin → `/forbidden`.
- `errorInterceptor` (mở rộng F0): 401 từ endpoint khác `/api/auth/login|register|me` → `AuthService.clear()`
  + điều hướng `/login?returnUrl=<url hiện tại>`; 403 → `/forbidden`. Vẫn ném `ApiError` như F0.
- `safeReturnUrl(url)`: chỉ nhận chuỗi bắt đầu `/`, không bắt đầu `//` hay `/\`, không phải `/login`,
  `/register`; ngược lại trả `/`.
- Form: Reactive forms, `Validators.required/email/minLength(8)/maxLength(128)`; lỗi hiện dưới ô với
  `aria-describedby` và `aria-invalid`; lỗi máy chủ hiện trong vùng `role="alert"`; nút gửi khoá khi đang
  gửi. Đăng ký gửi `Intl.DateTimeFormat().resolvedOptions().timeZone`.

## R11. Header ở 360px

> Sửa 2026-09-30: dòng 1 là "Luna" + nút giao diện (menu Sáng/Tối) + "Đăng xuất"; dòng 2 là email và các liên kết, cuộn ngang khi không đủ chỗ.

- **Decision**: Header dùng `flex-wrap`: dòng 1 là "Luna" + nhóm chọn giao diện; khi đã đăng nhập thêm dòng
  2 gồm email (cắt `…` nếu dài, có `title`), liên kết "Quản trị" (chỉ admin) và nút "Đăng xuất". Trên màn ≥
  640px tất cả nằm một dòng.
- **Rationale**: 360px không đủ cho brand + 3 nút 44px + email + 2 nút; hai dòng giữ vùng bấm 44px.

## R12. Tạo index khi Mongo có thể đang tắt

- **Decision**: `EnsureIndexes` chạy trong goroutine nền lúc khởi động, thử lại mỗi 10 giây tới khi thành
  công hoặc context bị huỷ; log `info` khi xong, `warn` mỗi lần lỗi. Index: `users.email` unique,
  `sessions.tokenHash` unique, `sessions.expiresAt` TTL 0.
- **Rationale**: Giữ quyết định F0 R3 (backend chạy khi Mongo tắt); tạo index là idempotent.

## R13. Endpoint admin tạm

- **Decision**: `GET /api/admin/ping` → 200 `{"ok":true}` qua `RequireAuth` + `RequireAdmin`. Trang `/admin`
  gọi endpoint này và hiện "Trang quản trị (đang xây dựng)". F2 thay bằng API thật.
- **Rationale**: FR-019/020 cần một dịch vụ quản trị để kiểm tra chặn ở API.
