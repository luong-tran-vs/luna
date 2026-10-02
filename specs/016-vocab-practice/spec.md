# Feature Specification: Trang chi tiết bài và luyện tập từ vựng

**Feature Branch**: `016-vocab-practice`

**Created**: 2026-10-02

**Status**: Draft

**Mã tính năng**: F17 | **Giai đoạn**: Giai đoạn 3 (`docs/phases/giai-doan-3.md`, tính năng không bắt buộc)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f17-luyen-tap-tu-vung.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Học từ vựng qua câu ví dụ và hội thoại mẫu (Priority: P1)

Người học mở một bài từ danh sách bài học (bài hôm nay hoặc bài đã học) và vào trang chi tiết bài. Trang có:

- thanh trên: nút đóng (về danh sách bài), thanh tiến độ 4 đoạn kèm chữ "1/4", streak ("3 ngày");
- đầu trang: "Bài N" + tiêu đề, dòng "Mục tiêu: …", nhãn mức độ;
- hai tab **Bài học** (4 bước luyện tập) và **Bài đọc** (bài đọc gốc và ghi chú ngữ pháp);
- nút **Tiếp theo →** cố định ở cuối màn hình.

Bước 1 – **Từ vựng quan trọng**: mỗi từ có phiên âm, nghĩa tiếng Việt, nút loa đọc từ và một câu ví dụ tiếng Anh nghe được.
Bước 2 – **Hội thoại mẫu**: đoạn hội thoại giữa 2 người dùng các từ của bài; nghe cả đoạn (đổi tốc độ) hoặc từng lượt, bật/tắt nghĩa
tiếng Việt.

**Why this priority**: Đây là phần người học nhận giá trị đầu tiên: thấy từ trong câu, trong hội thoại thật, nghe được cách đọc. Bước 3
và 4 đều dựa trên nội dung này.

**Independent Test**: Mở một bài đã có phần luyện tập. Thấy đầu trang đủ thông tin, bước 1 có câu ví dụ chứa đúng từ, bấm loa nghe
được từ và câu. Bấm Tiếp theo sang bước 2 (tiến độ "2/4"), nghe cả đoạn ở tốc độ 0.75×, nghe một lượt riêng, tắt rồi bật nghĩa tiếng Việt.

**Acceptance Scenarios**:

1. **Given** bài hôm nay hoặc bài đã học, **When** chạm bài trong danh sách bài học, **Then** mở trang chi tiết bài ở tab Bài học,
   bước 1, tiến độ "1/4".
2. **Given** bài sắp tới (khoá), **When** chạm bài hoặc mở thẳng đường dẫn của bài, **Then** không xem được nội dung bài như hiện nay.
3. **Given** trang chi tiết bài, **When** xem đầu trang, **Then** thấy "Bài N" (thứ tự của bài trong lộ trình chủ đề), tiêu đề, "Mục tiêu:
   …" và nhãn "Mức độ: Cơ bản" (A1–A2), "Trung cấp" (B1–B2) hoặc "Nâng cao" (C1–C2).
4. **Given** bước 1, **When** xem một từ, **Then** thấy từ, phiên âm, "— nghĩa tiếng Việt", nút loa và dòng "Ví dụ: …" bằng tiếng Anh
   chứa đúng từ đó. Bấm loa nghe từ, bấm câu ví dụ nghe câu.
5. **Given** bước 1, **When** bấm thu gọn, **Then** danh sách từ ẩn đi; bấm lần nữa thì hiện lại.
6. **Given** bước 2, **When** bấm phát cả đoạn, **Then** các lượt được đọc lần lượt; thanh phát hiện thời gian; đổi được tốc độ 0.75×, 1×,
   1.25×; dừng được giữa chừng.
7. **Given** bước 2, **When** xem danh sách lượt, **Then** mỗi lượt có tên người nói, câu tiếng Anh, nút nghe lượt đó và nghĩa tiếng Việt;
   nút bật/tắt nghĩa ẩn hoặc hiện nghĩa của mọi lượt.
8. **Given** tab Bài đọc, **When** chọn, **Then** thấy bài đọc gốc và ghi chú ngữ pháp; quay lại tab Bài học vẫn ở đúng bước đang làm.
9. **Given** bất kỳ bước nào, **When** bấm nút đóng, **Then** về danh sách bài học.

---

### User Story 2 - Phần luyện tập được sinh tự động và không làm hỏng bài (Priority: P1)

Sau khi bài được chú thích xong, hệ thống tự sinh phần luyện tập ở chế độ chạy nền, bằng đúng 1 yêu cầu AI riêng: mục tiêu bài, câu ví
dụ cho từng từ, hội thoại mẫu, mẹo ngữ pháp, các câu dịch. Sau đó tạo audio cho câu ví dụ và từng lượt hội thoại. Nội dung hỏng được
bỏ bớt. Bài chưa có phần luyện tập vẫn mở được bình thường.

**Why this priority**: Không có nội dung thì US1, US3, US4 không có gì để hiện. Đồng thời phải giữ nguyên tắc "AI là phần bổ sung".

**Independent Test**: Tạo một bài mới, đợi chú thích xong. Sau ít phút trang chi tiết bài có đủ 4 bước. Với AI giả lập lỗi: chú thích từ
và câu hỏi hiểu bài vẫn có; bước 1 vẫn hiện từ vựng (không có câu ví dụ); bước 2–4 báo "Bài này chưa có phần luyện tập".

**Acceptance Scenarios**:

1. **Given** chú thích bài vừa xong thành công, **When** vài phút sau mở trang chi tiết bài, **Then** có phần luyện tập; mỗi câu ví dụ
   chứa đúng từ của nó.
2. **Given** chú thích bài lỗi, **When** xem bài, **Then** không sinh phần luyện tập.
3. **Given** AI lỗi hoặc trả nội dung không dùng được khi sinh phần luyện tập, **When** xem bài, **Then** chú thích từ, câu hỏi hiểu bài,
   đề viết và các bước Ôn → Đọc → Nghe → Viết vẫn bình thường; chỉ phần luyện tập trống.
4. **Given** bài chưa có phần luyện tập (bài cũ, đang sinh, AI lỗi), **When** mở trang chi tiết bài, **Then** bước 1 hiện danh sách từ
   vựng không có câu ví dụ; bước 2, 3, 4 báo "Bài này chưa có phần luyện tập"; vẫn chuyển bước và xem tab Bài đọc được.
5. **Given** AI trả về câu ví dụ không chứa từ của nó, **When** lưu, **Then** câu đó bị bỏ, từ đó hiện không có câu ví dụ.
6. **Given** AI trả về hội thoại dưới 4 lượt, **When** lưu, **Then** bỏ cả hội thoại; bước 2 và 3 báo chưa có phần luyện tập.
7. **Given** một câu dịch có đáp án không ghép được từ các ô, **When** lưu, **Then** bỏ câu đó, các câu khác vẫn dùng.
8. **Given** quản trị viên sửa nội dung bài hoặc chạy lại chú thích, **When** xong, **Then** phần luyện tập cũ bị bỏ và được sinh lại
   theo nội dung mới.

---

### User Story 3 - Điền từ vào ô trống trong hội thoại (Priority: P2)

Bước 3 hiện lại hội thoại ở bước 2, các từ vựng của bài được thay bằng ô trống (tối đa 5 ô). Bên dưới là **Ngân hàng từ** gồm các đáp
án và 2 từ vựng khác của bài, xáo trộn. Người học chạm từ để điền vào ô đang chọn, gỡ được, rồi bấm **Kiểm tra**.

**Why this priority**: Luyện nhớ chủ động, nhưng cần nội dung của US1 và US2 trước.

**Independent Test**: Ở bước 3, điền đúng mọi ô, bấm Kiểm tra: thấy "Chính xác!". Làm lại, điền sai một ô: ô đó được đánh dấu sai
bằng chữ và thấy đáp án đúng.

**Acceptance Scenarios**:

1. **Given** bước 3, **When** xem, **Then** mỗi lượt có thẻ người nói với nút "Nghe câu", câu có ô trống, nghĩa tiếng Việt bên dưới;
   ô trống đầu tiên đang được chọn.
2. **Given** một ô đang chọn, **When** chạm một từ trong Ngân hàng từ, **Then** từ được điền vào ô đó, từ trong ngân hàng mờ đi (không
   chọn lại được), ô trống kế tiếp còn trống được chọn.
3. **Given** ô đã điền, **When** chạm ✕ trên ô, **Then** từ được gỡ, trở lại dùng được trong ngân hàng, ô đó được chọn.
4. **Given** ô đã điền, **When** chạm vào ô, **Then** ô đó được chọn; chạm từ khác thì thay từ cũ (từ cũ trở lại ngân hàng).
5. **Given** chưa điền đủ mọi ô, **When** xem nút Kiểm tra, **Then** nút bị khoá.
6. **Given** đã điền đủ, **When** bấm Kiểm tra, **Then** đúng hết thì hiện "Chính xác!"; có ô sai thì mỗi ô sai có dấu và chữ "Sai",
   kèm đáp án đúng; ô đúng có chữ "Đúng". So khớp không phân biệt hoa/thường.
7. **Given** đã kiểm tra, **When** xem bên dưới, **Then** thấy thẻ "Mẹo ngữ pháp".

---

### User Story 4 - Ghép câu dịch Việt → Anh và tổng kết (Priority: P2)

Bước 4 hiện lần lượt từng câu tiếng Việt (3–5 câu). Người học ghép "Câu dịch của bạn" bằng cách chạm các ô từ trong Ngân hàng từ (từ của
đáp án và từ gây nhiễu, xáo trộn), gỡ hoặc **Làm lại**, rồi **Kiểm tra**. Hết câu cuối, nút đổi thành **Hoàn thành** và hiện tổng kết.

**Why this priority**: Bài tập khó nhất, cần dùng từ chủ động; cũng là nơi kết thúc luồng 4 bước.

**Independent Test**: Ở bước 4, ghép đúng câu 1 → "Chính xác!"; ghép sai câu 2 → thấy câu đúng; ghép dở câu 3 rồi bấm Làm lại → câu
trống. Hoàn thành → tổng kết đúng số ô điền đúng và số câu dịch đúng; bấm Làm lại → về bước 1, mọi kết quả xoá.

**Acceptance Scenarios**:

1. **Given** bước 4, **When** xem, **Then** thấy câu tiếng Việt, thứ tự câu ("Câu 1/4"), vùng "Câu dịch của bạn" trống và Ngân hàng từ.
2. **Given** chạm một ô trong ngân hàng, **When** xong, **Then** ô được thêm vào cuối câu đang ghép và mờ đi trong ngân hàng; chạm ✕ trên
   một ô trong câu thì gỡ ô đó; **Làm lại** gỡ hết.
3. **Given** câu đang ghép trống, **When** xem nút Kiểm tra, **Then** nút bị khoá.
4. **Given** bấm Kiểm tra, **When** các ô đúng đáp án và đúng thứ tự, **Then** hiện "Chính xác!"; ngược lại hiện "Chưa đúng" kèm câu
   đúng. Sau khi kiểm tra, nút loa đọc câu tiếng Anh đúng dùng được.
5. **Given** đã kiểm tra một câu không phải câu cuối, **When** bấm Tiếp theo, **Then** sang câu kế tiếp (vẫn ở bước 4).
6. **Given** đã kiểm tra câu cuối, **When** bấm Hoàn thành, **Then** hiện tổng kết: "Điền đúng x/y ô", "Dịch đúng a/b câu", nút **Làm
   lại** và nút về bài.
7. **Given** tổng kết của bài hôm nay, **When** xem nút về bài, **Then** là "Học bài này" (vào luồng học hôm nay). Với bài đã học: các
   nút Đọc lại, Nghe lại, Bài viết.
8. **Given** tổng kết, **When** bấm Làm lại, **Then** về bước 1, mọi ô điền và câu ghép xoá hết, ngân hàng từ xáo lại.

---

### User Story 5 - Quản trị viên xem và tạo lại phần luyện tập (Priority: P3)

Ở trang chi tiết bài của khu quản trị có mục "Phần luyện tập": trạng thái (chưa có, đang sinh, xong, lỗi kèm lý do), nội dung đã sinh và
nút **Tạo lại phần luyện tập**.

**Why this priority**: Cần khi AI lỗi hoặc nội dung chưa tốt, nhưng không chặn người học.

**Independent Test**: Mở một bài có phần luyện tập lỗi ở khu quản trị, thấy trạng thái "Lỗi" và lý do. Bấm Tạo lại: trạng thái "Đang
sinh", sau ít phút "Xong" và xem được nội dung.

**Acceptance Scenarios**:

1. **Given** trang chi tiết bài quản trị, **When** xem mục Phần luyện tập, **Then** thấy trạng thái và, khi xong, toàn bộ nội dung: mục
   tiêu, câu ví dụ, hội thoại, mẹo ngữ pháp, câu dịch kèm từ gây nhiễu.
2. **Given** bấm Tạo lại phần luyện tập, **When** xong, **Then** tốn đúng 1 yêu cầu AI; phần cũ được thay khi phần mới sinh xong thành công.
3. **Given** đang sinh, **When** bấm Tạo lại lần nữa, **Then** không tạo thêm yêu cầu AI trùng.
4. **Given** bài chưa được chú thích xong, **When** xem mục, **Then** nút Tạo lại bị khoá kèm lý do.

---

### Edge Cases

- Bài có ít từ vựng: ngân hàng từ bước 3 thêm tối đa 2 từ khác, có bao nhiêu dùng bấy nhiêu.
- Hội thoại không chứa từ vựng nào của bài: bước 3 báo "Bài này chưa có phần luyện tập", bước 2 vẫn dùng được.
- Một từ vựng xuất hiện nhiều lần: mỗi ô trống có một thẻ đáp án riêng trong ngân hàng từ; ô trống được trải đều các lượt, tối đa 5.
- Hai ô từ cùng chữ (trong ngân hàng từ của bước 3 hoặc 4): thay thế được cho nhau khi chấm.
- Dấu câu dính theo từ ("you,", "Vietnam."): ô từ giữ nguyên dấu câu và chữ hoa; chấm câu dịch so đúng nội dung từng ô.
- Mọi câu dịch đều hỏng: bước 4 báo "Bài này chưa có phần luyện tập"; tổng kết chỉ hiện phần có làm.
- Audio của một câu chưa có hoặc lỗi: nút nghe của câu đó bị ẩn hoặc khoá, phần còn lại vẫn dùng được; phát cả đoạn bỏ qua lượt thiếu audio.
- Phần luyện tập sinh xong khi người học đang mở trang: không tự đổi nội dung đang làm; mở lại trang thì thấy.
- Rời trang hoặc tải lại giữa chừng: kết quả luyện tập mất, bắt đầu lại từ bước 1.
- Bấm Tiếp theo khi chưa kiểm tra ở bước 3 hoặc 4: vẫn chuyển bước được; ô/câu chưa kiểm tra tính là sai trong tổng kết.
- Bài thuộc chủ đề người học không còn theo: mở được nếu là bài đã học, như hiện nay.
- Ở 360px, sáng và tối: không cuộn ngang; nút Tiếp theo không che nội dung và không đè thanh tab đáy; vùng chạm ≥ 44px.

## Requirements *(mandatory)*

### Functional Requirements

**Nội dung luyện tập**

- **FR-001**: Khi chú thích bài xong thành công, hệ thống MUST tự xếp hàng sinh phần luyện tập chạy nền, dùng đúng 1 yêu cầu AI cho mỗi
  lần sinh, tách riêng với yêu cầu chú thích.
- **FR-002**: Phần luyện tập MUST gồm:
  - mục tiêu bài: 1 câu tiếng Việt;
  - câu ví dụ: mỗi từ vựng 1 câu tiếng Anh ngắn, đúng trình độ bài, chứa đúng từ đó;
  - hội thoại mẫu: 2 người có tên, 4–10 lượt, dùng nhiều từ vựng của bài, mỗi lượt có nghĩa tiếng Việt;
  - mẹo ngữ pháp: 1–2 câu tiếng Việt về một cách nói trong hội thoại;
  - câu dịch: 3–5 câu tiếng Việt, mỗi câu có 1 đáp án tiếng Anh dùng ít nhất một từ vựng của bài và 3–4 từ gây nhiễu.
- **FR-003**: Hệ thống MUST kiểm tra nội dung trước khi lưu và bỏ phần hỏng:
  - câu ví dụ không chứa từ của nó;
  - hội thoại dưới 4 lượt (bỏ cả đoạn); quá 10 lượt thì giữ 10 lượt đầu;
  - câu dịch có đáp án rỗng hoặc không dùng từ vựng nào của bài; giữ tối đa 5 câu, mỗi câu tối đa 4 từ gây nhiễu.
- **FR-004**: Ô trống của bước 3 MUST là các lần xuất hiện nguyên từ của từ vựng bài trong hội thoại (khớp dạng trong bài hoặc dạng
  gốc), tối đa 5, trải đều các lượt; không cần thêm yêu cầu AI.
- **FR-005**: Đáp án câu dịch MUST được tách thành các ô theo khoảng trắng, dấu câu dính theo từ.
- **FR-006**: Sau khi lưu phần luyện tập, hệ thống MUST tạo audio (cùng giọng với bài đọc) cho từng câu ví dụ, từng lượt hội thoại và
  đáp án mỗi câu dịch.
- **FR-007**: Phần luyện tập MUST có trạng thái: chưa có, đang sinh, xong, lỗi (kèm lý do).
- **FR-008**: AI lỗi khi sinh phần luyện tập MUST NOT ảnh hưởng chú thích từ, câu hỏi hiểu bài, đề viết hay bất kỳ bước nào của bài.
- **FR-009**: Khi nội dung bài được sửa hoặc chú thích được chạy lại, phần luyện tập cũ MUST bị bỏ và được sinh lại theo nội dung mới;
  kết quả sinh của nội dung cũ đến muộn MUST NOT được lưu.

**Trang chi tiết bài**

- **FR-010**: Trang chi tiết bài MUST mở được từ danh sách bài học cho bài hôm nay và bài đã học; bài sắp tới MUST vẫn bị chặn ở cả giao
  diện và phía máy chủ.
- **FR-011**: Trang MUST có thanh trên (đóng, tiến độ 4 đoạn kèm "n/4", streak hiện tại), đầu trang ("Bài N" theo thứ tự trong lộ trình
  chủ đề, tiêu đề, mục tiêu, nhãn mức độ Cơ bản/Trung cấp/Nâng cao theo A1–A2/B1–B2/C1–C2) và hai tab Bài học, Bài đọc. Thiếu mục
  tiêu thì ẩn dòng "Mục tiêu".
- **FR-012**: Nút Tiếp theo MUST cố định ở cuối màn hình, chuyển lần lượt 4 bước và cập nhật tiến độ; ở câu cuối bước 4 nút là "Hoàn
  thành".
- **FR-013**: Bước 1 MUST hiện mọi từ vựng của bài với phiên âm, nghĩa tiếng Việt, nút nghe từ và câu ví dụ nghe được (nếu có); thu gọn
  được.
- **FR-014**: Bước 2 MUST cho nghe cả đoạn (lần lượt từng lượt, có thời gian, tốc độ 0.75×/1×/1.25×, dừng được), nghe từng lượt, và
  bật/tắt nghĩa tiếng Việt.
- **FR-015**: Bước 3 MUST hoạt động như User Story 3: Ngân hàng từ gồm các đáp án và tối đa 2 từ vựng khác của bài, xáo trộn; điền, gỡ,
  thay; Kiểm tra chỉ khi đủ ô; chấm không phân biệt hoa/thường; hiện mẹo ngữ pháp.
- **FR-016**: Bước 4 MUST hoạt động như User Story 4: ghép ô theo thứ tự chạm, gỡ, Làm lại, Kiểm tra từng câu; đúng khi các ô khớp đáp
  án đúng thứ tự; sai thì hiện câu đúng; sau khi kiểm tra nghe được câu đúng.
- **FR-017**: Tổng kết MUST hiện số ô điền đúng/tổng số ô và số câu dịch đúng/tổng số câu, nút Làm lại (xoá mọi kết quả, về bước 1) và
  nút về bài theo loại bài (bài hôm nay: "Học bài này"; bài đã học: Đọc lại, Nghe lại, Bài viết).
- **FR-018**: Khi bài chưa có phần luyện tập, bước 1 MUST vẫn hiện từ vựng (không có câu ví dụ) và các bước 2–4 MUST báo "Bài này chưa có
  phần luyện tập".
- **FR-019**: Chấm điểm MUST diễn ra ngay trên trang; kết quả chỉ giữ trong lần mở trang, MUST NOT gửi hay lưu lên máy chủ.
- **FR-020**: Phần luyện tập MUST NOT ảnh hưởng các bước Ôn → Đọc → Nghe → Viết, tiến độ, streak hay thống kê; không có điểm XP.
- **FR-021**: Đúng/sai MUST phân biệt được bằng chữ và biểu tượng, không chỉ bằng màu. Mọi thao tác (chọn ô, chạm từ, gỡ, Kiểm tra, Tiếp
  theo, phát audio, đổi tab) MUST làm được bằng bàn phím; kết quả kiểm tra được thông báo cho trình đọc màn hình.
- **FR-022**: Trang MUST dùng tốt ở 360px, chế độ sáng và tối, chỉ dùng màu và font của `docs/design-system.md`.

**Quản trị**

- **FR-023**: Trang chi tiết bài quản trị MUST hiện trạng thái và nội dung phần luyện tập, và nút Tạo lại phần luyện tập (khoá khi bài
  chưa chú thích xong hoặc đang sinh). Tạo lại MUST tốn đúng 1 yêu cầu AI.

### Key Entities

- **Phần luyện tập của bài** (gắn với một phiên bản nội dung bài, dùng chung cho mọi người học):
  - trạng thái, lý do lỗi;
  - mục tiêu bài, mẹo ngữ pháp;
  - danh sách câu ví dụ (từ vựng, câu, audio);
  - hội thoại (2 tên người nói; các lượt: người nói, câu, nghĩa tiếng Việt, audio);
  - danh sách câu dịch (câu tiếng Việt, đáp án tiếng Anh, từ gây nhiễu, audio đáp án).
- **Bài điền ô trống** (suy ra từ hội thoại và từ vựng): các lượt có ô trống, đáp án mỗi ô, ngân hàng từ.
- **Kết quả luyện tập** (chỉ trong lần mở trang): ô đã điền, câu đã ghép, đúng/sai, tổng kết.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Với AI hoạt động, ít nhất 9/10 bài mới có phần luyện tập trong vòng 5 phút sau khi chú thích xong.
- **SC-002**: 100% câu ví dụ đã lưu chứa đúng từ của nó; 100% ô trống và đáp án ô trống là từ vựng của bài; 100% câu dịch đã lưu ghép
  được đúng đáp án từ các ô.
- **SC-003**: Mỗi lần sinh phần luyện tập tốn đúng 1 yêu cầu AI (kiểm qua nhật ký).
- **SC-004**: Khi AI lỗi lúc sinh phần luyện tập, 100% bài vẫn có chú thích từ, câu hỏi hiểu bài và học trọn các bước của bài như trước.
- **SC-005**: Người học đi hết 4 bước và thấy tổng kết của một bài trong dưới 10 phút.
- **SC-006**: Học xong phần luyện tập không làm thay đổi tiến độ, streak hay thống kê (so trước và sau).
- **SC-007**: Mọi thao tác của F17 làm được ở 360px, cả sáng và tối, và chỉ bằng bàn phím.

## Assumptions

- Dùng AI và giọng đọc đã cấu hình cho chú thích và bài đọc (F2, F4); không thêm cấu hình mới. Hội thoại dùng một giọng cho cả hai người.
- Trang `/lessons/:id` hiện có (bản đơn giản) được thay bằng trang chi tiết mới; các trang đọc lại, nghe lại, bài viết giữ nguyên.
- "Bài N" là thứ tự của bài trong lộ trình chủ đề mà bài thuộc về; streak lấy từ số liệu streak hiện có.
- Từ vựng của bài là danh sách từ có trong chú thích của bài (F2); bài không có từ vựng thì bước 1 báo chưa có từ vựng.
- Bài cũ (chú thích xong trước F17) được tự sinh phần luyện tập một lần khi backend khởi động, nếu đã cấu hình AI (đổi 2026-10-02);
  bài sinh lỗi không tự sinh lại, quản trị viên bấm Tạo lại.
- Phần luyện tập dùng chung cho mọi người học, không chứa dữ liệu cá nhân; đáp án được gửi về trang để chấm (chấp nhận được vì không có
  điểm số).
- Ngoài phạm vi: điểm XP, AI chấm câu dịch tự do, nhiều giọng đọc cho hội thoại, lưu và thống kê kết quả luyện tập, quản trị viên sửa
  tay từng câu luyện tập, các loại bài tập khác (ghép đôi, kiểm tra cuối bài).
- Phụ thuộc: F2 (chú thích, phiên bản nội dung bài, trang quản trị bài), F4 (audio), F15 (ghi chú ngữ pháp ở tab Bài đọc), L (quyền mở bài, streak), F14
  (lộ trình chủ đề).
