# Contract: Biến môi trường

## Backend

| Biến | Bắt buộc | Mặc định | Ví dụ | Ý nghĩa |
|---|---|---|---|---|
| `MONGO_URI` | Có | — | `mongodb://localhost:27017` | Chuỗi kết nối MongoDB |
| `HTTP_ADDR` | Không | `:8080` | `:8080` | Địa chỉ lắng nghe |
| `LOG_LEVEL` | Không | `info` | `debug` | `debug` \| `info` \| `warn` \| `error` |

Thiếu hoặc sai → backend ghi ra stderr một dòng cho mỗi lỗi, nêu tên biến (không in giá trị), rồi thoát
mã 1. Ví dụ:

```text
config: MONGO_URI is required
config: LOG_LEVEL must be one of debug, info, warn, error
```

## Docker Compose (`deploy/.env`, tuỳ chọn)

| Biến | Mặc định trong compose | Ý nghĩa |
|---|---|---|
| `MONGO_URI` | `mongodb://mongo:27017` | Truyền vào container backend |
| `LOG_LEVEL` | `info` | Truyền vào container backend |
| `WEB_PORT` | `8000` | Cổng trên máy để mở app |

`deploy/.env.example` liệt kê đủ các biến trên. `deploy/.env` và mọi `.env` nằm trong `.gitignore`.

## Frontend

Không có biến môi trường lúc chạy. API luôn gọi đường dẫn tương đối `/api/...`; `ng serve` chuyển tiếp
qua `proxy.conf.json`, bản build trong Docker do nginx chuyển tiếp.

## Chạy local: `backend/.env` (bổ sung 2026-09-29)

Khi khởi động, backend đọc `.env` trong thư mục làm việc (chạy `go run ./cmd/api` từ `backend/`) nếu file tồn
tại. Biến đã có trong môi trường không bị ghi đè. File mẫu: `backend/.env.example`; `backend/.env` bị git bỏ qua
và không được đưa vào image Docker (`.dockerignore`). Dòng sai định dạng → backend báo lỗi nêu số dòng và dừng.
