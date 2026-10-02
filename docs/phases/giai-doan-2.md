# Giai đoạn 2: AI xử lý chữ

> Nguồn: [mvp-features.md](../mvp-features.md) v9. Cần xong [giai đoạn 1](giai-doan-1.md).

**Mục tiêu:** AI giúp soạn bài; thêm kỹ năng **Viết**; bước Đọc có câu hỏi hiểu bài và ghi chú ngữ pháp.
**Các bước của bài:** Ôn → Đọc → Nghe → **Viết** (tuỳ chọn, có nút Bỏ qua; cập nhật 2026-10-02).
**AI:** Gemini gói miễn phí (ưu tiên Flash-Lite), dự phòng Groq, OpenRouter hoặc Ollama. AI lỗi thì mọi tính năng không cần AI vẫn chạy.
**Thứ tự làm:** F7 → F15 → F8 → F9.

---

## F7. AI sinh bài học

**Chức năng**
- Nút **Sinh bài bằng AI** trong trang lộ trình của một chủ đề (F14). Trình độ và chủ đề lấy theo lộ trình đang mở.
- Nhập: số bài (1–10; trước 2026-10-02 là 1–5), độ dài (số từ), dạng bài (bài đọc hoặc hội thoại), ý chính (không bắt buộc).
- AI tránh lặp nội dung các bài đã có trong chủ đề.
- Mỗi bài sinh ra là một **bản nháp**: quản trị viên xem, sửa, rồi **Lưu** hoặc **Bỏ** từng bài.
- Bài đã lưu có nguồn "AI sinh", đi qua quy trình của F2 (tách câu, chú thích) và được thêm vào **cuối lộ trình** của chủ đề.

**Tiêu chí nghiệm thu**
- [ ] Bản nháp đúng trình độ của chủ đề và đúng độ dài đã chọn (sai lệch quá 20% thì có cảnh báo; đổi 2026-10-02, trước đó bị loại).
- [ ] Không lưu tự động; bản nháp chưa lưu không xuất hiện trong danh sách bài hay lộ trình.
- [ ] Sinh nhiều bài một lượt: các bài khác nội dung nhau và khác các bài đã có trong chủ đề.
- [ ] AI lỗi hoặc hết lượt: thông báo rõ ràng, không mất các bản nháp đã sinh và dữ liệu đã nhập.

---

## F15. Câu hỏi hiểu bài và ghi chú ngữ pháp

**Chức năng**
- Khi chú thích bài (F2), AI trả về thêm trong **cùng một request**:
  - 3–5 câu hỏi trắc nghiệm hiểu bài (4 lựa chọn, 1 đáp án đúng, giải thích ngắn bằng tiếng Việt).
  - Một ghi chú ngữ pháp ngắn bằng tiếng Việt về một điểm ngữ pháp nổi bật trong bài, có ví dụ lấy từ bài.
  - Một đề viết cho bước Viết (F8).
- Quản trị viên xem và sửa được câu hỏi, ghi chú ngữ pháp và đề viết.
- Bài đã có từ trước: quản trị viên bấm **Chạy lại chú thích** để sinh thêm các phần này.
- Bước Đọc: sau khi đọc, người học trả lời câu hỏi; chọn xong mỗi câu thì thấy đúng/sai và giải thích. Mục **Ngữ pháp** hiện trong bước Đọc.

**Tiêu chí nghiệm thu**
- [ ] Chú thích một bài vẫn chỉ tốn 1 request AI.
- [ ] Bước Đọc hoàn thành khi đã trả lời hết câu hỏi (trả lời sai vẫn tính là xong).
- [ ] Bài chưa có câu hỏi (AI lỗi hoặc bài cũ): bước Đọc dùng nút "Đã đọc xong" như giai đoạn 1.
- [ ] Đúng/sai phân biệt được không cần nhìn màu; tỷ lệ trả lời đúng được lưu để thống kê.

---

## F8. Viết

**Chức năng**
- Bước **Viết** sau bước Nghe: người học viết theo đề của bài (F15) và bấm **Nộp**, hoặc bấm **Bỏ qua** để hoàn thành bài mà
  không viết (cập nhật 2026-10-02: Viết là tuỳ chọn).
- Bài viết nháp được tự lưu trong lúc gõ.
- AI chấm nền và trả về:
  - Điểm 1–5 và nhận xét tiếng Việt cho 4 tiêu chí: hoàn thành yêu cầu, ngữ pháp, từ vựng, mạch lạc.
  - Bản đã sửa, so sánh với bản gốc (đánh dấu chỗ thêm, bớt).
- Có kết quả thì hiện thông báo trong app. Trang **Bài viết** liệt kê các bài đã nộp và nhận xét.

**Tiêu chí nghiệm thu**
- [ ] Bước Viết hoàn thành ngay khi nộp, không chờ AI chấm; bấm Bỏ qua cũng hoàn thành bước (không có bài viết, không gọi AI).
- [ ] Có kết quả chấm thì hiện thông báo trong app.
- [ ] Chấm lỗi thì bài viết vẫn được lưu, có nút Chấm lại.
- [ ] Bài viết nháp được tự lưu, thoát ra vào lại không mất.
- [ ] Bài chưa có đề viết: dùng đề mặc định "Tóm tắt bài bằng 3–5 câu".

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

- **L. Luồng một ngày học:** thêm bước Viết (tuỳ chọn, bỏ qua được) sau bước Nghe; bước Đọc hoàn thành theo F15. Có nút
  **← Bước trước** để xem lại bước Đọc, Nghe đã xong mà không mất tiến độ.
- **F6:** hiện thêm thanh kỹ năng **Viết**; thống kê thêm tỷ lệ đúng câu hỏi hiểu bài, số bài viết và điểm trung bình.
- **F13:** file xuất dữ liệu có thêm câu trả lời hiểu bài và bài viết.
