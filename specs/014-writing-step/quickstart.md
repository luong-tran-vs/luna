# Quickstart: kiểm tra F8 (Bước Viết)

Chuẩn bị:

- `docker compose -f deploy/docker-compose.yml up --build -d`.
- Tài khoản `hoc@example.com` có mục tiêu đang học.
- Phần chấm thật cần `GEMINI_API_KEY` trong `deploy/.env`.

## Kiểm tra tự động

```bash
cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...
# golangci-lint chạy qua Docker
cd frontend && npx ng test --watch=false && npm run lint && npx ng build
```

## 1. Viết và nộp (US1)

1. Học bài hôm nay tới hết bước Nghe. Kiểm tra:
   - thanh tiến độ có 4 bước Ôn · Đọc · Nghe · Viết;
   - bước hiện tại là Viết.
2. Bước Viết hiện đề của bài và gợi ý độ dài theo trình độ. Bài chưa có đề thì hiện "Tóm tắt bài bằng 3–5 câu.".
3. Gõ khoảng 30 từ, đợi 1 giây. Kiểm tra:
   - hiện "Đã lưu nháp";
   - tải lại trang thì nháp vẫn còn;
   - mở trên trình duyệt khác cũng thấy nháp.
4. Xoá còn 4 từ: nút Nộp bị khoá và có gợi ý "ít nhất 5 từ".
5. Gõ đủ rồi bấm Nộp. Kiểm tra:
   - bài hôm nay "Đã xong";
   - mục tiêu +1, streak cập nhật;
   - hiện "Đã nộp, AI đang chấm". Bước này vẫn phải đúng khi AI tắt.
6. Gửi nộp lại bằng curl: phải nhận 409 `already_submitted`.

   ```bash
   curl -s -b hoc.txt -H 'Content-Type: application/json' \
     --data-binary '{"text":"one two three four five"}' \
     http://localhost:8000/api/lessons/<id>/writing/submit
   ```

## 2. Kết quả và thông báo (US2)

1. Khi AI bật: sau khi nộp, sang trang Sổ từ. Trong vòng 1 phút phải thấy:
   - header có "Bài viết" kèm dấu báo "1";
   - toast "Bài viết đã có kết quả" có nút Xem.
2. Bấm Xem. Kiểm tra:
   - có 4 tiêu chí với điểm "x/5" và nhận xét tiếng Việt;
   - có điểm trung bình, nhận xét chung, bản đã sửa;
   - phần so sánh dùng gạch chân cho chỗ thêm, gạch ngang cho chỗ bớt, có chú thích.
3. Quay lại: dấu báo biến mất.
4. Bật chế độ xám (DevTools › Rendering › Achromatopsia): vẫn phân biệt được thêm/bớt. Trình đọc màn hình đọc "thêm: …", "bớt: …".

## 3. Chấm lỗi (US3)

1. Đặt `AI_PROVIDER=none`, khởi động lại backend, rồi nộp bài hôm nay (dùng tài khoản khác hoặc ngày khác). Kiểm tra:
   - bước Viết vẫn xong;
   - trang bài viết hiện "Chấm lỗi: AI chưa được cấu hình" và nút **Chấm lại**;
   - toast báo lỗi chấm.
2. Bật lại AI, bấm Chấm lại: trạng thái chuyển "Đang chấm" rồi ra kết quả.
3. Chấm lại một bài đã chấm xong: phải nhận 409 `not_failed`.

## 4. Trang Bài viết và xem lại (US4)

1. Mở **Bài viết**: danh sách mới nhất trước, mỗi dòng có ngày, tên bài và điểm (hoặc trạng thái).
2. Tài khoản mới: hiện lời nhắc chưa có bài viết.
3. Vào **Bài học**, bấm **Bài viết** của một bài đã xong: thấy bài viết và kết quả, không có ô soạn hay nút Nộp.
4. Dùng cookie của tài khoản khác `GET /api/writings/<id>`: phải nhận 404.

## 5. Số liệu (US5)

1. Màn hình chính có thanh **Viết** cạnh Đọc và Nghe.
2. Thống kê hiện "n bài viết · điểm trung bình x,x" (chưa có bài chấm xong thì "—").
3. File xuất dữ liệu có `writings`.

## 6. 360px và bàn phím

- Ở 360 × 640:
  - ô soạn, đếm từ, nút Nộp, 4 thẻ nhận xét, phần so sánh và toast không cuộn ngang;
  - toast không che nút Nộp.
- Chỉ dùng bàn phím làm hết luồng: soạn, nộp, mở toast, Chấm lại.
