# Feature Specification: Bước Viết

**Feature Branch**: `014-writing-step`

**Created**: 2026-10-01

**Status**: Draft

**Mã tính năng**: F8 | **Giai đoạn**: Giai đoạn 2 (`docs/phases/giai-doan-2.md`)

**Input**: User description: "tạo spec theo khối 1 trong @docs/spec-inputs/f8-viet.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Viết và nộp bài ở bước Viết (Priority: P1)

Bài hôm nay có thêm bước **Viết** bắt buộc sau bước Nghe: Ôn → Đọc → Nghe → Viết. Người học thấy đề viết của bài (F15) và độ dài
gợi ý theo trình độ, gõ bài; bản nháp tự lưu trong lúc gõ. Bấm **Nộp**: bước Viết hoàn thành ngay, bài hôm nay xong (mục tiêu +1,
streak +1), không phải chờ AI chấm.

**Why this priority**: Là bước mới của luồng học hằng ngày; không có nó thì bài hôm nay không xong được sau khi F8 có mặt.

**Independent Test**: Học bài hôm nay tới hết bước Nghe; bước Viết mở ra với đề viết; gõ 30 từ, tải lại trang thấy nguyên bản
nháp; bấm Nộp: bài hôm nay chuyển "Đã xong", mục tiêu và streak tăng ngay cả khi AI đang tắt.

**Acceptance Scenarios**:

1. **Given** người học đã xong bước Nghe, **When** mở bài hôm nay, **Then** bước hiện tại là Viết; các bước theo thứ tự Ôn → Đọc →
   Nghe → Viết; bài chưa được tính là xong.
2. **Given** bước Viết, **When** mở, **Then** thấy đề viết của bài, độ dài gợi ý theo trình độ, ô soạn có đếm số từ.
3. **Given** bài chưa có đề viết, **When** mở bước Viết, **Then** đề là "Tóm tắt bài bằng 3–5 câu".
4. **Given** đang gõ, **When** dừng gõ một lúc, **Then** bản nháp được lưu và có báo "Đã lưu nháp"; thoát ra, tải lại hoặc mở trên
   thiết bị khác thấy đúng bản nháp.
5. **Given** bài viết dưới 5 từ hoặc trên 400 từ, **When** bấm Nộp, **Then** báo lỗi rõ ràng, không nộp.
6. **Given** bài viết hợp lệ, **When** bấm Nộp, **Then** bước Viết hoàn thành ngay, bài hôm nay xong, mục tiêu +1 bài và streak tính
   như giai đoạn 1; trang báo "Đã nộp, AI đang chấm".
7. **Given** đã nộp, **When** mở lại bước Viết hoặc gửi nộp lần nữa, **Then** không sửa hay nộp lại được.
8. **Given** chưa xong bước Nghe, **When** cố mở hoặc nộp bước Viết của bài hôm nay, **Then** bị chặn như các bước khác của L.

---

### User Story 2 - Nhận kết quả chấm và xem nhận xét (Priority: P1)

AI chấm bài ở nền. Khi có kết quả, app hiện thông báo; người học mở ra thấy điểm 1–5 và nhận xét tiếng Việt cho 4 tiêu chí (hoàn
thành yêu cầu, ngữ pháp, từ vựng, mạch lạc), nhận xét chung, bản đã sửa và phần so sánh với bản gốc (chỗ thêm, chỗ bớt).

**Why this priority**: Nhận xét cụ thể để sửa là giá trị chính của luyện viết; thiếu nó bước Viết chỉ là chép bài.

**Independent Test**: Nộp một bài có lỗi ngữ pháp; trong lúc chờ đi làm việc khác; khi chấm xong, đầu trang hiện dấu báo và thông
báo ngắn; mở bài thấy 4 điểm, nhận xét, bản sửa và phần so sánh đánh dấu chỗ thêm/bớt bằng ký hiệu và chữ, không chỉ bằng màu.

**Acceptance Scenarios**:

1. **Given** vừa nộp, **When** AI còn đang chấm, **Then** bài viết hiện trạng thái "Đang chấm"; người học dùng mọi phần khác của app
   bình thường.
2. **Given** AI chấm xong khi người học đang ở bất kỳ trang nào, **When** kết quả có, **Then** trong vòng 1 phút đầu trang hiện dấu báo
   kèm số kết quả mới chưa xem và một thông báo ngắn có nút mở bài viết.
3. **Given** mở bài viết đã chấm, **When** xem, **Then** thấy điểm 1–5 và nhận xét tiếng Việt cho từng tiêu chí trong 4 tiêu chí, điểm
   trung bình, nhận xét chung, bản gốc, bản đã sửa và phần so sánh.
4. **Given** phần so sánh, **When** xem (kể cả bằng trình đọc màn hình hoặc không phân biệt màu), **Then** chỗ thêm và chỗ bớt phân
   biệt được bằng kiểu chữ (gạch chân / gạch ngang) và nhãn đọc được, không chỉ bằng màu.
5. **Given** đã mở xem kết quả, **When** quay lại, **Then** kết quả đó không còn tính là "chưa xem" và dấu báo giảm tương ứng.

---

### User Story 3 - Chấm lỗi thì chấm lại (Priority: P2)

Khi AI lỗi, hết lượt hoặc trả về kết quả không dùng được, bài viết vẫn được lưu, hiện trạng thái lỗi kèm lý do, và có nút **Chấm
lại**.

**Why this priority**: AI gói miễn phí hay hết lượt; người học không được mất bài đã viết. Là tiêu chí nghiệm thu nhưng phụ thuộc
US1–US2.

**Independent Test**: Tắt AI rồi nộp: bước Viết vẫn hoàn thành; bài viết hiện "Chấm lỗi: AI chưa được cấu hình"; bật AI, bấm Chấm
lại: trạng thái về "Đang chấm" rồi có kết quả.

**Acceptance Scenarios**:

1. **Given** AI lỗi tạm thời, **When** chấm, **Then** hệ thống tự thử lại vài lần trước khi báo lỗi.
2. **Given** chấm lỗi hẳn (hết lượt, chưa cấu hình, lỗi lặp lại), **When** xem bài viết, **Then** thấy "Chấm lỗi" kèm lý do bằng tiếng
   Việt, bài viết vẫn nguyên, có nút Chấm lại.
3. **Given** bài viết đang lỗi, **When** bấm Chấm lại, **Then** trạng thái về "Đang chấm" và việc chấm chạy lại ở nền.
4. **Given** bài viết đang chấm hoặc đã chấm xong, **When** cố chấm lại, **Then** không cho (chỉ bài lỗi mới chấm lại được).
5. **Given** chấm lỗi, **When** xem tiến độ, **Then** bước Viết và bài hôm nay vẫn tính là xong.

---

### User Story 4 - Trang Bài viết (Priority: P2)

Trang **Bài viết** liệt kê các bài viết đã nộp: ngày, bài học, điểm trung bình hoặc trạng thái đang chấm / lỗi; mở từng bài xem
nhận xét. Mở lại bài đã học ở trang Bài học thấy bài viết và nhận xét cũ, không nộp lại.

**Why this priority**: Giúp xem lại tiến bộ; dùng ít hơn luồng hằng ngày.

**Independent Test**: Sau 3 ngày học, mở Bài viết: thấy 3 dòng mới nhất trước, mỗi dòng có ngày, tên bài và điểm hoặc trạng thái;
mở dòng thứ hai thấy đủ nhận xét; mở bài đã học từ trang Bài học thấy bài viết cũ và không có nút Nộp.

**Acceptance Scenarios**:

1. **Given** đã nộp nhiều bài viết, **When** mở trang Bài viết, **Then** thấy danh sách mới nhất trước với ngày nộp, tên bài học, điểm
   trung bình (hoặc "Đang chấm" / "Chấm lỗi").
2. **Given** chưa nộp bài viết nào, **When** mở trang, **Then** thấy lời nhắc học bài hôm nay để viết bài đầu tiên.
3. **Given** mở lại bài đã học từ trang Bài học, **When** sang phần Viết, **Then** thấy bài đã nộp và nhận xét (hoặc trạng thái chấm),
   không sửa hay nộp lại được.
4. **Given** hai người học, **When** mỗi người mở trang Bài viết hoặc một bài viết cụ thể, **Then** chỉ thấy bài của mình; mở bài của
   người khác như bài không tồn tại.

---

### User Story 5 - Màn hình chính, thống kê và xuất dữ liệu (Priority: P3)

Thanh kỹ năng **Viết** trên màn hình chính đếm số bài đã nộp bài viết trong mục tiêu; trang thống kê có thêm số bài viết và điểm
trung bình; file xuất dữ liệu (F13) có thêm bài viết và nhận xét.

**Why this priority**: Bổ sung số liệu, không chặn việc học.

**Independent Test**: Nộp 2 bài viết (điểm trung bình 3,5 và 4,0) trong mục tiêu A1 · Gia đình: màn hình chính có thanh Viết 2/N;
thống kê hiện "2 bài viết · điểm trung bình 3,8"; file xuất có 2 bài viết kèm nhận xét.

**Acceptance Scenarios**:

1. **Given** mục tiêu đang học, **When** mở màn hình chính, **Then** có thanh Viết cạnh Đọc / Nghe, đếm số bài trong mục tiêu đã nộp bài
   viết.
2. **Given** đã có bài viết chấm xong, **When** mở thống kê, **Then** thấy số bài viết đã nộp và điểm trung bình của các bài đã chấm;
   chưa có bài chấm xong thì điểm hiện "—".
3. **Given** xuất dữ liệu, **When** mở file, **Then** có mọi bài viết (đề, bài viết, trạng thái, điểm, nhận xét, bản sửa) của người học,
   không có của người khác.

---

### Edge Cases

- Bài đang học dở lúc F8 có mặt (đã xong Nghe nhưng chưa xong bài, hoặc đang ở Nghe): sau Nghe sang Viết như bài mới. Bài đã xong từ
  trước vẫn là xong, không phải viết bù.
- Nháp rỗng hoặc chưa gõ gì: không tạo bài viết; xoá hết chữ thì nháp được lưu rỗng.
- Hai tab cùng soạn: bản lưu sau cùng thắng; sau khi nộp ở một tab, tab kia báo đã nộp và không lưu nháp được nữa.
- Mất mạng khi lưu nháp: báo "Chưa lưu được nháp", tự thử lại ở lần gõ tiếp; bản trên màn hình không mất.
- Mất mạng khi nộp: báo lỗi, bài chưa nộp, bấm Nộp lại được.
- Nội dung bài học bị sửa sau khi đã nộp: bài viết đã nộp giữ nguyên đề và nhận xét lúc nộp.
- Đề viết đổi sau khi đã có nháp: nháp giữ nguyên; lúc nộp dùng đề đang hiện cho người học, đề đó được lưu cùng bài viết.
- Đếm từ theo khoảng trắng như các nơi khác của app; 5 và 400 từ là hợp lệ.
- AI trả về điểm ngoài 1–5, thiếu tiêu chí hoặc bản sửa rỗng: coi là chấm lỗi, có thể chấm lại.
- Kết quả về khi người học đang ở chính trang bài viết đó: trang tự hiện kết quả, không cần tải lại.
- Ở 360px: ô soạn, đếm từ, nút Nộp, nhận xét và phần so sánh không cuộn ngang; thao tác được chỉ bằng bàn phím.

## Requirements *(mandatory)*

### Functional Requirements

**Bước Viết trong luồng học**

- **FR-001**: Bài hôm nay MUST có 4 bước theo thứ tự Ôn → Đọc → Nghe → Viết; bước Viết bắt buộc và chỉ mở khi bước Nghe xong; bài hôm
  nay chỉ xong khi nộp bài viết.
- **FR-002**: Bước Viết MUST hiện đề viết của bài (F15) hoặc đề mặc định "Tóm tắt bài bằng 3–5 câu" khi bài chưa có đề, cùng độ dài
  gợi ý theo trình độ và số từ đang gõ.
- **FR-003**: Bản nháp MUST tự lưu theo người học và bài trong lúc gõ (sau khi dừng gõ khoảng 1 giây), có báo "Đã lưu nháp" / "Chưa lưu
  được nháp"; vào lại (kể cả thiết bị khác) MUST thấy bản nháp mới nhất.
- **FR-004**: Nộp MUST chỉ chấp nhận bài từ 5 đến 400 từ; ngoài khoảng này báo lỗi tiếng Việt và không nộp.
- **FR-005**: Nộp MUST hoàn thành bước Viết và bài hôm nay ngay lập tức (mục tiêu +1, cập nhật ngày học và streak như L), không phụ thuộc
  việc chấm thành công.
- **FR-006**: Sau khi nộp, bài viết MUST NOT sửa hay nộp lại được; mỗi người học có tối đa một bài viết đã nộp cho mỗi bài học.
- **FR-007**: Bài viết đã nộp MUST lưu đề viết, bài viết và thời điểm nộp tại lúc nộp, không đổi khi bài học bị sửa sau đó.

**Chấm bài**

- **FR-008**: Việc chấm MUST chạy nền sau khi nộp, có trạng thái đang chấm / xong / lỗi, tự thử lại khi lỗi tạm thời, và tốn 1 lần gọi AI
  mỗi lần chấm.
- **FR-009**: Kết quả chấm MUST gồm điểm nguyên 1–5 và nhận xét tiếng Việt cho đúng 4 tiêu chí: hoàn thành yêu cầu, ngữ pháp, từ vựng,
  mạch lạc; nhận xét chung tiếng Việt; bản đã sửa. Kết quả thiếu hoặc sai dạng MUST được coi là chấm lỗi.
- **FR-010**: Chấm lỗi MUST giữ nguyên bài viết, hiện lý do bằng tiếng Việt (ví dụ hết lượt AI) và cho phép **Chấm lại**; Chấm lại chỉ dùng
  được với bài đang lỗi.
- **FR-011**: AI lỗi hoặc chưa cấu hình MUST NOT chặn việc nộp, hoàn thành bài hay bất kỳ chức năng nào khác.

**Xem kết quả và thông báo**

- **FR-012**: Khi có kết quả chấm mới (xong hoặc lỗi), app MUST hiện thông báo trong app trong vòng 1 phút ở bất kỳ trang nào: dấu báo ở
  đầu trang kèm số kết quả chưa xem và một thông báo ngắn có liên kết tới bài viết; mở bài viết MUST đánh dấu đã xem.
- **FR-013**: Trang chi tiết bài viết MUST hiện đề, bài gốc, trạng thái, và khi đã chấm: điểm từng tiêu chí, điểm trung bình, nhận xét,
  bản đã sửa, phần so sánh bài gốc với bản sửa theo từ.
- **FR-014**: Phần so sánh MUST đánh dấu chỗ thêm và chỗ bớt bằng kiểu chữ (gạch chân cho thêm, gạch ngang cho bớt) và nhãn đọc được bởi
  trình đọc màn hình, không chỉ bằng màu.
- **FR-015**: Trang **Bài viết** MUST liệt kê bài viết đã nộp của người học, mới nhất trước, với ngày nộp, tên bài học, điểm trung bình
  hoặc trạng thái; header MUST có liên kết "Bài viết".
- **FR-016**: Mở lại bài đã học (trang Bài học) MUST hiện bài viết đã nộp và nhận xét, không cho soạn hay nộp lại.

**Dữ liệu và số liệu**

- **FR-017**: Bài viết, nháp và nhận xét MUST chỉ người viết xem được; bài của người khác coi như không tồn tại.
- **FR-018**: Màn hình chính MUST có thanh kỹ năng Viết đếm số bài trong mục tiêu đã nộp bài viết; trang thống kê MUST có số bài viết đã
  nộp và điểm trung bình (trung bình của điểm trung bình các bài đã chấm xong; chưa có thì "—").
- **FR-019**: File xuất dữ liệu (F13) MUST có mọi bài viết của người học (đề, bài viết, trạng thái, điểm, nhận xét, bản sửa, thời điểm),
  kể cả nháp chưa nộp.
- **FR-020**: Mọi màn hình của F8 MUST dùng được ở 360px không cuộn ngang và chỉ bằng bàn phím; trạng thái (đã lưu nháp, đang chấm, có
  kết quả) MUST được thông báo cho trình đọc màn hình.

### Key Entities

- **Bài viết**: người học, bài học (kèm phiên bản bài lúc viết), đề viết, nội dung, trạng thái (nháp / đã nộp), thời điểm tạo, nộp;
  mỗi người học tối đa một bài viết cho mỗi bài học.
- **Kết quả chấm** (thuộc bài viết đã nộp): trạng thái (đang chấm / xong / lỗi kèm lý do), 4 tiêu chí (tên, điểm 1–5, nhận xét), nhận
  xét chung, bản đã sửa, thời điểm chấm, đã xem hay chưa.
- **Tiến độ bài hôm nay** (L, mở rộng): thêm bước Viết vào thứ tự bước.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% lần nộp hợp lệ hoàn thành bước Viết và bài hôm nay ngay (dưới 1 giây với mạng bình thường), kể cả khi AI tắt hoặc lỗi.
- **SC-002**: Thoát ra rồi vào lại sau khi gõ (đợi quá 1 giây), 100% nội dung nháp còn nguyên.
- **SC-003**: Với AI hoạt động, ít nhất 9/10 bài viết có kết quả trong vòng 2 phút sau khi nộp.
- **SC-004**: Kết quả mới được báo trong app trong vòng 1 phút sau khi chấm xong, ở bất kỳ trang nào của app.
- **SC-005**: 100% bài chấm lỗi vẫn giữ nguyên bài viết và chấm lại được; sau khi chấm lại thành công có đủ 4 tiêu chí.
- **SC-006**: Kiểm tra ở chế độ xám và bằng trình đọc màn hình: 100% chỗ thêm / bớt trong phần so sánh vẫn phân biệt được.
- **SC-007**: Số liệu thanh Viết và thống kê khớp với số bài viết đã nộp và điểm tính tay từ dữ liệu của người học.
- **SC-008**: Mọi thao tác của F8 làm được ở 360px và chỉ bằng bàn phím.

## Assumptions

- Dùng AI và hàng đợi việc nền sẵn có (F2: Gemini gói miễn phí, thử lại tối đa 3 lần); không thêm cấu hình mới.
- Độ dài gợi ý theo trình độ là gợi ý, không chặn nộp: A1 30–60 từ, A2 50–80, B1 80–120, B2 120–180, C1 và C2 150–250; giới hạn cứng là
  5–400 từ.
- "Điểm trung bình" của một bài là trung bình 4 tiêu chí (làm tròn 1 chữ số thập phân); điểm trung bình trên thống kê là trung bình của
  các bài đã chấm xong.
- Thanh Viết trên màn hình chính dùng cùng tổng số bài của mục tiêu như các thanh Đọc / Nghe.
- Thông báo kết quả là thông báo trong app (dấu báo + thông báo ngắn), không gửi email hay thông báo đẩy.
- Bài đã hoàn thành trước khi F8 có mặt vẫn là hoàn thành, không bị yêu cầu viết bù; bài đang dở chuyển sang có bước Viết.
- Chỉ có bài viết gắn với bài học trong luồng hằng ngày (và bài đã học được mở lại nhưng chưa có bài viết thì chỉ xem, không viết bù).
- Ngoài phạm vi: viết tự do ngoài bài học, chấm theo thang IELTS, nhiều đề mỗi bài.
- Phụ thuộc: F15 (đề viết), L (thứ tự bước, hoàn thành bài, streak), F6 (màn hình chính, thống kê), F13 (xuất dữ liệu), F2 (AI và việc
  nền).
