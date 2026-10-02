# Giai đoạn 3: AI xử lý giọng nói

> Nguồn: [mvp-features.md](../mvp-features.md) v10. Cần xong [giai đoạn 2](giai-doan-2.md).

**Mục tiêu:** dùng app trên điện thoại ở mọi nơi qua HTTPS, và thêm kỹ năng **Nói**.
**Các bước của bài:** Ôn → Đọc → Nghe → Viết → **Nói** (bắt buộc).
**AI:** whisper.cpp chạy trên CPU, model `small.en` (máy 16GB RAM; dự phòng `base.en` nếu chậm). Hội thoại (F11) dùng Gemini; mọi chỗ nghe dùng giọng đọc của trình duyệt (bỏ Kokoro từ 2026-10-02).
**Thứ tự làm:** F16 → F10 → F11. F17 và F18 (thêm 2026-10-02) độc lập, làm lúc nào cũng được.

---

## F16. Vận hành: đặt lại mật khẩu và truy cập từ điện thoại

**Chức năng**
- Lệnh đặt lại mật khẩu chạy trong container backend: tạo mật khẩu tạm in ra một lần, huỷ mọi phiên đăng nhập của tài khoản đó, gỡ khoá đăng nhập.
- Lệnh liệt kê tài khoản (email, vai trò, ngày tạo) để biết cần đặt lại cho ai.
- Truy cập từ điện thoại qua **HTTPS bằng Tailscale** (`tailscale serve`): chỉ thiết bị trong tailnet của bạn vào được, không mở ra internet.
- README có hướng dẫn từng bước: cài Tailscale trên máy tính và điện thoại, bật `tailscale serve`, bật cookie Secure.

**Tiêu chí nghiệm thu**
- [ ] Đặt lại mật khẩu bằng một lệnh; đăng nhập được bằng mật khẩu tạm; các phiên cũ không dùng được nữa.
- [ ] Điện thoại (cài Tailscale, cùng tài khoản) mở được app qua `https://…ts.net`, không cảnh báo chứng chỉ, đăng nhập và học được.
- [ ] Thiết bị ngoài tailnet không truy cập được.
- [ ] Trên địa chỉ HTTPS đó, trình duyệt điện thoại cho phép dùng micro (cần cho F10).
- [ ] Cookie phiên có cờ Secure khi chạy qua HTTPS; `http://localhost` trên máy tính vẫn dùng được.

---

## F10. Nói (shadowing)

**Chức năng**
- Bước **Nói** sau bước Viết, bắt buộc. Với từng câu của bài: nghe câu mẫu (giọng đọc của trình duyệt), bấm **Ghi âm**, đọc theo, bấm **Dừng**.
- Whisper chuyển giọng nói thành chữ, so với câu gốc và tô từng từ: *đúng*, *sai*, *thiếu* (cùng quy ước F4).
- Nghe lại bản ghi của mình; ghi lại câu đó bao nhiêu lần cũng được, lấy lần gần nhất.
- Tỷ lệ đọc đúng của bài được lưu để thống kê.

**Tiêu chí nghiệm thu**
- [ ] Có kết quả trong vòng 5 giây với câu dưới 20 từ.
- [ ] Trình duyệt chưa cho quyền micro thì hiện hướng dẫn cấp quyền; không có micro thì báo rõ.
- [ ] Bước Nói hoàn thành khi đã ghi âm hết các câu; câu sai vẫn tính là xong.
- [ ] File ghi âm không lưu lâu dài, chỉ lưu kết quả so sánh.
- [ ] Whisper không chạy: báo lỗi rõ ràng, thử lại được; các bước khác của bài không bị ảnh hưởng.

---

## F11. Hội thoại nhập vai *(không bắt buộc)*

**Chức năng**
- Từ bài học, chọn **Luyện hội thoại**: AI đóng một vai trong một tình huống gắn với bài (gọi món, hỏi đường…), dùng lại từ vựng của bài.
- Người học trả lời bằng giọng nói (Whisper) hoặc gõ chữ; câu trả lời của AI được đọc bằng giọng đọc của trình duyệt.
- Mỗi phiên tối đa 10 lượt; kết thúc thì AI tóm tắt lỗi thường gặp và gợi ý cách nói tốt hơn.

**Tiêu chí nghiệm thu**
- [ ] Không bắt buộc để hoàn thành bài học; không ảnh hưởng tiến độ hay streak.
- [ ] Mỗi lượt trò chuyện tốn tối đa 1 request AI.
- [ ] AI lỗi hoặc hết lượt giữa phiên: báo rõ, giữ nội dung đã trò chuyện.

---

## F17. Trang chi tiết bài và luyện tập từ vựng *(không bắt buộc)*

> Thêm 2026-10-02. Bố cục theo `design/screen1.png`, `design/screen2.png`, `design/screen3.png` (giữ màu Oải hương và font Lexend).
> Không phụ thuộc F16/F10/F11, làm trước hay sau đều được.

**Chức năng**
- Sau khi chú thích bài, AI sinh thêm (1 request riêng) phần luyện tập xoay quanh từ vựng của bài: mục tiêu bài, câu ví dụ tiếng Anh
  cho mỗi từ, hội thoại mẫu 2 người dùng các từ đó (kèm nghĩa tiếng Việt), mẹo ngữ pháp, 3–5 câu dịch Việt → Anh. Từ, câu ví dụ, hội
  thoại và câu dịch được đọc bằng giọng đọc của trình duyệt.
- Trang chi tiết bài (`/lessons/:id`): thanh trên (đóng, tiến độ 1/4, streak), "Bài N" + tiêu đề, mục tiêu, mức độ; tab **Bài học**
  (4 bước) và **Bài đọc** (bài đọc gốc, ngữ pháp). Nút **Tiếp theo** cố định cuối màn hình.
  1. **Từ vựng quan trọng:** từ, phiên âm, nghĩa tiếng Việt, nút loa, câu ví dụ tiếng Anh.
  2. **Hội thoại mẫu:** nghe cả đoạn (tốc độ) hoặc từng lượt; lời ẩn sẵn (nghe trước, bấm "Hiện lời" để xem), mỗi lượt có dạng sóng
     tô theo tiến độ phát; bật/tắt nghĩa tiếng Việt.
  3. **Điền vào ô trống:** hội thoại có ô trống ở từ vựng, chạm từ trong ngân hàng từ để điền, Kiểm tra, mẹo ngữ pháp.
  4. **Dịch câu sang tiếng Anh:** ghép câu bằng các ô từ (có từ gây nhiễu), Làm lại, Kiểm tra từng câu.
- Tổng kết cuối cùng, làm lại được. Không có điểm XP.
- Quản trị viên xem phần luyện tập và bấm **Tạo lại phần luyện tập**.

**Tiêu chí nghiệm thu**
- [ ] Bài mới chú thích xong thì sau ít phút có phần luyện tập; mỗi câu ví dụ chứa đúng từ của nó; ô trống và đáp án đều là từ vựng
  của bài.
- [ ] Nghe được từng từ, từng câu ví dụ, từng lượt hội thoại và cả đoạn; đổi được tốc độ; bật/tắt được nghĩa tiếng Việt.
- [ ] Điền vào ô trống: chạm từ để điền, gỡ được; Kiểm tra thấy đúng/sai từng ô (có chữ, không chỉ màu) và đáp án.
- [ ] Dịch câu: ghép đúng thứ tự thì "Chính xác!", sai thì thấy câu đúng; Làm lại xoá câu đang ghép.
- [ ] Thanh tiến độ và nút Tiếp theo chuyển đúng 4 bước; cuối cùng thấy tổng kết và làm lại được.
- [ ] AI lỗi khi sinh phần luyện tập: chú thích từ, câu hỏi hiểu bài vẫn bình thường; bước 2–4 báo chưa có phần luyện tập.
- [ ] Sửa nội dung bài thì phần luyện tập được sinh lại theo nội dung mới.
- [ ] Không ảnh hưởng các bước của bài, tiến độ, streak, thống kê. Bài sắp tới vẫn không mở được.
- [ ] Dùng tốt ở 360px, chế độ sáng và tối.

---

## F18. Từ vựng theo chủ đề *(quản trị)*

> Thêm 2026-10-02. Dữ liệu ban đầu: [f18-topic-words.json](../spec-inputs/f18-topic-words.json), 1.257 từ tiếng Anh cho 42 chủ đề,
> tham khảo danh sách của Langmaster (chỉ lấy từ tiếng Anh). Không phụ thuộc F16/F10/F11; làm sau F17 thì trang chi tiết bài dùng ngay.

**Chức năng**
- Mỗi chủ đề có danh sách từ vựng tiếng Anh cốt lõi; nạp sẵn từ file dữ liệu một lần, quản trị viên xem, thêm, xoá ở trang Chủ đề.
- Trang Chủ đề hiện độ phủ "Từ vựng: đã dùng X/Y" (từ có trong bài của chủ đề) và đánh dấu từ đã dùng/chưa dùng.
- Sinh bài bằng AI: mỗi bài được giao một nhóm từ mục tiêu, ưu tiên từ chưa dùng; quản trị viên chỉnh nhóm từ trước khi sinh; bản
  nháp báo số từ mục tiêu đã dùng và từ còn thiếu.
- Chú thích bài đưa các từ của chủ đề có trong bài vào danh sách từ vựng của bài.

**Tiêu chí nghiệm thu**
- [ ] Lần khởi động đầu, 42 chủ đề trùng tên nhận danh sách từ; khởi động lại không nạp lại, không ghi đè danh sách đã sửa.
- [ ] Thêm nhiều từ một lần, xoá từ; từ trùng hoặc sai định dạng bị báo lỗi theo từng từ.
- [ ] Độ phủ đúng: từ có trong nội dung bài của chủ đề (kể cả cụm, hoa/thường, số nhiều) là "đã dùng".
- [ ] Sinh 3 bài với 8 từ mỗi bài: nhóm từ không trùng nhau, ưu tiên từ chưa dùng; vẫn 1 request AI; bản nháp báo từ còn thiếu.
- [ ] Đặt 0 từ mục tiêu hoặc chủ đề chưa có từ: sinh bài như trước.
- [ ] Bài mới chú thích xong có trong danh sách từ vựng mọi từ của chủ đề xuất hiện trong bài.
- [ ] Không ảnh hưởng luồng học, tiến độ, streak, thống kê; dùng tốt ở 360px, sáng và tối, bằng bàn phím.

---

## Thay đổi trên tính năng cũ

- **L. Luồng một ngày học:** thêm bước Nói (bắt buộc) sau bước Viết.
- **F6:** hiện thêm thanh kỹ năng **Nói**; thống kê thêm tỷ lệ đọc đúng.
- **F13:** file xuất dữ liệu có thêm kết quả bước Nói và lịch sử hội thoại.
