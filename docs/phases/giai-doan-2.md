# Giai đoạn 2: AI xử lý chữ

> Nguồn: [mvp-features.md](../mvp-features.md) v6. Cần xong [giai đoạn 1](giai-doan-1.md).

**Mục tiêu:** thêm kỹ năng **Viết** và để AI hỗ trợ soạn bài.
**Các bước của bài:** Ôn → Đọc → Nghe → **Viết**.
**AI:** Gemini gói miễn phí (ưu tiên Flash-Lite), dự phòng Groq, OpenRouter hoặc Ollama.

---

## F7. AI sinh bài học

**Chức năng**
- Nút **Sinh bài bằng AI** trong trang quản trị (F2).
- Nhập: chủ đề, trình độ CEFR, độ dài (số từ), dạng bài (bài đọc hoặc hội thoại).
- AI trả về bản nháp; quản trị viên sửa rồi mới lưu.
- Bài được ghi nguồn là "AI sinh". Sau khi lưu, bài đi qua đúng quy trình của F2 (tách câu, audio, chú thích).

**Tiêu chí nghiệm thu**
- [ ] Bản nháp đúng trình độ và độ dài đã chọn (sai lệch độ dài không quá 20%).
- [ ] Không lưu tự động; phải có thao tác lưu của quản trị viên.
- [ ] AI lỗi hoặc hết lượt: hiện thông báo rõ ràng, không mất dữ liệu đã nhập.

---

## F8. Viết

**Chức năng**
- Mỗi bài có 1 đề viết: trả lời câu hỏi, tóm tắt, hoặc viết tiếp. Đề do AI sinh khi tạo bài, quản trị viên sửa được.
- Người học viết và bấm **Nộp**.
- AI chấm nền và trả về:
  - Nhận xét theo 4 tiêu chí: hoàn thành yêu cầu, ngữ pháp, từ vựng, mạch lạc.
  - Bản đã sửa, so sánh với bản gốc (đánh dấu chỗ thêm, bớt).
- Xem lại các bài viết cũ và nhận xét.

**Tiêu chí nghiệm thu**
- [ ] Bước Viết hoàn thành ngay khi nộp, không chờ AI chấm.
- [ ] Có kết quả chấm thì hiện thông báo trong app.
- [ ] Chấm lỗi thì bài viết vẫn được lưu, có nút Chấm lại.
- [ ] Bài viết nháp được tự lưu, thoát ra vào lại không mất.

---

## F9. Hỏi AI về từ

**Chức năng**
- Trong popup tra từ (F3), với từ hoặc cụm **không có** trong chú thích của bài: nút **Hỏi AI**.
- AI giải thích nghĩa tiếng Việt theo đúng câu đang đọc.
- Kết quả được lưu lại cho lần tra sau.

**Tiêu chí nghiệm thu**
- [ ] Chỉ gọi AI khi người học bấm nút.
- [ ] Tra lại cùng từ trong cùng câu thì dùng kết quả đã lưu, không gọi AI lần nữa.

---

## Thay đổi trên tính năng cũ

- **L. Luồng một ngày học:** thêm bước Viết sau bước Nghe.
- **F6:** hiện thêm thanh kỹ năng **Viết**.

## Chờ quyết định

- Câu hỏi hiểu bài ở bước Đọc (3–5 câu, AI sinh cùng lúc chú thích).
- Ghi chú ngữ pháp ngắn cho mỗi bài.
