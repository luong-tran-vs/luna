# Luna: Tính năng MVP

> Phiên bản: v6 (2026-09-29). v6 chốt 3 câu hỏi mở: mục tiêu gồm thanh tổng và 4 thanh kỹ năng, AI sinh bài ở giai đoạn 2, thêm F12 Cài đặt và F13 Sao lưu.
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
5. **Máy không có GPU.** Tác vụ AI nặng chạy nền (bất đồng bộ) và lưu kết quả lại. Audio được sinh sẵn một lần rồi dùng lại.

## 4. Một ngày học

Đây là trải nghiệm cốt lõi của app. Các tính năng ở mục 5 đều phục vụ luồng này.

### 4.1. Màn hình chính

Vừa mở app, người học thấy ngay 3 thứ: đã đi được bao xa so với mục tiêu, bài của hôm nay, và đang ở bước nào trong bài.

```
┌────────────────────────────────┐
│ Luna               🔥 12 ngày  │
│                                │
│ Mục tiêu: Lộ trình A2 → B1     │
│ ████████░░░░░░░░░   12/30 bài  │   ← thanh mục tiêu
│                                │
│ Hôm nay: Bài 13, At the café   │
│  ✔─────✔─────●─────○─────○     │
│  Ôn   Đọc   Nghe  Nói   Viết   │
│ ██████░░░░░░░░░░░   2/5 bước   │   ← thanh tiến trình bài học
│                                │
│     [ Tiếp tục: Nghe ▶ ]       │
└────────────────────────────────┘
```

### 4.2. Quy tắc

**R1. Mục tiêu và thanh mục tiêu**
- Người học chọn một **lộ trình** gồm N bài học xếp theo thứ tự (mặc định 30 bài, tương đương khoảng 30 ngày).
- Thanh mục tiêu hiển thị số bài đã hoàn thành trên N.
- Bên dưới là **4 thanh nhỏ theo kỹ năng** Nghe, Nói, Đọc, Viết. Mỗi thanh đếm số bài đã hoàn thành bước của kỹ năng đó, để người học thấy kỹ năng nào đang bị bỏ lại. Thanh Nói và Viết chỉ hiện khi giai đoạn tương ứng đã làm xong.
- Đi hết lộ trình thì app chúc mừng và mời người học đặt mục tiêu mới.

**R2. Mỗi ngày một bài**
- Bài của hôm nay là bài **tiếp theo chưa hoàn thành** trong lộ trình.
- Học xong bài hôm nay thì bài kế tiếp chỉ mở vào ngày hôm sau. Trong lúc chờ, người học vẫn xem lại được bài cũ và ôn từ.
- Nghỉ một ngày thì bài **không bị dồn**: hôm sau vẫn chỉ học một bài, là bài kế tiếp. Chỉ có chuỗi ngày học (streak) bị reset.
- Một ngày được tính theo múi giờ của người học, sang ngày mới lúc 0h.

**R3. Ôn flashcard trước khi vào bài mới**
- Bước đầu tiên của mỗi bài là **Ôn**: ôn các thẻ từ vựng đã đến hạn theo lịch FSRS (xem F5).
- Phải ôn xong phần thẻ này thì bước Đọc mới mở.
- Mỗi ngày ôn tối đa **30 thẻ** (chỉnh được), để nghỉ vài ngày quay lại không bị dồn hàng trăm thẻ. Thẻ vượt giới hạn được chuyển sang hôm sau.
- Nếu không có thẻ nào đến hạn (ví dụ ngày học đầu tiên), bước Ôn tự đánh dấu hoàn thành.
- Vòng lặp này nối các bài học với nhau: từ lưu trong bài hôm nay sẽ xuất hiện ở bước Ôn của những ngày sau.

**R4. Thanh tiến trình bài học**
- Bài học gồm các bước làm theo thứ tự; bước sau chỉ mở khi bước trước xong.

| Bước | Hoàn thành khi | Có từ |
|---|---|---|
| Ôn | Ôn hết thẻ đến hạn trong giới hạn ngày | Giai đoạn 1 |
| Đọc | Đọc hết bài và bấm "Đã đọc xong" | Giai đoạn 1 |
| Nghe | Chép chính tả hết các câu. Câu sai vẫn tính là xong; tỷ lệ đúng được ghi vào thống kê | Giai đoạn 1 |
| Viết | Nộp bài viết, không cần chờ AI chấm xong | Giai đoạn 2 |
| Nói | Shadowing hết các câu | Giai đoạn 3 |

- Ở giai đoạn 1, thanh tiến trình chỉ có **Ôn → Đọc → Nghe**. Bước Viết và Nói được thêm vào khi làm xong giai đoạn tương ứng.
- Tiến độ được lưu liên tục. Thoát giữa chừng rồi quay lại thì tiếp tục đúng bước và đúng câu đang làm.

**R5. Hoàn thành bài hôm nay**
- Xong bước cuối thì thanh mục tiêu tăng thêm 1 bài và chuỗi ngày học (streak) tăng thêm 1.

**R6. Hết bài trong lộ trình**
- Nếu bài hôm nay chưa có sẵn: app báo cho quản trị viên để thêm bài trong trang quản trị (F2). Trong lúc chờ, người học vẫn làm bước Ôn được.

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
- Sinh audio cho từng câu bằng TTS chạy trên máy, lưu lại và dùng lại cho các lần sau.
- **Chú thích bài bằng AI:** khi tạo bài, gửi **một request** tới LLM để lấy các từ và cụm từ đáng học (ví dụ *give up*), kèm dạng gốc (*went → go*) và nghĩa tiếng Việt **theo đúng ngữ cảnh** trong bài. Kết quả được lưu vào cơ sở dữ liệu, sau đó không gọi lại nữa.
  - Chú thích chạy nền. Nếu AI lỗi hoặc hết lượt miễn phí, bài vẫn được tạo, và quản trị viên có thể bấm "Chú thích lại" sau.
  - Quản trị viên xem và sửa được phần chú thích trước khi bài được đưa vào lộ trình.
- Mỗi bài hiển thị trạng thái của audio và của phần chú thích (đang chạy, xong, lỗi), kèm nút chạy lại.
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
- Phát theo từng câu, có các nút câu trước, câu sau và lặp lại câu.
- Chỉnh tốc độ phát (0.5x đến 1.25x), ẩn hoặc hiện transcript.
- **Chép chính tả:** nghe một câu, gõ lại, rồi được so sánh từng từ để thấy chỗ sai và chỗ thiếu.

**F5. Sổ từ vựng và ôn tập**
- Mỗi thẻ gồm từ, phiên âm, nghĩa, câu ví dụ lấy từ bài học và audio.
- Người học sửa, xoá hoặc tự thêm thẻ được. Nghĩa do AI hay từ điển đưa ra có thể sai, nên việc sửa tay là cần thiết.
- Lịch ôn dùng thuật toán **FSRS** (mã nguồn mở, chính xác hơn SM-2).
- Có hai kiểu ôn: **nhìn từ đoán nghĩa**, và **nghe rồi gõ lại từ**. Sau mỗi thẻ, người học tự đánh giá mức Again, Hard, Good hoặc Easy.
- Bước Ôn ở đầu mỗi bài học dùng chính sổ từ này (R3). Ngoài ra, người học có thể vào ôn thêm bất cứ lúc nào.

**F6. Màn hình chính và tiến độ**
- Màn hình chính hiển thị thanh mục tiêu, bài hôm nay kèm thanh tiến trình, streak và nút "Tiếp tục" (mục 4.1).
- Thống kê: số từ đã học, số câu đã chép chính tả và tỷ lệ đúng, số bài hoàn thành theo từng kỹ năng.

**F12. Cài đặt**
- Chế độ giao diện: Sáng, Tối, hoặc Theo hệ thống (mặc định). Xem [design-system.md](design-system.md).
- Số thẻ ôn tối đa mỗi ngày (mặc định 30) và múi giờ dùng để tính ngày học.

**F13. Sao lưu và xuất dữ liệu**
- Tự động sao lưu MongoDB mỗi ngày, giữ lại 7 bản gần nhất.
- Nút "Xuất dữ liệu" tải về một file JSON gồm sổ từ, lịch ôn, tiến độ và bài học.

### Giai đoạn 2: AI chữ

**F7. AI sinh bài học**
- Là một nút trong trang quản trị (F2). Nhập chủ đề, trình độ CEFR và độ dài mong muốn; AI sinh bài đọc hoặc hội thoại.
- Quản trị viên xem lại và sửa trước khi lưu. Bài được đánh dấu nguồn là "AI sinh".

**F8. Viết**
- Mỗi bài học có sẵn đề viết (trả lời câu hỏi, tóm tắt, hoặc viết tiếp) để người học làm.
- AI nhận xét theo 4 tiêu chí: hoàn thành yêu cầu, ngữ pháp, từ vựng, mạch lạc. Kèm bản đã sửa và phần so sánh với bản gốc.
- Việc chấm chạy nền; người học có thể đi làm việc khác và quay lại xem khi có kết quả.

**F9. Hỏi AI về từ chưa được chú thích**
- Với từ hoặc cụm không có trong phần chú thích của bài (F2), người học bấm "Hỏi AI" để AI giải thích nghĩa trong đúng câu đang đọc, bằng tiếng Việt.
- Mỗi lần bấm tốn một request, nên chỉ gọi khi người học chủ động bấm, và kết quả được lưu lại.

### Giai đoạn 3: AI giọng nói

**F10. Nói (shadowing)**
- Nghe một câu, ghi âm lại ngay trên trình duyệt, rồi chuyển giọng nói thành chữ bằng Whisper.
- So với câu gốc và tô màu từ đọc đúng, đọc sai và bị bỏ sót.

**F11. Hội thoại nhập vai** *(có thể làm hoặc không)*
- Trò chuyện với AI theo một tình huống, dùng lại từ vựng của bài học.

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
| Cơ sở dữ liệu | **MongoDB**: bản Community tự chạy trong Docker, hoặc MongoDB Atlas gói M0 miễn phí (giới hạn 512MB). Có thể đổi sang cơ sở dữ liệu khác sau này, nên backend chỉ truy cập dữ liệu qua một lớp repository. File audio lưu trên ổ đĩa, không lưu trong cơ sở dữ liệu |
| Đọc thành giọng nói (TTS) | Piper hoặc Kokoro chạy trên CPU, sinh sẵn và lưu file |
| Giọng nói thành chữ (STT) | whisper.cpp ở chế độ server, dùng model `base.en` hoặc `small.en` trên CPU |
| Mô hình ngôn ngữ (LLM) | Chính: gói miễn phí của Gemini (ưu tiên model Flash-Lite vì hạn mức cao hơn). Dự phòng: Groq, OpenRouter hoặc Ollama chạy trên máy |
| Từ điển Anh–Việt | [minhqnd/dictionary](https://github.com/minhqnd/dictionary): file SQLite chạy offline, khoảng 357 nghìn mục từ, có IPA và ví dụ. Giấy phép dữ liệu CC BY-SA 4.0, phải ghi nguồn khi mở cho người khác |
| Triển khai | Docker Compose |

## 8. Câu hỏi còn mở

- [ ] Máy có bao nhiêu RAM? Câu trả lời quyết định chạy được mô hình Ollama và Whisper cỡ nào (chỉ ảnh hưởng giai đoạn 2 và 3).
- [ ] Điện thoại truy cập app từ bên ngoài bằng cách nào: Tailscale, Cloudflare Tunnel, hay chỉ dùng trong mạng nhà? (Chỉ cần chốt khi triển khai.)
- [ ] Có thêm câu hỏi hiểu bài ở bước Đọc và ghi chú ngữ pháp cho mỗi bài không? (Cả hai do AI sinh, nếu làm thì thuộc giai đoạn 2.)
