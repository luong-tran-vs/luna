# Contract: Biến môi trường mới (F1)

Bổ sung cho `specs/001-project-skeleton/contracts/config.md`.

| Biến | Bắt buộc | Mặc định | Ý nghĩa |
|---|---|---|---|
| `MONGO_DATABASE` | Không | `luna` | Tên database chứa `users`, `sessions` |
| `COOKIE_SECURE` | Không | `false` | `true` khi app chạy qua HTTPS; thêm cờ `Secure` cho cookie phiên |

`COOKIE_SECURE` chỉ nhận `true` hoặc `false` (không phân biệt hoa thường); giá trị khác → lỗi
`config: COOKIE_SECURE must be true or false`, backend dừng mã 1.

Cập nhật `deploy/.env.example`, `deploy/docker-compose.yml` (`${COOKIE_SECURE:-false}`,
`${MONGO_DATABASE:-luna}`) và bảng cấu hình trong `README.md`.
