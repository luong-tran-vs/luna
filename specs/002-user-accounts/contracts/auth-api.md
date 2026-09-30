# Contract: Auth API

Mọi phản hồi JSON có `Content-Type: application/json`. Lỗi có dạng chung:

```json
{ "error": "<code>", "message": "<tiếng Việt>" }
```

POST chỉ nhận `Content-Type: application/json` (khác → 415 `unsupported_media_type`), thân ≤ 16 KiB, không
có trường lạ (→ 400 `invalid_body`).

Cookie phiên: `luna_session=<token>; Path=/; HttpOnly; SameSite=Strict; Max-Age=2592000[; Secure]`.

Đối tượng `user`:

```json
{ "id": "66f9…", "email": "an@example.com", "role": "admin", "timezone": "Asia/Ho_Chi_Minh" }
```

## POST /api/auth/register

Request: `{ "email": "An@Example.com", "password": "matkhau123", "timezone": "Asia/Ho_Chi_Minh" }`
(`timezone` tuỳ chọn).

| Status | Body | Khi nào |
|---|---|---|
| 201 | `{ "user": {…} }` + Set-Cookie | tạo thành công, đã đăng nhập |
| 400 | `validation_failed` + `fields` | email sai định dạng, mật khẩu < 8 hoặc > 128 ký tự |
| 409 | `email_taken`, "Email này đã được dùng" | email đã có |

`fields` ví dụ: `{ "email": "Email không hợp lệ", "password": "Mật khẩu cần ít nhất 8 ký tự" }`.

## POST /api/auth/login

Request: `{ "email": "an@example.com", "password": "matkhau123" }`

| Status | Body | Khi nào |
|---|---|---|
| 200 | `{ "user": {…} }` + Set-Cookie | đúng |
| 400 | `validation_failed` | thiếu email hoặc mật khẩu |
| 401 | `invalid_credentials`, "Email hoặc mật khẩu không đúng" | sai email **hoặc** sai mật khẩu |
| 429 | `account_locked`, "Đăng nhập tạm khoá. Thử lại sau N phút", `retryAfterSeconds` + header `Retry-After` | lần sai thứ 5 liên tiếp (khoá bắt đầu), hoặc đang khoá |

## POST /api/auth/logout

| Status | Khi nào |
|---|---|
| 204 + Set-Cookie `Max-Age=0` | luôn luôn (có hoặc không có phiên) |

## GET /api/auth/me

| Status | Body | Khi nào |
|---|---|---|
| 200 | `{ "user": {…} }` | phiên hợp lệ |
| 401 | `unauthenticated`, "Bạn cần đăng nhập" | không có cookie, phiên sai, đã huỷ hoặc hết hạn |

## GET /api/admin/ping (tạm, thay ở F2)

| Status | Body | Khi nào |
|---|---|---|
| 200 | `{ "ok": true }` | quản trị viên |
| 401 | `unauthenticated` | chưa đăng nhập |
| 403 | `forbidden`, "Bạn không có quyền truy cập" | người học |

## GET /api/health

Không đổi so với F0, vẫn công khai.

## Test hợp đồng (backend `httptest`)

- register: 201 + cookie; 400 từng lỗi trường; 409 trùng email (khác hoa thường); 415 sai Content-Type.
- login: 200; 401 email lạ và 401 sai mật khẩu có body giống hệt; 429 sau 5 lần sai, kể cả khi đúng mật khẩu.
- logout: 204, sau đó `/me` với cookie cũ → 401.
- me: 200 đúng người; 401 không cookie; 401 phiên hết hạn.
- admin/ping: 200 admin, 403 learner, 401 ẩn danh.
