# Feature Specification: Luyện thi TOEIC (Nghe–Đọc)

**Feature Branch**: `022-toeic-practice` (làm trên `main`, như các spec 018–021)

**Created**: 2026-11-18

**Status**: Draft

**Mã tính năng**: F27 (mới, F26 là Trò chơi) | **Giai đoạn**: 3

**Input**: User description: "Dùng nội dung Khối 1 trong docs/spec-inputs/f27-luyen-thi-toeic.md (F27 – Luyện thi TOEIC
Nghe–Đọc), giao diện theo design/toeic/de-xuat-luyen-thi-toeic-2.png. Thư mục spec: specs/022-toeic-practice."

**Nguồn**: `docs/spec-inputs/f27-luyen-thi-toeic.md`. Đây là spec mới, độc lập; theo yêu cầu người dùng **không sửa vào
các spec cũ và tài liệu giai đoạn cũ** (`docs/phases/giai-doan-3.md`, `docs/mvp-features.md`), nên file đầu vào trên là
tài liệu sản phẩm của F27.

**Phác thảo đã duyệt**: `design/toeic/de-xuat-luyen-thi-toeic-2.png` (bản 2, 7 phần). Bản 1
`design/toeic/de-xuat-luyen-thi-toeic.png` không dùng.

## Tóm tắt

Người học có thêm khu **Luyện thi TOEIC** cho bài thi Listening & Reading, làm theo cách của các trang luyện thi chuyên
dụng: chọn đề trong **thư viện đề**, **luyện tập** từng Part hoặc **làm full test** trong một **phòng thi** giống thi thật
(đồng hồ, bảng câu, văn bản bên trái và câu hỏi bên phải, phần Nghe phát liền một mạch). Sau khi nộp có **trang kết quả**
với điểm ước tính hai phần, bảng phân tích theo dạng câu và **đáp án chi tiết** từng câu (transcript, bản dịch, giải
thích tiếng Việt). Câu sai vào **sổ câu sai**; người học luyện thêm theo Part, theo dạng câu và theo dõi điểm so với mục
tiêu. Nội dung do Claude soạn thành file và admin nhập, duyệt, đăng; mỗi đề có đủ ảnh như đề thật. Không làm phần Nói–Viết.

## Clarifications

### Session 2026-11-18

- Q: Tài khoản khách được làm những gì? → A: 1 Mini Test (đề Mini Test đầu tiên) và Luyện theo Part cho Part 1, 2, 5;
  phần còn lại khoá (thay cho quyết định cũ "lượt đầu của mỗi Part").
- Q: Sửa một cụm đã đăng và đã có người làm thì sao? → A: Nhập lại đè lên cụm cũ (giữ mã cụm); các lần làm cũ được chấm
  lại theo nội dung mới, sổ câu sai cập nhật theo.
- Q: Khi luyện tập phần Nghe, âm thanh phát thế nào? → A: Tự chạy liền như đề thật (Directions, các cụm, câu hỏi, khoảng
  nghỉ) nhưng có tạm dừng, nghe lại cụm đang phát và đổi tốc độ.
- Q: Làm lại một đề đã làm thì sao? → A: Làm lại không giới hạn, mọi lần lưu trong tab Kết quả, thẻ đề hiện lần gần nhất;
  chỉ lần đầu của mỗi đề được tính vào biểu đồ điểm và "điểm ước tính gần nhất".
- Q: Ở chế độ Luyện tập, khi nào được xem đáp án? → A: Luôn nộp xong mới chấm, kể cả khi luyện tập; không có công tắc xem
  đáp án ngay (chấm ngay từng câu chỉ có ở sổ câu sai).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Nhập nội dung và ghép đề (Priority: P1)

Admin mở trang quản trị TOEIC, chọn file nội dung (các cụm câu của một hoặc nhiều Part) và bấm chạy thử. Hệ thống báo
từng cụm hợp lệ, cụm trùng sẽ bỏ qua và lỗi theo từng trường; còn lỗi thì không lưu gì. Hết lỗi thì nhập thật, các cụm vào
ở dạng nháp. Admin xem trước một cụm như người học sẽ thấy, rồi bấm Đăng. Sau đó admin tạo một đề (đề đầy đủ hoặc Mini
Test) bằng cách ghép các cụm đã đăng; trang đề cho biết còn thiếu bao nhiêu câu mỗi Part và bao nhiêu ảnh.

**Why this priority**: mọi phần của người học đều cần nội dung đã đăng và đề đã ghép. Không có phần này thì không kiểm
tra được phần nào khác.

**Independent Test**: nhập một file có lỗi (thấy báo lỗi, không có gì được lưu), sửa rồi nhập lại, xem trước, đăng, ghép
một Mini Test; kiểm tra cụm nháp không hiện với người học còn cụm đã đăng thì có.

**Acceptance Scenarios**:

1. **Given** file có một câu với đáp án ngoài A–D, **When** admin chạy thử, **Then** hệ thống báo đúng cụm, đúng câu, đúng
   trường bị lỗi và không lưu cụm nào.
2. **Given** file hợp lệ có 1 cụm trùng nội dung đã có, **When** admin nhập thật, **Then** các cụm mới được lưu ở dạng nháp,
   cụm trùng bị bỏ qua và được liệt kê trong báo cáo.
3. **Given** một cụm đang là nháp, **When** người học mở luyện tập Part của cụm đó, **Then** cụm không xuất hiện.
4. **Given** admin bấm Đăng một cụm, **When** người học luyện tập Part đó, **Then** cụm có thể xuất hiện; admin bấm Ẩn
   thì cụm không còn được chọn cho lượt mới.
5. **Given** admin ghép một đề đầy đủ còn thiếu 4 câu Part 3, **When** xem trang đề, **Then** trang ghi rõ số câu đã có
   và còn thiếu của từng Part, và nút Đăng đề bị khoá.
6. **Given** đề đủ 200 câu nhưng thiếu 2 ảnh Part 1, **When** xem trang đề, **Then** trang ghi "thiếu ảnh câu 5, 6" và
   nút Đăng đề vẫn khoá đến khi đủ ảnh.
7. **Given** cụm đã đăng có đáp án sai và đã có người làm, **When** admin nhập lại cụm đó ở chế độ thay thế với đáp án
   đúng, **Then** cụm giữ mã và trạng thái đăng, các lần làm cũ được chấm lại và sổ câu sai của người học cập nhật theo.

---

### User Story 2 - Luyện tập phần Đọc trong phòng thi (Priority: P1)

Người học mở mục Luyện, chọn TOEIC và thấy thư viện đề. Mở một đề, ở tab **Luyện tập** người học tick Part 5 và Part 6,
chọn giới hạn 20 phút rồi bắt đầu. Phòng thi hiện đồng hồ, bảng câu và câu hỏi; ở Part 6/7 văn bản nằm bên trái, câu hỏi
bên phải (trên điện thoại thì xếp dọc). Người học chọn đáp án, đánh dấu vài câu để xem lại, rồi nộp. Trang kết quả hiện số
câu đúng/sai/bỏ qua, độ chính xác, bảng phân loại câu hỏi theo dạng câu và danh sách đáp án. Bấm vào một câu để xem đáp
án chi tiết có bản dịch và giải thích tiếng Việt.

**Why this priority**: đây là vòng luyện thi cốt lõi (chọn đề → làm → xem kết quả → hiểu vì sao sai) và phần Đọc làm
được mà chưa cần âm thanh hay ảnh.

**Independent Test**: với một đề đã đăng đủ Part 5–7, luyện tập Part 5 + 6 có giới hạn giờ, nộp, kiểm tra trang kết quả và
đáp án chi tiết; kiểm tra đáp án không có trong dữ liệu gửi về trước khi nộp.

**Acceptance Scenarios**:

1. **Given** có đề đã đăng, **When** người học mở thư viện đề, **Then** mỗi đề hiện tên, thời gian, số phần, số câu và
   trạng thái (chưa làm / đang làm dở kèm câu đang làm / đã làm kèm điểm hoặc số câu đúng gần nhất).
2. **Given** tab Luyện tập của một đề, **When** người học tick Part 5 và 6, **Then** mỗi Part hiện số câu và các #dạng câu
   có trong đề, mức giờ gợi ý đổi theo tổng số câu đã chọn.
3. **Given** đang làm, **When** người học bấm một số trong bảng câu, **Then** phòng thi nhảy tới câu đó; câu đã làm, câu
   đánh dấu, câu chưa làm và câu đang xem có màu khác nhau.
4. **Given** đồng hồ về 0, **When** hết giờ, **Then** bài tự nộp với các câu đã trả lời.
5. **Given** còn 3 câu chưa làm, **When** bấm Nộp bài, **Then** hệ thống hỏi lại và cho quay lại làm tiếp.
6. **Given** đã nộp, **When** xem kết quả, **Then** có số câu đúng / sai / bỏ qua, độ chính xác, thời gian làm, bảng phân
   loại câu hỏi theo từng Part (mỗi dạng câu: đúng, sai, bỏ qua, độ chính xác, danh sách số câu tô màu) và danh sách
   "đáp án đúng : bạn chọn".
7. **Given** trang kết quả, **When** bấm số một câu, **Then** hiện đáp án chi tiết: văn bản với câu chứa đáp án được tô,
   bản dịch, giải thích tiếng Việt (vì sao đúng, các phương án khác sai ở đâu, bẫy thường gặp), từ vựng kèm nút thêm vào
   sổ từ.
8. **Given** đang làm Part 7, **When** bôi đen một đoạn văn bản, **Then** đoạn đó được tô vàng trong suốt bài làm.

---

### User Story 3 - Sổ câu sai (Priority: P2)

Mỗi câu làm sai (hoặc bỏ qua) sau khi nộp được đưa vào sổ câu sai. Người học mở sổ, lọc theo Part, bấm "Làm lại 10
câu". Ở đây mỗi câu được chấm ngay. Câu đúng 2 lần liên tiếp thì ra khỏi sổ; sai lại thì đếm lại từ đầu. Trang kết quả
cũng có nút "Làm lại các câu sai" chỉ gồm câu sai của lần làm đó.

**Why this priority**: biến kết quả thành việc ôn cụ thể; đây là thói quen chính của người luyện thi.

**Independent Test**: làm sai 3 câu, mở sổ (có 3 câu), làm đúng một câu 2 lần liên tiếp (câu ra khỏi sổ), làm sai lại một
câu đã đúng 1 lần (đếm về 0); mở trên máy khác thấy cùng sổ.

**Acceptance Scenarios**:

1. **Given** nộp bài có câu 104 sai và câu 109 bỏ qua, **When** mở sổ câu sai, **Then** có cả hai câu, kèm Part, #dạng
   câu và số lần đúng liên tiếp (0/2).
2. **Given** câu 104 đang 1/2, **When** làm lại và đúng, **Then** câu ra khỏi sổ.
3. **Given** câu 104 đang 1/2, **When** làm lại và sai, **Then** câu về 0/2.
4. **Given** câu sai thuộc một cụm Part 7, **When** làm lại, **Then** hiện cả văn bản của cụm nhưng chỉ chấm câu đó.
5. **Given** trang kết quả có 71 câu sai, **When** bấm "Làm lại các câu sai", **Then** mở một lượt chỉ gồm các câu sai đó.

---

### User Story 4 - Phần Nghe (Priority: P2)

Người học luyện tập Part 1–4. Mỗi Part bắt đầu bằng màn Directions tiếng Anh như đề thật, được đọc lên, kèm một dòng tóm
tắt tiếng Việt (bỏ qua được khi luyện tập). Âm thanh được đọc theo kịch bản như đề thật: số câu, nội dung, câu hỏi kèm
khoảng nghỉ; mỗi nhân vật một giọng. Part 1 có ảnh chụp và 4 câu mô tả chỉ nghe; Part 2 chỉ có "Mark your answer" và 3 đáp
án chỉ nghe; Part 3/4 hiện câu hỏi và phương án bằng chữ, có câu "Look at the graphic" kèm sơ đồ, bảng hoặc biểu đồ. Sau
khi nộp, đáp án chi tiết có transcript ghi người nói và giọng, nghe lại được từng cụm.

**Why this priority**: phần Nghe chiếm nửa điểm thi nhưng cần thêm giọng đọc, ảnh và biểu đồ nên làm sau phần Đọc.

**Independent Test**: luyện tập Part 1–4 của một đề đủ ảnh; kiểm tra Directions, ảnh Part 1, Part 2 không hiện chữ đáp án,
hội thoại Part 3 dùng ít nhất 2 giọng, câu biểu đồ hiển thị rõ ở 360px, transcript sau khi nộp.

**Acceptance Scenarios**:

1. **Given** bắt đầu Part 1 khi luyện tập, **When** màn Directions hiện, **Then** lời hướng dẫn được đọc lên, có tóm tắt
   tiếng Việt và nút bỏ qua.
2. **Given** câu Part 1, **When** audio chạy, **Then** màn hình chỉ có ảnh và 4 nút A–D, không hiện chữ các câu mô tả.
3. **Given** câu Part 2, **When** audio chạy, **Then** màn hình chỉ có "Mark your answer" và 3 nút A–C.
4. **Given** máy có nhiều giọng tiếng Anh, **When** phát hội thoại Part 3 có 2–3 người, **Then** mỗi người một giọng khác
   nhau, ưu tiên giọng Mỹ, Anh, Úc, Canada theo kịch bản.
5. **Given** câu "Look at the graphic", **When** hiển thị trên điện thoại 360px ở chế độ sáng và tối, **Then** sơ đồ, bảng
   hoặc biểu đồ đọc rõ, không bị cắt.
6. **Given** đã nộp, **When** mở đáp án chi tiết một câu Part 3, **Then** có transcript ghi người nói và giọng, câu chứa đáp
   án được tô và ghi số câu hỏi, bản dịch, giải thích tiếng Việt, và nghe lại được cả cụm.
7. **Given** đang luyện tập Part 3, **When** cụm 41–43 phát xong và hết khoảng nghỉ, **Then** âm thanh tự sang cụm 44–46;
   người học tạm dừng, nghe lại cụm đang phát và đổi tốc độ được (chỉ full test mới cấm).

---

### User Story 5 - Full test, Mini Test và điểm ước tính (Priority: P3)

Người học mở tab **Làm full test** của một đề: thấy cảnh báo điều kiện thi thật, thời lượng từng phần, mục kiểm tra tai
nghe và nút nghe thử giọng. Bắt đầu thi: phần Nghe phát liền khoảng 45 phút, không tạm dừng, không tua, không quay lại;
xong phần Nghe thì sang phần Đọc 75 phút. Đóng app giữa chừng thì lần sau (kể cả trên máy khác) làm tiếp. Nộp bài thì
có điểm Listening và Reading ước tính và tổng điểm. Mini Test và đề ngẫu nhiên dùng cùng cách làm với số câu ít hơn.

**Why this priority**: là trải nghiệm "thi thử" hoàn chỉnh nhưng dựa trên phòng thi và phần Nghe đã có ở US2, US4.

**Independent Test**: làm full test một đề: kiểm tra không tạm dừng/tua được phần Nghe, các tab phần Đọc khoá khi đang
nghe, đóng app ở phần Đọc rồi mở trên máy khác (đúng câu, đúng giờ còn lại), nộp và xem điểm ước tính.

**Acceptance Scenarios**:

1. **Given** tab Làm full test, **When** người học mở, **Then** thấy cảnh báo (liền 120 phút, Nghe phát một lần, không tạm
   dừng, không tua, xem đáp án sau khi nộp cả bài), thời lượng từng phần và nút nghe thử giọng.
2. **Given** đang ở phần Nghe của full test, **When** người học tìm cách tạm dừng, tua hoặc nhảy câu, **Then** không làm
   được; chỉ sửa được đáp án các câu của cụm đang phát; các tab Part 5–7 bị khoá.
3. **Given** đang ở phần Đọc với 48 phút còn lại, **When** đóng app và mở lại trên máy khác sau 1 giờ, **Then** làm tiếp
   đúng câu đang làm với 48 phút còn lại (thời gian tạm dừng không bị trừ).
4. **Given** đang ở phần Nghe, **When** đóng app rồi mở lại, **Then** làm tiếp từ đầu cụm câu đang nghe, các đáp án đã chọn
   được giữ.
5. **Given** nộp full test, **When** xem kết quả, **Then** có điểm Listening và Reading (5–495 mỗi phần, bước 5) và tổng
   điểm, đều ghi "ước tính", so với điểm mục tiêu nếu có.
6. **Given** người học tạo đề ngẫu nhiên 50 câu, **When** hệ thống dựng đề, **Then** đề có đủ 7 Part theo tỉ lệ của đề thật
   và ưu tiên câu người học chưa làm.
7. **Given** nộp Mini Test 100 câu, **When** xem kết quả, **Then** điểm được quy đổi theo tỉ lệ số câu đúng của từng phần.
8. **Given** đã làm full test Luna Test 01 được 610, **When** làm lại và được 700, **Then** thẻ đề hiện 700, tab Kết quả có
   cả hai lần (lần sau có nhãn "làm lại"), còn biểu đồ điểm và "điểm ước tính gần nhất" vẫn dùng 610.

---

### User Story 6 - Luyện theo dạng câu, thống kê và mục tiêu (Priority: P4)

Người học đặt điểm mục tiêu và ngày thi. Trang "Luyện theo dạng câu" liệt kê các #dạng câu kèm độ chính xác, yếu nhất
lên đầu; bấm một dạng để làm 10 câu dạng đó. Trang thống kê có biểu đồ điểm ước tính qua các đề (có đường mục tiêu) và độ
chính xác theo Part và dạng câu. Dạng câu ngữ pháp có link sang điểm ngữ pháp tương ứng trong phần Ngữ pháp của Luna.

**Why this priority**: giúp luyện đúng chỗ yếu và giữ động lực, nhưng cần dữ liệu từ nhiều lần làm bài trước đó.

**Independent Test**: sau vài lần làm bài, đặt mục tiêu 750 và ngày thi; kiểm tra thư viện đề hiện mục tiêu và số ngày
còn lại; danh sách dạng câu xếp đúng thứ tự độ chính xác; bấm một dạng mở lượt luyện đúng dạng đó.

**Acceptance Scenarios**:

1. **Given** người học chọn mục tiêu 750 và ngày thi 15/01/2027, **When** mở thư viện đề, **Then** cột bên hiện mục tiêu,
   số ngày còn lại và điểm ước tính gần nhất so với mục tiêu.
2. **Given** đã làm nhiều lượt, **When** mở Luyện theo dạng câu, **Then** các dạng xếp theo độ chính xác tăng dần, mỗi dạng
   có Part và số câu sẵn có.
3. **Given** bấm "#Từ loại", **When** lượt luyện mở ra, **Then** mọi câu đều có dạng #Từ loại, ưu tiên câu chưa làm.
4. **Given** dạng "#Giới từ" có điểm ngữ pháp tương ứng, **When** xem dạng đó, **Then** có link mở điểm ngữ pháp đó.
5. **Given** đã làm 3 đề, **When** mở thống kê, **Then** biểu đồ có 3 điểm theo thời gian và đường mục tiêu.

---

### Edge Cases

- Chưa có đề nào được đăng: thư viện đề báo "Chưa có đề", các tab luyện chỉ hiện các Part đã có câu.
- Một Part trong đề không đủ câu cho lựa chọn luyện tập: chỉ cho tick các Part có câu.
- Máy không có giọng đọc tiếng Anh hoặc chỉ có một giọng: vẫn phát được; mọi nhân vật dùng chung giọng có sẵn và màn kiểm
  tra trước khi thi báo điều này.
- Trình duyệt chặn tự phát âm thanh: phòng thi yêu cầu người học bấm một nút trước khi phần Nghe bắt đầu.
- Mất mạng khi đang làm: đáp án vẫn chọn được và được gửi lại khi có mạng; nộp bài khi mất mạng thì báo và thử lại, không mất
  bài làm.
- Đang làm dở một đề mà bắt đầu đề khác: bài dở vẫn được giữ; mỗi đề tối đa một bài đang làm dở.
- Admin ẩn một cụm khi người học đang làm dở bài có cụm đó: bài dở vẫn làm và chấm bình thường với nội dung lúc bắt đầu.
- Admin sửa (nhập lại) một cụm đã có người làm: các lần làm đã nộp được chấm lại theo đáp án mới (kết quả, điểm ước tính,
  thống kê đổi theo); câu trước đây sai nay thành đúng thì ra khỏi sổ câu sai, câu trước đúng nay thành sai thì vào sổ. Bài
  đang làm dở dùng nội dung mới khi mở lại.
- Nhập lại đổi số câu của một cụm đang nằm trong đề đã đăng: chạy thử báo lỗi, vì đề sẽ không còn đúng số câu mỗi Part.
- Hết giờ phần Đọc khi đang mất mạng: bài được nộp với các đáp án đã chọn ngay khi có mạng lại, thời gian không cộng thêm.
- Câu sai trong sổ thuộc cụm đã bị ẩn: vẫn làm lại được trong sổ cho đến khi ra khỏi sổ.
- Ảnh Part 1 không tải được: hiện ô báo lỗi ảnh và nút thử lại; câu vẫn trả lời được.
- Khách mở một đề bị khoá bằng đường dẫn trực tiếp: hiện ghi chú "chỉ dành cho thành viên", không lộ nội dung đề.

## Requirements *(mandatory)*

### Functional Requirements

**Lối vào và thư viện đề**

- **FR-001**: Khu người học MUST có ô **TOEIC** trong thẻ của nút giữa **Luyện**, mở trang **thư viện đề**.
- **FR-002**: Thư viện đề MUST hiện lưới thẻ đề (tên, thời gian, số phần, số câu, trạng thái chưa làm / đang làm dở kèm câu
  đang làm / đã làm kèm kết quả gần nhất), thẻ **Đề ngẫu nhiên**, các tab **Đề thi, Luyện theo Part, Luyện theo dạng câu,
  Kết quả của tôi, Sổ câu sai** (kèm số câu), và cột bên có mục tiêu, ngày thi, điểm ước tính gần nhất, các lần làm gần đây.
- **FR-003**: Tên đề MUST do Luna đặt (ví dụ "Luna Test 01", "Mini Test 05"); không dùng tên hay nội dung đề của ETS hoặc
  sách đã xuất bản.

**Chi tiết đề**

- **FR-004**: Trang chi tiết đề MUST có ba tab: **Luyện tập**, **Làm full test**, **Kết quả** (mọi lần làm đề này: ngày, chế
  độ, các Part, kết quả, thời gian; lần thứ hai trở đi có nhãn "làm lại").
- **FR-004a**: Người học MUST làm lại một đề (luyện tập hoặc full test) không giới hạn số lần. Thẻ đề trong thư viện hiện kết
  quả lần gần nhất. Chỉ **lần làm full test / Mini Test đầu tiên** của mỗi đề (và mỗi đề ngẫu nhiên) được tính vào biểu đồ
  điểm và "điểm ước tính gần nhất"; các lần làm lại vẫn có trang kết quả đầy đủ, vẫn cập nhật sổ câu sai và độ chính xác theo
  dạng câu.
- **FR-005**: Tab Luyện tập MUST cho tick các Part (mỗi Part hiện số câu và các #dạng câu có trong đề) và chọn giới hạn thời
  gian: một mức gợi ý theo số câu đã chọn, vài mức khác hoặc không giới hạn.
- **FR-006**: Tab Làm full test MUST hiện cảnh báo điều kiện thi thật, thời lượng từng phần, mục kiểm tra trước khi thi (tai
  nghe, nơi yên tĩnh) và nút nghe thử giọng đọc.

**Phòng thi**

- **FR-007**: Phòng thi MUST có: tên đề và chế độ, nút Thoát, hàng tab Part, đồng hồ đếm ngược, nút **Nộp bài**, bảng câu
  chia theo Part với bốn trạng thái (đã làm, đánh dấu xem lại, chưa làm, đang xem), nút đánh dấu xem lại cho mỗi câu.
- **FR-008**: Trên màn rộng, Part 6/7 MUST hiện văn bản bên trái (cuộn riêng) và câu hỏi bên phải; Part 5 hiện nhiều câu một
  trang. Trên điện thoại MUST xếp dọc, thanh dưới có Bảng câu và Nộp bài, và có **phiếu trả lời** dạng ô tròn tô kín.
- **FR-009**: Người học MUST tô vàng được đoạn văn bản đã bôi đen; phần tô giữ suốt bài làm.
- **FR-010**: Trong phòng thi MUST không chấm từng câu, ở mọi chế độ (luyện tập, full test, Mini Test, đề ngẫu nhiên, luyện
  theo Part, luyện theo dạng câu); không có tuỳ chọn xem đáp án ngay. Hệ thống MUST không gửi đáp án hay giải thích về máy
  người học trước khi lượt làm được nộp. Chỉ lượt làm lại trong sổ câu sai mới chấm ngay từng câu (FR-031).
- **FR-011**: Nộp khi còn câu chưa làm MUST hỏi lại. Hết giờ MUST tự nộp với các đáp án đã chọn.
- **FR-012**: Bài làm dở MUST được lưu trên server (đáp án, đánh dấu, vị trí, thời gian còn lại) và làm tiếp được trên máy
  khác. Thời gian không chạy khi không làm, trừ khi đang ở full test (xem FR-020). Mỗi đề có tối đa một bài làm dở cho mỗi
  người học.
- **FR-013**: Mỗi Part MUST bắt đầu bằng màn **Directions** tiếng Anh như đề thật, được đọc lên ở phần Nghe, kèm một dòng
  tóm tắt tiếng Việt; khi luyện tập MUST bỏ qua được.

**Phần Nghe**

- **FR-014**: Âm thanh phần Nghe MUST dùng giọng đọc có sẵn trên máy người học, đọc theo kịch bản như đề thật: lời hướng
  dẫn, câu dẫn ("Questions 41 through 43 refer to the following conversation"), nội dung, rồi đọc từng câu hỏi kèm khoảng
  nghỉ để trả lời (khoảng 8 giây). Không cần file âm thanh lưu sẵn.
- **FR-015**: Mỗi người nói trong kịch bản MUST có một giọng (Mỹ, Anh, Úc, Canada); máy có nhiều giọng tiếng Anh thì các
  người nói khác nhau trong một cụm MUST dùng giọng khác nhau.
- **FR-016**: Part 1 MUST chỉ hiện ảnh và các nút A–D; Part 2 chỉ hiện "Mark your answer" và các nút A–C; Part 3/4 hiện câu
  hỏi và phương án bằng chữ.
- **FR-017**: Câu "Look at the graphic" MUST hiện sơ đồ, bảng giá, lịch hoặc biểu đồ dựng từ dữ liệu của cụm, đọc rõ ở
  360px và ở cả chế độ sáng và tối.
- **FR-018**: Khi luyện tập, âm thanh MUST tự chạy liền theo kịch bản như đề thật (Directions nếu không bỏ qua, các cụm, câu
  hỏi, khoảng nghỉ) và tự sang cụm tiếp theo; người học MUST tạm dừng / tiếp tục, nghe lại cụm đang phát và đổi tốc độ
  (0.75×, 1×, 1.25×) được. Nhảy tới câu khác trong bảng câu thì âm thanh chuyển tới đầu cụm của câu đó.

**Full test và Mini Test**

- **FR-019**: Full test MUST đi theo thứ tự: phần Nghe (Part 1–4, khoảng 45 phút theo kịch bản) rồi phần Đọc (Part 5–7, 75
  phút). Khi đang ở phần nào thì các tab của phần kia MUST bị khoá.
- **FR-020**: Phần Nghe của full test MUST phát liền: không tạm dừng, không tua, không nhảy câu; chỉ sửa được đáp án các câu
  của cụm đang phát. Đóng app giữa chừng thì làm tiếp từ đầu cụm đang nghe, đáp án đã chọn được giữ.
- **FR-021**: Phần Đọc của full test MUST cho nhảy câu tự do trong phần Đọc; đóng app thì làm tiếp đúng câu với thời gian
  còn lại lúc đóng.
- **FR-022**: Mini Test MUST có 50 câu (38 phút) hoặc 100 câu (60 phút), đủ 7 Part theo tỉ lệ của đề thật, cùng cách làm như
  full test.
- **FR-023**: **Đề ngẫu nhiên** MUST rút 50 hoặc 100 câu đã đăng theo tỉ lệ các Part, ưu tiên câu người học chưa làm; đề
  ngẫu nhiên được lưu như một lần làm để xem lại kết quả.

**Kết quả và đáp án chi tiết**

- **FR-024**: Trang kết quả MUST có: số câu đúng, sai, bỏ qua; độ chính xác; thời gian làm; với bài có phần Nghe hoặc Đọc đủ
  để quy đổi thì có điểm Listening và Reading (5–495, bước 5) và tổng điểm, đều ghi chữ **"ước tính"**, so với mục tiêu nếu
  có.
- **FR-025**: Điểm MUST được quy đổi từ số câu đúng mỗi phần theo một bảng quy đổi gần đúng cố định; bài ít hơn 100 câu một
  phần thì quy đổi theo tỉ lệ đúng. Lượt luyện tập chỉ vài Part thì chỉ hiện số câu đúng và độ chính xác, không hiện điểm
  thang 990.
- **FR-026**: Trang kết quả MUST có bảng **Phân loại câu hỏi** theo từng Part: mỗi #dạng câu có số đúng, sai, bỏ qua, độ
  chính xác và danh sách số câu tô màu theo kết quả; bấm số MUST mở đáp án chi tiết câu đó.
- **FR-027**: Trang kết quả MUST có danh sách "đáp án đúng : bạn chọn" theo Part và nút **Làm lại các câu sai**.
- **FR-028**: Đáp án chi tiết MUST gồm: văn bản hoặc transcript (ghi người nói và giọng) với câu làm bằng chứng được tô và
  ghi số câu, bản dịch tiếng Việt, giải thích tiếng Việt (vì sao đúng, các phương án khác sai ở đâu, bẫy thường gặp), #dạng
  câu, từ vựng đáng chú ý kèm nút thêm vào sổ từ; câu phần Nghe nghe lại được cả cụm.

**Luyện thêm, sổ câu sai, mục tiêu**

- **FR-029**: **Luyện theo Part** MUST cho chọn một Part và số câu (hoặc số cụm), lấy câu đã đăng từ mọi đề, ưu tiên câu chưa
  làm, làm trong phòng thi ở chế độ luyện tập. (2026-11-23) Mỗi lượt MUST có đồng hồ 15 hoặc 20 phút (mặc định 15, không có
  "không giới hạn"), hết giờ tự nộp.
- **FR-030**: **Luyện theo dạng câu** MUST liệt kê các #dạng câu có câu đã đăng kèm độ chính xác của người học, yếu nhất lên
  đầu (dạng chưa làm xếp cuối); bấm một dạng mở lượt 10 câu (hoặc vài cụm) dạng đó. Dạng câu ngữ pháp có điểm ngữ pháp tương
  ứng trong Luna MUST có link sang điểm đó.
- **FR-031**: Câu sai hoặc bỏ qua sau khi nộp MUST vào **sổ câu sai**. Sổ lọc được theo Part; làm lại trong sổ thì chấm ngay
  từng câu; đúng 2 lần liên tiếp thì ra khỏi sổ, sai thì đếm lại từ 0. Câu thuộc cụm thì hiện cả cụm nhưng chỉ chấm câu đó.
- **FR-032**: Làm lại các câu sai từ trang kết quả MUST mở một lượt chỉ gồm các câu sai của lần làm đó; kết quả lượt này cũng
  cập nhật sổ câu sai.
- **FR-033**: Người học MUST đặt được điểm mục tiêu (450, 600, 750, 900) và ngày thi (không bắt buộc), sửa hoặc xoá được.
- **FR-034**: Trang **thống kê** MUST có biểu đồ điểm ước tính của các lần làm đầu tiên có điểm (theo FR-004a, có đường mục
  tiêu) và độ chính xác theo Part và theo dạng câu (tính từ mọi lần làm).
- **FR-035**: Hoàn thành (nộp) một lượt luyện, một đề hoặc một lượt làm lại câu sai MUST được tính là một ngày học (chuỗi
  ngày học chung của Luna).

**Ảnh trong đề**

- **FR-036**: Mỗi câu Part 1 MUST có một ảnh kiểu ảnh chụp thật (văn phòng, kho, công trường, nhà hàng, đường phố…), không
  phải kiểu hoạt hình, không có chữ trong ảnh. Ảnh được vẽ bằng AI chỉ khi admin bấm, admin MUST chọn nhà cung cấp ảnh mỗi
  lần (Gemini hoặc Codex, không có mặc định); có Vẽ lại từng ảnh và tự tải ảnh lên.
- **FR-037**: Yêu cầu vẽ ảnh MUST dùng mô tả cảnh của cụm cùng câu mô tả đúng, không gửi các câu mô tả sai, để ảnh không vô
  tình khớp với câu sai. Mọi lần vẽ được ghi vào trang Sử dụng AI như các chức năng AI khác.
- **FR-038**: Văn bản Part 6/7 MUST hiện có bố cục giống văn bản thật theo loại (email, thư, thông báo, quảng cáo, trang web,
  hoá đơn, mẫu đơn, lịch, tin nhắn/chat), có thể kèm ảnh minh hoạ; cụm có nhiều văn bản MUST chuyển qua lại được giữa các văn
  bản.
- **FR-039**: Một đề MUST không đăng được khi còn câu Part 1 chưa có ảnh.

**Nội dung và quản trị**

- **FR-040**: Admin MUST nhập được nội dung từ file có cấu trúc. Chạy thử MUST kiểm tra toàn bộ, báo cáo từng cụm (hợp lệ,
  trùng sẽ bỏ qua, lỗi theo từng trường) và không lưu gì. Nhập thật MUST từ chối cả file khi còn lỗi; cụm mới lưu ở dạng nháp.
- **FR-040a**: Cụm có mã trùng cụm đã có MUST được hiểu là bản sửa khi admin chọn "thay thế" (không chọn thì báo trùng và bỏ
  qua): nội dung mới đè lên cụm cũ, giữ mã cụm và trạng thái đăng. Mọi lần làm đã nộp có câu của cụm đó MUST được chấm lại
  theo đáp án mới và sổ câu sai cập nhật theo. Bản sửa MUST giữ nguyên số câu của cụm nếu cụm đang nằm trong một đề đã đăng.
- **FR-041**: Một cụm câu MUST gồm: Part, mức độ khó (theo mốc điểm), ngữ liệu theo Part (văn bản có loại và bố cục; kịch
  bản lời nói có người nói và giọng; mô tả cảnh tiếng Anh cho Part 1; dữ liệu biểu đồ khi có), và các câu hỏi (số thứ tự
  trong cụm, câu hỏi, phương án, đáp án, câu làm bằng chứng, bản dịch, giải thích tiếng Việt, một hoặc vài #dạng câu từ
  danh sách cố định). Số câu và số phương án MUST đúng với Part (Part 2: 3 phương án; Part 3/4: 3 câu một cụm; Part 6: 4 câu
  một văn bản; Part 7: 2–5 câu một cụm).
- **FR-042**: Admin MUST xem trước một cụm như người học thấy (kể cả đáp án và giải thích), Đăng và Ẩn từng cụm, lọc danh sách
  cụm theo Part và trạng thái.
- **FR-042a**: Trang đầu quản trị TOEIC MUST chỉ là danh sách đề (mỗi đề một thẻ: loại, trạng thái, số câu đã có/cần, từng
  Part đủ/thiếu, lý do chưa đăng; bấm thẻ mở trang đề) cùng nút Tạo đề và số liệu kho cụm câu. Mọi cụm đã nhập MUST nằm ở
  trang riêng **Kho cụm câu** (`/admin/toeic/groups`): Nhập JSON, xem trước, Đăng/Ẩn, lọc theo Part, trạng thái và "Thuộc
  đề" (có "Chưa thuộc đề nào"). Trang đề và trang kho có dòng đường dẫn "TOEIC › …".
- **FR-042b**: Trong Kho cụm câu, admin MUST nhờ AI (op `toeic_groups`, mặc định gemini-3.6-flash, đổi được ở trang Model AI) soạn khoảng 10 câu của một Part mỗi lần bấm (Part 1, 2, 5: 10 cụm; Part 3, 4: 4 cụm = 12 câu; Part 6: 3 văn bản = 12 câu; Part 7: các cụm cộng lại đúng 10 câu), kèm mức điểm và gợi ý tuỳ chọn. Kho cụm câu MUST lọc được theo nguồn (AI soạn / tự nhập). Mỗi cụm MUST qua đúng kiểm tra của lần nhập; cụm qua được lưu nháp, nguồn `ai`, có nhãn AI trong bảng; cụm hỏng được báo theo trường và không lưu. Cụm trùng với một cụm đã có của Part (kể cả nháp) hoặc với cụm khác trong cùng lần soạn — so trên máy chủ, không gọi AI: câu Part 5, câu hỏi Part 2, cảnh + câu mô tả Part 1, lời thoại Part 3/4, văn bản Part 6/7, giống từ 80% trở lên — MUST bị bỏ và báo "Trùng với cụm …". Admin vẫn phải xem trước và đăng.
- **FR-042c**: Trong Kho cụm câu, admin MUST xoá được một cụm (nút "Xoá" trên dòng) hoặc nhiều cụm (ô chọn từng dòng, ô chọn cả trang, "Xoá N cụm đã chọn…"), luôn qua hộp thoại xác nhận; xoá cụm thì xoá cả ảnh của nó. Cụm đang thuộc một đề (mọi trạng thái) hoặc đã có bài làm đã nộp của người học MUST được giữ lại và báo lý do (gỡ khỏi đề trước / ẩn thay vì xoá), để không làm hỏng đề, kết quả và sổ lỗi sai.
- **FR-043**: Admin MUST tạo đề đầy đủ (6/25/39/30/30/16/54 câu theo Part 1–7) hoặc Mini Test bằng cách ghép các cụm đã đăng.
  Trang đề MUST hiện số câu đã có và còn thiếu mỗi Part, số ảnh Part 1 đã có, số biểu đồ và văn bản; chỉ đăng được đề khi đủ
  câu và đủ ảnh. Đề đã đăng MUST ẩn được.
- **FR-043a**: **Ghép đề tự động** MUST lấy cụm đã đăng cho đúng số câu từng Part theo loại đề (Mini Test lệch tối đa một cụm), ưu tiên cụm chưa thuộc đề nào rồi cụm ít dùng nhất, cho xem trước số câu, số cụm, số cụm dùng lại từng Part rồi mới tạo đề nháp (tên, mã tự đặt nếu bỏ trống). Part không đủ câu MUST được nêu rõ; admin chọn "vẫn tạo" thì Part đó để trống. (2026-11-23) Mỗi Part MUST trộn mức điểm theo tỉ lệ số câu 20/30/30/20% (450/600/750/900, gần nhất mà cỡ cụm cho phép, không lệch hẳn về một mức qua nhiều đề) và xếp câu dễ trước. Đề đầy đủ MUST theo cấu trúc đề thật: Part 3 có đúng 3 cụm có biểu đồ, Part 4 đúng 2, đặt cuối Part; Part 7 có 29 câu bài đơn, rồi đúng 2 cụm 2 bài và 3 cụm 3 bài. Mini Test theo bản thu nhỏ (Mini 100: 1 cụm biểu đồ ở Part 3 và Part 4, Part 7 có 1 cụm 2 bài và 1 cụm 3 bài; Mini 50: như vậy nhưng Part 7 chỉ 1 cụm 2 bài). Kho không đủ cụm cho cấu trúc thì Part đó ghép theo số câu như trước và xem trước MUST ghi chú điều đó; xem trước MUST cho thấy số câu từng mức của mỗi Part.
- **FR-043b**: (2026-11-23) Admin MUST xoá được đề chưa đăng (ở thẻ đề và trang đề, có hỏi lại); các cụm của đề vẫn ở trong kho. Đề đang đăng MUST ẩn trước khi xoá; đề đã có bài làm của người học (đã nộp hay đang làm) MUST không xoá được, báo "hãy ẩn thay vì xoá".
- **FR-044**: Khi người học làm bài, hệ thống MUST không gọi AI.

**Quyền và dữ liệu**

- **FR-045**: Tài khoản khách MUST làm được **1 Mini Test** (đề Mini Test đầu tiên trong thư viện) và **Luyện theo Part** cho
  Part 1, 2, 5. Các đề khác, các Part khác, sổ câu sai, luyện theo dạng câu và thống kê MUST hiện khoá kèm ghi chú "chỉ dành
  cho thành viên"; server MUST từ chối các yêu cầu đó của khách.
- **FR-046**: Chỉ admin MUST nhập, xem trước, đăng, ẩn nội dung, ghép đề và vẽ ảnh.
- **FR-047**: Bài làm, sổ câu sai và mục tiêu là dữ liệu của từng người học: MUST chỉ người đó xem được, MUST có trong file
  xuất dữ liệu và MUST bị xoá khi xoá tài khoản.

**Giao diện**

- **FR-048**: Giao diện MUST theo hệ thống thiết kế hiện có (bảng màu, font Lexend, sáng/tối), dùng được ở 360px và ở màn
  rộng; đúng/sai có biểu tượng ✓/✗ ngoài màu; nút chạm tối thiểu 44px trên điện thoại.

### Key Entities

- **Cụm câu (TOEIC item group)**: đơn vị nội dung nhỏ nhất được nhập và đăng: một Part, mức độ khó, ngữ liệu (văn bản, kịch
  bản lời nói, mô tả cảnh, dữ liệu biểu đồ), một hoặc nhiều câu hỏi, trạng thái nháp / đã đăng / ẩn, nguồn (tên file).
- **Câu hỏi**: thuộc một cụm; câu hỏi, phương án, đáp án, câu làm bằng chứng, bản dịch, giải thích tiếng Việt, #dạng câu.
- **Dạng câu (question type)**: danh sách cố định theo Part (ví dụ Từ loại, Thì động từ, Giới từ, Liên từ, Từ vựng, Câu chèn,
  Ý chính, Chi tiết, Suy luận, NOT true, Hàm ý câu nói, Biểu đồ, Câu hỏi WH, Yes/No, Câu trả lời gián tiếp, Tranh tả người,
  Tranh tả vật…); một số dạng nối với điểm ngữ pháp của Luna.
- **Ảnh câu**: ảnh của một câu Part 1 (hoặc ảnh minh hoạ văn bản Part 7), có nguồn (AI hay tải lên).
- **Đề (test)**: tên, loại (đầy đủ / Mini Test), danh sách cụm theo Part và thứ tự, trạng thái nháp / đã đăng / ẩn.
- **Bài làm (attempt)**: của một người học: đề (hoặc đề ngẫu nhiên, lượt luyện theo Part/dạng câu/câu sai), chế độ (luyện
  tập / full test), các Part, đáp án từng câu, câu đánh dấu, vị trí, thời gian còn lại, trạng thái đang làm / đã nộp, kết quả
  (đúng/sai/bỏ qua từng câu, điểm ước tính), thời điểm bắt đầu và nộp.
- **Mục sổ câu sai**: người học, câu hỏi, số lần đúng liên tiếp, lần sai gần nhất.
- **Mục tiêu TOEIC**: người học, điểm mục tiêu, ngày thi.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Từ thanh tab, người học mở được thư viện đề trong 2 lần chạm và bắt đầu một lượt luyện tập trong không quá 4
  lần chạm nữa.
- **SC-002**: Chọn đáp án, đánh dấu và nhảy câu phản hồi ngay (dưới 0,1 giây cảm nhận), kể cả khi mạng chậm.
- **SC-003**: Làm dở rồi mở lại trên máy khác, người học tiếp tục đúng câu với sai lệch thời gian còn lại không quá 5 giây
  (phần Đọc), 100% trường hợp có mạng.
- **SC-004**: 0 lần đáp án hoặc giải thích có trong dữ liệu gửi về máy người học trước khi nộp (kiểm tra trên mọi chế độ làm
  đề).
- **SC-005**: Trang kết quả hiện đầy đủ trong vòng 2 giây sau khi nộp một đề 200 câu.
- **SC-006**: Một full test chạy hết phần Nghe theo kịch bản trong khoảng 40–50 phút, không có chỗ dừng ngoài các khoảng
  nghỉ của kịch bản.
- **SC-007**: Admin nhập và kiểm tra một file 50 cụm (chạy thử, sửa lỗi, nhập, đăng) trong dưới 10 phút; mọi lỗi chạy thử chỉ
  rõ cụm, câu và trường.
- **SC-008**: 100% đề đã đăng có đủ câu theo Part và đủ ảnh Part 1.
- **SC-009**: Toàn bộ màn hình (thư viện, chi tiết đề, phòng thi, kết quả, đáp án chi tiết, sổ câu sai, thống kê, quản trị)
  dùng được ở 360px và màn rộng, sáng và tối, không có cuộn ngang.
- **SC-010**: Người học ra khỏi sổ câu sai đúng sau 2 lần đúng liên tiếp trong 100% trường hợp kiểm tra.

## Assumptions

- **Phát hành theo 4 đợt**, mỗi đợt dùng được độc lập: Đợt 1 = US1 + US2 + US3 (thư viện đề, luyện tập phần Đọc, kết quả, đáp
  án chi tiết, sổ câu sai, quản trị nhập và ghép đề); Đợt 2 = US4 (phần Nghe, ảnh, biểu đồ, Directions); Đợt 3 = US5 (full
  test, Mini Test, đề ngẫu nhiên, điểm ước tính); Đợt 4 = US6. Ở đợt 1, đề chỉ cần đủ Part 5–7 để luyện tập; tab Làm full
  test và các Part chưa có hiện "sắp có".
- **Điểm ước tính** dùng một bảng quy đổi gần đúng cố định (số câu đúng → điểm mỗi phần), không phải bảng chính thức của
  ETS (ETS không công bố bảng cố định); luôn ghi "ước tính".
- **Nội dung** do Claude soạn mới hoàn toàn theo định dạng và độ khó của đề thật, không chép đề ETS hay sách đã xuất bản. Mục
  tiêu ban đầu: đủ cho ít nhất 2 đề đầy đủ và 2 Mini Test; sau đó bổ sung dần.
- **Giọng đọc** là giọng có sẵn trên trình duyệt/thiết bị của người học, giống mọi chỗ nghe khác trong Luna; chất lượng và
  số giọng khác nhau giữa các máy. Không lưu file âm thanh.
- **Ảnh Part 1** vẽ bằng nhà cung cấp ảnh AI đang dùng trong Luna (có tính phí theo lượt, người dùng đã đồng ý cho ảnh, như
  ảnh từ vựng và ảnh bài học); chỉ chạy khi admin bấm. Đây là ngoại lệ đã có của nguyên tắc chi phí 0 đồng.
- **Ảnh minh hoạ Part 7** (ảnh sản phẩm…) là tuỳ chọn; biểu đồ Part 3/4 và văn bản Part 6/7 không cần ảnh vì được dựng từ dữ
  liệu.
- **Thời gian làm dở**: ở chế độ luyện tập và phần Đọc của full test, đồng hồ dừng khi rời phòng thi; phần Nghe của full
  test làm tiếp từ đầu cụm đang nghe nên không cần lưu thời gian chính xác.
- **Bố cục** ưu tiên màn rộng cho phòng thi (như các trang luyện thi), đồng thời vẫn làm được trên điện thoại theo nguyên
  tắc mobile-first của Luna.
- **Sổ câu sai** của TOEIC tách riêng với thẻ ôn từ vựng; chỉ từ vựng người học bấm "thêm vào sổ từ" mới vào thẻ ôn.
- **Chuỗi ngày học**: dùng chuỗi chung hiện có của Luna, thêm "nộp một lượt/đề TOEIC" như một hoạt động tính ngày học.
- **Ngoài phạm vi**: phần Nói–Viết của TOEIC; bảng xếp hạng hay so sánh giữa người học; bình luận dưới câu hỏi; lộ trình học
  theo ngày tự động (chỉ có mục tiêu, thống kê và luyện theo dạng câu yếu); AI sinh đề tự động; tải đề về máy hay in đề.
