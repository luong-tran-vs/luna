# Đầu vào cho F16: Vận hành

Nguồn: [giai-doan-3.md › F16](../phases/giai-doan-3.md). Làm trước F10, vì micro trên điện thoại cần HTTPS.

## Khối 1: `/speckit-specify`

```text
F16 – Vận hành: đặt lại mật khẩu và truy cập từ điện thoại (Giai đoạn 3) cho Luna.

Mục tiêu: không bị khoá ngoài app khi quên mật khẩu, và dùng app trên điện thoại ở bất cứ đâu một cách an toàn, qua HTTPS.

Kịch bản (người vận hành):
- Quên mật khẩu: chạy một lệnh với email → nhận một mật khẩu tạm in ra một lần; mọi phiên đăng nhập cũ của tài khoản đó bị huỷ và khoá đăng nhập (nếu có) được gỡ. Đăng nhập bằng mật khẩu tạm.
- Liệt kê tài khoản: một lệnh in email, vai trò, ngày tạo.
- Dùng từ điện thoại: cài Tailscale trên máy tính chạy app và trên điện thoại (cùng tài khoản Tailscale), bật chia sẻ HTTPS → điện thoại mở app qua địa chỉ https riêng, chứng chỉ hợp lệ, đăng nhập và học như trên máy tính, kể cả khi không ở nhà.

Quy tắc:
- Chỉ thiết bị trong mạng Tailscale của người dùng truy cập được; không mở app ra internet công khai.
- Qua HTTPS, cookie phiên có cờ Secure; trên máy tính vẫn mở được bằng http://localhost.
- Trình duyệt điện thoại phải cho phép dùng micro trên địa chỉ HTTPS này (chuẩn bị cho F10).
- Lệnh vận hành không cần mạng ngoài và không làm gián đoạn app đang chạy.
- Mật khẩu tạm đủ mạnh (ít nhất 12 ký tự ngẫu nhiên), không ghi vào log.
- README có hướng dẫn từng bước bằng tiếng Việt.

Ngoài phạm vi: quên mật khẩu qua email, đổi mật khẩu trong giao diện, tên miền riêng, mở app công khai.

Tiêu chí nghiệm thu: theo mục F16 trong docs/phases/giai-doan-3.md.
```

## Khối 2: `/speckit-plan`

```text
Backend
- Binary thứ hai backend/cmd/luna-admin (cùng module, dùng lại internal/auth và internal/storage/mongo): lệnh `reset-password --email <email>` và `list-users`. Đọc MONGO_URI như API. In mật khẩu tạm ra stdout, không ghi log.
- internal/auth: thêm ResetPassword(ctx, email) → (tempPassword, error): sinh 16 ký tự ngẫu nhiên (crypto/rand), băm argon2id, xoá sessions của user, xoá khoá đăng nhập. Khoá đăng nhập đang nằm trong bộ nhớ của process API nên ghi thêm trường lockClearedAt trên user để API bỏ qua các lần sai trước thời điểm đó.
- Dockerfile backend build cả hai binary; chạy: `docker compose -f deploy/docker-compose.yml exec backend /luna-admin reset-password --email ...`.
- Test: ResetPassword (email không tồn tại → lỗi, phiên bị huỷ, mật khẩu mới đăng nhập được, khoá bị gỡ), CLI parse tham số.

Deploy
- Tailscale chạy trên máy host (Windows), không thêm container: `tailscale serve --bg http://localhost:8000` → https://<máy>.<tailnet>.ts.net.
- deploy/.env.example: COOKIE_SECURE=true (Secure vẫn hoạt động trên http://localhost vì trình duyệt coi localhost là ngữ cảnh an toàn).
- nginx: thêm header bảo mật cơ bản (Strict-Transport-Security chỉ khi qua HTTPS bằng X-Forwarded-Proto, X-Content-Type-Options, Referrer-Policy, Permissions-Policy microphone=(self)).
- README: mục "Dùng trên điện thoại" (cài Tailscale, bật HTTPS cho tailnet trong trang quản trị Tailscale, `tailscale serve`, kiểm tra) và mục "Quên mật khẩu".
- Kiểm tra thủ công: điện thoại qua 4G mở được, thiết bị ngoài tailnet không mở được, `navigator.mediaDevices.getUserMedia` được phép trên địa chỉ ts.net.
```
