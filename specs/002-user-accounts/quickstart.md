# Quickstart: kiểm tra F1

Chạy app: `docker compose -f deploy/docker-compose.yml up --build -d`, mở <http://localhost:8000>.
Bắt đầu từ database trống: `docker compose -f deploy/docker-compose.yml down -v` trước khi `up`.

## 1. Đăng ký, tài khoản đầu là quản trị viên

1. Mở `/` → được đưa tới `/login`; bấm "Đăng ký".
2. Nhập `Admin@Example.com` / `matkhau123` → vào trang chủ; header có email `admin@example.com`, "Quản trị",
   "Đăng xuất".
3. Kiểm tra DB:

   ```bash
   docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval 'db.users.find({}, {email:1, role:1, timezone:1, passwordHash:1}).toArray()'
   ```

   **Mong đợi**: `role: "admin"`, `timezone` = múi giờ trình duyệt, `passwordHash` bắt đầu `$argon2id$`.
4. Đăng xuất, đăng ký `hoc@example.com` → không có liên kết "Quản trị"; DB `role: "learner"`.
5. Đăng ký lại `HOC@example.com` → "Email này đã được dùng". Mật khẩu 7 ký tự → lỗi dưới ô.

## 2. Giữ đăng nhập, đăng xuất

1. Đăng nhập, đóng hẳn trình duyệt, mở lại `/` → vẫn đăng nhập. DevTools › Application › Cookies:
   `luna_session` có HttpOnly, SameSite Strict, hết hạn sau 30 ngày.
2. Mở 2 tab, đăng xuất ở tab 1, thao tác ở tab 2 (tải lại) → về `/login`.

## 3. Hết phiên giữa chừng, quay lại đúng trang

1. Đăng nhập admin, mở `/admin`.
2. Xoá phiên ở DB: `... mongosh luna --quiet --eval 'db.sessions.deleteMany({})'`.
3. Tải lại `/admin` → `/login?returnUrl=%2Fadmin`; đăng nhập → quay về `/admin`.
4. Thử `/login?returnUrl=https://evil.example` → đăng nhập xong về `/`.

## 4. Lỗi chung và tạm khoá

```bash
for i in 1 2 3 4 5 6; do curl -s -w " %{http_code}\n" -H 'Content-Type: application/json' \
  -d '{"email":"hoc@example.com","password":"sai-mat-khau"}' http://localhost:8000/api/auth/login; done
curl -s -w " %{http_code}\n" -H 'Content-Type: application/json' \
  -d '{"email":"khong-co@example.com","password":"sai-mat-khau"}' http://localhost:8000/api/auth/login
```

**Mong đợi**: lần 1–4 → 401 cùng body; lần 5 và 6 → 429 `account_locked`; email lạ → 401 với body giống
hệt lần 1. Đăng nhập đúng mật khẩu trên giao diện → thông báo tạm khoá kèm số phút.
Khởi động lại backend (`restart backend`) để xoá khoá khi cần thử tiếp.

## 5. Chặn người học ở giao diện và API

1. Đăng nhập `hoc@example.com`, gõ `/admin` → trang "Bạn không có quyền truy cập trang này".
2. Lấy cookie `luna_session` của người học trong DevTools, rồi:

   ```bash
   curl -s -w " %{http_code}\n" -b "luna_session=<token>" http://localhost:8000/api/admin/ping   # 403
   curl -s -w " %{http_code}\n" http://localhost:8000/api/admin/ping                             # 401
   ```

## 6. Không lộ dữ liệu

- `curl -b "luna_session=<token người học>" http://localhost:8000/api/auth/me` → chỉ email người học, không
  có `passwordHash`.
- `docker compose ... logs backend | grep -i -E "matkhau|password\"|luna_session="` → không có kết quả.

## 7. 360px, sáng và tối

DevTools 360px: trang `/login`, `/register`, `/`, `/admin`, `/forbidden` không cuộn ngang; header xuống 2
dòng khi đăng nhập; lỗi form đọc được ở cả hai chế độ.
