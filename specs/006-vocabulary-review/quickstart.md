# Quickstart: kiểm tra F5

Chuẩn bị: `docker compose -f deploy/docker-compose.yml up --build -d`; tài khoản người học (`hoc@example.com`), một bài đã
có chú thích "Xong" (ví dụ "Reading test": went → go, gave up → give up).

## 1. Mục Từ vựng của bài (US3)

1. Mở bước Đọc của bài → mở mục **Từ vựng**: thấy go, give up… với IPA, nghĩa, nút nghe, nút Lưu; từ đã có trong sổ hiện ✓.
2. Bấm Lưu một từ → ✓ ngay. Bấm **Lưu tất cả** → "Đã lưu N từ"; bấm lại → không thêm thẻ nào.
3. Tra và lưu một từ bằng popup (F3) → mục Từ vựng cập nhật ✓.
4. Mở bài mới tạo, chú thích chưa xong → "Chưa có danh sách từ vựng".

## 2. Sổ từ (US2)

1. Header → **Sổ từ**: thẻ nhóm "Hôm nay", "Hôm qua", ngày cụ thể; mỗi thẻ có từ, IPA, nghĩa, câu ví dụ, nút nghe.
2. Tìm "giv" → chỉ "give up". Lọc theo bài; lọc "Thẻ tự thêm".
3. **Thêm thẻ** "serendipity" / "sự tình cờ may mắn" → vào nhóm Hôm nay; thêm "Serendipity" lần nữa → "Từ này đã có trong sổ".
4. Sửa nghĩa một thẻ đã ôn → nghĩa mới hiện; `due`/`reps` không đổi:
   `docker compose -f deploy/docker-compose.yml exec mongo mongosh luna --quiet --eval 'db.cards.find({}, {text:1, due:1, reps:1, state:1}).toArray()'`
5. Xoá một thẻ đã ôn → thẻ mất; `db.review_logs.countDocuments({cardId: ObjectId("…")})` = 0.

## 3. Ôn tập (US1)

1. Thẻ lưu hôm nay chưa đến hạn. Để thử, đặt một thẻ đến hạn:
   `db.cards.updateOne({text: "went"}, {$set: {due: new Date(Date.now() - 60000)}})`.
2. **Ôn tập** → chọn **Xem từ đoán nghĩa** → chạm lật → 4 nút có khoảng cách ("1 phút", "5 phút", "10 phút", "N ngày") → chọn
   Good → thẻ tiếp theo. Dùng bàn phím: Space lật, phím 3 = Good.
3. Chọn Again cho một thẻ → thẻ hiện lại cuối phiên. Kết thúc → "Đã ôn N thẻ" + số lần từng mức.
4. Kiểu **Nghe rồi gõ**: audio tự phát; gõ "WENT" → đúng; gõ "want" → sai, hiện từ đúng.
5. Không còn thẻ đến hạn → "Không có thẻ nào đến hạn" + thời điểm thẻ sớm nhất.
6. DevTools › Network › Offline, chọn đánh giá → báo lỗi, bấm Thử lại khi có mạng → không bị tính hai lần
   (`db.review_logs.countDocuments({cardId: …})` tăng đúng 1).

## 4. Múi giờ

Đổi `timezone` của tài khoản (`db.users.updateOne({email: "hoc@example.com"}, {$set: {timezone: "America/New_York"}})`),
tải lại sổ từ → nhóm ngày theo giờ New York; thẻ mới lưu có `due` = 0 giờ hôm sau giờ New York. Đặt lại `Asia/Ho_Chi_Minh`.

## 5. Điện thoại 360px, sáng và tối

Sổ từ, form thêm/sửa, phiên ôn (hai kiểu, bàn phím ảo mở khi gõ) không cuộn ngang; nút đánh giá ≥ 44px; đúng/sai không chỉ bằng
màu.
