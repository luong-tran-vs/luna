# Giai đoạn 1: Nền tảng

> Nguồn: [mvp-features.md](../mvp-features.md) v6, [design-system.md](../design-system.md) v1.

**Mục tiêu:** học được mỗi ngày với chu trình **Ôn → Đọc → Nghe**.
**AI:** chỉ dùng ở F2 (chú thích bài). AI lỗi thì mọi tính năng khác vẫn chạy.
**Thứ tự làm:** Khung dự án (dựng Angular, Go, Docker Compose theo [architecture.md](../architecture.md)) → F1 → F2 → F3 → F4 → F5 → L → F6 → F12 → F13.

---

## F1. Tài khoản

**Chức năng**
- Đăng ký, đăng nhập, đăng xuất bằng email và mật khẩu.
- Hai vai trò: **người học** và **quản trị viên**. Tài khoản đăng ký đầu tiên là quản trị viên, các tài khoản sau là người học.
- Mọi dữ liệu cá nhân (sổ từ, tiến độ, cài đặt) gắn với tài khoản.

**Tiêu chí nghiệm thu**
- [ ] Mật khẩu tối thiểu 8 ký tự, được lưu dạng băm.
- [ ] Sai email hoặc mật khẩu thì báo lỗi chung, không nói rõ sai phần nào.
- [ ] Người học không vào được trang quản trị (bị chặn cả ở giao diện và API).
- [ ] Hai tài khoản khác nhau không thấy dữ liệu của nhau.

---

## F2. Trang quản trị bài học

**Chức năng**
- Thêm, sửa, xoá, xem trước bài học.
- Thông tin bài: tiêu đề, nội dung (dán văn bản), trình độ CEFR (A1–C2), chủ đề, nguồn, giấy phép.
- Khi lưu bài, hệ thống chạy nền 3 việc:
  1. Tách bài thành từng câu.
  2. Sinh audio cho từng câu (TTS chạy trên máy).
  3. Gửi **1 request** AI để chú thích các từ và cụm từ đáng học: dạng gốc (*went → go*) và nghĩa tiếng Việt theo ngữ cảnh.
- Mỗi bài hiện trạng thái của audio và của chú thích: *đang chạy*, *xong*, *lỗi*, kèm nút **Chạy lại**.
- Quản trị viên xem và sửa được phần chú thích.
- **Lộ trình:** danh sách bài có thứ tự, kéo thả để đổi thứ tự.
- Cảnh báo khi lộ trình còn dưới 3 bài chưa học.

**Tiêu chí nghiệm thu**
- [ ] Lưu bài thành công ngay cả khi AI lỗi hoặc hết lượt; trạng thái chú thích là *lỗi* và bấm Chạy lại được.
- [ ] Mỗi câu có một file audio riêng; audio sinh một lần, không sinh lại khi mở bài.
- [ ] Sửa nội dung bài thì tách câu, audio và chú thích được làm lại.
- [ ] Không xoá được bài mà người học đang học dở (phải gỡ khỏi lộ trình trước).
- [ ] Trang quản trị dùng được trên điện thoại (từ 360px).

---

## F3. Đọc

**Chức năng**
- Hiển thị bài; bấm vào một từ hoặc bôi đen nhiều từ để tra.
- Popup tra từ gồm: từ, phiên âm IPA, nghĩa, nút nghe phát âm, nút **Lưu vào sổ từ**, và nhãn nguồn nghĩa.
- Thứ tự lấy nghĩa:
  1. Chú thích AI của bài → nhãn "AI · theo ngữ cảnh".
  2. Từ điển Anh–Việt offline (SQLite) → nhãn "Từ điển".
  3. Không có → người học tự nhập.
- Lưu từ thì lưu kèm câu chứa từ đó.
- Từ đã có trong sổ được tô màu ở mọi bài.
- Nút **Đã đọc xong** để hoàn thành bước.

**Tiêu chí nghiệm thu**
- [ ] Tra từ trả kết quả dưới 300ms (không gọi AI khi tra).
- [ ] Tra được dạng biến đổi của từ (*went* ra *go*) nhờ chú thích hoặc từ điển.
- [ ] Lưu một từ đã có trong sổ thì không tạo thẻ trùng.
- [ ] Nút Đã đọc xong chỉ bật khi đã cuộn hết bài.

---

## F4. Nghe

**Chức năng**
- Phát theo từng câu: câu trước, câu sau, lặp lại câu.
- Tốc độ: 0.5x, 0.75x, 1x, 1.25x.
- Ẩn hoặc hiện transcript.
- **Chép chính tả:** nghe một câu, gõ lại, bấm Kiểm tra. So sánh từng từ và đánh dấu *đúng*, *sai* (kèm từ đúng), *thiếu*.

**Tiêu chí nghiệm thu**
- [ ] So sánh không phân biệt hoa thường và bỏ qua dấu câu.
- [ ] Mỗi câu được nghe lại không giới hạn số lần trước khi kiểm tra.
- [ ] Bước Nghe hoàn thành khi đã kiểm tra hết các câu; câu sai vẫn tính là xong.
- [ ] Tỷ lệ đúng của mỗi bài được lưu để thống kê.
- [ ] Sai và thiếu phân biệt được mà không cần nhìn màu (gạch ngang, gạch chân chấm).

---

## F5. Sổ từ và ôn tập

**Chức năng**
- Thẻ gồm: từ, IPA, nghĩa, câu ví dụ từ bài học, audio.
- Xem danh sách thẻ; sửa, xoá, tự thêm thẻ.
- Lịch ôn theo thuật toán **FSRS**.
- Hai kiểu ôn:
  - **Xem từ đoán nghĩa:** hiện từ, bấm để lật xem nghĩa.
  - **Nghe rồi gõ:** nghe audio, gõ lại từ.
- Sau mỗi thẻ, người học chọn: **Again**, **Hard**, **Good**, **Easy**.
- Ôn tự do được bất cứ lúc nào, ngoài bước Ôn bắt buộc.

**Tiêu chí nghiệm thu**
- [ ] Thẻ mới lưu hôm nay đến hạn ôn lần đầu vào hôm sau.
- [ ] Đánh giá của người học cập nhật ngày ôn tiếp theo đúng theo FSRS.
- [ ] Kiểu Nghe rồi gõ không phân biệt hoa thường.
- [ ] Sửa nghĩa trên thẻ không làm mất lịch ôn của thẻ.

---

## L. Luồng một ngày học

**Chức năng**
- **Mục tiêu:** lộ trình N bài (mặc định 30). Hoàn thành hết thì mời đặt mục tiêu mới.
- **Mỗi ngày một bài:** bài hôm nay là bài tiếp theo chưa hoàn thành trong lộ trình.
- **Các bước của bài:** Ôn → Đọc → Nghe. Bước sau chỉ mở khi xong bước trước.
- **Bước Ôn:** ôn các thẻ đến hạn, tối đa 30 thẻ mỗi ngày (chỉnh ở F12). Không có thẻ đến hạn thì bước tự hoàn thành.
- **Hoàn thành bài:** thanh mục tiêu +1 bài, streak +1 ngày.

**Tiêu chí nghiệm thu**
- [ ] Học xong bài hôm nay thì bài tiếp theo chỉ mở vào ngày hôm sau (sang ngày lúc 0h theo múi giờ ở F12).
- [ ] Nghỉ một hoặc nhiều ngày: hôm quay lại chỉ có một bài, streak về 0.
- [ ] Thẻ đến hạn vượt quá giới hạn ngày được chuyển sang hôm sau.
- [ ] Thoát giữa chừng rồi vào lại thì tiếp tục đúng bước và đúng câu đang làm.
- [ ] Lộ trình hết bài: người học thấy thông báo "chưa có bài mới", vẫn ôn được; quản trị viên thấy cảnh báo ở F2.

---

## F6. Màn hình chính và tiến độ

**Chức năng**
- **Thanh mục tiêu:** số bài đã xong trên N.
- **Thanh kỹ năng:** Nghe và Đọc (Viết và Nói hiện ở giai đoạn sau), mỗi thanh đếm số bài đã xong bước của kỹ năng đó.
- **Bài hôm nay:** tên bài, trình độ, chủ đề, thanh tiến trình các bước, nút **Tiếp tục: <bước>**.
- **Streak** (số ngày học liên tiếp) và số thẻ đến hạn ngày mai.
- **Thống kê:** số từ đã học, số câu đã chép chính tả, tỷ lệ đúng, số bài hoàn thành theo kỹ năng.

**Tiêu chí nghiệm thu**
- [ ] Mở app là thấy ngay thanh mục tiêu, bài hôm nay và nút Tiếp tục, không phải cuộn trên điện thoại 360px.
- [ ] Nút Tiếp tục đưa thẳng đến bước đang dở.
- [ ] Số liệu cập nhật ngay sau khi hoàn thành một bước.

---

## F12. Cài đặt

**Chức năng**
- Chế độ giao diện: **Sáng**, **Tối**, **Theo hệ thống** (mặc định).
- Số thẻ ôn tối đa mỗi ngày (mặc định 30).
- Múi giờ để tính ngày học (mặc định lấy từ trình duyệt).

**Tiêu chí nghiệm thu**
- [ ] Đổi chế độ giao diện có hiệu lực ngay, không cần tải lại trang.
- [ ] Cài đặt lưu theo tài khoản, đăng nhập thiết bị khác vẫn giữ nguyên.
- [ ] Màu và font đúng theo [design-system.md](../design-system.md) ở cả hai chế độ.

---

## F13. Sao lưu và xuất dữ liệu

**Chức năng**
- Tự sao lưu cơ sở dữ liệu mỗi ngày lúc 3h sáng, giữ 7 bản gần nhất.
- Nút **Xuất dữ liệu**: tải file JSON gồm sổ từ, lịch ôn, tiến độ, bài học của tài khoản.

**Tiêu chí nghiệm thu**
- [ ] Khôi phục được từ một bản sao lưu bằng một lệnh có hướng dẫn.
- [ ] Bản sao lưu thứ 8 tự xoá bản cũ nhất.
- [ ] File xuất chỉ chứa dữ liệu của tài khoản đang đăng nhập.

---

## Yêu cầu chung

- Web responsive, dùng tốt từ 360px.
- Không tốn phí: font OFL, AI gói miễn phí, TTS và từ điển chạy trên máy.
- Nội dung bài học ghi nguồn và giấy phép.
- Truy cập dữ liệu qua lớp repository (MongoDB hiện tại, có thể đổi sau).
