# Quickstart: kiểm tra L

Chuẩn bị: `docker compose -f deploy/docker-compose.yml up --build -d`; quản trị viên tạo chủ đề "A1 · Gia đình" với lộ trình ≥ 3
bài có audio và chú thích xong, chủ đề "A1 · Mua sắm" ≥ 1 bài; tài khoản người học `hoc@example.com` có vài thẻ trong sổ.

Đổi ngày để thử (không cần chờ tới mai): đổi múi giờ tài khoản sang múi giờ đã qua nửa đêm, hoặc lùi `dayKey` trong DB:

```bash
docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval '
  const u = db.users.findOne({email: "hoc@example.com"})._id;
  db.study_days.updateMany({userId: u}, [{$set: {dayKey: "2026-09-29"}}]);
  db.lesson_progress.updateMany({userId: u}, [{$set: {dayKey: "2026-09-29"}}])'
```

## 1. Đặt mục tiêu (US1)

Trang chủ → **Học hôm nay** → chọn A1 → danh sách chủ đề A1 kèm số bài → chọn "Gia đình" → bài hôm nay là bài 1.

## 2. Học bài hôm nay (US2)

1. Thanh bước Ôn ● · Đọc 🔒 · Nghe 🔒; phiên ôn gồm thẻ đến hạn (≤ 30). Đặt thẻ đến hạn:
   `db.cards.updateMany({}, {$set: {due: new Date(Date.now() - 60000)}})`.
2. Ôn xong → Đọc mở. Gõ `/lessons/<id bài 2>/read` → "Bài này sẽ mở khi tới lượt".
3. Đọc tới giữa bài, tải lại trang → đúng vị trí. "Đã đọc xong" → Nghe.
4. Nghe tới câu 3, đóng tab, mở lại → Nghe câu 3. Kiểm tra hết → "Đã xong bài hôm nay", streak 1, mục tiêu 1/N.
5. Không có thẻ đến hạn (ngày đầu): bước Ôn tự xong, vào thẳng Đọc.

## 3. Một bài mỗi ngày và streak (US3)

1. Cùng ngày mở lại → "Đã xong bài hôm nay"; ôn tự do vẫn vào được.
2. Lùi ngày (lệnh trên) → bài 2 là bài hôm nay; streak hiện 1 (hôm qua xong). Lùi thêm 2 ngày → vẫn một bài, streak 0.
3. Lộ trình hết bài → "Chưa có bài mới" + nút ôn tự do + chọn chủ đề khác.
4. 45 thẻ đến hạn → bước Ôn 30 thẻ; hôm sau 15 thẻ còn lại.

## 4. Đổi chủ đề (US4)

1. Trước khi làm bước nào: đổi sang "A1 · Mua sắm" → bài hôm nay đổi ngay.
2. Sau khi xong bước Ôn: đổi chủ đề → "Chủ đề mới bắt đầu từ ngày mai"; bài hôm nay giữ nguyên.
3. Quay lại "Gia đình" → học tiếp bài chưa xong; streak không đổi.

## 5. Trang Bài học (US5)

Header → **Bài học**: Hôm nay, Đã học (mở lại đọc/nghe — tiến độ, streak không đổi), Sắp tới (🔒, không mở được).

## 6. 360px, bàn phím, sáng và tối

Chọn mục tiêu, bài hôm nay (cả ba bước), trang Bài học: không cuộn ngang; thanh bước phân biệt được không cần màu.
