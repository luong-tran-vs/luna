# Đầu vào cho F27: Luyện thi TOEIC (Nghe–Đọc)

Spec mới, độc lập; không sửa vào các spec và tài liệu giai đoạn cũ. Chốt sau buổi thảo luận với người dùng ngày 2026-11-18.
Phác thảo đã duyệt: `design/toeic/de-xuat-luyen-thi-toeic-2.png` (bản 2, theo kiểu các trang luyện thi như study4, Prep,
phần mềm thi trên máy của IIG). Bản 1 `design/toeic/de-xuat-luyen-thi-toeic.png` chỉ để tham khảo, không dùng. Ảnh thiết kế
TOEIC sau này cũng để trong `design/toeic/`. Màu và font theo `docs/design-system.md`. Thư mục spec dự kiến:
`specs/022-toeic-practice`.

## Khối 1: `/speckit-specify`

```text
F27 – Luyện thi TOEIC (Nghe–Đọc) cho Luna.

Mục tiêu: người học luyện bài thi TOEIC Listening & Reading như trên các trang luyện thi chuyên dụng: chọn đề trong thư viện,
luyện tập từng Part hoặc làm full test trong phòng thi giống thi thật, xem kết quả phân tích theo dạng câu, đáp án chi tiết có
transcript và giải thích tiếng Việt, ôn lại câu sai và theo dõi điểm ước tính so với mục tiêu. Chỉ làm Nghe–Đọc; không làm
Nói–Viết. Giao diện theo design/toeic/de-xuat-luyen-thi-toeic-2.png.

Cấu trúc một đề đầy đủ (theo đề thật, 200 câu, 120 phút):
- Listening (100 câu, khoảng 45 phút): Part 1 tả ảnh (6), Part 2 hỏi–đáp (25), Part 3 hội thoại (39), Part 4 bài nói ngắn (30).
- Reading (100 câu, 75 phút): Part 5 điền câu (30), Part 6 điền đoạn văn (16), Part 7 đọc hiểu một, hai hoặc ba văn bản (54).
- Mini Test: 50 câu (38 phút) hoặc 100 câu (60 phút), đủ 7 Part theo tỉ lệ.

1. Thư viện đề (/toeic, vào từ mục Luyện):
- Lưới thẻ đề: tên (Luna Test 01…, Mini Test 05…; không dùng tên đề ETS), thời gian, số phần, số câu, trạng thái (chưa làm /
  đang làm dở kèm câu đang làm / đã làm kèm điểm). Thẻ "Đề ngẫu nhiên": rút 50 hoặc 100 câu chưa làm từ kho.
- Tab: Đề thi, Luyện theo Part, Luyện theo dạng câu, Kết quả của tôi, Sổ câu sai (kèm số câu).
- Cột bên: mục tiêu điểm, ngày thi, điểm ước tính gần nhất, các lần làm gần đây.

2. Chi tiết đề, hai cách làm:
- Luyện tập: tick các Part muốn làm (mỗi Part liệt kê các #dạng câu có trong đề), chọn giới hạn thời gian (mức gợi ý theo số
  câu hoặc không giới hạn). Được nghe lại, tạm dừng, xem đáp án khi nộp.
- Làm full test: cảnh báo điều kiện thi thật (liền 120 phút, phần Nghe phát một lần, không tạm dừng, không tua, xem đáp án sau
  khi nộp cả bài), thời lượng từng phần, mục kiểm tra trước khi thi (tai nghe, nghe thử giọng đọc).
- Tab Kết quả: mọi lần làm đề này (ngày, chế độ, các Part, điểm, thời gian).

3. Phòng thi (dùng chung cho luyện tập và full test):
- Thanh trên: tên đề, chế độ, nút Thoát; hàng tab Part (các Part của phần Nghe khoá khi đang làm phần Đọc và ngược lại).
- Máy tính: văn bản bên trái (cuộn riêng, bôi đen để tô vàng), câu hỏi bên phải (số câu tròn, nút chọn A–D); cột phải có đồng
  hồ đếm ngược, nút NỘP BÀI, bảng câu chia theo Part (đã làm / đánh dấu xem lại / chưa làm / câu đang xem), bấm số để nhảy câu
  (chỉ trong phần Đọc). Part 5 không có cột văn bản, nhiều câu một trang.
- Điện thoại: xếp dọc, thanh dưới có Bảng câu và Nộp bài, phiếu trả lời dạng ô tròn tô kín như phiếu thi giấy.
- Không chấm từng câu; API không trả đáp án trước khi nộp. Nộp khi còn câu chưa làm thì hỏi lại.
- Làm dở: lưu trên server, mở lại trên máy khác vẫn làm tiếp. Phần Đọc làm tiếp đúng câu và đúng thời gian còn lại; phần Nghe
  làm tiếp từ đầu cụm câu đang nghe.
- Màn hướng dẫn (Directions) đầu mỗi Part, tiếng Anh như đề thật, được đọc lên, kèm một dòng tóm tắt tiếng Việt; khi luyện tập
  bỏ qua được.

4. Phần Nghe:
- Dùng giọng đọc của trình duyệt, đọc theo kịch bản như đề thật: lời hướng dẫn mỗi Part, "Questions 41 through 43 refer to the
  following conversation", nội dung, rồi đọc câu hỏi kèm khoảng nghỉ (khoảng 8 giây). Khi full test, audio chạy liền một mạch,
  thanh audio luôn hiện tiến độ; chỉ sửa được đáp án các câu của cụm đang phát.
- Mỗi nhân vật một giọng, ưu tiên giọng Mỹ, Anh, Úc, Canada nếu máy có.
- Part 1: một ảnh, 4 câu mô tả chỉ nghe. Part 2: chỉ "Mark your answer", 3 đáp án A–C chỉ nghe. Part 3/4: câu hỏi và phương án
  hiện bằng chữ.

5. Ảnh trong đề (mỗi đề đều phải có):
- Part 1: 6 ảnh, kiểu ảnh chụp thật (văn phòng, kho, công trường, nhà hàng, đường phố), không phải kiểu hoạt hình. AI vẽ khi admin
  bấm, phải chọn Gemini hay Codex mỗi lần; có Vẽ lại từng ảnh và tự tải ảnh lên. Prompt dùng mô tả cảnh trong JSON cùng câu mô tả
  đúng, không gửi các câu nhiễu.
- Part 3/4: câu "Look at the graphic" (sơ đồ tầng, bảng giá, lịch, biểu đồ cột) dựng bằng HTML từ dữ liệu JSON.
- Part 7: văn bản có bố cục giống thật (trang web, hoá đơn, mẫu đơn, quảng cáo, email, tin nhắn), có thể kèm ảnh sản phẩm; các
  văn bản trong một cụm chuyển qua lại bằng nút.
- Đề thiếu ảnh Part 1 thì admin không đăng được.

6. Trang kết quả:
- Số câu đúng / sai / bỏ qua, độ chính xác, thời gian hoàn thành; điểm Listening và Reading (5–495), tổng điểm, luôn ghi "ước
  tính" (bảng quy đổi gần đúng; đề không đủ 200 câu quy đổi theo tỉ lệ), so với mục tiêu.
- Bảng "Phân loại câu hỏi" theo từng Part: mỗi #dạng câu có số đúng, sai, bỏ qua, độ chính xác và danh sách số câu tô màu; bấm số
  để mở câu.
- Danh sách đáp án (đáp án đúng : bạn chọn) với link chi tiết. Nút "Làm lại các câu sai" tạo một lượt luyện chỉ gồm câu sai.

7. Đáp án chi tiết từng câu: audio của cụm để nghe lại, transcript ghi người nói và giọng, tô câu chứa đáp án, bản dịch tiếng
   Việt, giải thích tiếng Việt (vì sao đúng, các phương án khác sai ở đâu, bẫy thường gặp), từ vựng có nút đưa vào sổ từ để ôn.

8. Luyện thêm:
- Luyện theo Part: chọn một Part, làm câu từ mọi đề, ưu tiên câu chưa làm.
- Luyện theo dạng câu: danh sách #dạng câu kèm độ chính xác, yếu nhất lên đầu; bấm để làm 10 câu (hoặc vài cụm) dạng đó. Dạng ngữ
  pháp có link sang điểm ngữ pháp tương ứng trong phần Ngữ pháp của Luna.
- Sổ câu sai: câu làm sai được lưu, lọc theo Part; làm đúng 2 lần liên tiếp thì ra khỏi sổ, sai lại thì đếm lại từ đầu. Câu thuộc
  cụm (Part 3/4/6/7) thì mở cả cụm nhưng chỉ chấm câu đó. Đây là chỗ duy nhất chấm ngay từng câu.
- Thống kê: biểu đồ điểm ước tính các đề, có đường mục tiêu; độ chính xác theo Part và dạng câu.
- Mục tiêu: chọn điểm mục tiêu (450 / 600 / 750 / 900) và ngày thi (không bắt buộc).
- Hoàn thành một lượt luyện hoặc một đề tính là một ngày học (streak).

9. Nội dung và quản trị:
- Nội dung do Claude soạn thành file JSON trong content/toeic/, admin nhập qua API: chạy thử (dryRun) báo lỗi từng trường, bỏ qua
  câu trùng, không lưu khi còn lỗi; nhập thật thì lưu nháp; đăng rồi người học mới thấy, có nút Ẩn. Không chép đề thật của ETS.
  Không gọi AI khi người học làm bài.
- Một cụm câu gồm: Part, độ khó, ngữ liệu (văn bản có bố cục, lời nói có người nói và giọng, mô tả ảnh tiếng Anh, dữ liệu biểu
  đồ), danh sách câu hỏi (câu hỏi, phương án, đáp án, câu làm bằng chứng, giải thích tiếng Việt, bản dịch, #dạng câu lấy từ danh
  sách cố định).
- Trang admin TOEIC: danh sách cụm câu lọc theo Part và trạng thái; nhập JSON; trang mỗi đề đếm câu theo Part và ảnh (Part 1,
  biểu đồ, văn bản), vẽ ảnh còn thiếu; tạo đề đầy đủ hoặc Mini Test bằng cách ghép các cụm đã đăng đúng số câu mỗi Part.

10. Quyền:
- Khách làm được 1 Mini Test và phần Luyện tập Part 1, 2, 5; các đề còn lại, sổ câu sai, thống kê và luyện theo dạng câu chỉ dành
  cho thành viên (API trả 403), hiện khoá kèm ghi chú "chỉ dành cho thành viên".
- Bài làm, sổ câu sai và mục tiêu thuộc về từng người học: có trong file xuất và bị xoá cùng tài khoản.

Làm theo 4 đợt:
- Đợt 1: thư viện đề, chi tiết đề, phòng thi phần Đọc (Part 5–7), trang kết quả, đáp án chi tiết, sổ câu sai, admin nhập JSON và
  ghép đề.
- Đợt 2: phần Nghe (Part 1–4): kịch bản đọc liền, nhiều giọng, ảnh Part 1, biểu đồ, màn Directions.
- Đợt 3: full test liền 200 câu (khoá phần Nghe/Đọc, làm tiếp trên máy khác), Mini Test, đề ngẫu nhiên, quy đổi điểm.
- Đợt 4: luyện theo dạng câu, thống kê, mục tiêu, link sang Ngữ pháp và sổ từ.

Tiêu chí nghiệm thu:
- Từ thanh tab mở được thư viện đề trong 2 lần chạm; phòng thi dùng được ở máy tính (hai cột) và điện thoại 360px, sáng và tối.
- Nhập JSON sai thì chạy thử báo rõ trường lỗi và không lưu gì; cụm câu nháp không hiện với người học; đề thiếu ảnh Part 1 không
  đăng được.
- Trong phòng thi, API không trả đáp án trước khi nộp; làm dở, đóng app, mở lại trên máy khác thì làm tiếp đúng chỗ và đúng
  thời gian còn lại.
- Full test: phần Nghe phát liền, không tạm dừng hay tua; hội thoại Part 3 dùng ít nhất 2 giọng khác nhau khi máy có nhiều giọng.
- Trang kết quả có số đúng/sai/bỏ qua, điểm hai phần ghi "ước tính", bảng phân loại câu hỏi theo Part và dạng câu.
- Câu sai có trong sổ trên mọi máy và ra khỏi sổ sau 2 lần đúng liên tiếp.
- Khách chỉ làm được 1 Mini Test và luyện tập Part 1, 2, 5; API trả 403 cho phần còn lại.
- Bài làm, sổ câu sai và mục tiêu có trong file xuất và bị xoá cùng tài khoản.
```

## Gợi ý kỹ thuật cho `/speckit-plan`

```text
Backend: package mới internal/toeic theo mẫu internal/games (deps.go ports, service.go, handler.go Register(mux, auth) với
auth/member/admin), wiring ở internal/service/toeic.go và routes.go, getter trong storage.Store và storage/factory, cài đặt
storage/mongo/toeic.go và storage/mysql/toeic.go, migration 047_toeic trong mysql/migrate.go (chỉ thêm bảng: dev và production
dùng chung một DB). Dữ liệu: toeic_groups (cụm câu), toeic_images (ảnh Part 1, JPEG thu nhỏ như word_images), toeic_tests (đề:
danh sách cụm theo Part), toeic_attempts (bài làm: chế độ, các Part, câu trả lời, đánh dấu, vị trí và thời gian còn lại, kết
quả), toeic_mistakes, toeic_goals; ba bảng có người dùng thêm vào export/model.go và danh sách xoá tài khoản (mongo/users.go,
mysql/users.go). Nhập JSON theo mẫu games/spot_admin.go (dryRun, báo cáo từng mục). Sinh ảnh theo mẫu ảnh cảnh của bài
(internal/ai, chọn engine Gemini/Codex, EngineDialog requireChoice). Streak qua progress.StudyService.MarkStudied; thêm từ vào
sổ qua vocab.Service.Save.
Frontend: features/toeic (library, test-detail, exam-room với listening-player dựng kịch bản đọc bằng SpeechService.speak(...,
voice), result, answer-detail, practice, mistakes, stats, shared: question, passage/document renderer, graphic renderer, answer
sheet, question grid, timer), core/services/toeic-api.service.ts, core/models/toeic.ts, route trong app.routes.ts, ô trong
practice-menu.ts và isPracticePath(); admin features/admin/toeic dùng lại features/admin/json-import, mục sidebar trong
admin-layout.ts. Không thêm thư viện mới.
```
