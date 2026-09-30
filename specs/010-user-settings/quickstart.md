# Quickstart: kiểm tra F12

Chuẩn bị: `docker compose -f deploy/docker-compose.yml up --build -d`, mở http://localhost:8000, đăng nhập `hoc@example.com`.

Kiểm tra tự động: `cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`; `cd frontend && npx ng test --watch=false &&
npm run lint && npx ng build`.

## 1. Giao diện (US1)

1. Thanh trên → **Cài đặt** → nhóm Giao diện → chọn **Tối**: cả trang đổi sang tối ngay, không tải lại; "Đã lưu".
2. Nhóm Giao diện chỉ có **Sáng** và **Tối**. Tài khoản chưa chọn lần nào: đổi chế độ của hệ điều hành (DevTools → Rendering → `prefers-color-scheme`) → app đổi theo.
3. Chọn **Tối**. Mở cửa sổ ẩn danh (localStorage trống, chưa chọn nên theo thiết bị, thiết bị đang sáng) → đăng nhập cùng tài khoản →
   app chuyển tối sau khi đăng nhập. Đăng xuất → trang đăng nhập vẫn tối (lựa chọn cuối trên trình duyệt đó).
4. Dùng nút giao diện nhanh ở thanh trên chọn **Sáng** → mở Cài đặt: lựa chọn là Sáng; `GET /api/settings` trả `"theme":"light"`.
5. Xem trang chủ, bài đọc, sổ từ ở hai chế độ: màu Oải hương và font Lexend đúng `docs/design-system.md`.

## 2. Giới hạn thẻ (US2)

1. Có ≥ 25 thẻ đến hạn (`db.cards.updateMany({}, {$set: {due: new Date(Date.now() - 60000)}})`, xem README), chưa ôn hôm nay.
2. Cài đặt → Số thẻ ôn mỗi ngày = 10 → **Lưu** → mở Bài hôm nay: bước Ôn có 10 thẻ.
3. Nhập 4, 201, để trống → **Lưu**: lỗi "Số thẻ từ 5 đến 200" dưới ô, `GET /api/settings` vẫn là 10.
4. Ôn hết 10 thẻ; đặt 5 → Bài hôm nay: bước Ôn đã đủ. Đặt 15 → mở lại: còn 5 thẻ.

## 3. Múi giờ (US3)

1. Cài đặt → Múi giờ đang chọn là múi giờ lúc đăng ký. Gõ "london" → chọn Europe/London (GMT+1) → **Lưu**.
2. Đang học dở bước Đọc (cùng ngày theo London): mở Bài hôm nay vẫn đúng bài, Ôn ✓, Đọc mở ở câu đang đọc.
3. Lùi ngày: sau khi xong bài hôm nay (giờ Việt Nam, sau 0 giờ), đổi sang `America/Los_Angeles` (vẫn là hôm qua ở đó) → màn hình chính
   "Đã xong bài hôm nay", không có bài mới, streak không đổi.
4. `curl` PUT `{"timezone":"Mars/Olympus"}` → 400 "Múi giờ không hợp lệ".

## 4. API bằng curl

```bash
curl -s -b hoc.txt http://localhost:8000/api/settings
printf '{"dailyReviewLimit":20,"timezone":"Europe/London"}' > s.json
curl -s -b hoc.txt -X PUT -H "Content-Type: application/json" --data-binary @s.json http://localhost:8000/api/settings
printf '{"dailyReviewLimit":201}' > bad.json
curl -s -b hoc.txt -X PUT -H "Content-Type: application/json" --data-binary @bad.json http://localhost:8000/api/settings  # 400
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8000/api/settings   # 401
curl -s -b hoc.txt http://localhost:8000/api/auth/me     # user.timezone = Europe/London
```

## 5. Migration

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
  printjson(db.users.find({}, {email: 1, timezone: 1, settings: 1}).toArray());
  printjson(db.migrations.find().toArray())'
```

Không user nào còn trường `timezone` gốc; mọi user có `settings.timezone` đúng múi giờ cũ.
