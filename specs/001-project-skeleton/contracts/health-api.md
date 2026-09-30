# Contract: GET /api/health

Kiểm tra tình trạng máy chủ và kết nối cơ sở dữ liệu. Không cần xác thực, không trả dữ liệu người dùng
hay chi tiết lỗi nội bộ.

## Request

```http
GET /api/health HTTP/1.1
X-Request-ID: <tuỳ chọn, ≤ 64 ký tự [A-Za-z0-9-]>
```

## Responses

Mọi phản hồi có `Content-Type: application/json` và header `X-Request-ID` (lấy từ request hoặc sinh mới).

### 200 OK: máy chủ và cơ sở dữ liệu hoạt động

```json
{ "status": "ok", "database": "up" }
```

### 503 Service Unavailable: cơ sở dữ liệu không phản hồi trong 2 giây hoặc ping lỗi

```json
{ "status": "degraded", "database": "down" }
```

### 405 Method Not Allowed: phương thức khác GET

Do `http.ServeMux` trả mặc định (kèm header `Allow: GET, HEAD`).

### 500 Internal Server Error: panic được middleware Recover bắt

```json
{ "error": "internal_error" }
```

## Phía frontend hiểu phản hồi

| Phản hồi | Trạng thái giao diện |
|---|---|
| 200 + `database: "up"` | Đã kết nối |
| 503 + `database: "down"` | Mất kết nối cơ sở dữ liệu |
| Lỗi mạng, quá 5 giây, 502/504, body sai hợp đồng, mã khác | Không kết nối được máy chủ |

## Test hợp đồng

- Backend `internal/health/handler_test.go`: Pinger giả trả nil → 200 + body trên; trả lỗi → 503 + body
  trên; Pinger chậm hơn 2 giây → 503 (context hết hạn).
- Frontend `features/home` spec: giả lập ba trường hợp 200, 503, lỗi mạng (và 502) → đúng chữ hiển thị.
