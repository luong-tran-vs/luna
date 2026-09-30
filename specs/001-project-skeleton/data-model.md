# Data Model: Khung dự án Luna (F0)

F0 chưa lưu dữ liệu nghiệp vụ nào vào MongoDB. Các "thực thể" dưới đây là kiểu dữ liệu trao đổi và
trạng thái trong bộ nhớ.

## 1. HealthStatus (backend → frontend)

Kết quả kiểm tra tình trạng, tính mỗi lần gọi `GET /api/health`, không lưu.

| Trường | Kiểu | Giá trị | Ghi chú |
|---|---|---|---|
| `status` | string | `ok` \| `degraded` | `ok` khi mọi phụ thuộc hoạt động |
| `database` | string | `up` \| `down` | `down` khi ping lỗi hoặc quá 2 giây |

Quy tắc: `database = up` ⇔ `status = ok` ⇔ HTTP 200; `database = down` ⇔ `status = degraded` ⇔ HTTP 503.

Go: `health.Response { Status string \`json:"status"\`; Database string \`json:"database"\` }`.
TypeScript: `core/models/health.ts` → `interface HealthResponse { status: 'ok' | 'degraded'; database: 'up' | 'down' }`.

## 2. ConnectionStatus (frontend, trang chủ)

| Giá trị | Chữ hiển thị | Token màu biểu tượng | Biểu tượng |
|---|---|---|---|
| `checking` | Đang kiểm tra kết nối… | `--color-text-muted` | vòng xoay |
| `connected` | Đã kết nối | `--color-ok` | ✓ |
| `database-down` | Mất kết nối cơ sở dữ liệu | `--color-warn` | ! |
| `server-unreachable` | Không kết nối được máy chủ | `--color-bad` | ✕ |

Chuyển trạng thái: khi trang mở → `checking` → một trong ba trạng thái cuối (theo bảng ánh xạ ở
research R7). Tải lại trang thì bắt đầu lại từ `checking`.

## 3. ThemePreference (frontend, trình duyệt)

| Trường | Kiểu | Giá trị | Mặc định |
|---|---|---|---|
| preference | string | `light` \| `dark` \| `system` | `system` |
| resolved (tính ra) | string | `light` \| `dark` | theo `prefers-color-scheme` khi `system` |

- Lưu ở localStorage, khoá `luna.theme`. Đọc lỗi, không có, hoặc giá trị lạ → `system`.
- Ghi lỗi (trình duyệt chặn) → bỏ qua, lựa chọn vẫn có hiệu lực trong phiên.
- Sẽ chuyển sang lưu theo tài khoản ở F12.

## 4. Config (backend)

| Trường | Biến môi trường | Bắt buộc | Mặc định | Kiểm tra |
|---|---|---|---|---|
| MongoURI | `MONGO_URI` | Có | — | không rỗng |
| HTTPAddr | `HTTP_ADDR` | Không | `:8080` | — |
| LogLevel | `LOG_LEVEL` | Không | `info` | một trong `debug`, `info`, `warn`, `error` |

Mọi lỗi được gom lại và trả cùng lúc; backend in lỗi rồi thoát mã 1.
