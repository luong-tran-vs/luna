# Giai đoạn 3: AI xử lý giọng nói

> Nguồn: [mvp-features.md](../mvp-features.md) v6. Cần xong [giai đoạn 2](giai-doan-2.md).

**Mục tiêu:** thêm kỹ năng **Nói**.
**Các bước của bài:** Ôn → Đọc → Nghe → Viết → **Nói**.
**AI:** whisper.cpp chạy trên máy (model `base.en` hoặc `small.en`, CPU). Chưa chốt: RAM máy.

---

## F10. Nói (shadowing)

**Chức năng**
- Với từng câu của bài: nghe audio mẫu, bấm **Ghi âm**, đọc theo, bấm **Dừng**.
- Whisper chuyển giọng nói thành chữ, so sánh với câu gốc.
- Tô màu từng từ: *đúng*, *sai*, *thiếu* (cùng quy ước với F4).
- Nghe lại bản ghi của mình, ghi lại bao nhiêu lần cũng được.

**Tiêu chí nghiệm thu**
- [ ] Có kết quả trong vòng 5 giây với câu dưới 20 từ.
- [ ] Trình duyệt chưa cho quyền micro thì hiện hướng dẫn cấp quyền.
- [ ] Bước Nói hoàn thành khi đã ghi âm hết các câu; câu sai vẫn tính là xong.
- [ ] File ghi âm không lưu lâu dài, chỉ lưu kết quả so sánh.

---

## F11. Hội thoại nhập vai *(không bắt buộc)*

**Chức năng**
- Chọn một tình huống gắn với bài học (gọi món, hỏi đường…).
- Trò chuyện với AI bằng giọng nói hoặc gõ chữ; AI dùng lại từ vựng của bài.
- Kết thúc thì AI tóm tắt lỗi thường gặp.

**Tiêu chí nghiệm thu**
- [ ] Không bắt buộc để hoàn thành bài học.
- [ ] Mỗi lượt trò chuyện tốn tối đa 1 request AI.

---

## Thay đổi trên tính năng cũ

- **L. Luồng một ngày học:** thêm bước Nói sau bước Viết.
- **F6:** hiện thêm thanh kỹ năng **Nói**.
