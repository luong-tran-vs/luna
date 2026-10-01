# Feature Specification: Câu hỏi hiểu bài và ghi chú ngữ pháp

**Feature Branch**: `013-reading-comprehension`

**Created**: 2026-10-01

**Status**: Draft

**Mã tính năng**: F15 | **Giai đoạn**: Giai đoạn 2 (`docs/phases/giai-doan-2.md`)

**Input**: User description: "tạo spec theo khối 1 trong @docs/spec-inputs/f15-hieu-bai-ngu-phap.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trả lời câu hỏi hiểu bài ở bước Đọc (Priority: P1)

Ở bước Đọc của một bài có câu hỏi, người học đọc bài rồi trả lời lần lượt 3–5 câu hỏi trắc nghiệm (4 lựa chọn). Chọn xong mỗi
câu thì thấy ngay đúng hay sai, đáp án đúng và lời giải thích bằng tiếng Việt. Câu đã trả lời không đổi được. Trả lời hết thì
bước Đọc hoàn thành (kể cả khi có câu sai) và hiện số câu đúng.

**Why this priority**: Là mục tiêu chính của F15: biết người học có hiểu bài hay không, thay cho nút "Đã đọc xong" chỉ dựa vào
lời tự nhận.

**Independent Test**: Với một bài đã có 4 câu hỏi, người học mở bước Đọc, trả lời 3 câu (1 câu sai), thoát ra, vào lại: tiếp
tục ở câu 4; trả lời xong thì bước Đọc chuyển "Xong" và hiện "Đúng 3/4 câu".

**Acceptance Scenarios**:

1. **Given** bài có câu hỏi, **When** người học mở bước Đọc, **Then** thấy nội dung bài và phần câu hỏi; không có nút "Đã đọc
   xong".
2. **Given** đang ở một câu hỏi, **When** chọn một lựa chọn, **Then** ngay lập tức thấy kết quả "Đúng" hoặc "Sai" bằng chữ và ký
   hiệu (✓/✗), lựa chọn đúng được đánh dấu, lời giải thích hiện ra; các lựa chọn của câu đó bị khoá.
3. **Given** câu đã trả lời, **When** người học cố chọn lại (kể cả gửi lại từ nơi khác), **Then** câu trả lời đầu tiên được giữ,
   không đổi kết quả.
4. **Given** đã trả lời một phần, **When** thoát ra rồi vào lại (hoặc tải lại trang, hoặc mở trên thiết bị khác), **Then** các câu
   đã trả lời hiện lại kết quả cũ và người học tiếp tục ở câu chưa trả lời đầu tiên.
5. **Given** đã trả lời câu cuối, **When** kết quả hiện ra, **Then** bước Đọc được tính hoàn thành, hiện "Đúng x/y câu", và người
   học đi tiếp được sang bước sau như giai đoạn 1.
6. **Given** trả lời sai mọi câu, **When** trả lời câu cuối, **Then** bước Đọc vẫn hoàn thành.
7. **Given** người dùng chế độ màu bất kỳ hoặc không phân biệt màu, **When** xem kết quả, **Then** đúng/sai phân biệt được nhờ chữ
   và ký hiệu, không chỉ nhờ màu.

---

### User Story 2 - Ghi chú ngữ pháp trong bước Đọc (Priority: P1)

Bước Đọc có mục **Ngữ pháp**: một ghi chú ngắn bằng tiếng Việt về một điểm ngữ pháp nổi bật trong bài, có ví dụ lấy từ chính bài.
Người học mở hoặc thu gọn mục này bất cứ lúc nào, trước hay sau khi trả lời câu hỏi.

**Why this priority**: Là phần dạy thêm của F15, ít công sức nhưng có giá trị mỗi ngày học.

**Independent Test**: Mở bước Đọc của bài có ghi chú ngữ pháp: thấy mục "Ngữ pháp" có tiêu đề, giải thích tiếng Việt và ví dụ;
thu gọn rồi mở lại được; bài không có ghi chú thì không có mục này.

**Acceptance Scenarios**:

1. **Given** bài có ghi chú ngữ pháp, **When** mở bước Đọc, **Then** mục "Ngữ pháp" hiện với tiêu đề điểm ngữ pháp, phần giải thích
   tiếng Việt và 1–3 ví dụ là câu (hoặc cụm) có trong bài.
2. **Given** mục Ngữ pháp, **When** bấm vào tiêu đề mục, **Then** mục thu gọn hoặc mở ra; thao tác được bằng bàn phím.
3. **Given** bài không có ghi chú ngữ pháp, **When** mở bước Đọc, **Then** không hiện mục Ngữ pháp và không có chỗ trống.

---

### User Story 3 - AI sinh câu hỏi, ngữ pháp và đề viết khi chú thích bài (Priority: P1)

Khi chú thích một bài (lúc tạo bài, sửa nội dung, hoặc bấm **Chạy lại chú thích**), AI trả về thêm trong **cùng một lần gọi**:
3–5 câu hỏi hiểu bài (4 lựa chọn, 1 đáp án đúng, giải thích tiếng Việt), một ghi chú ngữ pháp kèm ví dụ lấy từ bài, và một đề
viết dùng cho bước Viết (F8). Bài có từ trước F15 được bổ sung bằng nút **Chạy lại chú thích**.

**Why this priority**: Không có phần này thì US1, US2 không có dữ liệu; nguồn duy nhất là AI để tiết kiệm công soạn.

**Independent Test**: Tạo bài mới với AI hoạt động: sau khi chú thích xong, trang chi tiết bài có 3–5 câu hỏi hợp lệ, một ghi chú
ngữ pháp có ví dụ nằm trong bài và một đề viết; nhật ký cho thấy chỉ một lần gọi AI cho bài đó.

**Acceptance Scenarios**:

1. **Given** bài mới được lưu, **When** việc chú thích chạy xong, **Then** bài có chú thích từ, 3–5 câu hỏi, ghi chú ngữ pháp và đề
   viết, tất cả từ một lần gọi AI.
2. **Given** AI trả về câu hỏi hỏng (thiếu lựa chọn, lựa chọn trùng nhau, đáp án không nằm trong 4 lựa chọn), **When** lưu kết quả,
   **Then** câu hỏng bị bỏ, các câu hợp lệ được giữ; dưới 3 câu hợp lệ thì vẫn giữ các câu hợp lệ đó.
3. **Given** ví dụ ngữ pháp không có trong bài, **When** lưu kết quả, **Then** ví dụ đó bị bỏ; không còn ví dụ nào thì bỏ cả ghi chú
   ngữ pháp.
4. **Given** AI chỉ trả về một phần (ví dụ thiếu ghi chú ngữ pháp), **When** lưu kết quả, **Then** phần thiếu để trống, các phần
   khác (kể cả chú thích từ) vẫn được lưu; chú thích không bị coi là lỗi vì thiếu phần mới.
5. **Given** bài có từ trước F15 đã chú thích xong, **When** quản trị viên bấm **Chạy lại chú thích**, **Then** bài được chú thích
   lại và có thêm câu hỏi, ngữ pháp, đề viết.
6. **Given** bài có chú thích từ hoặc câu hỏi / ngữ pháp / đề viết đã sửa tay, **When** bấm Chạy lại chú thích, **Then** hệ thống
   cảnh báo phần đã sửa tay sẽ bị thay và chỉ chạy khi quản trị viên xác nhận.
7. **Given** AI lỗi, **When** chú thích, **Then** xử lý như F2 (trạng thái lỗi, chạy lại được); bài không có câu hỏi và bước Đọc
   dùng nút "Đã đọc xong".

---

### User Story 4 - Quản trị viên sửa câu hỏi, ngữ pháp và đề viết (Priority: P2)

Ở trang chi tiết bài, quản trị viên xem và sửa: câu hỏi (thêm, xoá, sửa câu hỏi, lựa chọn, đổi đáp án, sửa giải thích), ghi chú
ngữ pháp (tiêu đề, giải thích, ví dụ) và đề viết.

**Why this priority**: AI có thể sai; quản trị viên phải sửa được trước khi người học gặp. Ít dùng hơn luồng chính.

**Independent Test**: Mở chi tiết bài có 4 câu hỏi: đổi đáp án câu 2, xoá câu 4, thêm một câu mới, sửa đề viết, lưu; tải lại thấy
đúng như đã sửa; người học thấy câu hỏi mới.

**Acceptance Scenarios**:

1. **Given** trang chi tiết bài, **When** quản trị viên sửa và lưu, **Then** câu hỏi, ngữ pháp, đề viết được lưu và đánh dấu "đã sửa
   tay".
2. **Given** đang sửa câu hỏi, **When** để trống câu hỏi hoặc lựa chọn, có hai lựa chọn trùng nhau, không chọn đáp án, hoặc có
   quá 5 câu, **Then** báo lỗi ngay chỗ sai và không lưu.
3. **Given** bài không có câu hỏi, **When** quản trị viên thêm câu hỏi và lưu, **Then** bài có câu hỏi và bước Đọc chuyển sang dùng
   câu hỏi.
4. **Given** quản trị viên xoá hết câu hỏi, **When** lưu, **Then** bước Đọc của bài quay về nút "Đã đọc xong".
5. **Given** người học đã trả lời câu hỏi của bài, **When** quản trị viên sửa câu hỏi, **Then** câu trả lời cũ của người học không
   còn áp dụng cho bộ câu hỏi mới (bước Đọc đã hoàn thành trước đó vẫn giữ hoàn thành).

---

### User Story 5 - Xem lại và thống kê tỷ lệ đúng (Priority: P3)

Người học mở lại một bài đã học (trang Bài học) thấy câu trả lời cũ và kết quả, không trả lời lại và không tính lại. Tỷ lệ trả
lời đúng được lưu; màn hình chính / thống kê (F6) hiện thêm tỷ lệ này.

**Why this priority**: Bổ sung giá trị nhưng không chặn việc học hằng ngày.

**Independent Test**: Học xong bài có câu hỏi (đúng 3/4), mở lại từ trang Bài học: thấy các câu đã chọn và kết quả, không chọn lại
được; trang thống kê hiện "Hiểu bài: 75%".

**Acceptance Scenarios**:

1. **Given** bài đã trả lời hết câu hỏi, **When** mở lại từ trang Bài học, **Then** thấy lựa chọn đã chọn, đúng/sai, đáp án đúng và
   giải thích của mọi câu; không chọn lại được.
2. **Given** đã trả lời câu hỏi ở nhiều bài, **When** mở thống kê, **Then** thấy tỷ lệ trả lời đúng (số câu đúng / số câu đã trả
   lời) và số câu đã trả lời.
3. **Given** chưa trả lời câu nào, **When** mở thống kê, **Then** tỷ lệ hiện "—" kèm lời nhắc, không hiện 0%.
4. **Given** hai người học, **When** mỗi người xem câu trả lời và thống kê, **Then** chỉ thấy dữ liệu của mình.

---

### Edge Cases

- Bài có câu hỏi nhưng người học đã hoàn thành bước Đọc trước khi bài có câu hỏi (bấm "Đã đọc xong"): bước vẫn hoàn thành; khi xem
  lại thấy câu hỏi và trả lời được nhưng không ảnh hưởng tiến độ; câu trả lời vẫn tính vào thống kê.
- Nội dung bài được sửa (câu và chú thích làm lại): câu hỏi cũ bị thay khi chú thích xong; câu trả lời cũ không còn áp dụng; trong
  lúc chú thích đang chạy, bài tạm thời không có câu hỏi và bước Đọc dùng nút "Đã đọc xong".
- Câu hỏi bị quản trị viên sửa trong lúc người học đang trả lời: câu trả lời gửi lên cho bộ câu hỏi cũ bị từ chối với thông báo
  "Câu hỏi vừa được cập nhật" và trang tải lại bộ mới.
- Mất mạng khi chọn đáp án: lựa chọn chưa bị khoá, báo lỗi, chọn lại được.
- Hai tab cùng trả lời một câu: chỉ câu trả lời đầu tiên được ghi; tab còn lại thấy kết quả đã ghi.
- Người học không trả lời hết: bước Đọc chưa hoàn thành, các bước sau vẫn theo quy tắc mở khoá của L (như khi chưa bấm "Đã đọc xong").
- Đề viết có nhưng F8 chưa làm: đề viết chỉ hiện ở trang quản trị, chưa hiện cho người học.
- Ở 360px: câu hỏi, lựa chọn (≥ 44px), giải thích và mục Ngữ pháp không cuộn ngang; thao tác được chỉ bằng bàn phím; trình đọc màn
  hình đọc được kết quả đúng/sai.

## Requirements *(mandatory)*

### Functional Requirements

**Sinh bởi AI**

- **FR-001**: Khi chú thích một bài, hệ thống MUST nhận chú thích từ, câu hỏi hiểu bài, ghi chú ngữ pháp và đề viết trong **một**
  lần gọi AI duy nhất.
- **FR-002**: Mỗi câu hỏi MUST có nội dung câu hỏi, đúng 4 lựa chọn khác nhau, đúng 1 đáp án nằm trong 4 lựa chọn và lời giải
  thích tiếng Việt; câu không đạt MUST bị bỏ; giữ tối đa 5 câu.
- **FR-003**: Ghi chú ngữ pháp MUST có tiêu đề, phần giải thích tiếng Việt và 1–3 ví dụ có trong nội dung bài; ví dụ không có trong
  bài MUST bị bỏ; không còn ví dụ thì bỏ cả ghi chú.
- **FR-004**: Thiếu hoặc hỏng câu hỏi / ngữ pháp / đề viết MUST NOT làm chú thích từ thất bại; phần thiếu để trống.
- **FR-005**: **Chạy lại chú thích** MUST dùng được cho bài đã chú thích xong (không chỉ bài lỗi), trừ khi chú thích đang chạy.
- **FR-006**: Trước khi chạy lại chú thích cho bài có phần đã sửa tay (chú thích từ, câu hỏi, ngữ pháp, đề viết), hệ thống MUST
  cảnh báo phần nào sẽ bị thay và chỉ chạy khi quản trị viên xác nhận.

**Quản trị**

- **FR-007**: Quản trị viên MUST xem và sửa được câu hỏi (thêm, xoá, sửa nội dung, lựa chọn, đáp án, giải thích), ghi chú ngữ pháp
  (tiêu đề, giải thích, ví dụ) và đề viết ở trang chi tiết bài.
- **FR-008**: Khi lưu, hệ thống MUST kiểm tra theo FR-002 (tối đa 5 câu, có thể 0 câu) và FR-003 (ví dụ có trong bài); sai thì báo
  lỗi theo từng ô bằng tiếng Việt và không lưu.
- **FR-009**: Phần đã sửa tay MUST được đánh dấu để dùng cho cảnh báo FR-006.

**Bước Đọc của người học**

- **FR-010**: Bài có ít nhất một câu hỏi MUST hiện phần câu hỏi trong bước Đọc thay cho nút "Đã đọc xong"; bài không có câu hỏi
  MUST giữ nút "Đã đọc xong" như giai đoạn 1.
- **FR-011**: Người học MUST NOT nhìn thấy đáp án đúng hay lời giải thích của câu chưa trả lời (kể cả khi xem dữ liệu trang).
- **FR-012**: Chọn một lựa chọn MUST ghi lại câu trả lời và trả về ngay: đúng/sai, đáp án đúng, lời giải thích; câu đã trả lời
  MUST NOT đổi được (lần trả lời sau bị từ chối, giữ kết quả đầu).
- **FR-013**: Đúng/sai MUST được thể hiện bằng chữ ("Đúng"/"Sai") và ký hiệu (✓/✗), không chỉ bằng màu, và được thông báo cho trình
  đọc màn hình.
- **FR-014**: Tiến độ trả lời MUST được lưu ở máy chủ theo người học và bài: vào lại (tải lại, thiết bị khác) thấy các câu đã trả lời
  và tiếp tục ở câu chưa trả lời đầu tiên.
- **FR-015**: Bước Đọc MUST được tính hoàn thành khi người học đã trả lời hết câu hỏi của bài (đúng hay sai đều được); máy chủ tự xác
  định điều này, không tin vào báo cáo của trình duyệt. Khi hoàn thành MUST hiện "Đúng x/y câu".
- **FR-016**: Mục **Ngữ pháp** MUST hiện trong bước Đọc khi bài có ghi chú, thu gọn / mở ra được bằng chuột và bàn phím.
- **FR-017**: Mở lại bài đã học (trang Bài học) MUST hiện câu trả lời cũ và kết quả, không cho trả lời lại câu đã trả lời và không
  tính lại tiến độ.
- **FR-018**: Câu trả lời chỉ áp dụng cho đúng phiên bản câu hỏi đã trả lời; khi nội dung bài hoặc câu hỏi thay đổi, câu trả lời cũ
  MUST NOT được hiện như câu trả lời của bộ câu hỏi mới. Bước Đọc đã hoàn thành trước đó vẫn giữ hoàn thành.

**Thống kê và dữ liệu**

- **FR-019**: Mỗi câu trả lời MUST được lưu kèm đúng/sai để tính tỷ lệ trả lời đúng; màn hình thống kê (F6) MUST hiện tỷ lệ đúng và
  số câu đã trả lời; chưa có câu nào thì hiện "—".
- **FR-020**: Câu trả lời và tỷ lệ MUST chỉ thuộc về người học đã trả lời; người khác không xem hay ghi được; dữ liệu này MUST có
  trong file xuất dữ liệu (F13).
- **FR-021**: Đề viết MUST chỉ hiện cho quản trị viên ở F15 (người học dùng ở F8).

### Key Entities

- **Câu hỏi hiểu bài** (thuộc bài): nội dung câu hỏi, 4 lựa chọn, chỉ số đáp án đúng, giải thích tiếng Việt; 0–5 câu mỗi bài, có thứ
  tự.
- **Ghi chú ngữ pháp** (thuộc bài, tối đa 1): tiêu đề, giải thích tiếng Việt, 1–3 ví dụ lấy từ bài.
- **Đề viết** (thuộc bài, tối đa 1): đoạn văn ngắn yêu cầu viết, dùng ở F8.
- **Đánh dấu đã sửa tay** (thuộc bài): cho biết câu hỏi / ngữ pháp / đề viết đã được quản trị viên sửa.
- **Câu trả lời của người học**: người học, bài, phiên bản câu hỏi, câu số mấy, lựa chọn, đúng/sai, thời điểm; mỗi câu chỉ một câu
  trả lời cho mỗi phiên bản.
- **Tỷ lệ hiểu bài** (tính ra): số câu đúng / số câu đã trả lời của một người học.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Chú thích một bài (kể cả phần câu hỏi, ngữ pháp, đề viết) tốn đúng 1 lần gọi AI, kiểm được qua nhật ký.
- **SC-002**: Với AI hoạt động, ít nhất 9/10 bài mới có từ 3 câu hỏi hợp lệ trở lên và có ghi chú ngữ pháp với ví dụ có trong bài.
- **SC-003**: 100% bài có câu hỏi: bước Đọc hoàn thành ngay sau câu trả lời cuối cùng, kể cả khi mọi câu sai; 100% bài không có câu
  hỏi vẫn hoàn thành bằng nút "Đã đọc xong".
- **SC-004**: Người học thấy kết quả đúng/sai của một câu trong dưới 1 giây sau khi chọn (mạng bình thường).
- **SC-005**: Kiểm tra với chế độ xám (không màu) và trình đọc màn hình: 100% kết quả đúng/sai vẫn phân biệt được.
- **SC-006**: Thoát giữa chừng rồi vào lại (cùng hoặc khác thiết bị): 100% câu đã trả lời được giữ, tiếp tục đúng câu kế tiếp.
- **SC-007**: Tỷ lệ đúng trên thống kê khớp với số câu đúng / số câu đã trả lời tính tay từ dữ liệu của người học.
- **SC-008**: Mọi thao tác của F15 làm được ở 360px và chỉ bằng bàn phím.

## Assumptions

- Dùng AI và việc chú thích nền đã có từ F2 (Gemini gói miễn phí); không thêm cấu hình mới. Phần câu hỏi, ngữ pháp, đề viết được sinh
  cùng lúc và lưu cùng chú thích từ, nên không tốn thêm request.
- Câu hỏi và đề viết bằng tiếng Anh, đúng trình độ của bài; giải thích và ghi chú ngữ pháp bằng tiếng Việt.
- "Ví dụ có trong bài" được kiểm bằng cách tìm nguyên văn trong nội dung bài (không phân biệt hoa thường, bỏ khoảng trắng thừa).
- Bài có từ trước F15 không được tự chạy lại; quản trị viên chủ động bấm Chạy lại chú thích từng bài để tiết kiệm hạn mức AI.
- Sửa câu hỏi bằng tay tạo một phiên bản câu hỏi mới; câu trả lời cho phiên bản cũ vẫn tính vào thống kê nhưng không hiện lại.
- Câu hỏi sinh mới khi chạy lại chú thích thay toàn bộ câu hỏi cũ (sau khi đã cảnh báo nếu có phần sửa tay).
- Tỷ lệ hiểu bài tính trên mọi câu đã trả lời (không theo khoảng thời gian); chi tiết theo ngày/tuần ngoài phạm vi.
- Ngoài phạm vi: câu hỏi tự luận, nhiều điểm ngữ pháp mỗi bài, bài tập ngữ pháp riêng, hiện đề viết cho người học (F8).
- Phụ thuộc: F2 (chú thích, trang chi tiết bài), F3 (bước Đọc), L (tiến độ bài trong ngày, mở khoá bước), F6 (thống kê), F13 (xuất
  dữ liệu).
