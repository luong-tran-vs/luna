# Quickstart: kiểm tra F7 (AI sinh bài học)

Chuẩn bị:
- `deploy/.env` có `AI_PROVIDER=gemini` và `GEMINI_API_KEY` hợp lệ.
- Chạy `docker compose -f deploy/docker-compose.yml up --build -d`.
- Đăng nhập `admin@example.com` tại http://localhost:8000.
- Có ít nhất một chủ đề A1 (ví dụ "A1 · Gia đình") và vài bài sẵn trong đó.

## Kiểm tra tự động

```bash
cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...
# golangci-lint như các tính năng trước (docker golangci/golangci-lint:v2.14.0)
cd frontend && npx ng test --watch=false && npm run lint && npx ng build
```

## 1. Sinh bản nháp (US1)

1. Vào **Quản trị → Lộ trình**, chọn "A1 · Gia đình", bấm **Sinh bài bằng AI**. Kiểm tra:
   - dialog ghi A1 · Gia đình;
   - số bài 3, độ dài 120, dạng bài "Bài đọc", ý chính trống.
2. Nhập số bài 6 → lỗi "Số bài từ 1 đến 5", không gửi. Sửa lại 3.
3. Bấm **Sinh bài**. Trong lúc chờ phải thấy:
   - nút "Đang sinh…";
   - các ô bị khoá;
   - không bấm sinh được lần nữa.
4. Khi xong:
   - có tối đa 3 bản nháp, mỗi bản ghi số từ trong khoảng 96–144;
   - tiêu đề khác nhau và khác tiêu đề các bài đã có trong chủ đề.
5. Sinh 2 bài dạng **Hội thoại**: mỗi dòng có dạng `Tên: câu nói`.
6. Mở **Bài học** (danh sách) và lộ trình phía người học: chưa có bản nháp nào.

Kiểm bằng curl (cookie admin):

```bash
curl -s -b admin.txt -H 'Content-Type: application/json' \
  --data-binary '{"count":2,"words":120,"kind":"reading","idea":""}' \
  http://localhost:8000/api/admin/topics/<topicId>/generate
# Cookie người học → 403 forbidden; count 0 → 400 validation_failed fields.count
```

## 2. Duyệt và lưu (US2)

1. Sửa tiêu đề bản 1 → số từ không đổi; sửa nội dung → số từ cập nhật.
2. Bấm **Lưu** bản 1. Kiểm tra:
   - bản 1 biến khỏi danh sách nháp;
   - lộ trình tải lại, bài ở cuối, chip audio và chú thích "Đang chạy";
   - mở bài thấy nguồn "AI sinh", giấy phép "Nội dung do AI tạo".
3. **Bỏ** bản 2: không còn ở đâu.
4. Sinh thêm 3 bản, bấm **Lưu tất cả**: 3 bài nối cuối lộ trình đúng thứ tự hiển thị.
5. Xoá trống tiêu đề một bản rồi Lưu tất cả:
   - bản đó báo "Vui lòng nhập tiêu đề" và vẫn ở lại;
   - các bản khác được lưu.
6. Đợi vài phút: audio và chú thích của bài mới chuyển "Xong" như bài tạo tay.

## 3. AI lỗi (US3)

1. Có bản nháp đã sửa. Đặt `GEMINI_API_KEY=` (rỗng) hoặc `AI_PROVIDER=none`, rồi khởi động lại backend.
2. Sinh tiếp. Kiểm tra:
   - dialog báo "AI chưa được cấu hình. Liên hệ người vận hành.";
   - các lựa chọn vẫn giữ;
   - bản nháp cũ (kể cả phần đã sửa) vẫn còn.
3. Khoá sai → "Khoá AI không hợp lệ…".
4. Hết lượt: lặp lại nhiều lượt hoặc dùng test 429 của gemini → "Đã hết lượt AI…".
5. Còn bản nháp, bấm link **Bài học** trên menu:
   - hiện hộp xác nhận;
   - chọn ở lại thì giữ nguyên trang;
   - chọn rời thì sang trang khác.
6. Còn bản nháp, đổi chủ đề trong ô chọn → cũng hỏi xác nhận.
7. Còn bản nháp, nhấn F5 → trình duyệt hỏi rời trang.
8. Không còn bản nháp → rời trang không bị hỏi.

## 4. 360px và bàn phím

- DevTools 360 × 640:
  - dialog, danh sách bản nháp và các nút không cuộn ngang;
  - mọi nút ≥ 44px.
- Chỉ dùng Tab / Shift+Tab / Enter / Space / Escape:
  - mở dialog, nhập, sinh, sửa, lưu, bỏ, xác nhận rời trang;
  - focus không thoát ra sau dialog.
