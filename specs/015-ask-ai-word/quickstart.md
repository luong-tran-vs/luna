# Quickstart: kiểm tra F9 (Hỏi AI về từ)

## Chuẩn bị

- Chạy `docker compose -f deploy/docker-compose.yml up --build -d`.
- Đăng nhập `hoc@example.com`.
- Để hỏi AI thật, ghi `GEMINI_API_KEY` vào `deploy/.env`.

## Kiểm tra tự động

```bash
cd backend && CGO_ENABLED=1 GOTOOLCHAIN=local go test -race ./...
cd frontend && npx ng test --watch=false && npm run lint && npx ng build
```

## 1. Hỏi AI (US1)

1. Mở bước Đọc. Tra một từ đã có trong chú thích của bài: nhãn "AI · theo ngữ cảnh", không có nút Hỏi AI.
2. Tra một cụm không có trong chú thích. Popup hiện nghĩa từ điển hoặc "Chưa có nghĩa", kèm nút **Hỏi AI**.
3. Mở log backend:

   ```bash
   docker compose logs -f backend | grep '"op":"explain"'
   ```

   Tra thêm vài từ nhưng không bấm nút: log không có dòng mới.
4. Bấm **Hỏi AI**:
   - popup hiện "Đang hỏi AI…" và nút bị khoá;
   - sau đó hiện nghĩa theo câu, dạng gốc, câu giải thích và nhãn "AI · theo ngữ cảnh";
   - log có đúng 1 dòng `op=explain`.
5. Bấm **Lưu vào sổ từ**: sổ từ có thẻ với nghĩa AI và câu đang đọc.

## 2. Dùng lại kết quả (US2)

1. Đóng popup rồi tra lại cùng cụm trong cùng câu: kết quả AI hiện ngay, không có nút Hỏi AI, log không có dòng mới.
2. Đăng nhập tài khoản khác được mở bài này (hoặc admin) và tra cùng cụm, cùng câu: thấy kết quả đã lưu.
3. Tra cùng cụm ở câu khác: lại có nút Hỏi AI.
4. Bấm hỏi nhiều lần liền nhau, gửi qua curl trong hai terminal cùng lúc:

   ```bash
   curl -s -b hoc.txt -H 'Content-Type: application/json' \
     --data-binary '{"text":"make up for","sentenceIndex":3}' \
     http://localhost:8000/api/lessons/<id>/ask
   ```

   Log chỉ có 1 dòng `op=explain`.
5. Quản trị viên sửa nội dung bài. Tra lại: kết quả cũ không còn, lại có nút Hỏi AI.

## 3. AI lỗi (US3)

1. Đặt `AI_PROVIDER=none` rồi khởi động lại backend. Bấm Hỏi AI:
   - popup báo "AI chưa được cấu hình";
   - nghĩa từ điển hoặc ô tự nhập vẫn còn;
   - Lưu vẫn được.
2. Gửi `text` không có trong câu: nhận 400 `fields.text`.

## 4. 360px và bàn phím

- Thu màn hình về 360px: nút Hỏi AI, câu giải thích và thông báo lỗi không gây cuộn ngang.
- Làm toàn bộ bằng bàn phím: mở popup, Tab tới **Hỏi AI**, Enter để hỏi, Tab tới **Lưu**.
- Trình đọc màn hình phải đọc "Đang hỏi AI…" và đọc kết quả.
