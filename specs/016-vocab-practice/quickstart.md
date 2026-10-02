# Quickstart: kiểm tra F17 (Trang chi tiết bài và luyện tập từ vựng)

## Chuẩn bị

- Chạy `docker compose -f deploy/docker-compose.yml up --build -d`.
- Đăng nhập quản trị viên và người học `hoc@example.com`.
- Để sinh phần luyện tập thật, ghi `GEMINI_API_KEY` vào `deploy/.env`; TTS (Kokoro) chạy trong compose.

## Kiểm tra tự động

```bash
cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./... && make lint
cd frontend && npx ng test --watch=false && npm run lint && npx ng build
```

## 1. Sinh phần luyện tập (US2)

1. Mở log: `docker compose logs -f backend | grep '"op":"practice"'`.
2. Quản trị viên tạo một bài mới trong lộ trình chủ đề của người học. Đợi chú thích xong: mục "Phần luyện tập" ở trang quản trị chuyển
   "Đang sinh" → "Xong" sau ít phút; log có đúng 1 dòng `op=practice`.
3. Xem nội dung ở trang quản trị: mỗi câu ví dụ chứa đúng từ của nó; hội thoại 4–10 lượt; 3–5 câu dịch, mỗi câu ≤ 4 từ nhiễu.
4. Kiểm tra file audio: `ls data/audio/<id>/<rev>/practice/<ver>/` có `example-*`, `turn-*`, `answer-*`.
5. Sửa nội dung bài: phần luyện tập biến mất, sau khi chú thích xong thì được sinh lại theo nội dung mới (revision mới).
6. Bấm **Tạo lại phần luyện tập**: trạng thái "Đang sinh", bấm lần nữa bị khoá; xong thì nội dung mới, log thêm đúng 1 dòng.

## 2. AI lỗi (US2)

1. Đặt `AI_PROVIDER=none`, khởi động lại backend, bấm Tạo lại: trạng thái "Lỗi" kèm lý do; chú thích, câu hỏi hiểu bài, đề viết không đổi.
2. Với bài chưa có phần luyện tập, mở `/lessons/<id>` bằng tài khoản người học: bước 1 có từ vựng (không có "Ví dụ"), bước 2–4 báo
   "Bài này chưa có phần luyện tập"; luồng Ôn → Đọc → Nghe → Viết ở `/today` vẫn học được.

## 3. Trang chi tiết bài (US1)

1. Từ trang Bài học, chạm bài hôm nay: thấy nút đóng, tiến độ "1/4", streak, "Bài N", tiêu đề, "Mục tiêu: …", "Mức độ: …".
2. Bước 1: bấm loa nghe từ, bấm câu ví dụ nghe câu; thu gọn rồi mở lại danh sách.
3. **Tiếp theo** → "2/4". Phát cả đoạn ở 0.75×, đổi sang 1.25× giữa chừng, dừng; nghe một lượt riêng; tắt rồi bật nghĩa tiếng Việt.
4. Tab **Bài đọc**: thấy bài đọc gốc và ghi chú ngữ pháp; quay lại tab Bài học vẫn ở bước 2.
5. Mở thẳng `/lessons/<id bài sắp tới>`: báo "Bài này sẽ mở khi tới lượt", không có nội dung; API trả 403.

## 4. Điền vào ô trống (US3)

1. Bước 3: ô đầu tiên đang chọn. Chạm từ trong Ngân hàng từ → điền vào ô, từ mờ đi, ô kế tiếp được chọn.
2. Chạm ✕ trên một ô: từ trở lại ngân hàng. Chạm ô đã điền rồi chạm từ khác: thay từ.
3. Điền sai một ô, bấm Kiểm tra: ô sai có ✗ và chữ "Sai", kèm "Đáp án: …"; ô đúng có "Đúng". Thấy thẻ "Mẹo ngữ pháp".

## 5. Dịch câu và tổng kết (US4)

1. Bước 4: ghép đúng câu 1 → "Chính xác!", nút loa đọc câu đúng dùng được.
2. Câu 2: ghép sai thứ tự → "Chưa đúng" + câu đúng. Câu 3: ghép dở rồi bấm Làm lại → câu trống.
3. Câu cuối → **Hoàn thành**: "Điền đúng x/y ô", "Dịch đúng a/b câu". Bài hôm nay có "Học bài này"; bài đã học có Đọc lại / Nghe lại /
   Bài viết.
4. Bấm **Làm lại**: về bước 1, mọi kết quả xoá, ngân hàng từ xáo lại.
5. So sánh trang chủ trước và sau: tiến độ, streak, thống kê không đổi.

## 6. 360px, sáng/tối, bàn phím

- Thu màn hình về 360px, cả chế độ sáng và tối: không cuộn ngang; nút Tiếp theo nằm trên thanh tab đáy, không che nội dung cuối trang.
- Làm toàn bộ 4 bước chỉ bằng bàn phím: Tab tới ô trống/ô từ, Enter để chọn, mũi tên đổi tab Bài học/Bài đọc và tốc độ phát.
- Trình đọc màn hình đọc được kết quả Kiểm tra ("Chính xác!", "Đúng 3/5 ô").
