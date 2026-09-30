# Quickstart: kiểm tra F6

Chuẩn bị: `docker compose -f deploy/docker-compose.yml up --build -d`, mở http://localhost:8000. Dùng dữ liệu của L: chủ đề "A1 · Gia
đình" có lộ trình ≥ 3 bài (audio + chú thích xong); `hoc@example.com` có vài thẻ trong sổ. Đổi ngày để thử như quickstart của L
(lùi `dayKey` trong `study_days` và `lesson_progress`).

Kiểm tra tự động: `cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...`; `cd frontend && npx ng test --watch=false &&
npm run lint && npx ng build`.

## 1. Màn hình chính khi đang học (US1)

1. DevTools → thiết bị 360 × 640. Đăng nhập `hoc@example.com` (mục tiêu A1 · Gia đình, bài hôm nay chưa bắt đầu).
2. Không cuộn, thấy: "🔥 N ngày", thanh mục tiêu "A1 · Gia đình" + "x/y bài", thanh Đọc và Nghe, tên bài + "A1 · Gia đình", thanh bước
   (Ôn hiện tại, Đọc/Nghe khoá) và nút **Bắt đầu: Ôn** (hoặc **Bắt đầu: Đọc** nếu không có thẻ đến hạn).
3. Bấm nút → `/today` đúng bước. Ôn xong, quay về `/` (link header hoặc Back): nút thành **Tiếp tục: Đọc**, thanh bước Ôn ✓.
4. Đọc tới giữa bài, quay về, bấm **Tiếp tục: Đọc** → mở lại đúng câu đang đọc.
5. Lặp lại bước 2 ở chế độ tối (F12 chưa có: DevTools → Rendering → `prefers-color-scheme: dark`).

## 2. Số liệu cập nhật ngay (US3)

1. Ghi lại "Đọc a/y", "Nghe b/y", "x/y bài", streak.
2. Hoàn thành Đọc → về `/`: Đọc a+1, nút "Tiếp tục: Nghe", không cần F5.
3. Hoàn thành Nghe → về `/`: Nghe b+1, mục tiêu x+1, streak +1, trạng thái "Đã xong bài hôm nay, hẹn bạn ngày mai".
4. Chỉ có Đọc và Nghe, không có Viết/Nói.

## 3. Trạng thái đặc biệt (US2)

- Tài khoản mới (đăng ký tài khoản mới): lời mời + **Chọn chủ đề**, không có thanh mục tiêu/bài hôm nay.
- Xong bài hôm nay: thông báo, thanh bước đủ ✓, **Ôn tự do** (→ `/vocabulary/review`), "Ngày mai: N thẻ cần ôn".
- Hết lộ trình: đổi mục tiêu sang chủ đề mà mọi bài đã xong (hoặc quản trị viên bỏ bớt bài khỏi lộ trình) → "Chưa có bài mới" +
  **Ôn tự do** + **Chọn chủ đề khác**.
- Lỗi kết nối: `docker compose -f deploy/docker-compose.yml stop mongo`, tải lại → chân trang báo lỗi cơ sở dữ liệu;
  `docker compose -f deploy/docker-compose.yml start mongo`, điều hướng sang trang khác rồi về → dòng lỗi biến mất. Khi bình thường
  không có dòng trạng thái nào.

## 4. Thẻ ngày mai

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
  const u = db.users.findOne({email: "hoc@example.com"})._id;
  const tz = db.users.findOne({_id: u}).timezone || "Asia/Ho_Chi_Minh";
  print(tz, db.cards.countDocuments({userId: u, due: {$exists: true}}))'
```

Đặt hạn một thẻ vào 23:00 ngày mai và một thẻ vào 01:00 ngày kia (giờ địa phương) bằng `updateOne({_id}, {$set: {due: ISODate(...)}})`
→ `tomorrowCards` tăng 1 (chỉ thẻ thứ nhất).

## 5. Thống kê (US4)

1. Màn hình chính → **Xem thống kê** → `/stats`.
2. So với DB:

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
  const u = db.users.findOne({email: "hoc@example.com"})._id;
  printjson({
    cards: db.cards.countDocuments({userId: u}),
    dictation: db.dictation_results.aggregate([{$match: {userId: u}},
      {$group: {_id: null, n: {$sum: 1}, c: {$sum: "$correctWords"}, t: {$sum: "$totalWords"}}}]).toArray(),
    read: db.lesson_progress.countDocuments({userId: u, "steps.read": true}),
    listen: db.lesson_progress.countDocuments({userId: u, "steps.listen": true}),
    completed: db.lesson_progress.countDocuments({userId: u, completedAt: {$exists: true}})
  })'
```

3. Tài khoản mới: các số 0, tỷ lệ "—".

## 6. API bằng curl

```bash
curl -s -b hoc.txt http://localhost:8000/api/dashboard
curl -s -b hoc.txt http://localhost:8000/api/stats
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8000/api/dashboard   # 401
```

Gọi `/api/dashboard` hai lần khi bài hôm nay chưa bắt đầu và không có thẻ đến hạn: `study_days` của hôm nay vẫn không có (dashboard chỉ
đọc), `action` là `{"kind":"start","step":"read"}`.

## 7. Index

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
  const u = db.users.findOne({email: "hoc@example.com"})._id;
  printjson(db.cards.find({userId: u, due: {$lt: new Date()}}).explain().queryPlanner.winningPlan)'
```

Kế hoạch dùng `IXSCAN` trên `userId_1_due_1`.
