# Luna: Tính năng MVP

> Phiên bản: v10 (2026-10-01). v10: chốt giai đoạn 3: thêm F16 Vận hành (đặt lại mật khẩu, HTTPS qua Tailscale), bước Nói bắt buộc, Whisper small.en (máy 16GB RAM).
> v9 (2026-09-30): chốt giai đoạn 2: F7 sinh nhiều bài trong lộ trình chủ đề, thêm F15 câu hỏi hiểu bài và ghi chú ngữ pháp, bước Viết bắt buộc.
> v8: học theo chủ đề: mỗi cặp trình độ và chủ đề là một lộ trình riêng (R1, R6), thêm F14 Chủ đề và lộ trình theo trình độ.
> v7 (2026-09-29) thêm cho người học: trang Bài học (R2), mục Từ vựng của bài và sổ từ theo ngày/bài (F5).
> v6 chốt 3 câu hỏi mở: mục tiêu gồm thanh tổng và 4 thanh kỹ năng, AI sinh bài ở giai đoạn 2, thêm F12 Cài đặt và F13 Sao lưu.
> v5 chốt giao diện: bảng màu Oải hương, font Lexend, có chế độ tối (chi tiết ở [design-system.md](design-system.md)).
> v4 thêm trang quản trị để tự thêm bài học (F2), mục tiêu theo dõi theo 4 kỹ năng (R1), cơ sở dữ liệu dùng MongoDB (mục 7), quy trình phát triển dùng spec-kit (thư mục gốc dự án).
> v3 chốt nguồn nghĩa tiếng Việt: AI chú thích một lần khi tạo bài, từ điển SQLite offline làm dự phòng (F2, F3, F5, F9, mục 7).
> v2 thêm mục 4 "Một ngày học": thanh mục tiêu, mỗi ngày một bài, thanh tiến trình bài học, ôn flashcard trước khi học bài mới.
> Đây là **nguồn sự thật** về sản phẩm. Tài liệu kỹ thuật, spec hay task mâu thuẫn với file này thì sửa theo file này.

## 1. Tầm nhìn

Ứng dụng web học tiếng Anh luyện đủ **4 kỹ năng Nghe, Nói, Đọc, Viết** xoay quanh **một nội dung duy nhất cho mỗi bài học**. Người học đọc, nghe, nhại theo và viết về cùng một văn bản, nên gặp lại cùng bộ từ vựng qua nhiều kênh. Các từ đã lưu được đưa vào sổ từ để ôn tập ngắt quãng (SRS).

```
        ┌─ Đọc  : đọc bài, bấm để tra từ và lưu từ
        ├─ Nghe : nghe theo từng câu, chép chính tả
Bài học ┤
        ├─ Nói  : shadowing, chấm đúng sai theo từng từ
        └─ Viết : trả lời hoặc tóm tắt, AI nhận xét
                 │
                 ▼
        Sổ từ vựng + ôn tập ngắt quãng (SRS)
```

## 2. Người dùng và bối cảnh

- **Hiện tại:** một người dùng (chủ dự án) vừa là **người học**, vừa là **quản trị viên** tự soạn bài học trong trang quản trị.
- **Tương lai:** có thể mở cho người khác. Vì vậy mọi dữ liệu cá nhân đều gắn `user_id` ngay từ đầu. Quyền chỉ có hai mức (người học, quản trị viên); chưa làm phân quyền chi tiết hay thanh toán.
- **Thiết bị:** web **responsive**, dùng tốt trên điện thoại (màn hình từ 360px) lẫn máy tính.

## 3. Nguyên tắc sản phẩm

1. **Chi phí 0 đồng là ưu tiên.** AI dùng mô hình chạy trên máy hoặc gói miễn phí. Không có tính năng nào bắt buộc phải có dịch vụ trả phí.
2. **AI là phần bổ sung, không phải điều kiện để app chạy.** Khi AI lỗi, chậm hoặc hết lượt miễn phí, các tính năng Đọc, Nghe và Ôn từ vẫn hoạt động bình thường.
3. **Nhà cung cấp AI thay được.** Mọi lời gọi AI đi qua một lớp trung gian chung, nên đổi từ Gemini sang Ollama hay dịch vụ khác chỉ cần sửa cấu hình.
4. **Nội dung ghi rõ nguồn và giấy phép.** Ưu tiên nguồn cho phép tái sử dụng để sau này mở cho người khác không vướng bản quyền.
5. **Máy không có GPU.** Tác vụ AI nặng chạy nền (bất đồng bộ) và lưu kết quả lại. Phần nghe trên giao diện dùng giọng đọc của trình duyệt (Web Speech API), không chờ audio sinh sẵn.

## 4. Một ngày học

Đây là trải nghiệm cốt lõi của app. Các tính năng ở mục 5 đều phục vụ luồng này.

### 4.1. Màn hình chính

Vừa mở app, người học thấy ngay 3 thứ: đã đi được bao xa so với mục tiêu, bài đang học, và đang ở bước nào trong bài.

```
┌────────────────────────────────┐
│ Luna               🔥 12 ngày  │
│                                │
│ Mục tiêu: A1 · Gia đình        │
│ ████████░░░░░░░░░   12/30 bài  │   ← thanh mục tiêu
│                                │
│ Đang học: Bài 13, At the café  │
│  ✔─────●─────○─────○           │
│  Đọc  Nghe  Viết  Nói          │
│ ██████░░░░░░░░░░░   1/4 bước   │   ← thanh tiến trình bài học
│                                │
│     [ Tiếp tục: Nghe ▶ ]       │
└────────────────────────────────┘
```

### 4.2. Quy tắc

**R1. Mục tiêu và thanh mục tiêu**
- Người học tự chọn **trình độ** (A1–C2), rồi chọn **một chủ đề** của trình độ đó. Mỗi cặp trình độ và chủ đề là một **lộ trình** riêng, thứ tự bài do quản trị viên xếp (F14).
- Mục tiêu là hoàn thành lộ trình đã chọn. Thanh mục tiêu ghi tên lộ trình (ví dụ "A1 · Gia đình") và số bài đã hoàn thành trên tổng số bài của lộ trình.
- Chỉ học bài thuộc trình độ đang chọn, không xen kẽ các trình độ.
- Đổi chủ đề hoặc trình độ: có hiệu lực ngay (cập nhật 2026-10-02). Tiến độ của từng lộ trình được lưu riêng, quay lại thì học tiếp bài đang dở. Streak không bị ảnh hưởng.
- Bên dưới là **4 thanh nhỏ theo kỹ năng** Nghe, Nói, Đọc, Viết. Mỗi thanh là **độ chính xác** của kỹ năng đó (cập nhật 2026-10-05, trước đây đếm số bài đã qua bước, mà bước Đọc và Nghe bắt buộc nên các thanh gần như luôn bằng nhau): Đọc = tỷ lệ trả lời đúng câu hỏi hiểu bài, Nghe = tỷ lệ từ đúng khi chép chính tả, Viết = điểm trung bình bài viết trên 5, Nói = tỷ lệ từ đọc đúng (khi có). Kỹ năng chưa có dữ liệu ghi "Chưa có". Nhờ vậy người học thấy kỹ năng nào đang yếu. Thanh Nói và Viết chỉ hiện khi giai đoạn tương ứng đã làm xong.
- Đi hết lộ trình thì app chúc mừng và mời chọn chủ đề khác cùng trình độ, hoặc lên trình độ tiếp theo.

**R2. Học lần lượt từng bài** (cập nhật 2026-10-02, thay cho "mỗi ngày một bài")
- Bài đang học là bài **tiếp theo chưa hoàn thành** trong lộ trình. Không giới hạn số bài mỗi ngày: học xong bài 1 thì bài 2 mở ngay.
- Không có trang "Hôm nay" riêng: người học học ngay trong trang chi tiết của bài (F17).
- Trang **Bài học** liệt kê các bài đã học (mở lại tự do), bài đang học và các bài sắp tới (khoá, chỉ hiện tên). Bài nào có trong lộ trình thì hiện bài đó, không có ô "bài hôm nay" trống.
- Streak đếm số ngày liên tiếp có hoàn thành ít nhất một bài. Một ngày được tính theo múi giờ của người học, sang ngày mới lúc 0h.
- **Một ngày nghỉ được tha** (cập nhật 2026-10-05): bỏ lỡ đúng một ngày giữa hai ngày học thì chuỗi không đứt (ngày nghỉ không được tính vào số ngày). Lần tha tiếp theo cần ít nhất 7 ngày học kể từ lần tha trước. Bỏ lỡ hai ngày liền thì chuỗi về 0. Không lưu trạng thái, quy tắc chỉ đọc các ngày đã xong bài.

**R3. Ôn flashcard** (cập nhật 2026-10-02)
- Ôn thẻ đến hạn theo lịch FSRS (xem F5) không còn là bước của bài. Người học ôn ở mục **Từ vựng › Ôn tập** bất cứ lúc nào; trang chủ nhắc số thẻ đến hạn hôm nay (thẻ nổi bật trên bài đang học) và ngày mai.
- **Ôn trước bài mới** (cập nhật 2026-10-05): vì học bao nhiêu bài một ngày cũng được, xong bài mà còn thẻ đến hạn thì thẻ hoàn thành bài mời ôn trước (nút chính), nói rõ khi thẻ tồn nhiều (từ 30 thẻ); **Sang bài tiếp theo** vẫn một chạm. Không chặn người học.
- **Lỗi sai vào lịch ôn** (cập nhật 2026-10-05): từ làm sai ở phần luyện tập của bài đang học (điền ô trống, câu dịch) thành thẻ đến hạn ngay, hoặc thẻ đã có được kéo về bây giờ (giữ trạng thái FSRS).
- Vòng lặp vẫn nối các bài học với nhau: từ lưu trong bài sẽ đến hạn ôn ở những ngày sau.

**R4. Thanh tiến trình bài học**
- Bài học gồm các bước làm theo thứ tự; bước sau chỉ mở khi bước trước xong.

| Bước | Hoàn thành khi | Có từ |
|---|---|---|
| Đọc | Đọc hết bài và bấm "Đã đọc xong" | Giai đoạn 1 |
| Nghe | Chép chính tả hết các câu. Câu sai vẫn tính là xong; tỷ lệ đúng được ghi vào thống kê | Giai đoạn 1 |
| Viết | Nộp bài viết, không cần chờ AI chấm xong | Giai đoạn 2 |
| Nói | Shadowing hết các câu | Giai đoạn 3 |

- Ở giai đoạn 1, thanh tiến trình chỉ có **Đọc → Nghe** (bước Ôn bỏ từ 2026-10-02). Bước Viết và Nói được thêm vào khi làm xong giai đoạn tương ứng.
- Tiến độ được lưu liên tục. Thoát giữa chừng rồi quay lại thì tiếp tục đúng bước và đúng câu đang làm.

**R5. Hoàn thành bài**
- Xong bước cuối thì thanh mục tiêu tăng thêm 1 bài, ngày hôm đó được tính vào chuỗi ngày học (streak) và bài kế tiếp mở ngay; trang bài có nút **Sang bài tiếp theo**.

**R6. Hết bài trong lộ trình**
- Nếu lộ trình đang học chưa có bài tiếp theo: app báo cho quản trị viên để thêm bài (F2, F14). Trong lúc chờ, người học vẫn ôn được hoặc chọn chủ đề khác.

## 5. Tính năng theo giai đoạn

### Giai đoạn 1: nền tảng

Giai đoạn này chỉ dùng AI ở một chỗ: chú thích nghĩa khi tạo bài học (F2). Nếu AI lỗi thì mọi thứ vẫn chạy, chỉ là nghĩa lấy từ từ điển (F3).

**F1. Tài khoản**
- Đăng ký và đăng nhập bằng email và mật khẩu. Một người dùng là đủ, nhưng dữ liệu vẫn tách riêng theo từng người.
- Tài khoản có vai trò **người học** hoặc **quản trị viên**. Chỉ quản trị viên mới vào được trang quản trị (F2).

**F2. Trang quản trị bài học**
- Trang riêng dành cho quản trị viên để thêm, sửa, xoá và xem trước bài học. Trang này cũng phải dùng được trên điện thoại.
- Thêm bài bằng cách dán văn bản vào. Thông tin gồm tiêu đề, trình độ CEFR (A1–C2), chủ đề, nguồn và giấy phép.
- Tự động tách bài thành từng câu.
- Không sinh audio: các câu được đọc bằng giọng đọc của trình duyệt (bỏ Kokoro từ 2026-10-02).
- **Chú thích bài bằng AI:** khi tạo bài, gửi **một request** tới LLM để lấy các từ và cụm từ đáng học (ví dụ *give up*), kèm dạng gốc (*went → go*) và nghĩa tiếng Việt **theo đúng ngữ cảnh** trong bài. Kết quả được lưu vào cơ sở dữ liệu, sau đó không gọi lại nữa.
  - Chú thích chạy nền. Nếu AI lỗi hoặc hết lượt miễn phí, bài vẫn được tạo, và quản trị viên có thể bấm "Chú thích lại" sau.
  - Quản trị viên xem và sửa được phần chú thích trước khi bài được đưa vào lộ trình.
- Mỗi bài hiển thị trạng thái của phần chú thích (đang chạy, xong, lỗi), kèm nút chạy lại.
  - Với khoảng 1 bài mỗi ngày, mỗi ngày chỉ tốn khoảng 1–3 request. Mức này thấp hơn nhiều so với hạn mức miễn phí (thấp nhất khoảng 20 request/ngày).
- Danh sách bài học lọc được theo trình độ, chủ đề và trạng thái (chưa học, đang học, đã xong).
- **Lộ trình:** quản trị viên xếp các bài vào một danh sách có thứ tự (kéo thả để đổi thứ tự) để app lấy ra làm bài của mỗi ngày (xem R1, R2).
- Trang quản trị hiện cảnh báo khi lộ trình chỉ còn ít bài chưa học (ví dụ dưới 3 bài), để quản trị viên kịp thêm bài.

**F3. Đọc**
- Hiển thị bài; bấm vào một từ thì mở popup gồm phiên âm, nghĩa, nút nghe phát âm và nút lưu vào sổ từ.
- Bôi đen nhiều từ liền nhau để tra cả cụm (ví dụ *look forward to*).
- **Thứ tự lấy nghĩa khi tra:**
  1. Phần chú thích AI của bài, nếu có từ hoặc cụm đó: hiện nghĩa theo ngữ cảnh.
  2. Nếu không có, tra từ điển Anh–Việt offline (file SQLite): hiện các nghĩa kèm IPA và ví dụ.
  3. Cả hai đều không có: người học tự nhập nghĩa.
- Popup luôn ghi rõ nghĩa lấy từ đâu ("AI – theo ngữ cảnh" hay "Từ điển").
- Khi lưu từ, **câu chứa từ đó** cũng được lưu làm ngữ cảnh.
- Từ đã có trong sổ được tô màu khi gặp lại trong bất kỳ bài nào.

**F4. Nghe**
- Phát theo từng câu, có các nút câu trước, câu sau và lặp lại câu; câu do giọng đọc của trình duyệt đọc, kèm dạng sóng minh hoạ.
- Chỉnh tốc độ phát (0.5x đến 1.25x), ẩn hoặc hiện transcript.
- **Chép chính tả:** nghe một câu, gõ lại, rồi được so sánh từng từ để thấy chỗ sai và chỗ thiếu.

**F5. Sổ từ vựng và ôn tập**
- Mỗi thẻ gồm từ, phiên âm, nghĩa, câu ví dụ lấy từ bài học; nghe được bằng giọng đọc của trình duyệt.
- Người học sửa, xoá hoặc tự thêm thẻ được. Nghĩa do AI hay từ điển đưa ra có thể sai, nên việc sửa tay là cần thiết.
- Lịch ôn dùng thuật toán **FSRS** (mã nguồn mở, chính xác hơn SM-2).
- Có hai kiểu ôn: **nhìn từ đoán nghĩa**, và **nghe rồi gõ lại từ**. Sau mỗi thẻ, người học tự đánh giá mức Quên, Khó, Nhớ hoặc Dễ (tương ứng Again, Hard, Good, Easy của FSRS; nhãn tiếng Việt từ 2026-10-05).
- Người học ôn sổ từ này ở mục Từ vựng › Ôn tập bất cứ lúc nào (R3).
- Sổ từ nhóm theo ngày lưu và lọc theo bài học.
- **Từ vựng của bài:** bước Đọc có mục liệt kê các từ đã được AI chú thích, lưu từng từ hoặc "Lưu tất cả".

**F6. Màn hình chính và tiến độ**
- Màn hình chính hiển thị thanh mục tiêu, bài đang học kèm thanh tiến trình, streak và nút "Tiếp tục" (mục 4.1).
- Thống kê: số từ đã học, số câu đã chép chính tả và tỷ lệ đúng, số bài hoàn thành theo từng kỹ năng.

**F12. Cài đặt**
- Chế độ giao diện: Sáng hoặc Tối (khi chưa chọn thì theo thiết bị). Xem [design-system.md](design-system.md).
- Số thẻ ôn tối đa mỗi ngày (mặc định 30) và múi giờ dùng để tính ngày học.

**F13. Sao lưu và xuất dữ liệu**
- Tự động sao lưu MongoDB mỗi ngày, giữ lại 7 bản gần nhất.
- Nút "Xuất dữ liệu" tải về một file JSON gồm sổ từ, lịch ôn, tiến độ và bài học.

**F14. Chủ đề và lộ trình theo trình độ**
- Quản trị viên quản lý danh mục chủ đề; mỗi chủ đề thuộc một trình độ (ví dụ "A1 · Gia đình").
- Mỗi bài thuộc đúng một chủ đề; trình độ của bài là trình độ của chủ đề.
- Mỗi chủ đề có lộ trình riêng, kéo thả để xếp thứ tự. Thay cho lộ trình chung duy nhất ở F2.

### Giai đoạn 2: AI chữ

**F7. AI sinh bài học**
- Nút trong trang lộ trình của một chủ đề (F14): trình độ và chủ đề lấy sẵn. Chọn số bài (1–10), độ dài, dạng bài (bài đọc hoặc hội thoại); AI tránh lặp nội dung đã có.
- Quản trị viên duyệt từng bản nháp rồi lưu; bài có nguồn "AI sinh" và được thêm vào cuối lộ trình.

**F15. Câu hỏi hiểu bài và ghi chú ngữ pháp**
- Cùng request chú thích bài, AI sinh thêm 3–5 câu hỏi trắc nghiệm, một ghi chú ngữ pháp tiếng Việt và đề viết cho F8.
- Bước Đọc hoàn thành khi trả lời hết câu hỏi; bài chưa có câu hỏi thì dùng nút "Đã đọc xong".

**F8. Viết**
- Bước Viết sau bước Nghe, **tuỳ chọn** (cập nhật 2026-10-02): viết và nộp thì được AI chấm; bấm **Bỏ qua** thì bài vẫn hoàn thành. Đề viết do AI sinh cùng lúc chú thích (F15), quản trị viên sửa được.
- Trong trang bài có nút **← Bước trước** để xem lại các bước đã xong (Đọc, Nghe), rồi quay về bước đang làm.
- AI nhận xét theo 4 tiêu chí: hoàn thành yêu cầu, ngữ pháp, từ vựng, mạch lạc. Kèm bản đã sửa và phần so sánh với bản gốc.
- Việc chấm chạy nền; người học có thể đi làm việc khác và quay lại xem khi có kết quả.

**F9. Hỏi AI về từ chưa được chú thích**
- Với từ hoặc cụm không có trong phần chú thích của bài (F2), người học bấm "Hỏi AI" để AI giải thích nghĩa trong đúng câu đang đọc, bằng tiếng Việt.
- Mỗi lần bấm tốn một request, nên chỉ gọi khi người học chủ động bấm, và kết quả được lưu lại.

### Giai đoạn 3: AI giọng nói

**F16. Vận hành: đặt lại mật khẩu và truy cập từ điện thoại**
- Lệnh đặt lại mật khẩu và liệt kê tài khoản chạy trong container backend.
- Truy cập từ điện thoại qua HTTPS bằng Tailscale; HTTPS cần để dùng micro ở F10.

**F10. Nói (shadowing)**
- Bước Nói bắt buộc sau bước Viết: nghe một câu, ghi âm lại ngay trên trình duyệt, rồi chuyển giọng nói thành chữ bằng Whisper.
- So với câu gốc và tô màu từ đọc đúng, đọc sai và bị bỏ sót.

**F11. Hội thoại nhập vai** *(không bắt buộc)*
- Trò chuyện với AI theo tình huống của bài (tối đa 10 lượt), trả lời bằng giọng nói hoặc gõ chữ; câu của AI được đọc bằng giọng đọc của trình duyệt; cuối phiên tóm tắt lỗi.

## 6. Ngoài phạm vi MVP

- App native iOS/Android (thay bằng web responsive, sau này có thể nâng lên PWA).
- Chấm phát âm tới từng âm vị như ELSA.
- Bảng xếp hạng, tính năng xã hội, thanh toán, phân quyền chi tiết (ngoài hai vai trò người học và quản trị viên).
- Tự động lấy nội dung từ website có bản quyền.
- Bài test xếp trình độ đầu vào (người học tự chọn trình độ khi tạo lộ trình).
- Nhắc học hằng ngày bằng thông báo đẩy (cần PWA).

## 7. Hướng kỹ thuật (tóm tắt)

| Thành phần | Lựa chọn |
|---|---|
| Frontend | Angular, responsive, có chế độ sáng và tối. Màu và font theo [design-system.md](design-system.md): bảng màu Oải hương, font Lexend, IPA dùng Noto Sans |
| Backend | Go (REST API) |
| Cơ sở dữ liệu | **MongoDB**: bản Community tự chạy trong Docker, hoặc MongoDB Atlas gói M0 miễn phí (giới hạn 512MB). Có thể đổi sang cơ sở dữ liệu khác sau này, nên backend chỉ truy cập dữ liệu qua một lớp repository |
| Đọc thành giọng nói (TTS) | Giọng đọc của trình duyệt (Web Speech API) cho mọi chỗ nghe; không có dịch vụ TTS ở backend (bỏ Kokoro từ 2026-10-02) |
| Giọng nói thành chữ (STT) | whisper.cpp ở chế độ server trên CPU, model **`small.en`** (khoảng 1GB RAM; máy có 16GB). Dự phòng `base.en` nếu chậm |
| Mô hình ngôn ngữ (LLM) | Chính: gói miễn phí của Gemini (ưu tiên model Flash-Lite vì hạn mức cao hơn). Dự phòng: Groq, OpenRouter hoặc Ollama chạy trên máy |
| Từ điển Anh–Việt | [minhqnd/dictionary](https://github.com/minhqnd/dictionary): file SQLite chạy offline, khoảng 357 nghìn mục từ, có IPA và ví dụ. Giấy phép dữ liệu CC BY-SA 4.0, phải ghi nguồn khi mở cho người khác |
| Triển khai | Docker Compose |

## 8. Câu hỏi còn mở

- [x] Điện thoại truy cập app từ bên ngoài: **Tailscale** (`tailscale serve`, HTTPS, chỉ thiết bị của mình), xem F16.
