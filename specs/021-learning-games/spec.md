# Feature Specification: Trò chơi — vừa chơi vừa học

**Feature Branch**: `021-learning-games` (làm trên `main`, như các spec 018–020)

**Created**: 2026-11-13

**Status**: Draft

**Mã tính năng**: F26 (mới, F23–F25 đã dùng) | **Giai đoạn**: 3 (`docs/phases/giai-doan-3.md`)

**Input**: User description: "Khu trò chơi 'vừa chơi vừa học' phía người học, dùng lại dữ liệu đã có và ghi kết quả về
lịch ôn. 6 trò theo ưu tiên: Ảnh nhanh tay, Đấu động từ, Chữ rơi, Lật thẻ ghi nhớ, Bắn lỗi ngữ pháp, Thử thách hằng
ngày. Có màn chọn trò, kỷ lục cá nhân, mỗi lượt 1–3 phút, chơi tốt trên điện thoại một tay."

**Phác thảo đã duyệt**: `design/de-xuat-tro-choi.gif` (trò 1–4), `design/de-xuat-tro-choi-2.gif` (Bắn lỗi, Thử thách
hằng ngày).

## Tóm tắt

Người học có thêm mục **Trò chơi**: các lượt chơi ngắn (1–3 phút), có đếm giờ, combo, mạng và kỷ lục, để ôn lại những
gì đã học vào lúc không muốn học bài. Trò chơi **chủ yếu dùng nội dung có sẵn**: từ và ảnh trong Kho từ vựng, danh sách
động từ bất quy tắc và bài tập ngữ pháp đã duyệt. Chỉ riêng Bắn lỗi ngữ pháp có thêm nguồn câu do AI soạn khi admin bấm
và admin duyệt. Từ chơi sai được đưa về hàng ôn để gặp lại sớm. Mỗi trò là một phần
riêng, làm và phát hành độc lập theo thứ tự ưu tiên P1 → P6.

## Clarifications

### Session 2026-11-13

- Q: Bảng xếp hạng của Thử thách hằng ngày gồm những ai? → A: Chỉ so với chính mình (hôm qua, trung bình 7 ngày, kỷ
  lục), không có bảng giữa người dùng.
- Q: Vào mục Trò chơi từ đâu? → A: Thêm ô "Trò chơi" vào thẻ của nút giữa Luyện, mở trang danh sách trò; Thử thách hằng
  ngày có thẻ riêng ở Trang chủ.
- Q: Lấy câu cho Bắn lỗi ngữ pháp ở đâu? → A: Bài tập "chọn đáp án" (điền phương án sai vào chỗ trống) và bài "sửa câu"
  khác đúng một chữ, kèm nút để admin nhờ AI soạn thêm câu và duyệt.
- Q: Có lưu lại từng lượt chơi không? → A: Không. Chỉ lưu kỷ lục mỗi trò và kết quả thử thách mỗi ngày; lượt thường
  không lưu sau khi xử lý xong.
- Q: Tài khoản khách được chơi những gì? → A: Mọi trò trừ Bắn lỗi (khoá, "chỉ dành cho thành viên"); Thử thách ngày
  của khách không có câu ngữ pháp; nội dung chỉ từ các bài khách được học.
- Q: Mỗi câu trong Thử thách hằng ngày diễn ra thế nào? → A: Bản rút gọn từng loại (Ảnh 4 đáp án, Động từ gõ V2+V3,
  Ghép chữ xáo trộn không rơi, Bắn lỗi 1 câu), không có Lật thẻ, mỗi câu giới hạn 20 giây, không có đồng hồ chung.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ảnh nhanh tay (Priority: P1)

Người học mở mục Trò chơi, chọn "Ảnh nhanh tay". Mỗi câu hiện một ảnh và 4 từ tiếng Anh, người học chạm vào từ đúng.
Lượt chơi dài 60 giây. Đúng liên tiếp thì combo tăng và điểm mỗi câu được nhân theo combo. Sai thì combo về x1 và hiện
đáp án đúng trong chốc lát. Hết giờ thì hiện màn kết quả: điểm, số câu đúng/sai, kỷ lục và danh sách từ đã sai.

**Why this priority**: tốn ít công nhất, dùng ngay hàng trăm ảnh trong Kho từ. Làm trò này sẽ dựng luôn phần dùng
chung cho mọi trò: màn chọn trò, đếm giờ, điểm, combo, màn kết quả, kỷ lục, ghi từ sai về hàng ôn.

**Independent Test**: chơi một lượt 60 giây với tài khoản đã học ít nhất một bài, kiểm tra điểm, combo, màn kết quả,
kỷ lục mới được lưu và từ chơi sai xuất hiện trong lượt ôn hôm nay.

**Acceptance Scenarios**:

1. **Given** người học đã học từ có ảnh, **When** mở "Ảnh nhanh tay", **Then** mỗi câu hiện một ảnh, 4 đáp án khác
   nhau, trong đó đúng 1 đáp án khớp ảnh, và đồng hồ đếm ngược 60 giây.
2. **Given** combo đang là x3, **When** chọn đúng, **Then** combo lên x4 và điểm câu đó bằng điểm gốc nhân 4.
3. **Given** combo đang là x3, **When** chọn sai, **Then** combo về x1, đáp án đúng được tô xanh khoảng 1 giây rồi mới
   sang câu mới.
4. **Given** hết 60 giây, **When** lượt kết thúc, **Then** hiện điểm, số câu đúng/sai, kỷ lục cá nhân (báo "Kỷ lục
   mới" nếu vượt) và danh sách từ đã sai kèm nghĩa.
5. **Given** người học chọn sai từ "bicycle", **When** lượt kết thúc, **Then** thẻ ôn của "bicycle" đến hạn ôn ngay
   hôm nay.
6. **Given** người học thoát giữa lượt, **When** quay lại, **Then** lượt đó không được tính điểm hay kỷ lục, nhưng từ
   đã sai vẫn được đưa về hàng ôn.

---

### User Story 2 - Đấu động từ (Priority: P2)

Người học thấy một động từ nguyên mẫu (V1) kèm ảnh và nghĩa, gõ V2 rồi V3. Có 3 mạng: sai một ô thì mất 1 mạng và hiện
đáp án đúng. Chơi đến khi hết mạng. Số câu đúng liên tiếp là điểm của lượt, và kỷ lục là chuỗi dài nhất.

**Why this priority**: đúng chỗ người học hay quên, dùng lại danh sách và ảnh động từ đã có, chỉ cần gõ nên dễ làm.

**Independent Test**: chơi đến hết 3 mạng, kiểm tra chuỗi, đáp án hiện khi sai, kỷ lục và việc động từ sai đến hạn ôn.

**Acceptance Scenarios**:

1. **Given** lượt mới, **When** bắt đầu, **Then** động từ lấy từ các động từ người học đã học. Nếu đã học dưới 10
   động từ thì bổ sung động từ thông dụng nhất.
2. **Given** động từ "go", **When** gõ "went" và "gone", **Then** cả hai ô chuyển xanh và chuỗi tăng 1.
3. **Given** động từ có hai dạng đúng (ví dụ "learned/learnt"), **When** gõ dạng nào trong số đó, **Then** đều tính
   đúng.
4. **Given** động từ "run", **When** gõ "runned", **Then** mất 1 mạng, hiện "V2 ran · V3 run", và động từ này được
   đưa về hàng ôn động từ.
5. **Given** còn 1 mạng, **When** sai tiếp, **Then** lượt kết thúc và hiện màn kết quả có chuỗi, kỷ lục và các động
   từ đã sai.
6. **Given** người học gõ đúng trong trò chơi, **When** lượt kết thúc, **Then** lịch ôn của động từ đó không thay đổi
   (chơi đúng không đẩy lịch ôn ra xa).

---

### User Story 3 - Chữ rơi (Priority: P3)

Màn hình hiện nghĩa tiếng Việt (ví dụ "ngôi nhà"), các chữ cái của từ cần ghép rơi xuống cùng vài chữ nhiễu. Người học
chạm các chữ theo đúng thứ tự (h-o-u-s-e). Ghép xong thì hiện ảnh và từ đúng, rồi sang từ tiếp theo. Mỗi màn rơi nhanh
hơn và thêm chữ nhiễu. Có 3 mạng: chạm sai chữ hoặc để chữ cần chạm rơi hết màn hình thì mất 1 mạng.

**Why this priority**: vui nhất nhóm từ vựng và giúp nhớ chính tả, nhưng phần chuyển động tốn công hơn hai trò trên.

**Independent Test**: chơi qua ít nhất 2 màn, kiểm tra tốc độ tăng, mất mạng khi chạm sai, ghép đúng thì hiện ảnh.

**Acceptance Scenarios**:

1. **Given** từ cần ghép là "house", **When** chạm lần lượt h, o, u, s, e, **Then** các ô bên dưới điền dần, đủ chữ
   thì hiện ảnh và "house ✓" rồi sang từ mới.
2. **Given** đang chờ chữ "o", **When** chạm chữ khác, **Then** mất 1 mạng và chữ đó không được điền.
3. **Given** từ có chữ lặp (ví dụ "apple"), **When** chạm "p" lần thứ nhất và thứ hai, **Then** cả hai lần đều được
   tính theo thứ tự.
4. **Given** đã xong một màn (5 từ), **When** sang màn sau, **Then** chữ rơi nhanh hơn và có thêm chữ nhiễu.
5. **Given** từ có khoảng trắng hoặc dấu gạch (cụm từ), **When** chọn từ cho trò, **Then** những từ đó bị bỏ qua.
   Chỉ dùng từ một chữ, dài 3–8 ký tự.

---

### User Story 4 - Lật thẻ ghi nhớ (Priority: P4)

Lưới 12 thẻ úp (6 cặp). Mỗi cặp gồm một thẻ ảnh và một thẻ từ tiếng Anh. Người học lật 2 thẻ: khớp thì giữ lại, lệch thì
2 thẻ rung rồi úp xuống. Ghép đủ 6 cặp là xong lượt. Điểm tính theo số lượt lật và thời gian, kỷ lục là kết quả tốt
nhất.

**Why this priority**: dễ làm, hợp làm phần "chơi thêm" ngay sau bài học, nhưng ít tạo cảm giác thử thách hơn.

**Independent Test**: chơi xong một lưới, kiểm tra cặp khớp được giữ, cặp lệch úp lại, số lượt và thời gian đúng.

**Acceptance Scenarios**:

1. **Given** người học vừa học xong một bài, **When** mở "Lật thẻ" từ màn xong bài, **Then** 6 cặp lấy từ từ vựng của
   bài đó (ưu tiên từ có ảnh).
2. **Given** mở "Lật thẻ" từ mục Trò chơi, **When** bắt đầu, **Then** 6 cặp lấy từ các từ đã học, ưu tiên từ đến hạn
   ôn.
3. **Given** đã lật 2 thẻ lệch, **When** người học chạm thẻ thứ ba, **Then** 2 thẻ lệch úp lại ngay và thẻ thứ ba được
   lật (không bắt chờ).
4. **Given** từ không có ảnh, **When** tạo cặp, **Then** thẻ thứ hai hiện nghĩa tiếng Việt thay cho ảnh.
5. **Given** ghép xong 6 cặp, **When** lượt kết thúc, **Then** hiện số lượt, thời gian, điểm và kỷ lục.

---

### User Story 5 - Bắn lỗi ngữ pháp (Priority: P5)

Mỗi câu có thể có **một chữ sai** hoặc không có lỗi. Câu có lỗi: người học chạm vào chữ sai, chọn cách sửa trong 3 phương
án. Câu đúng: người học bấm "Không có lỗi". Sau mỗi câu hiện quy tắc ngắn và nhãn chủ điểm ngữ pháp. Lượt gồm 10 câu,
mỗi câu có thanh đếm giờ riêng, có 3 mạng.

**Why this priority**: luyện ngữ pháp theo cách mới, nhưng cần câu có đúng một chữ sai và phương án sửa, nên phải lọc
hoặc bổ sung nội dung.

**Independent Test**: chơi một lượt 10 câu gồm cả câu có lỗi và câu đúng, kiểm tra chạm chữ sai, chọn sửa, quy tắc,
mạng và kết quả.

**Acceptance Scenarios**:

1. **Given** câu "She go to school every day.", **When** chạm "go", **Then** chữ chuyển đỏ và hiện 3 phương án
   (ví dụ goes / going / went).
2. **Given** đã chạm đúng chữ sai, **When** chọn "goes", **Then** chữ thay thành "goes" màu xanh, cộng điểm và hiện
   quy tắc kèm nhãn "Hiện tại đơn".
3. **Given** câu đúng "They are playing football now.", **When** bấm "Không có lỗi", **Then** được tính đúng và hiện
   lý do câu đúng.
4. **Given** câu có lỗi, **When** chạm vào một chữ không sai hoặc bấm "Không có lỗi", **Then** mất 1 mạng, hiện chữ
   sai và cách sửa đúng.
5. **Given** người học chỉ học tới một số điểm ngữ pháp, **When** tạo lượt, **Then** câu ưu tiên các điểm ngữ pháp
   người học đã học, có thể lẫn điểm khác cùng trình độ.
6. **Given** bài tập chọn đáp án "Hoa says to ___" (they / them / their, đúng là them), **When** tạo câu có lỗi, **Then**
   câu hiện "Hoa says to they" hoặc "…their", và 3 phương án sửa là they / them / their.
7. **Given** admin mở một điểm ngữ pháp có dưới 10 câu Bắn lỗi, **When** bấm nhờ AI soạn thêm, **Then** nhận tối đa 10
   câu nháp. Câu nháp chưa xuất hiện trong trò chơi cho đến khi admin duyệt.

---

### User Story 6 - Thử thách hằng ngày (Priority: P6)

Mỗi ngày có một thử thách 10 câu, mỗi câu là bản rút gọn của một trò (ảnh, động từ, ghép chữ, bắn lỗi; xem FR-027a), lấy nội dung từ những gì
người học đã học. Màn mở đầu hiện chuỗi ngày thử thách và các ngày đã chơi trong tuần. Chơi xong hiện điểm, hàng 10 ô
xanh/đỏ, chuỗi ngày tăng, câu sai được đưa về hàng ôn, và nút "Chia sẻ" để sao chép kết quả dạng chữ. Mỗi ngày chỉ
lượt đầu tiên được tính.

**Why this priority**: giữ người học quay lại mỗi ngày tốt nhất, nhưng phụ thuộc vào các trò P1–P5 nên làm sau cùng.

**Independent Test**: chơi thử thách hôm nay, kiểm tra 10 câu, kết quả, chuỗi ngày tăng 1, chơi lại không đổi kết quả
đã tính, và hôm sau có bộ câu mới.

**Acceptance Scenarios**:

1. **Given** hôm nay chưa chơi, **When** mở Thử thách hằng ngày, **Then** hiện ngày, chuỗi hiện tại, các ngày đã chơi
   trong tuần và nút "Bắt đầu".
2. **Given** đang chơi, **When** trả lời mỗi câu, **Then** hàng chấm tiến độ tô xanh (đúng) hoặc đỏ (sai), và câu sau
   có thể thuộc trò khác.
3. **Given** đã chơi xong hôm qua, **When** chơi xong hôm nay, **Then** chuỗi tăng 1. Bỏ một ngày thì chuỗi về 1 khi
   chơi lại.
4. **Given** đã chơi xong hôm nay, **When** mở lại thử thách, **Then** thấy lại kết quả đã tính. Có thể chơi lại để
   luyện nhưng điểm và chuỗi đã tính không đổi.
5. **Given** màn kết quả, **When** bấm "Chia sẻ", **Then** một đoạn chữ ngắn được sao chép (ví dụ "Luna 13/11 · 9/10
   🔥13" và hàng ô 🟩/🟥), không chứa đáp án hay thông tin cá nhân.
6. **Given** màn kết quả, **When** xem phần so sánh, **Then** thấy điểm hôm nay so với hôm qua, với trung bình 7 ngày
   gần nhất và với kỷ lục thử thách. Không có bảng xếp hạng giữa người dùng (chốt 2026-11-13).
7. **Given** xong thử thách hôm nay, **When** xem trang chủ, **Then** hôm nay được tính là ngày học (chuỗi học tập
   chung) và thẻ thử thách trên trang chủ báo "Đã xong".

---

### Edge Cases

- **Chưa đủ nội dung**: người học mới chưa học từ nào → trò từ vựng lấy từ của bài 1 chủ đề đầu tiên trong trình độ của
  người học. Nếu vẫn không đủ (ví dụ cần 6 cặp nhưng chỉ có 4 từ có ảnh) thì trò đó hiện "Học thêm vài bài để mở trò
  này" thay vì vào lượt thiếu câu.
- **Đáp án nhiễu trùng nghĩa**: 4 đáp án của Ảnh nhanh tay không được có hai từ cùng nghĩa hoặc là dạng khác của nhau
  (apple / apples), và không dùng từ có ảnh dễ lẫn với ảnh đang hiện (ví dụ cùng ảnh).
- **Mất mạng giữa lượt**: lượt đang chơi không bị mất khi mạng chập chờn. Kết quả được gửi lại khi có mạng. Nếu không
  gửi được thì báo nhẹ và giữ kết quả trên máy để gửi lần sau.
- **Ra khỏi trang giữa lượt** (chuyển tab, khoá màn hình): đồng hồ tạm dừng và hiện màn "Tạm dừng". Thử thách hằng
  ngày cũng tạm dừng, không bị tính là bỏ.
- **Đổi ngày giữa lượt**: thử thách bắt đầu trước nửa đêm vẫn được tính cho ngày bắt đầu.
- **Nhiều thiết bị**: kỷ lục và kết quả thử thách lưu trên server nên thấy giống nhau trên mọi máy. Hai máy cùng chơi
  thử thách hôm nay thì chỉ lượt gửi lên trước được tính.
- **Tài khoản khách (guest)**: xem FR-033. Khách mới chưa học bài nào thì dùng từ của bài 1 các chủ đề như người học
  mới.
- **Bàn phím điện thoại** (Đấu động từ): ô gõ không bị bàn phím che, không tự sửa chính tả, không tự viết hoa.
- **Người dùng bật giảm chuyển động**: Chữ rơi vẫn chơi được (chữ di chuyển từng bước, không hiệu ứng nảy), các hiệu
  ứng lật/rung thay bằng đổi màu.
- **Từ sai nhiều lần trong một lượt**: chỉ đưa về hàng ôn một lần.

## Requirements *(mandatory)*

### Functional Requirements

**Phần chung (làm cùng P1)**

- **FR-001**: Khu người học MUST có trang **Trò chơi**: danh sách các trò đã phát hành, mỗi trò có tên, mô tả một dòng,
  thời lượng ước tính và kỷ lục cá nhân. Thử thách hằng ngày (khi có) đứng đầu danh sách, kèm trạng thái hôm nay. Lối
  vào là ô "Trò chơi" trong thẻ của nút giữa **Luyện** (cạnh Ngữ pháp, Động từ, Nói chuyện với AI). Khi đang ở trang
  Trò chơi, tab Luyện được tô là tab hiện tại. Màn đang chơi chiếm cả màn hình điện thoại (không có thanh tab) và có nút
  đóng riêng, như các trang học khác.
- **FR-002**: Mỗi trò MUST có màn kết quả chung: điểm, số câu đúng/sai, kỷ lục (báo khi lập kỷ lục mới), danh sách mục
  đã sai kèm đáp án/nghĩa, nút "Chơi lại" và "Về danh sách trò".
- **FR-003**: Hệ thống MUST lưu kỷ lục cá nhân theo từng trò trên server và chỉ cập nhật khi lượt chơi kết thúc bình
  thường (không tính lượt bỏ giữa chừng). Lượt chơi thường MUST NOT được lưu thành lịch sử: chỉ giữ kỷ lục mỗi trò và
  kết quả Thử thách hằng ngày. Trang Thống kê không có phần trò chơi.
- **FR-004**: Mục chơi sai MUST được đưa về hàng ôn: từ vựng đã có thẻ ôn thì đến hạn ngay hôm nay; động từ đã học thì
  đến hạn ôn động từ ngay hôm nay. Chơi đúng MUST NOT đổi lịch ôn. Mục chưa có thẻ ôn thì không tự tạo thẻ mới.
- **FR-005**: Nội dung trò chơi MUST lấy từ dữ liệu đã có (Kho từ vựng và ảnh, từ vựng các bài đã học, danh sách và ảnh
  động từ bất quy tắc, bài tập ngữ pháp đã duyệt). Trò chơi MUST NOT gọi AI khi người học chơi.
- **FR-006**: Mỗi lượt chơi MUST trong khoảng 1–3 phút. Mọi thao tác MUST làm được bằng một ngón cái trên điện thoại:
  vùng chạm chính nằm ở nửa dưới màn hình, nút tối thiểu 44×44 px.
- **FR-007**: Đồng hồ MUST tạm dừng khi trang bị ẩn và hiện màn "Tạm dừng" với nút "Tiếp tục" / "Thoát".
- **FR-008**: Giao diện MUST theo hệ thống thiết kế hiện có (màu Oải hương, font Lexend, sáng/tối) và có phản hồi rõ
  khi đúng/sai không chỉ dựa vào màu (có thêm ✓/✗).
- **FR-009**: Trò chơi MUST tôn trọng cài đặt giảm chuyển động của thiết bị.

**P1 — Ảnh nhanh tay**

- **FR-010**: Lượt 60 giây. Mỗi câu một ảnh và 4 đáp án tiếng Anh, đúng 1 đáp án khớp. Đáp án nhiễu cùng chủ đề hoặc
  cùng trình độ, không trùng nghĩa, không là dạng khác của đáp án đúng.
- **FR-011**: Điểm mỗi câu đúng = 100 × combo. Combo tăng 1 sau mỗi câu đúng (tối đa x5), về x1 khi sai.
- **FR-012**: Ảnh lấy theo thứ tự: ảnh Kho từ vựng của từ đã học, rồi ảnh của từ đến hạn ôn, rồi từ cùng chủ đề đang
  học. Không lặp lại một từ trong cùng lượt khi còn từ khác.

**P2 — Đấu động từ**

- **FR-013**: Mỗi câu hiện V1, ảnh động từ (nếu có), nghĩa tiếng Việt và hai ô gõ V2, V3. Chấp nhận mọi dạng đúng đã có
  trong danh sách động từ, không phân biệt hoa thường và khoảng trắng thừa.
- **FR-014**: 3 mạng, sai ô nào thì mất 1 mạng và hiện đáp án đúng của cả hai ô. Điểm lượt = số động từ đúng cả hai
  ô. Kỷ lục = điểm lượt cao nhất.
- **FR-015**: Thứ tự động từ: động từ đến hạn ôn và động từ hay sai trước, rồi động từ đã học, rồi động từ thông dụng
  chưa học.

**P3 — Chữ rơi**

- **FR-016**: Mỗi từ hiện nghĩa tiếng Việt, các chữ của từ rơi lẫn chữ nhiễu. Người học chạm theo đúng thứ tự để điền
  các ô. Chỉ dùng từ một chữ, 3–8 ký tự.
- **FR-017**: Mỗi màn 5 từ. Màn sau tăng tốc độ rơi và số chữ nhiễu. 3 mạng, mất mạng khi chạm sai hoặc chữ cần chạm rơi
  khỏi màn hình (chữ đó rơi lại từ trên).
- **FR-018**: Ghép xong một từ thì hiện ảnh (nếu có) và từ đúng trong chốc lát. Điểm theo từ đã ghép và màn đạt được.

**P4 — Lật thẻ ghi nhớ**

- **FR-019**: Lưới 3×4 = 12 thẻ, 6 cặp (ảnh hoặc nghĩa tiếng Việt ↔ từ tiếng Anh), xếp ngẫu nhiên.
- **FR-020**: Mở từ màn xong bài thì dùng từ của bài đó. Mở từ mục Trò chơi thì dùng từ đã học, ưu tiên từ đến hạn ôn.
- **FR-021**: Điểm tính từ số lượt lật và thời gian (ít lượt, nhanh thì điểm cao). Kỷ lục lưu theo điểm.
- **FR-022**: Màn xong bài MUST có lối vào "Chơi Lật thẻ với từ của bài này".

**P5 — Bắn lỗi ngữ pháp**

- **FR-023**: Câu lấy từ ba nguồn, đều là nội dung đã duyệt:
  (a) bài tập **chọn đáp án** có một chỗ trống: điền một phương án sai vào chỗ trống tạo câu có lỗi, chỗ sai là cả phương
  án đó (có thể là cụm, ví dụ "she aren't"), 3 phương án sửa là các phương án của bài tập. Điền đáp án đúng thì ra câu
  "Không có lỗi";
  (b) bài tập **sửa câu** mà câu sai và câu đúng khác nhau đúng một chữ hoặc một cụm liền nhau;
  (c) câu **Bắn lỗi soạn thêm** đã được admin duyệt (FR-026b).
  Câu "Không có lỗi" còn lấy từ câu đúng của (b), (c) và câu ví dụ của điểm ngữ pháp. Khoảng 30 % số câu không có lỗi.
- **FR-024**: Mỗi câu có lỗi có 3 phương án sửa gồm chữ đúng và 2 chữ nhiễu. Với nguồn (a) đó là các phương án của bài
  tập. Với nguồn (b) và (c), chữ nhiễu lấy từ phần soạn sẵn của câu, hoặc là chữ sai cùng một chữ nhiễu cùng gốc. Chữ
  nhiễu MUST NOT tạo ra câu đúng khác. Bài tập chọn đáp án có câu dạng hỏi–đáp (có " – ") vẫn dùng được, chạm được ở cả
  hai vế.
- **FR-025**: Sau mỗi câu hiện nhãn chủ điểm ngữ pháp và quy tắc ngắn (lấy từ phần giải thích của bài tập).
- **FR-026**: Lượt 10 câu, mỗi câu có giới hạn thời gian, 3 mạng. Hết giờ một câu tính là sai.
- **FR-026a**: Trang quản trị Ngữ pháp MUST cho biết số câu Bắn lỗi có được của mỗi trình độ và mỗi điểm ngữ pháp (tách
  theo nguồn a/b/c), và đánh dấu những điểm có dưới 10 câu.
- **FR-026b**: Admin MUST nhờ được AI soạn thêm câu Bắn lỗi cho một điểm ngữ pháp: một lần bấm gửi một request, nhận về
  tối đa 10 câu nháp. Mỗi câu gồm câu sai, chỗ sai, chữ đúng, 2 chữ nhiễu, quy tắc ngắn tiếng Việt. Câu nháp chỉ dùng
  trong trò chơi sau khi admin duyệt; admin sửa hoặc xoá được từng câu. Hệ thống tự loại câu nháp có chỗ sai không nằm
  trong câu, chữ đúng trùng chữ sai, hoặc chữ nhiễu trùng chữ đúng.

**P6 — Thử thách hằng ngày**

- **FR-027**: Mỗi ngày (theo múi giờ của người học) mỗi người học có một bộ 10 câu cố định trong ngày, trộn từ các trò đã phát
  hành, lấy nội dung từ những gì người học đã học, ưu tiên mục đến hạn ôn.
- **FR-027a**: Mỗi câu của thử thách là **bản rút gọn một câu** của một loại: Ảnh (1 ảnh, 4 đáp án), Động từ (gõ V2 và
  V3), **Ghép chữ** (chạm các chữ cái đã xáo trộn theo đúng thứ tự, có chữ nhiễu, chữ không rơi), Bắn lỗi (1 câu). Lật
  thẻ không có trong thử thách. Mỗi câu giới hạn **20 giây**, hết giờ tính sai. Không có đồng hồ chung, không có mạng.
  Câu đúng được cộng điểm thêm theo thời gian còn lại.
- **FR-028**: Chỉ lượt đầu tiên xong trong ngày được tính điểm và chuỗi. Chơi lại được nhưng chỉ để luyện.
- **FR-029**: Chuỗi thử thách tăng 1 khi xong thử thách ngày liên tiếp, về 1 nếu bỏ một ngày. Xong thử thách cũng được
  tính là một ngày học trong chuỗi học tập chung.
- **FR-030**: Màn kết quả có điểm, hàng 10 ô đúng/sai, chuỗi, câu sai đã đưa về hàng ôn, nút "Chia sẻ" sao chép kết quả
  dạng chữ (ngày, số câu đúng, chuỗi, hàng ô 🟩/🟥) không lộ đáp án.
- **FR-031**: Màn kết quả so điểm hôm nay với hôm qua, trung bình 7 ngày gần nhất (các ngày đã chơi) và kỷ lục thử thách.
  Không có bảng xếp hạng giữa người dùng; khi Luna có bạn bè thì làm sau.
- **FR-032**: Trang chủ MUST có thẻ Thử thách hằng ngày (chưa chơi / đã xong + chuỗi).

**Quyền truy cập**

- **FR-033**: Tài khoản khách (guest) chơi được mọi trò **trừ Bắn lỗi ngữ pháp**: trò này hiện trong danh sách nhưng
  khoá, kèm ghi chú "chỉ dành cho thành viên" (giống khoá phần Ngữ pháp). Thử thách hằng ngày của khách chỉ trộn câu từ
  các trò từ vựng và động từ. Nội dung trò chơi của khách chỉ lấy từ những bài khách được học. Thành viên và admin chơi
  được mọi trò.

### Key Entities

- **Kết quả lượt chơi (game result)**: người học, trò, điểm, số đúng/sai, đã xong hay bỏ dở, danh sách mục đã sai. Gửi
  lên khi lượt kết thúc, chỉ dùng để cập nhật kỷ lục và đưa mục sai về hàng ôn, **không lưu lại** sau khi xử lý xong.
- **Kỷ lục (personal best)**: người học, trò, điểm cao nhất, ngày lập. Một bản ghi cho mỗi cặp (người học, trò).
- **Câu hỏi trò chơi**: một mục cần trả lời (từ + ảnh + đáp án nhiễu; động từ + các dạng đúng; câu sai + chữ sai +
  phương án sửa + quy tắc). Dựng từ dữ liệu có sẵn khi tạo lượt, không lưu thành nội dung riêng.
- **Câu Bắn lỗi soạn thêm**: điểm ngữ pháp, trình độ, câu sai, chỗ sai, chữ đúng, 2 chữ nhiễu, quy tắc, trạng thái (nháp
  / đã duyệt), ngày soạn. Chỉ câu đã duyệt được đưa vào trò chơi.
- **Thử thách ngày**: người học, ngày, bộ 10 câu, kết quả lượt được tính (điểm, hàng đúng/sai), thời điểm xong.
- **Chuỗi thử thách**: số ngày thử thách liên tiếp hiện tại, dài nhất, ngày chơi gần nhất.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Từ bất kỳ trang nào có thanh tab, người học mở được danh sách trò trong 2 lần chạm (Luyện → Trò chơi) và
  vào một lượt chơi ở lần chạm thứ 3. Từ Trang chủ vào Thử thách hằng ngày chỉ cần 1 lần chạm.
- **SC-002**: Mọi trò có lượt chơi dài 1–3 phút với người chơi trung bình.
- **SC-003**: 100 % mục chơi sai (đã có thẻ ôn) xuất hiện trong lượt ôn của chính ngày hôm đó.
- **SC-004**: Mọi trò chơi trọn lượt được bằng một tay trên màn hình điện thoại rộng 360 px, không phải cuộn trang.
- **SC-005**: Phản hồi đúng/sai hiện ngay khi chạm (người học không cảm thấy độ trễ), kể cả khi mạng chậm.
- **SC-006**: Không có câu hỏi nào có hai đáp án đúng hoặc đáp án đúng bị lặp trong cùng một câu (kiểm tra trên toàn bộ
  dữ liệu hiện có).
- **SC-007**: Sau 2 tuần dùng thử, người học quay lại chơi Thử thách hằng ngày ít nhất 5 ngày mỗi tuần (đo bằng chuỗi
  thử thách của chính người dùng).

## Assumptions

- Phát hành theo thứ tự P1 → P6; mỗi trò chỉ hiện trong mục Trò chơi khi đã làm xong. Thử thách hằng ngày chỉ trộn các
  trò đã phát hành.
- "Đã học" nghĩa là: từ vựng có thẻ ôn (từ bài đã học, Đã biết, hay thêm tay); động từ đã có tiến độ ôn; điểm ngữ pháp
  đã mở học. Dữ liệu này đã có sẵn.
- Đưa về hàng ôn chỉ đổi hạn ôn sang hôm nay, không đổi mức độ thuộc của thẻ; chơi đúng không ảnh hưởng lịch ôn (giống
  chế độ luyện tập của động từ hiện nay).
- Từ chưa có thẻ ôn (ví dụ từ nhiễu, từ bổ sung khi người học mới) không tự tạo thẻ khi chơi sai.
- Chuỗi thử thách là chuỗi riêng của trò chơi; chuỗi học tập chung vẫn giữ cách tính hiện tại và có thêm "xong thử thách
  hằng ngày".
- Ngày tính theo múi giờ lưu trong tài khoản người học (mặc định Asia/Ho_Chi_Minh), giống cách Luna đang tính ngày học.
- Nếu câu Bắn lỗi của các điểm đã học không đủ 10, trò lấy thêm câu của các điểm khác cùng trình độ, rồi của trình độ
  thấp hơn. Soạn thêm câu bằng AI chỉ diễn ra khi admin bấm, không chạy tự động, và được ghi vào trang Sử dụng AI như các chức năng AI khác.
- Âm thanh hiệu ứng và rung để ngoài phạm vi; có thể thêm sau.
- Không có tiền ảo, cửa hàng, huy hiệu hay phần thưởng ngoài điểm, kỷ lục và chuỗi.
