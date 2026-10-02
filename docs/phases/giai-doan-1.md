# Giai đoạn 1: Nền tảng

> Nguồn: [mvp-features.md](../mvp-features.md) v6, [design-system.md](../design-system.md) v1.

**Mục tiêu:** học được mỗi ngày với chu trình **Ôn → Đọc → Nghe**.
**AI:** chỉ dùng ở F2 (chú thích bài). AI lỗi thì mọi tính năng khác vẫn chạy.
**Thứ tự làm:** F0 → F1 → F2 → F3 → F4 → F5 → F14 → L → F6 → F12 → F13.

---

## F0. Khung dự án

**Chức năng**
- Dựng frontend, backend và cơ sở dữ liệu theo [architecture.md](../architecture.md); chạy toàn bộ bằng một lệnh.
- Khung giao diện: thanh trên cùng có tên **Luna**, trang chủ tạm, nút chọn giao diện **Sáng / Tối** (hiện chế độ đang dùng, bấm vào mở menu).
- Màu, font, cỡ chữ lấy từ token trong [design-system.md](../design-system.md); font được đóng gói kèm app.
- Backend có endpoint kiểm tra tình trạng (server và kết nối cơ sở dữ liệu); trang chủ tạm hiển thị tình trạng này.
- Cấu hình bằng biến môi trường, có file mẫu; có test mẫu và lint cho cả hai phía; có README hướng dẫn chạy.
- Chưa có tính năng nghiệp vụ, chưa có dịch vụ AI, TTS, STT (thêm khi tính năng cần).

**Tiêu chí nghiệm thu**
- [ ] Máy mới chỉ cần cài Docker, chạy một lệnh là mở được app trên trình duyệt.
- [ ] Trang chủ tạm hiện "Đã kết nối" khi backend và cơ sở dữ liệu chạy; tắt cơ sở dữ liệu thì hiện "Mất kết nối cơ sở dữ liệu", app không bị trắng trang.
- [ ] Chọn Sáng hoặc Tối có hiệu lực ngay (khi chưa chọn thì theo thiết bị) và được giữ sau khi tải lại trang (lưu trên trình duyệt cho tới khi có F12).
- [ ] Hiển thị đúng ở 360px, cả hai chế độ; màu và font đúng design-system; chữ tiếng Việt có dấu hiển thị đúng khi mất mạng (font không tải từ ngoài).
- [ ] Lệnh test và lint chạy được cho cả frontend và backend, không cần mạng.
- [ ] Repo không chứa bí mật; thiếu biến môi trường bắt buộc thì backend báo lỗi rõ ràng và dừng.
- [ ] Dừng backend (Ctrl+C hoặc dừng container) thì server tắt an toàn, không cắt ngang request đang xử lý.

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
- [ ] Đăng nhập được giữ 30 ngày trên thiết bị; sai mật khẩu 5 lần liên tiếp thì tạm khoá đăng nhập 15 phút.
- [ ] Khi đăng ký, múi giờ của trình duyệt được lưu vào tài khoản.

---

## F2. Trang quản trị bài học

**Chức năng**
- Thêm, sửa, xoá, xem trước bài học.
- Thông tin bài: tiêu đề, nội dung (dán văn bản), trình độ CEFR (A1–C2), chủ đề, nguồn, giấy phép.
- Khi lưu bài, hệ thống:
  1. Tách bài thành từng câu.
  2. Chạy nền **1 request** AI để chú thích các từ và cụm từ đáng học: dạng gốc (*went → go*) và nghĩa tiếng Việt theo ngữ cảnh.
- Mỗi bài hiện trạng thái của chú thích: *đang chạy*, *xong*, *lỗi*, kèm nút **Chạy lại**.
- *Cập nhật 2026-10-02:* bỏ sinh audio bằng TTS (Kokoro); mọi chỗ nghe dùng giọng đọc của trình duyệt.
- Quản trị viên xem và sửa được phần chú thích.
- **Lộ trình:** danh sách bài có thứ tự, kéo thả để đổi thứ tự.
- Cảnh báo khi lộ trình còn dưới 3 bài chưa học.

**Tiêu chí nghiệm thu**
- [ ] Lưu bài thành công ngay cả khi AI lỗi hoặc hết lượt; trạng thái chú thích là *lỗi* và bấm Chạy lại được.
- [ ] Sửa nội dung bài thì tách câu và chú thích được làm lại.
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
- Phát theo từng câu: câu trước, câu sau, lặp lại câu. Câu được đọc bằng giọng đọc của trình duyệt (cập nhật 2026-10-02, thay audio
  tạo sẵn); chữ ẩn sẵn, kèm dạng sóng minh hoạ chạy theo câu đang đọc.
- Tốc độ: 0.5x, 0.75x, 1x, 1.25x.
- Ẩn hoặc hiện transcript.
- **Chép chính tả:** nghe một câu, gõ lại, bấm Kiểm tra. So sánh từng từ và đánh dấu *đúng*, *sai* (kèm từ đúng), *thiếu*.

**Tiêu chí nghiệm thu**
- [ ] So sánh không phân biệt hoa thường và bỏ qua dấu câu (giữ dấu nháy trong từ như *don't*); từ gõ thừa tính là sai.
- [ ] Mỗi câu được nghe lại không giới hạn số lần trước khi kiểm tra.
- [ ] Bước Nghe hoàn thành khi đã kiểm tra hết các câu; câu sai vẫn tính là xong.
- [ ] Tỷ lệ đúng của mỗi bài được lưu để thống kê.
- [ ] Sai và thiếu phân biệt được mà không cần nhìn màu (gạch ngang, gạch chân chấm).

---

## F5. Sổ từ và ôn tập

**Chức năng**
- Thẻ gồm: từ, IPA, nghĩa, câu ví dụ từ bài học; nghe bằng giọng đọc của trình duyệt.
- Xem danh sách thẻ; sửa, xoá, tự thêm thẻ.
- Lịch ôn theo thuật toán **FSRS**.
- Hai kiểu ôn:
  - **Xem từ đoán nghĩa:** hiện từ, bấm để lật xem nghĩa.
  - **Nghe rồi gõ:** nghe từ, gõ lại từ.
- Sau mỗi thẻ, người học chọn: **Again**, **Hard**, **Good**, **Easy**.
- Ôn tự do được bất cứ lúc nào, ngoài bước Ôn bắt buộc.
- **Sổ từ theo ngày và theo bài:** danh sách thẻ nhóm theo ngày lưu ("Hôm nay", "Hôm qua", ngày cụ thể); lọc theo bài học.
- **Từ vựng của bài:** ở bước Đọc có mục "Từ vựng" liệt kê các từ và cụm từ đã được chú thích của bài (nghĩa, IPA, nút nghe). Mỗi từ có nút **Lưu**, và có nút **Lưu tất cả**; từ đã có trong sổ hiện dấu ✓.

**Tiêu chí nghiệm thu**
- [ ] "Lưu tất cả" chỉ thêm những từ chưa có trong sổ, không tạo thẻ trùng.
- [ ] Mục Từ vựng của bài không gọi AI (dùng chú thích đã có); bài chưa có chú thích thì hiện "Chưa có danh sách từ vựng".
- [ ] Nhóm theo ngày tính theo múi giờ của người học.
- [ ] Thẻ mới lưu hôm nay đến hạn ôn lần đầu vào hôm sau.
- [ ] Đánh giá của người học cập nhật ngày ôn tiếp theo đúng theo FSRS.
- [ ] Kiểu Nghe rồi gõ không phân biệt hoa thường.
- [ ] Sửa nghĩa trên thẻ không làm mất lịch ôn của thẻ.

---

## F14. Chủ đề và lộ trình theo trình độ

**Chức năng**
- Danh mục chủ đề: quản trị viên thêm, sửa, xoá chủ đề. Mỗi chủ đề có tên, trình độ (A1–C2), mô tả ngắn.
- Mỗi bài thuộc đúng một chủ đề (bắt buộc). Form bài chọn chủ đề từ danh sách, thay cho ô chủ đề gõ tự do và ô trình độ riêng.
- Mỗi chủ đề có **lộ trình riêng**, kéo thả để xếp thứ tự. Thay cho lộ trình chung duy nhất của F2.
- Danh sách bài lọc theo trình độ và chủ đề.
- Chuyển dữ liệu cũ: tạo chủ đề từ các cặp trình độ và chủ đề đang có; bài chưa có chủ đề vào chủ đề "Chung" cùng trình độ; lộ trình chung cũ được chia theo chủ đề, giữ thứ tự tương đối.

**Tiêu chí nghiệm thu**
- [ ] Lộ trình của một chủ đề chỉ chứa bài của chủ đề đó (cùng trình độ).
- [ ] Không xoá được chủ đề còn bài.
- [ ] Cảnh báo "dưới 3 bài chưa học" tính riêng cho từng lộ trình.
- [ ] Chuyển dữ liệu cũ không mất bài nào và không đổi thứ tự tương đối của các bài.

---

## L. Luồng một ngày học

**Chức năng**
- **Mục tiêu:** người học tự chọn trình độ, rồi chọn một chủ đề của trình độ đó; mục tiêu là hoàn thành lộ trình của chủ đề (F14). Xong thì mời chọn chủ đề khác cùng trình độ hoặc lên trình độ tiếp theo.
- **Đổi chủ đề hoặc trình độ:** có hiệu lực ngay nếu bài hôm nay chưa bắt đầu, nếu đã bắt đầu thì từ hôm sau. Tiến độ từng lộ trình lưu riêng.
- **Mỗi ngày một bài:** bài hôm nay là bài tiếp theo chưa hoàn thành trong lộ trình đang học.
- **Các bước của bài:** Ôn → Đọc → Nghe. Bước sau chỉ mở khi xong bước trước.
- **Bước Ôn:** ôn các thẻ đến hạn, tối đa 30 thẻ mỗi ngày (chỉnh ở F12). Không có thẻ đến hạn thì bước tự hoàn thành.
- **Hoàn thành bài:** thanh mục tiêu +1 bài, streak +1 ngày.
- **Danh sách bài học:** trang "Bài học" gồm bài hôm nay, các bài đã học (mở lại để đọc, nghe bất cứ lúc nào) và các bài sắp tới ở trạng thái khoá (chỉ hiện tên).

**Tiêu chí nghiệm thu**
- [ ] Chỉ học bài thuộc trình độ đang chọn; không có bài của trình độ khác xen vào.
- [ ] Đổi sang chủ đề khác rồi quay lại: học tiếp đúng bài đang dở; streak không đổi.
- [ ] Mở lại bài đã học không làm thay đổi tiến độ, streak hay mục tiêu.
- [ ] Bài sắp tới không mở được trước ngày của nó.
- [ ] Học xong bài hôm nay thì bài tiếp theo chỉ mở vào ngày hôm sau (sang ngày lúc 0h theo múi giờ ở F12).
- [ ] Nghỉ một hoặc nhiều ngày: hôm quay lại chỉ có một bài, streak về 0.
- [ ] Thẻ đến hạn vượt quá giới hạn ngày được chuyển sang hôm sau.
- [ ] Thoát giữa chừng rồi vào lại thì tiếp tục đúng bước và đúng câu đang làm.
- [ ] Lộ trình hết bài: người học thấy thông báo "chưa có bài mới", vẫn ôn được hoặc chọn chủ đề khác; quản trị viên thấy cảnh báo ở F14.

---

## F6. Màn hình chính và tiến độ

**Chức năng**
- **Thanh mục tiêu:** tên lộ trình (ví dụ "A1 · Gia đình") và số bài đã xong trên tổng số bài của lộ trình.
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
- Chế độ giao diện: **Sáng** hoặc **Tối** (khi chưa chọn thì theo thiết bị).
- Số thẻ ôn tối đa mỗi ngày: từ 5 đến 200 (mặc định 30).
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
- [x] Khôi phục được từ một bản sao lưu bằng một lệnh có hướng dẫn.
- [x] Bản sao lưu thứ 8 tự xoá bản cũ nhất.
- [x] File xuất chỉ chứa dữ liệu của tài khoản đang đăng nhập.

---

## Yêu cầu chung

- Web responsive, dùng tốt từ 360px.
- Không tốn phí: font OFL, AI gói miễn phí, giọng đọc của trình duyệt, từ điển chạy trên máy.
- Nội dung bài học ghi nguồn và giấy phép.
- Truy cập dữ liệu qua lớp repository (MongoDB hiện tại, có thể đổi sau).
