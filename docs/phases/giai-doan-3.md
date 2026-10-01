# Giai đoạn 3: AI xử lý giọng nói

> Nguồn: [mvp-features.md](../mvp-features.md) v10. Cần xong [giai đoạn 2](giai-doan-2.md).

**Mục tiêu:** dùng app trên điện thoại ở mọi nơi qua HTTPS, và thêm kỹ năng **Nói**.
**Các bước của bài:** Ôn → Đọc → Nghe → Viết → **Nói** (bắt buộc).
**AI:** whisper.cpp chạy trên CPU, model `small.en` (máy 16GB RAM; dự phòng `base.en` nếu chậm). Hội thoại (F11) dùng Gemini và Kokoro.
**Thứ tự làm:** F16 → F10 → F11.

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
- Bước **Nói** sau bước Viết, bắt buộc. Với từng câu của bài: nghe audio mẫu, bấm **Ghi âm**, đọc theo, bấm **Dừng**.
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
- Người học trả lời bằng giọng nói (Whisper) hoặc gõ chữ; câu trả lời của AI có audio (Kokoro).
- Mỗi phiên tối đa 10 lượt; kết thúc thì AI tóm tắt lỗi thường gặp và gợi ý cách nói tốt hơn.

**Tiêu chí nghiệm thu**
- [ ] Không bắt buộc để hoàn thành bài học; không ảnh hưởng tiến độ hay streak.
- [ ] Mỗi lượt trò chuyện tốn tối đa 1 request AI.
- [ ] AI lỗi hoặc hết lượt giữa phiên: báo rõ, giữ nội dung đã trò chuyện.

---

## Thay đổi trên tính năng cũ

- **L. Luồng một ngày học:** thêm bước Nói (bắt buộc) sau bước Viết.
- **F6:** hiện thêm thanh kỹ năng **Nói**; thống kê thêm tỷ lệ đọc đúng.
- **F13:** file xuất dữ liệu có thêm kết quả bước Nói và lịch sử hội thoại.
