# Quickstart: kiểm tra F2

Chạy: `docker compose -f deploy/docker-compose.yml up --build -d` (lần đầu tải image Kokoro vài GB). Đăng nhập
bằng tài khoản quản trị viên, mở <http://localhost:8000/admin>.

Muốn có chú thích AI: tạo key miễn phí tại Google AI Studio, ghi `GEMINI_API_KEY=...` vào `deploy/.env`, rồi
`up -d` lại backend.

## 1. Tạo bài và tách câu

Thêm bài: tiêu đề "Test", trình độ B1, nguồn "Tự viết", giấy phép "CC BY 4.0", nội dung:

```text
Mr. Smith arrived at 9.30 a.m. He was late! Why?
"Stop!" she said. J. K. Rowling wrote it... Really?
```

**Mong đợi**: xem trước có 6 câu: "Mr. Smith arrived at 9.30 a.m." · "He was late!" · "Why?" · "\"Stop!\" she
said." · "J. K. Rowling wrote it..." · "Really?". Để trống nguồn → lỗi tiếng Việt, không lưu.

## 2. Audio và chú thích chạy nền

1. Ngay sau khi lưu: danh sách hiện "Đang chạy" cho cả hai; trong ≤ 10 giây sau khi xong tự đổi "Xong" (không
   tải lại).
2. Mở chi tiết, bấm ▶ từng câu → nghe đúng câu.
3. `docker compose ... exec backend ls /data/audio/<id>/1` → 6 file; mở lại bài vài lần → số file không đổi,
   `docker compose ... logs backend | grep -c "tts synthesize"` không tăng.
4. Không có `GEMINI_API_KEY`: bài vẫn lưu, audio "Xong", chú thích "Lỗi: AI chưa được cấu hình". Thêm key, khởi
   động lại backend, bấm "Chạy lại" → "Xong". `logs backend | grep -c "ai request"` tăng đúng 1.
5. `docker compose ... stop kokoro`, tạo bài mới → audio thử lại rồi "Lỗi"; `start kokoro`, "Chạy lại" → "Xong".
6. Khởi động lại backend khi audio đang chạy → sau khi lên lại, audio tiếp tục và "Xong".

## 3. Lộ trình

1. Thêm 4 bài vào lộ trình; kéo bài cuối lên đầu (bằng tay nắm), tải lại → thứ tự giữ nguyên.
2. Dùng nút Lên/Xuống bằng bàn phím (Tab + Enter) → đổi chỗ được.
3. Gỡ 2 bài → cảnh báo "Lộ trình chỉ còn 2 bài chưa học".
4. Thêm lại một bài đã có → không trùng.

## 4. Sửa chú thích

Sửa nghĩa một mục, thêm mục "was late" (có trong bài), thêm mục "banana" (không có) → lỗi "Cụm từ không có trong
bài"; xoá một mục; lưu, tải lại → đúng thay đổi, mục sửa/thêm có nhãn "Đã sửa tay".

## 5. Sửa và xoá bài

1. Đổi tiêu đề → audio và chú thích giữ "Xong", file audio không đổi.
2. Đổi nội dung → hộp thoại cảnh báo chú thích sửa tay; xác nhận → câu tách lại, hai trạng thái "Đang chạy", audio
   mới ở thư mục revision 2, thư mục revision 1 bị xoá khi xong.
3. Xoá bài trong lộ trình → "Gỡ bài khỏi lộ trình trước khi xoá". Gỡ rồi xoá → bài và `/data/audio/<id>` biến mất.

## 6. Quyền

Đăng nhập người học: `/admin` → trang không có quyền; `curl -b "luna_session=<token người học>"
http://localhost:8000/api/admin/lessons` → 403; không cookie → 401; mở URL audio không cookie → 401.

## 7. 360px, sáng và tối

Danh sách, form, chi tiết (bảng chú thích), lộ trình: không cuộn ngang; chip trạng thái có chữ + biểu tượng; tay
nắm kéo và nút Lên/Xuống ≥ 44px.
