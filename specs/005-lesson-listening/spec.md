# Feature Specification: Nghe

**Feature Branch**: `005-lesson-listening`

**Created**: 2026-09-29

**Status**: Draft

**Mã tính năng**: F4 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f4-nghe.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Nghe từng câu (Priority: P1)

Người học mở bước Nghe của một bài. Bài được phát theo từng câu: có nút phát/dừng, câu trước, câu sau, lặp lại câu. Người
học chọn tốc độ 0.5x, 0.75x, 1x hoặc 1.25x, và ẩn hoặc hiện chữ của câu (transcript).

**Why this priority**: Nghe từng câu là nền của chép chính tả; không phát được thì bước Nghe không có gì.

**Independent Test**: Mở bài đã có audio, phát câu 1, chuyển câu sau/trước, lặp lại, đổi tốc độ; bật/tắt transcript.

**Acceptance Scenarios**:

1. **Given** bài có audio cho mọi câu, **When** mở bước Nghe, **Then** thấy "Câu 1/N", nút phát, câu trước (vô hiệu ở câu
   đầu), câu sau, lặp lại, chọn tốc độ (mặc định 1x) và transcript đang ẩn.
2. **Given** đang ở câu 3, **When** bấm câu sau, **Then** chuyển sang câu 4 và phát câu 4.
3. **Given** một câu đang hoặc đã phát, **When** bấm lặp lại, **Then** câu đó phát lại từ đầu; lặp bao nhiêu lần cũng được.
4. **Given** chọn tốc độ 0.75x, **When** phát câu, **Then** câu phát chậm hơn và tốc độ được giữ khi chuyển câu.
5. **Given** transcript đang ẩn, **When** bấm "Hiện chữ", **Then** thấy chữ của câu đang nghe; bấm lại thì ẩn.
6. **Given** audio của bài chưa sẵn sàng, **When** mở bước Nghe, **Then** thấy "Audio của bài chưa sẵn sàng" và không có ô chép
   chính tả.

---

### User Story 2 - Chép chính tả và xem lỗi (Priority: P1)

Với mỗi câu, người học nghe (bao nhiêu lần cũng được), gõ lại câu nghe được và bấm Kiểm tra. Kết quả hiện từng từ: đúng;
sai (gạch ngang, kèm từ đúng bên cạnh); thiếu (gạch chân chấm). Từ gõ thừa tính là sai.

**Why this priority**: Là mục tiêu của bước Nghe: biết mình nghe sai ở đâu.

**Independent Test**: Với câu "I don't like green apples.", gõ lần lượt câu đúng, câu thiếu một từ, câu sai một từ, câu thừa
một từ, câu viết hoa khác và không dấu câu; kiểm tra kết quả hiển thị.

**Acceptance Scenarios**:

1. **Given** câu "I don't like green apples.", **When** gõ "i dont like green apples" và kiểm tra, **Then** *dont* là sai
   (hiện *don't* bên cạnh), các từ còn lại đúng.
2. **Given** câu trên, **When** gõ "I don't like apples", **Then** *green* được báo thiếu (gạch chân chấm) ở đúng vị trí,
   các từ còn lại đúng.
3. **Given** câu trên, **When** gõ "I don't like the green apples", **Then** *the* là từ thừa, báo sai (gạch ngang) và không
   có từ đúng đi kèm.
4. **Given** câu trên, **When** gõ "i DON'T like green apples!!", **Then** mọi từ đúng (không phân biệt hoa thường, bỏ qua dấu
   câu, dấu nháy trong *don't* được giữ).
5. **Given** đã kiểm tra, **When** nhìn kết quả, **Then** câu đúng được hiện đầy đủ; sai và thiếu phân biệt được bằng kiểu chữ
   (gạch ngang, gạch chân chấm) và nhãn cho trình đọc màn hình, không chỉ bằng màu.
6. **Given** ô nhập trống, **When** bấm Kiểm tra, **Then** thấy nhắc "Hãy gõ câu bạn nghe được" và câu chưa được tính.
7. **Given** đã kiểm tra một câu, **When** sửa và kiểm tra lại, **Then** kết quả mới thay kết quả cũ.
8. **Given** điện thoại 360px đang mở bàn phím ảo, **When** gõ câu, **Then** nút Kiểm tra vẫn nhìn thấy và bấm được.

---

### User Story 3 - Hoàn thành bước Nghe và lưu tỷ lệ đúng (Priority: P2)

Khi mọi câu đã được kiểm tra ít nhất một lần, bước Nghe hoàn thành (câu sai vẫn tính là xong). Kết quả từng câu và tỷ lệ
đúng của bài được lưu để thống kê. Quay lại bài thì thấy các câu đã kiểm tra.

**Why this priority**: Cần cho luồng một ngày học (L) và thống kê (F6), nhưng luyện nghe vẫn dùng được nếu chưa lưu.

**Independent Test**: Kiểm tra hết các câu của một bài ngắn (có câu sai) → thấy hoàn thành và tỷ lệ đúng; tải lại trang →
các câu vẫn hiện đã kiểm tra, tỷ lệ không đổi.

**Acceptance Scenarios**:

1. **Given** bài 4 câu, đã kiểm tra 3 câu, **When** kiểm tra câu thứ 4 (dù sai), **Then** thấy "Đã hoàn thành bước Nghe" kèm tỷ
   lệ đúng của bài.
2. **Given** chưa kiểm tra hết, **When** nhìn tiến độ, **Then** thấy "Đã kiểm tra X/N câu" và từng câu đã/chưa kiểm tra.
3. **Given** đã kiểm tra vài câu, **When** tải lại trang, **Then** các câu đó vẫn hiện đã kiểm tra kèm kết quả lần gần nhất và
   bắt đầu ở câu đầu tiên chưa kiểm tra.
4. **Given** tài khoản A đã có kết quả, **When** tài khoản B mở cùng bài, **Then** B không thấy kết quả của A.

---

### Edge Cases

- Dấu nháy cong (’) và thẳng (') coi như nhau; *It’s* = *it's*.
- Gạch nối tách từ: *well-known* và *well known* coi như nhau.
- Số viết bằng chữ số (*9.30*, *2026*) được so như viết; dấu chấm giữa hai chữ số không bị bỏ.
- Người học gõ nhiều khoảng trắng, xuống dòng hoặc dấu câu thừa: không ảnh hưởng kết quả.
- Câu chỉ có một từ ("Why?"): so sánh vẫn đúng.
- Người học gõ lệch nhiều (gần như toàn sai): kết quả vẫn căn đúng các từ trùng, còn lại là sai/thiếu, không lỗi.
- Một câu thiếu audio (các câu khác có): câu đó báo "Chưa có audio" nhưng vẫn cho gõ theo transcript và kiểm tra.
- Mất kết nối khi lưu kết quả: kết quả vẫn hiện; thông báo "Chưa lưu được, sẽ thử lại" và gửi lại khi kiểm tra câu tiếp theo.
- Bài bị sửa nội dung sau khi đã chép (số câu thay đổi): kết quả cũ không còn khớp thì bị bỏ qua, bắt đầu lại.
- Chuyển câu khi audio đang phát: dừng câu cũ, phát câu mới, không chồng tiếng.

## Requirements *(mandatory)*

### Functional Requirements

**Nghe**

- **FR-001**: Người đã đăng nhập PHẢI mở được bước Nghe của một bài có audio và nghe từng câu theo thứ tự.
- **FR-002**: PHẢI có: phát/dừng câu hiện tại, câu trước, câu sau, lặp lại câu; hiển thị "Câu i/N".
- **FR-003**: PHẢI chọn được tốc độ 0.5x, 0.75x, 1x, 1.25x (mặc định 1x); tốc độ giữ nguyên khi chuyển câu.
- **FR-004**: Mỗi câu PHẢI nghe lại được không giới hạn số lần, trước và sau khi kiểm tra.
- **FR-005**: PHẢI ẩn/hiện được transcript của câu hiện tại; mặc định ẩn.
- **FR-006**: Chỉ một câu phát tại một thời điểm; chuyển câu thì dừng câu đang phát.

**Chép chính tả**

- **FR-007**: Mỗi câu PHẢI có ô gõ và nút Kiểm tra; ô trống thì không kiểm tra và hiện nhắc.
- **FR-008**: So sánh PHẢI không phân biệt hoa thường, bỏ qua dấu câu, giữ dấu nháy bên trong từ (*don't*, *it's*), coi dấu
  nháy cong và thẳng như nhau, coi gạch nối là khoảng trắng, giữ dấu chấm giữa hai chữ số.
- **FR-009**: Kết quả PHẢI căn từ gõ với từ đúng sao cho số lỗi ít nhất, và gán cho mỗi vị trí một trạng thái: *đúng*; *sai*
  (kèm từ đúng; từ thừa là sai không kèm từ đúng); *thiếu* (từ đúng mà người học không gõ).
- **FR-010**: Hiển thị PHẢI phân biệt được không cần màu: sai = gạch ngang + từ đúng bên cạnh; thiếu = gạch chân chấm; mỗi trạng
  thái có nhãn đọc được bởi trình đọc màn hình.
- **FR-011**: Sau khi kiểm tra PHẢI hiện câu đúng đầy đủ và số từ đúng trên tổng số từ của câu.
- **FR-012**: Người học PHẢI sửa và kiểm tra lại được; lần kiểm tra mới nhất là kết quả của câu.
- **FR-013**: Trên điện thoại, ô gõ và nút Kiểm tra PHẢI nhìn thấy được khi bàn phím ảo mở; Enter trong ô gõ tương đương bấm
  Kiểm tra.

**Hoàn thành và lưu**

- **FR-014**: Bước Nghe PHẢI hoàn thành khi mọi câu đã được kiểm tra ít nhất một lần, bất kể đúng sai; báo hoàn thành cho luồng
  học (nối vào L sau) và hiện xác nhận.
- **FR-015**: Kết quả lần kiểm tra mới nhất của mỗi câu (số từ đúng, tổng số từ) PHẢI được lưu theo người học và bài.
- **FR-016**: Tỷ lệ đúng của bài PHẢI bằng tổng số từ đúng chia tổng số từ của các câu đã kiểm tra, và được lưu để thống kê
  (F6).
- **FR-017**: Mở lại bài PHẢI thấy các câu đã kiểm tra, kết quả gần nhất, tiến độ "X/N" và bắt đầu ở câu đầu tiên chưa kiểm tra.
- **FR-018**: Kết quả chép chính tả PHẢI tách riêng theo người học: không ai đọc hay ghi kết quả của người khác (nguyên tắc V).
- **FR-019**: Cho tới khi có L, bước Nghe PHẢI mở được từ trang chi tiết bài của quản trị viên và bằng đường dẫn trực tiếp.

### Key Entities

- **Kết quả chép chính tả (Dictation result)**: thuộc một người học, một bài, một câu; số từ đúng, tổng số từ, thời điểm kiểm
  tra. Mỗi câu chỉ giữ lần mới nhất.
- **Tổng kết chép chính tả của bài (Dictation summary)**: số câu đã kiểm tra trên tổng số câu, tổng từ đúng, tổng từ, tỷ lệ đúng,
  đã hoàn thành hay chưa.
- **So sánh từng từ (Word comparison)**: danh sách vị trí với trạng thái đúng/sai/thiếu, từ người học gõ và từ đúng. Chỉ tính
  trên máy người học, không lưu.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Với bộ mẫu kiểm thử (khớp hoàn toàn; thiếu đầu, giữa, cuối; thừa; sai chính tả; dấu nháy; hoa thường; dấu câu;
  gạch nối; số), 100% kết quả so sánh đúng như mong đợi.
- **SC-002**: Kết quả so sánh hiện ngay khi bấm Kiểm tra (dưới 100ms với câu 40 từ), không cần chờ máy chủ.
- **SC-003**: Người học nghe lại một câu bao nhiêu lần cũng được; mỗi lần bấm lặp lại câu bắt đầu phát trong dưới 1 giây.
- **SC-004**: 100% bài có mọi câu đã kiểm tra được đánh dấu hoàn thành, kể cả khi có câu sai.
- **SC-005**: Tải lại trang giữ nguyên 100% kết quả đã lưu và tỷ lệ đúng.
- **SC-006**: Người dùng khác không đọc được kết quả của người học ở 100% lần thử.
- **SC-007**: Ở 360px có bàn phím ảo, nút Kiểm tra luôn nhìn thấy; không cuộn ngang; sai và thiếu phân biệt được khi xem ở chế độ
  thang xám.

## Assumptions

- Chưa gắn với luồng một ngày học: F4 báo "đã hoàn thành" và hiện xác nhận; mở bước tiếp theo và thanh tiến trình thuộc L.
- Mọi người đã đăng nhập đều mở được bước Nghe của mọi bài có audio (L sẽ chọn bài của ngày).
- Tổng số từ của một câu = số từ đúng cần gõ cộng số từ gõ thừa, để từ thừa làm giảm tỷ lệ đúng.
- Transcript mặc định ẩn để không lộ đáp án; sau khi kiểm tra, câu đúng luôn hiện.
- Kiểm tra lại một câu thay kết quả cũ; không lưu lịch sử từng lần.
- Tốc độ phát chỉ nhớ trong lúc đang ở trang (cài đặt lâu dài thuộc F12).
- Kết quả gắn với bài theo vị trí câu; nếu bài bị sửa nội dung làm số câu thay đổi thì kết quả cũ của bài đó không dùng nữa.
- Ngoài phạm vi: luồng một ngày học (L), luyện nói/shadowing (giai đoạn 3), phát liên tục cả bài không dừng.
