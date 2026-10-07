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
  3. **Điền vào ô trống:** hội thoại có ô trống ở từ vựng; mỗi ô là ô nhập chữ, gõ từ vào (cập nhật 2026-10-02), hoặc bấm từ
     trong ngân hàng từ (gợi ý) để điền vào ô đang chọn; Kiểm tra, mẹo ngữ pháp.
  4. **Dịch câu sang tiếng Anh:** ghép câu bằng các ô từ (có từ gây nhiễu), Làm lại, Kiểm tra từng câu.
- Tổng kết cuối cùng, làm lại được. Không có điểm XP.
- **Học bài ngay trong trang này** (cập nhật 2026-10-02, thay trang "Hôm nay"): với bài đang học, thứ tự là **Từ vựng → Đọc → Nghe → Hội thoại →
  Điền ô trống → Dịch câu → Viết** (cập nhật 2026-10-05: hiểu bài trước, rồi mới luyện lại; Viết tuỳ chọn, có **Bỏ qua**), cùng một thanh tiến độ; nút **← Bước trước** để xem lại bước đã qua. Vào lại
  bài thì tiếp tục đúng bước đang dở. Xong bước cuối thì bài hoàn thành và có nút **Sang bài tiếp theo** (hoặc chúc mừng khi hết lộ trình).
  Bài đã học thì chỉ có phần luyện tập, tổng kết có **Đọc lại / Nghe lại / Bài viết**.
- Quản trị viên xem phần luyện tập và bấm **Tạo lại phần luyện tập**.

**Tiêu chí nghiệm thu**
- [ ] Bài mới chú thích xong thì sau ít phút có phần luyện tập; mỗi câu ví dụ chứa đúng từ của nó; ô trống và đáp án đều là từ vựng
  của bài.
- [ ] Nghe được từng từ, từng câu ví dụ, từng lượt hội thoại và cả đoạn; đổi được tốc độ; bật/tắt được nghĩa tiếng Việt.
- [ ] Điền vào ô trống: gõ được vào từng ô (Enter sang ô kế), bấm từ gợi ý thì điền vào ô đang chọn, xoá/sửa được; Kiểm tra thấy đúng/sai từng ô (có chữ, không chỉ màu) và đáp án.
- [ ] Dịch câu: ghép đúng thứ tự thì "Chính xác!", sai thì thấy câu đúng; Làm lại xoá câu đang ghép.
- [ ] Thanh tiến độ và nút Tiếp theo chuyển đúng 4 bước; cuối cùng thấy tổng kết và làm lại được.
- [ ] AI lỗi khi sinh phần luyện tập: chú thích từ, câu hỏi hiểu bài vẫn bình thường. Trang chi tiết bài chỉ hiện các bước có nội dung
  (thanh tiến độ theo số bước thật); không có bước nào thì không có tab Bài học, chỉ hiện bài đọc (cập nhật 2026-10-02).
- [ ] Sửa nội dung bài thì phần luyện tập được sinh lại theo nội dung mới.
- [ ] Kết quả luyện tập (điền, dịch) không tính vào tiến độ, streak, thống kê. Bài sắp tới vẫn không mở được.
- [ ] Bài đang học (cập nhật 2026-10-05): từ làm sai ở bước điền ô trống hoặc có trong câu dịch làm sai trở thành thẻ đến hạn ngay (hoặc thẻ đã có được kéo về bây giờ, giữ trạng thái FSRS), mỗi từ một lần, trang ghi số từ đã đưa vào lịch ôn; lỗi gửi thì bỏ qua. Bài không phải bài đang học không gửi gì.
- [ ] Xong bài mà còn thẻ đến hạn: thẻ hoàn thành bài mời "Ôn trước khi học tiếp" (số thẻ, thời gian ước tính, nói rõ khi từ 30 thẻ), vẫn có **Sang bài tiếp theo**; trang Ôn tập mở từ đó có nút **Học bài tiếp theo**. Không lấy được số thẻ thì không hiện gợi ý.
- [ ] Trang chủ: thẻ "Hôm nay: N thẻ cần ôn" khi có thẻ đến hạn; ba thanh kỹ năng là độ chính xác (Đọc, Nghe, Viết trên 5), "Chưa có" khi thiếu dữ liệu.
- [ ] Streak: nghỉ đúng một ngày thì chuỗi giữ, lần tha sau cách ít nhất 7 ngày học, nghỉ hai ngày liền thì về 0.
- [ ] Bài đang học: Từ vựng → Đọc → Nghe → Hội thoại → Điền ô trống → Dịch câu → Viết trong cùng trang; bài chưa có luyện tập thì Đọc → Nghe → Viết. Hội thoại, ô trống và câu dịch bám sát bài đọc (cùng tình huống, dùng lại câu của bài), không phải một cảnh khác (cập nhật 2026-10-05).
  Thoát giữa chừng rồi vào lại thì tiếp tục đúng bước. Xong (nộp hoặc bỏ qua bước Viết) thì bấm **Sang bài tiếp theo** mở được ngay
  bài kế tiếp, cùng ngày.
- [ ] Dùng tốt ở 360px, chế độ sáng và tối.

---

## F18. Từ vựng theo chủ đề *(quản trị)*

> **Sửa 2026-10-07**: mỗi từ có trình độ (hoặc để trống); sinh bài trình độ X chỉ giao từ có trình độ ≤ X hoặc để trống.
> Xem `specs/017-topic-vocabulary/spec.md`.

> Thêm 2026-10-02. Dữ liệu ban đầu: [f18-topic-words.json](../spec-inputs/f18-topic-words.json), 1.257 từ tiếng Anh cho 42 chủ đề,
> tham khảo danh sách của Langmaster (chỉ lấy từ tiếng Anh). Không phụ thuộc F16/F10/F11; làm sau F17 thì trang chi tiết bài dùng ngay.

**Chức năng**
- Mỗi chủ đề có danh sách từ vựng tiếng Anh cốt lõi; nạp sẵn từ file dữ liệu một lần, quản trị viên xem, thêm, xoá ở trang Chủ đề.
- Trang Chủ đề hiện độ phủ "Từ vựng: đã dùng X/Y" (từ có trong bài của chủ đề) và đánh dấu từ đã dùng/chưa dùng.
- Sinh bài bằng AI: mỗi bài được giao một nhóm từ mục tiêu, ưu tiên từ chưa dùng; quản trị viên chỉnh nhóm từ trước khi sinh; bản
  nháp báo số từ mục tiêu đã dùng và từ còn thiếu.
- Thiếu từ chưa dùng: nút "Bổ sung bằng AI" gợi ý từ mới cho chủ đề rồi chia nhóm lại (thêm 2026-10-02).
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

- **L. Luồng học** (cập nhật 2026-10-02): bỏ trang "Hôm nay" và giới hạn một bài mỗi ngày; học lần lượt từng bài ngay trong trang bài (F17): Từ vựng, Đọc → Nghe, các phần luyện tập có nội dung (hội thoại, điền ô trống, dịch câu; đổi thứ tự 2026-10-05, trước đây đứng trước bước Đọc), rồi Viết (tuỳ chọn). Học xong bài thì bài kế tiếp mở ngay. Bỏ bước Ôn khỏi bài (ôn ở Từ vựng › Ôn tập). Trang Khóa học: nút ở thẻ "Khóa học đang học" mở bài đang học; chủ đề đã hết bài mới thì nút **Xem lại bài gần nhất** mở bài học xong gần nhất. Sau này thêm bước Nói (bắt buộc) sau bước Viết.
- **F6:** hiện thêm thanh kỹ năng **Nói**; thống kê thêm tỷ lệ đọc đúng.
- **F13:** file xuất dữ liệu có thêm kết quả bước Nói và lịch sử hội thoại.

---

## F19. Giáo trình ngữ pháp theo trình độ *(quản trị)*

> Thêm 2026-10-05, từ một cuộc thảo luận về việc đặt ngữ pháp trong bài học. Quyết định: làm bước nhỏ trước (mỗi bài được gán một điểm của
> giáo trình), chưa làm lộ trình ngữ pháp riêng. Không phụ thuộc F16/F10/F11.

**Vấn đề:** AI tự chọn điểm ngữ pháp cho mỗi bài nên bài nào cũng ra "thì hiện tại đơn", không có tiến trình trong một trình độ.

**Chức năng**
- **Giáo trình** (`backend/internal/grammar/syllabus.json`, nhúng vào binary): 67 điểm xếp theo thứ tự dạy (A1 15, A2 14, B1 13, B2 11, C1 8,
  C2 6). Mỗi điểm có mã ổn định (ví dụ `a1-to-be`), tên tiếng Việt và tiếng Anh, cấu trúc mẫu, gợi ý một dòng, ví dụ. Sửa file này để thêm
  hoặc đổi thứ tự điểm; không đổi hay dùng lại mã đã gán cho bài.
- Mỗi bài có tối đa **một** điểm, thuộc đúng trình độ của chủ đề (đổi sang chủ đề khác trình độ thì điểm cũ bị bỏ, hoặc bị từ chối khi sửa bài).
- **Sinh bài bằng AI:** hộp thoại có ô "Điểm ngữ pháp" (mặc định "AI tự chọn", chọn điểm cụ thể là tuỳ chọn); điểm được dùng cho mọi bài
  trong lượt sinh, và AI phải dùng điểm đó rõ ràng trong bài.
- **Chú thích:** ghi chú ngữ pháp của bài dạy đúng điểm được giao; tên ghi chú luôn là tên trong giáo trình (đặt bằng mã, không phụ thuộc AI).
  Bài không có điểm thì giữ cách cũ (AI tự chọn).
- **Thêm/sửa bài:** ô chọn điểm lọc theo trình độ của chủ đề; trang chi tiết bài hiện điểm đã gán.
- API (quản trị): `GET /api/admin/grammar?level=A1&topicId=…` (điểm kèm số bài đang dùng); `grammarPointId` ở tạo/sửa bài (khi sửa, vắng mặt
  nghĩa là giữ nguyên, `""` là bỏ gán), ở sinh bài và ở bản nháp; chi tiết bài trả thêm `grammarPointTitle`.
- Đổi điểm mà không đổi nội dung bài thì ghi chú cũ **không** tự cập nhật: bấm **Chạy lại chú thích**.

**Tiêu chí nghiệm thu**
- [ ] Bài tạo với một điểm lưu đúng điểm đó (Mongo và MySQL); điểm không tồn tại hoặc sai trình độ bị báo lỗi theo trường `grammarPointId`.
- [ ] Sinh bài với một điểm: AI nhận điểm trong prompt và các bài dùng điểm đó nhiều lần; mỗi bản nháp mang `grammarPointId`.
- [ ] Chú thích bài có điểm: ghi chú ngữ pháp mang tên của điểm trong giáo trình.
- [ ] Sửa bài không gửi `grammarPointId` thì giữ nguyên điểm; gửi `""` thì xoá; đổi chủ đề sang trình độ khác thì điểm cũ bị từ chối.
- [ ] Hộp thoại sinh bài và form bài nạp danh sách điểm theo trình độ; nạp lỗi thì vẫn sinh/lưu bài được.

---

## F20. Phần học ngữ pháp riêng

> Thêm 2026-10-05, tiếp F19. Quyết định của chủ dự án: ngữ pháp phải có trang học và luyện riêng, không chỉ là ghi chú trong bài. Hướng này là "hai lộ trình
> liên kết" trong cuộc thảo luận về kiến trúc ngữ pháp (lộ trình Chủ đề và lộ trình Ngữ pháp, không khoá cứng nhau).

**Chức năng**
- Mỗi điểm của giáo trình (F19) có thể có một **bài ngữ pháp**: mục tiêu, giải thích, khi nào dùng, cấu trúc, ví dụ, lỗi thường gặp, bài luyện tập (6–12 bài)
  và bài kiểm tra mức nắm vững (5–10 bài). Ba dạng bài tập: trắc nghiệm (4 lựa chọn), điền từ (một chỗ trống), sắp xếp câu (ô từ, có thể có từ gây nhiễu).
- **Quản trị:** mục Ngữ pháp (`/admin/grammar`) liệt kê điểm theo trình độ với trạng thái Chưa có bài / Bản nháp / Đã đăng; **Sinh bằng AI** (đúng 1 request
  AI, lọc bài tập hỏng, lưu bản nháp; đã sửa tay hoặc đã đăng thì hỏi trước khi thay), xem trước, sửa nội dung bằng JSON có kiểm tra theo từng trường, Đăng và Gỡ.
- **Người học:** tab Ngữ pháp (`/grammar`): chọn trình độ, gợi ý "Học tiếp", trạng thái Mới / Đang học / Đã nắm vững kèm điểm tốt nhất, điểm chưa đăng ghi
  "Sắp có". Trang học có ba tab: **Học**, **Luyện tập** (phản hồi ngay từng câu, "Luyện lại các câu sai"), **Kiểm tra** (không báo đúng sai từng câu, chấm cuối,
  đạt từ 80% là đã nắm vững và giữ mãi, làm lại được).
- Bài tập chấm trên trình duyệt; máy chủ nhận điểm (`POST /api/grammar/{id}/attempts`), kiểm tra số câu và id câu sai rồi cập nhật tiến độ. Không gọi AI khi làm bài.
- Liên kết hai chiều với bài học theo chủ đề: ghi chú ngữ pháp trong bài có "Học kỹ điểm này →"; trang ngữ pháp cho biết có bao nhiêu bài học dùng điểm đó.
- Dữ liệu: `grammar_lessons` (một bài mỗi điểm) và `grammar_progress` (mỗi người học và điểm) ở cả Mongo và MySQL; `grammar_progress` nằm trong file xuất dữ liệu.

**Tiêu chí nghiệm thu**
- [ ] Sinh bài ngữ pháp cho một điểm: bản nháp có đủ các phần, bài tập đúng giới hạn và đúng dạng; người học chưa thấy cho tới khi đăng.
- [ ] Đăng: người học mở được bài; Gỡ: bài trở lại "Sắp có" và mở trực tiếp trả 404.
- [ ] Luyện tập: phản hồi ngay có chữ Đúng/Sai và giải thích; điểm và câu sai được lưu; luyện lại các câu sai chạy được.
- [ ] Kiểm tra: dưới 80% thì chưa đạt và nói còn thiếu bao nhiêu câu; từ 80% thì "Đã nắm vững", điểm tốt nhất được lưu, trạng thái không bao giờ về lại.
- [ ] Sửa bài bằng JSON sai (đáp án ngoài phạm vi, thiếu chỗ trống...) bị báo lỗi theo từng trường, không lưu.
- [ ] Dùng được ở 360px, chế độ sáng và tối, bằng bàn phím; không truyền thông tin chỉ bằng màu.

---

## F21. Kiểm soát chất lượng bài ngữ pháp

> Thêm 2026-10-05, tiếp F20. AI có thể sai nội dung (đáp án đánh dấu nhầm, câu có hai đáp án đúng, đáp án điền từ thiếu biến thể); hệ thống chỉ kiểm tra được cấu trúc. Ba lớp bảo vệ.

**Chức năng**
- **AI giải lại độc lập (quản trị):** nút **Kiểm tra bằng AI** gửi bài tập (không kèm đáp án) cho AI giải, rồi so với đáp án đã lưu. Câu lệch hoặc mơ hồ bị gắn cờ kèm ghi chú
  tiếng Việt, hiện trong phần xem trước và thành nhãn "N câu cần xem" ở danh sách. Bài vừa sinh tự được kiểm tra một lần (lỗi kiểm tra không làm hỏng việc sinh bài).
  Sửa nội dung thì cờ bị xoá ("Chưa kiểm tra"). `POST /api/admin/grammar-lessons/{id}/check`.
- **Cổng đăng:** còn cờ thì Đăng hỏi xác nhận; máy chủ trả 409 `grammar_flags_unresolved` trừ khi gửi `acknowledgeFlags: true`. Bài chưa từng kiểm tra không bị chặn.
- **Mở rộng đáp án điền từ bằng mã:** `grammar.ExpandAnswers` thêm dạng viết tắt/đầy đủ hai chiều (`is not` ↔ `isn't`, `I am` ↔ `I'm`...), chuẩn hoá `’`. Áp dụng khi trả bài cho
  người học và khi kiểm tra; nội dung lưu không đổi. Trình duyệt cũng chuẩn hoá `’`, khoảng trắng thừa, hoa/thường khi chấm.
- **Người học báo lỗi:** nút "Báo lỗi câu này" (lý do + ghi chú ≤300 ký tự) sau mỗi câu luyện tập và trong phần "Xem lại từng câu" của bài kiểm tra.
  `POST /api/grammar/{id}/reports`. Quản trị thấy mục **Câu bị báo lỗi** ở danh sách và số lượt/lý do ở từng bài tập, bấm **Đã xử lý** để đóng.
- Dữ liệu: `checks` và `checkedAt` nằm trong bài; collection/bảng `grammar_reports` (duy nhất theo người học, điểm, bài tập) ở cả Mongo và MySQL.

**Tiêu chí nghiệm thu**
- [ ] Sinh bài xong có `checkedAt`; sửa một đáp án sai rồi bấm kiểm tra thì câu đó bị gắn cờ `mismatch`.
- [ ] Đăng bài còn cờ không kèm `acknowledgeFlags` trả 409; kèm thì đăng được.
- [ ] Đáp án điền từ viết tắt hoặc dấu nháy cong được chấp nhận.
- [ ] Báo lỗi: bài tập không có trong bài bị từ chối 400; báo lại thì mở lại; quản trị thấy và đóng được.

**F21b. Xử lý kết quả kiểm tra (bổ sung cùng ngày)**
- Câu AI không trả lời bị gắn cờ `unchecked` ("Chưa kiểm tra được"), không còn im lặng.
- Mỗi câu có cờ có **Xác nhận đúng** (`POST …/checks/{exerciseId}/confirm`), **Sửa câu này** (`PUT …/exercises/{exerciseId}`, không đổi dạng bài) và **Xoá câu này**
  (`DELETE …/exercises/{exerciseId}`, không xuống dưới 6 bài luyện tập / 5 bài kiểm tra). Sửa hoặc xoá chỉ gỡ cờ của câu đó; id các câu khác giữ nguyên.
- Nút **Xác nhận đã kiểm tra xong** (`POST …/verify`, cần đã kiểm tra bằng AI) đặt `verifiedAt` và xác nhận mọi cờ còn lại; danh sách hiện nhãn "Đã xác nhận".
  Sửa nội dung hoặc kiểm tra lại thì xoá trạng thái này. Cổng Đăng và số "N câu cần xem" chỉ tính cờ chưa xác nhận.

---

## F22. Kiểm tra bằng AI cho bài học

> Thêm 2026-10-05, tiếp F21. Bài học cũng do AI viết (câu của bài, chú thích, câu hỏi đọc hiểu, câu dịch ở phần luyện tập) và có thể sai: đáp án đánh dấu nhầm, nghĩa chú thích lệch theo câu, câu dịch tiếng Việt và tiếng Anh không cùng nghĩa. Trang quản trị bài học (`/admin/lessons/:id`) có cùng cách kiểm tra như bài ngữ pháp.

**Chức năng**
- **AI đọc lại độc lập (quản trị):** nút **Kiểm tra bằng AI** gửi cả bài trong một lần gọi (nhiệt độ 0,1) cho một AI thứ hai làm như giáo viên tiếng Anh cẩn thận: trả lời các câu hỏi
  đọc hiểu mà không thấy đáp án, và chỉ báo những câu, chú thích, câu dịch mà nó chắc là sai. Bài cần chú thích xong trước (409 `annotation_not_done`).
  `POST /api/admin/lessons/{id}/check`. **Không tự chạy** sau khi sinh hoặc chú thích, để tiết kiệm hạn mức AI; chỉ chạy khi quản trị bấm.
- **Cờ:** mỗi cờ có `area` (`sentence`, `annotation`, `question`, `translation`), `index` (vị trí trong mảng tương ứng của bài), `kind` và ghi chú tiếng Việt `noteVi`.
  Câu hỏi: `mismatch` (AI chọn đáp án khác), `ambiguous` (mơ hồ, nhiều đáp án hoặc không có), `unchecked` (AI không trả lời). Câu, chú thích, câu dịch: `wrong`. Chỉ số ngoài phạm vi bị bỏ.
- **Xử lý:** **Xác nhận đúng** từng cờ (`POST …/check/confirm`, body `{"area":"question","index":2}`), sửa hoặc xoá bằng các màn sửa sẵn có, rồi **Xác nhận đã kiểm tra xong**
  (`POST …/check/verify`). Chưa kiểm tra thì 409 `not_checked`; còn cờ chưa xác nhận thì 409 `flags_unresolved` kèm `count`.
- **Cờ gắn với chỉ số**, nên đổi nội dung bài, chú thích lại hoặc tạo lại phần luyện tập thì xoá kết quả kiểm tra (sửa câu hỏi, chú thích, câu dịch thì giữ cờ, xem F22b) ("Chưa kiểm tra"). Đổi tiêu đề, trình độ, chủ đề không xoá.
  Nếu bài đổi trong lúc AI đang đọc, kết quả cũ bị bỏ và bài được trả về như hiện tại.
- Danh sách `GET /api/admin/lessons` thêm `flags` (chỉ cờ chưa xác nhận), `checked`, `verified`; JSON bài của quản trị thêm `review` (`null` hoặc `checkedAt`, `verifiedAt`, `flags`).
  Học viên không thấy gì của phần này.
- Dữ liệu: trường lồng `review` trong bài (Mongo) / cột `review JSON NULL` (MySQL, migration `016_lesson_review`, chạy lại an toàn).

**Tiêu chí nghiệm thu**
- [ ] Bài chưa chú thích xong bấm kiểm tra trả 409 `annotation_not_done`; lỗi AI (503, 429, 502) không làm đổi bài.
- [ ] Sửa một đáp án sai rồi kiểm tra thì câu hỏi đó bị gắn cờ `mismatch`; chú thích hoặc câu dịch sai bị gắn cờ `wrong`.
- [ ] Còn cờ chưa xác nhận thì xác nhận xong bị 409 `flags_unresolved`; xác nhận hết rồi mới đặt được `verifiedAt`.
- [ ] Sửa nội dung bài thì `review` về `null`; sửa tiêu đề thì giữ.

**F22b. Sửa ngay chỗ bị cờ (bổ sung cùng ngày)**
- **Giữ cờ qua lần sửa phần phụ:** sửa câu hỏi (`PUT …/extras`), chú thích (`PUT …/annotations`) hoặc câu dịch thì bài vẫn còn `review`: `checkedAt` giữ nguyên, `verifiedAt` về rỗng (nội dung đã đổi), cờ của các phần khác giữ nguyên.
  Cờ của phần vừa sửa được dời theo **nội dung**: mục còn y nguyên thì giữ cờ (và trạng thái đã xác nhận) và theo chỉ số mới (xoá một câu hỏi thì các câu sau dịch lên); mục bị sửa hoặc bị xoá thì mất cờ. Ghi lại cờ lỗi thì chỉ ghi log, lần sửa vẫn thành công.
- **Xoá một chú thích hoặc câu hỏi** dùng chính PUT annotations/extras với danh sách không còn mục đó.
- **Sửa câu dịch:** `PUT /api/admin/lessons/{id}/practice/translations`, body `{"translations":[{"vi","en","distractors"}]}`, trả `{"lesson":…}`. Cùng luật với khi AI sinh (2–15 ô chữ, tối đa 5 câu và 4 từ gây nhiễu; admin tự sửa nên không bắt `en` chứa từ của bài)
  nhưng báo lỗi 400 theo trường (`translations.N.vi|en|distractors`) thay vì bỏ lặng lẽ. Danh sách rỗng là xoá hết câu dịch. 409 `no_practice` (chưa có phần luyện tập), `practice_running`, `practice_changed`.
