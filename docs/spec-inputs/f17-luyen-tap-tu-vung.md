# Đầu vào cho F17: Trang chi tiết bài và luyện tập từ vựng

Nguồn: [giai-doan-3.md › F17](../phases/giai-doan-3.md). Bố cục theo `design/screen1.png` (từ vựng, hội thoại mẫu),
`design/screen2.png` (điền vào ô trống) và `design/screen3.png` (dịch câu); màu và font giữ theo `docs/design-system.md` (Oải hương,
Lexend), không lấy màu xanh của ảnh. Sửa F2 (chú thích, trang quản trị bài) và trang chi tiết bài `/lessons/:id` đã có (bản đơn giản).

## Khối 1: `/speckit-specify`

```text
F17 – Trang chi tiết bài và luyện tập từ vựng cho Luna.

Mục tiêu: mỗi bài có một trang chi tiết dạng 4 bước giúp người học nắm chắc từ vựng của bài: xem từ trong câu ví dụ, nghe hội thoại
mẫu dùng các từ đó, điền từ vào hội thoại, rồi ghép câu dịch sang tiếng Anh. Mọi nội dung xoay quanh danh sách từ vựng của bài.

Nội dung (AI sinh, 1 request riêng, chạy nền sau khi chú thích bài xong):
- Mục tiêu bài: 1 câu tiếng Việt, ví dụ "Bạn có thể chào hỏi và giới thiệu bản thân."
- Câu ví dụ: mỗi từ vựng có 1 câu tiếng Anh ngắn, đơn giản, đúng trình độ, chứa đúng từ đó.
- Hội thoại mẫu: 2 người (có tên, ví dụ Minh và Anna), 6–10 lượt, dùng nhiều từ vựng của bài; mỗi lượt có nghĩa tiếng Việt.
- Mẹo ngữ pháp: 1–2 câu tiếng Việt về một cách nói trong hội thoại.
- Dịch câu: 3–5 câu tiếng Việt đơn giản, mỗi câu có 1 đáp án tiếng Anh dùng ít nhất một từ vựng của bài, cùng 3–4 từ gây nhiễu.
- Câu ví dụ và từng lượt hội thoại có audio (cùng một giọng như bài đọc).

Trang chi tiết bài (/lessons/:id), mở từ danh sách bài (bài hôm nay và bài đã học; bài sắp tới vẫn khoá):
- Thanh trên: nút đóng (về danh sách bài), thanh tiến độ 4 đoạn "1/4", streak "3 ngày".
- Đầu trang: "Bài N" (thứ tự trong lộ trình chủ đề) + tiêu đề, dòng "Mục tiêu: …", nhãn "Mức độ: Cơ bản / Trung cấp / Nâng cao"
  (A1–A2 / B1–B2 / C1–C2). Hai tab: "Bài học" (4 bước bên dưới) và "Bài đọc" (nội dung bài đọc gốc và ghi chú ngữ pháp).
- Bước 1 – Từ vựng quan trọng (thu gọn được): mỗi từ có từ, phiên âm, "— nghĩa tiếng Việt", nút loa đọc từ, dòng "Ví dụ: …" bằng
  tiếng Anh (bấm nghe được).
- Bước 2 – Hội thoại mẫu: thanh phát cả đoạn (thời gian, tốc độ 0.75×/1×/1.25×), danh sách lượt nói có tên người nói, nghe từng lượt,
  bật/tắt nghĩa tiếng Việt.
- Bước 3 – Điền vào ô trống: đoạn hội thoại ở bước 2 với các từ vựng của bài được thay bằng ô trống (tối đa 5 ô); thẻ người nói có
  nút "Nghe câu"; nghĩa tiếng Việt bên dưới; "Ngân hàng từ" gồm các đáp án và 2 từ vựng khác của bài, xáo trộn. Chạm từ để điền vào ô
  trống đang chọn, chạm ✕ để gỡ; từ đã dùng mờ đi. Bấm Kiểm tra: đúng thì "Chính xác!", sai thì chỉ ra ô sai và đáp án. Thẻ "Mẹo
  ngữ pháp" hiện bên dưới.
- Bước 4 – Dịch câu sang tiếng Anh: câu tiếng Việt (có nút loa đọc đáp án tiếng Anh sau khi kiểm tra); "Câu dịch của bạn" ghép bằng
  cách chạm các ô từ trong "Ngân hàng từ" (từ của đáp án + từ gây nhiễu, xáo trộn); chạm ✕ để gỡ, "Làm lại" để xoá hết. Kiểm tra:
  đúng thứ tự thì "Chính xác!", sai thì hiện câu đúng. Lần lượt từng câu.
- Nút "Tiếp theo →" cố định ở cuối màn hình chuyển bước; ở bước cuối là "Hoàn thành": hiện tổng kết (điền đúng x/y ô, dịch đúng
  a/b câu) với nút "Làm lại" và nút về bài (bài hôm nay → "Học bài này"; bài đã học → Đọc lại / Nghe lại / Bài viết).
- Quản trị viên xem phần luyện tập ở trang chi tiết bài quản trị và bấm "Tạo lại phần luyện tập".

Quy tắc:
- Phần luyện tập là tự chọn: không ảnh hưởng các bước Ôn → Đọc → Nghe → Viết, tiến độ, streak hay thống kê. Không có điểm XP.
- Sinh phần luyện tập tốn đúng 1 request AI cho mỗi bài; AI lỗi thì chỉ phần luyện tập trống, chú thích từ và các phần khác không bị
  ảnh hưởng.
- Bài chưa có phần luyện tập (bài cũ, AI lỗi, đang sinh): bước 1 vẫn hiện từ vựng (không có câu ví dụ); các bước 2–4 báo "Bài này chưa
  có phần luyện tập".
- Sửa nội dung bài hoặc chạy lại chú thích → phần luyện tập cũ bỏ đi và sinh lại.
- Kiểm tra hợp lệ, câu nào hỏng thì bỏ câu đó: câu ví dụ phải chứa từ của nó; hội thoại dưới 4 lượt thì bỏ cả đoạn; câu dịch phải
  ghép được từ các ô (đáp án tách theo khoảng trắng, dấu câu dính theo từ như "you," "Vietnam.").
- Chấm tự động ở trình duyệt: điền từ so đúng từ (không phân biệt hoa/thường); dịch câu so đúng thứ tự các ô.
- Đúng/sai phân biệt được không cần nhìn màu. Dùng được bằng bàn phím. Dùng tốt ở 360px, sáng và tối.
- Kết quả luyện tập không lưu lên máy chủ (chỉ trong lần mở trang).

Ngoài phạm vi: điểm XP, AI chấm câu dịch tự do, nhiều giọng đọc cho hội thoại, lưu và thống kê kết quả luyện tập, quản trị viên sửa tay
từng câu luyện tập, các loại bài tập khác (ghép đôi, kiểm tra cuối bài).

Tiêu chí nghiệm thu: theo mục F17 trong docs/phases/giai-doan-3.md.
```

## Khối 2: `/speckit-plan`

```text
Backend (internal/ai, internal/lesson, internal/storage/mongo, internal/job, internal/tts)
- ai.Provider thêm Practice(ctx, PracticeInput{level, title, sentences, words [{lemma, text, meaningVi}]}) -> Practice. Một request
  Gemini riêng (JSON mode + responseSchema), không đổi Annotate. Practice {objectiveVi, examples [{lemma, sentence}],
  dialogue {speakers [2 tên], turns [{speaker 0|1, text, meaningVi}]}, grammarTipVi, translations [{vi, en, distractors []}]}.
- lesson: CleanPractice bỏ phần không hợp lệ (xem Quy tắc); giới hạn 10 lượt hội thoại, 5 câu dịch, 4 từ nhiễu mỗi câu.
  Ô trống bước 3 do backend tính, không cần AI: các lần xuất hiện của từ vựng trong lượt hội thoại (khớp text hoặc lemma, nguyên từ),
  tối đa 5, trải đều các lượt; trả về turns dạng parts [{text} | {blank: index}] + blanks [{answer}] + wordBank.
- Job mới TypePractice: xếp hàng khi ProcessAnnotate xong thành công và khi admin bấm tạo lại. Lưu vào lesson: practice,
  practiceStatus (none|pending|done|failed), practiceError; gắn revision như annotations; sửa bài thì xoá.
- TTS: sau khi lưu practice, tạo audio cho câu ví dụ, từng lượt hội thoại và đáp án câu dịch vào
  {audioDir}/{id}/{rev}/practice/{kind}-{i}.mp3, phục vụ qua GET /api/audio/{lessonId}/{revision}/practice/{kind}/{i};
  JSON trả audioUrl (null khi chưa có). Phát cả đoạn hội thoại = phát lần lượt các file ở client.
- API người học: GET /api/lessons/{id}/practice (cùng guard CanOpen như GET /api/lessons/{id}) → {status, objectiveVi,
  lessonNumber, examples, dialogue, fill {turns, blanks, wordBank}, grammarTipVi, translations [{vi, answer[], tiles[]}]}.
  Đáp án gửi kèm (chấm ở client). Không có endpoint nộp kết quả.
- API quản trị: POST /api/admin/lessons/{id}/practice/regenerate; GET lesson quản trị trả thêm practice + practiceStatus.
- Test: CleanPractice, tính ô trống, job (thành công, AI lỗi không làm hỏng chú thích, revision cũ bị bỏ), handler (guard 403 bài sắp
  tới, 404), gemini schema/prompt với fake server.

Frontend
- features/lesson/lesson-detail: thanh trên (đóng, tiến độ 4 đoạn, streak), đầu trang (Bài N, mục tiêu, mức độ, tab Bài học | Bài
  đọc), 4 bước: vocab-step, dialogue-step (player cả đoạn + từng lượt + tốc độ), fill-step (ô trống + ngân hàng từ + mẹo ngữ pháp),
  translate-step (ghép ô từ, Làm lại); nút Tiếp theo cố định phía trên thanh tab (dùng --tabbar-h); tổng kết và làm lại.
- core/models: Practice, PracticeStatus; reading-api.service thêm practice(id).
- features/admin/lesson-detail: mục "Phần luyện tập" (trạng thái, xem nội dung, nút Tạo lại).
- Test Vitest: chuyển bước và tiến độ, điền từ (chọn ô, gỡ, kiểm tra đúng/sai), ghép câu (thứ tự đúng/sai, Làm lại), tổng kết, trạng
  thái chưa có phần luyện tập, bàn phím.
```
