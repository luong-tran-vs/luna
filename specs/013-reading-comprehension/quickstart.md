# Quickstart: kiểm tra F15 (câu hỏi hiểu bài, ngữ pháp)

## Chuẩn bị

- `deploy/.env` có `GEMINI_API_KEY`, rồi chạy `docker compose -f deploy/docker-compose.yml up --build -d`.
- Tài khoản: `admin@example.com` và `hoc@example.com` (mật khẩu `matkhau123`).

## Kiểm tra tự động

```bash
cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...
# golangci-lint chạy qua Docker như các tính năng trước
cd frontend && npx ng test --watch=false && npm run lint && npx ng build
```

## 1. AI sinh câu hỏi, ngữ pháp, đề viết (US3)

1. Quản trị viên tạo một bài A1 mới, đợi chú thích "Xong", rồi mở chi tiết bài. Phải thấy:
   - 3–5 câu hỏi, mỗi câu có 4 lựa chọn và một đáp án đã chọn;
   - mục Ngữ pháp có 1–3 ví dụ nằm trong bài;
   - một đề viết.
2. `docker compose logs backend | grep '"ai request"'`: bài này chỉ có **1** dòng `op=annotate`.
3. Mở một bài có từ trước F15 (chú thích đã "Xong"). Bấm **Chạy lại chú thích**:
   - nếu bài có chú thích đã sửa tay thì hiện cảnh báo trước khi chạy;
   - xong thì bài có thêm câu hỏi.
4. Tắt AI (`AI_PROVIDER=none`), tạo bài mới:
   - chú thích "Lỗi";
   - bước Đọc của bài này có nút "Đã đọc xong".

## 2. Quản trị viên sửa (US4)

1. Trên chi tiết bài, làm lần lượt:
   - đổi đáp án câu 2;
   - xoá câu cuối;
   - thêm một câu mới;
   - sửa ví dụ ngữ pháp thành một câu **không** có trong bài.
2. Bấm Lưu: phải báo "Ví dụ phải có trong bài" ngay dưới ô ví dụ.
3. Sửa lại ví dụ cho đúng rồi Lưu:
   - thấy nhãn "Đã sửa tay";
   - tải lại trang, các thay đổi vẫn còn.
4. Xoá hết câu hỏi rồi Lưu: bước Đọc của người học quay về nút "Đã đọc xong".

## 3. Người học trả lời (US1, US2)

1. Đăng nhập `hoc@example.com` và mở bài hôm nay ở bước Đọc. Phải thấy:
   - không có nút "Đã đọc xong";
   - mục "Ngữ pháp" thu gọn và mở ra được (cả bằng phím Enter);
   - phần câu hỏi "Câu 1/4".
2. Chọn một lựa chọn sai:
   - hiện "✗ Sai", lựa chọn đúng ghi "Đáp án đúng", có lời giải thích;
   - không chọn lại được.
3. Trả lời câu 2, tải lại trang (F5): câu 1–2 vẫn hiện kết quả, đang ở câu 3.
4. Trả lời hết. Kiểm tra:
   - hiện "Đúng x/4 câu";
   - bước Đọc "Xong", sang được bước Nghe.
5. Gọi API hoàn thành bước Đọc khi chưa trả lời hết:

   ```bash
   curl -s -b hoc.txt -X POST http://localhost:8000/api/today/steps/read/complete
   ```

   Phải nhận 409 `read_incomplete`. Muốn thử, dùng một bài khác mới bắt đầu.
6. Gửi lại câu trả lời của một câu đã trả lời:

   ```bash
   curl -s -b hoc.txt -H 'Content-Type: application/json' \
     --data-binary '{"version":N,"questionIndex":0,"choice":2}' \
     http://localhost:8000/api/lessons/<id>/answers
   ```

   Phải nhận 409 `already_answered`, kết quả cũ không đổi.
7. Xem nguồn JSON của `GET /api/lessons/<id>`: các câu chưa trả lời không có `answerIndex` hay `explanationVi`.
8. Bật chế độ xám (DevTools › Rendering › Emulate vision deficiencies › Achromatopsia): vẫn phân biệt được đúng/sai.

## 4. Xem lại và thống kê (US5)

1. Trang **Bài học** → **Đọc lại** bài vừa học: thấy mọi câu trả lời cũ, không chọn lại được.
2. Trang **Thống kê** hiện "Hiểu bài" là tỷ lệ % đúng kèm số câu đã trả lời; khớp với số câu đúng / số câu đã trả lời.
3. Tài khoản admin chưa trả lời câu nào: tỷ lệ hiện "—".
4. **Cài đặt → Xuất dữ liệu**: file xuất có `readingAnswers` và chỉ có câu trả lời của người đang đăng nhập.

## 5. 360px và bàn phím

- DevTools 360 × 640:
  - quiz, giải thích, mục ngữ pháp và form sửa câu hỏi không cuộn ngang;
  - các nút lựa chọn ≥ 44px.
- Chỉ dùng bàn phím:
  - trả lời quiz bằng Tab + Enter;
  - mở/đóng mục ngữ pháp;
  - sửa và lưu câu hỏi ở trang quản trị.
