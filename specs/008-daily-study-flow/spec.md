# Feature Specification: Luồng một ngày học

**Feature Branch**: `008-daily-study-flow`

**Created**: 2026-09-30

**Status**: Draft

**Mã tính năng**: L | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`, `docs/mvp-features.md` mục 4)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/l-luong-mot-ngay-hoc.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đặt mục tiêu: chọn trình độ và chủ đề (Priority: P1)

Người học mới (hoặc chưa có mục tiêu) chọn trình độ A1–C2, rồi chọn một chủ đề của trình độ đó. Danh sách chủ đề hiện số bài
của mỗi chủ đề và tiến độ nếu đã học dở. Mục tiêu là hoàn thành lộ trình của chủ đề theo thứ tự quản trị viên xếp (F14).

**Why this priority**: Không có mục tiêu thì không biết bài hôm nay là bài nào; mọi thứ khác của L dựa vào đây.

**Independent Test**: Đăng nhập tài khoản chưa có mục tiêu, chọn A1 → "Gia đình"; thấy mục tiêu "A1 · Gia đình, 0/N bài" và bài
hôm nay là bài đầu tiên của lộ trình.

**Acceptance Scenarios**:

1. **Given** người học chưa có mục tiêu, **When** mở "Học hôm nay", **Then** được dẫn tới bước chọn trình độ.
2. **Given** chọn trình độ A1, **When** sang bước chọn chủ đề, **Then** chỉ thấy các chủ đề A1, mỗi chủ đề kèm số bài trong lộ
   trình; chủ đề chưa có bài hiện "Chưa có bài" nhưng vẫn chọn được.
3. **Given** chọn "A1 · Gia đình", **When** xác nhận, **Then** mục tiêu là "A1 · Gia đình" với tiến độ 0/N bài và bài hôm nay
   là bài đầu tiên của lộ trình.
4. **Given** đã học 3 bài của "A1 · Gia đình" rồi chuyển sang chủ đề khác, **When** mở lại danh sách chủ đề A1, **Then**
   "Gia đình" hiện tiến độ "3/N bài".
5. **Given** học xong bài cuối của lộ trình, **When** hoàn thành bài, **Then** thấy lời chúc mừng và lời mời chọn chủ đề khác cùng
   trình độ hoặc lên trình độ tiếp theo.

---

### User Story 2 - Học bài hôm nay: Ôn → Đọc → Nghe (Priority: P1)

Người học mở "Bài hôm nay" (bài tiếp theo chưa hoàn thành của lộ trình). Bài gồm ba bước theo thứ tự: Ôn (thẻ đến hạn, tối đa 30
thẻ/ngày), Đọc (F3), Nghe (F4). Bước sau chỉ mở khi bước trước xong. Thanh tiến trình cho biết đang ở bước nào. Thoát giữa
chừng rồi vào lại thì tiếp tục đúng bước và đúng câu đang làm.

**Why this priority**: Là trải nghiệm cốt lõi của app (mục 4 của `mvp-features.md`).

**Independent Test**: Với mục tiêu đã chọn và vài thẻ đến hạn: ôn xong, đọc xong, nghe tới câu 3 rồi đóng trình duyệt; mở lại
thấy đang ở bước Nghe câu 3; làm hết thì bài hoàn thành.

**Acceptance Scenarios**:

1. **Given** có 12 thẻ đến hạn, **When** mở bài hôm nay, **Then** thấy thanh bước "Ôn · Đọc · Nghe" với Ôn là bước hiện tại, Đọc
   và Nghe bị khoá, và phiên ôn gồm 12 thẻ.
2. **Given** có 45 thẻ đến hạn và chưa ôn thẻ nào hôm nay, **When** vào bước Ôn, **Then** phiên ôn chỉ gồm 30 thẻ quá hạn lâu
   nhất; 15 thẻ còn lại chờ tới hôm sau.
3. **Given** không có thẻ đến hạn, **When** mở bài hôm nay, **Then** bước Ôn tự hoàn thành và bước hiện tại là Đọc.
4. **Given** đang ở bước Ôn, **When** tìm cách mở bước Đọc (kể cả gõ thẳng địa chỉ), **Then** không được; thấy "Hoàn thành bước
   Ôn trước".
5. **Given** ôn xong, **When** bước Ôn hoàn thành, **Then** bước Đọc mở và trở thành bước hiện tại.
6. **Given** đang ở bước Nghe câu 3/8, **When** đóng trình duyệt rồi mở lại "Bài hôm nay", **Then** vào thẳng bước Nghe câu 3.
7. **Given** đang đọc tới câu 10, **When** thoát rồi vào lại, **Then** bước Đọc mở ở đúng vị trí câu 10.
8. **Given** kiểm tra xong câu cuối của bước Nghe, **When** bước Nghe hoàn thành, **Then** bài hôm nay xong: thanh mục tiêu tăng 1
   bài, chuỗi ngày học tăng 1, và thấy "Đã xong bài hôm nay, hẹn bạn ngày mai".

---

### User Story 3 - Mỗi ngày một bài và chuỗi ngày học (Priority: P1)

Học xong bài hôm nay thì bài tiếp theo chỉ mở vào ngày hôm sau (0 giờ theo múi giờ của người học). Nghỉ một hay nhiều ngày thì
hôm quay lại vẫn chỉ có một bài và chuỗi ngày học (streak) về 0. Khi lộ trình chưa có bài tiếp theo, người học thấy "Chưa có
bài mới" nhưng vẫn ôn được hoặc chọn chủ đề khác.

**Why this priority**: Nhịp một bài mỗi ngày và streak là cách app giữ thói quen học.

**Independent Test**: Hoàn thành bài hôm nay lúc 23:50; tới 0:05 cùng đêm thấy bài mới; bỏ hai ngày rồi quay lại thấy đúng một
bài mới, streak 0 (thành 1 sau khi học xong).

**Acceptance Scenarios**:

1. **Given** đã xong bài hôm nay, **When** mở "Bài hôm nay" trong cùng ngày, **Then** thấy "Đã xong bài hôm nay", bài tiếp theo
   chưa mở, vẫn vào được ôn tự do và các bài đã học.
2. **Given** xong bài lúc 23:50 giờ của người học, **When** mở lại lúc 0:05, **Then** bài tiếp theo là bài hôm nay.
3. **Given** streak đang 5 và học đều mỗi ngày, **When** hôm nay học xong bài, **Then** streak thành 6; nếu hôm nay chưa học xong
   thì streak vẫn hiện 5.
4. **Given** streak 5, **When** nghỉ hai ngày rồi quay lại, **Then** chỉ có một bài (bài tiếp theo), streak hiện 0; học xong thì
   streak 1.
5. **Given** lộ trình không còn bài chưa học (quản trị viên chưa thêm), **When** mở "Bài hôm nay", **Then** thấy "Chưa có bài mới"
   kèm nút ôn tự do và nút chọn chủ đề khác; quản trị viên thấy cảnh báo ở F14.
6. **Given** hôm nay đã ôn 30 thẻ ở bước Ôn, **When** có thêm thẻ đến hạn trong ngày, **Then** chúng chờ tới hôm sau cho bước Ôn
   (ôn tự do vẫn dùng được).

---

### User Story 4 - Đổi chủ đề hoặc trình độ và quay lại (Priority: P2)

Người học đổi chủ đề hoặc trình độ bất cứ lúc nào. Nếu bài hôm nay chưa bắt đầu thì chủ đề mới có hiệu lực ngay; nếu đã bắt
đầu thì từ hôm sau. Tiến độ của mỗi lộ trình lưu riêng: quay lại chủ đề cũ thì học tiếp bài đang dở. Streak không bị ảnh hưởng.

**Why this priority**: Cần cho người học linh hoạt, nhưng luồng chính vẫn chạy với một mục tiêu.

**Independent Test**: Học 2 bài "A1 · Gia đình", đổi sang "A1 · Mua sắm" trước khi bắt đầu bài hôm nay → bài hôm nay là bài 1 của
Mua sắm; hôm sau quay lại Gia đình → bài hôm nay là bài 3 của Gia đình; streak không đổi.

**Acceptance Scenarios**:

1. **Given** bài hôm nay chưa bắt đầu bước nào, **When** đổi sang chủ đề khác, **Then** bài hôm nay đổi ngay thành bài tiếp theo
   của chủ đề mới.
2. **Given** đã xong bước Ôn của bài hôm nay, **When** đổi chủ đề, **Then** thấy "Chủ đề mới bắt đầu từ ngày mai"; hôm nay vẫn học
   tiếp bài đang dở; hôm sau bài hôm nay thuộc chủ đề mới.
3. **Given** đã học 2 bài của "A1 · Gia đình" rồi sang chủ đề khác, **When** quay lại "A1 · Gia đình", **Then** bài hôm nay là bài
   thứ 3 (bài chưa hoàn thành đầu tiên), tiến độ vẫn 2/N.
4. **Given** đổi sang một chủ đề B1 khi đang học A1, **When** bài hôm nay hiện ra, **Then** là bài của chủ đề B1; không có bài A1
   nào xen vào.
5. **Given** streak 4, **When** đổi chủ đề hoặc trình độ, **Then** streak vẫn 4.

---

### User Story 5 - Trang "Bài học": xem lại bài cũ, bài sắp tới bị khoá (Priority: P2)

Trang "Bài học" gồm bài hôm nay, các bài đã học (mở lại để đọc, nghe bất cứ lúc nào) và các bài sắp tới của lộ trình đang học ở
trạng thái khoá (chỉ hiện tên). Mở lại bài đã học không làm thay đổi tiến độ, streak hay mục tiêu.

**Why this priority**: Giúp ôn lại bài cũ, nhưng không cần cho luồng học hằng ngày.

**Independent Test**: Sau 3 bài, mở trang Bài học: thấy bài hôm nay, 3 bài đã học, các bài sắp tới có biểu tượng khoá; mở lại
bài đã học để nghe hết mà tiến độ và streak không đổi; mở thẳng địa chỉ của một bài sắp tới thì bị chặn.

**Acceptance Scenarios**:

1. **Given** đã học 3 bài, **When** mở trang "Bài học", **Then** thấy mục "Hôm nay", "Đã học" (3 bài, mới nhất trước) và "Sắp
   tới" (tên bài kèm biểu tượng khoá, theo thứ tự lộ trình).
2. **Given** một bài đã học, **When** mở lại để đọc hoặc nghe và làm hết, **Then** tiến độ, streak và mục tiêu không đổi và không
   có thông báo hoàn thành bước.
3. **Given** một bài sắp tới, **When** bấm vào hoặc gõ thẳng địa chỉ, **Then** không mở được; thấy "Bài này sẽ mở khi tới lượt".
4. **Given** bài đã học ở chủ đề cũ (đã đổi chủ đề), **When** mở trang "Bài học", **Then** vẫn thấy bài đó trong "Đã học".

---

### Edge Cases

- Đổi múi giờ tài khoản: ngày mới tính theo múi giờ mới từ lúc đổi; không tạo thêm bài trong cùng một ngày theo lịch cũ.
- Hai thiết bị cùng mở bài hôm nay: hoàn thành một bước ở thiết bị này thì thiết bị kia thấy bước đó đã xong khi tải lại; hoàn
  thành bước hai lần không cộng tiến độ hai lần.
- Quản trị viên đổi thứ tự hoặc thêm bài vào lộ trình giữa ngày: bài hôm nay đã bắt đầu thì giữ nguyên; chưa bắt đầu thì tính
  lại theo thứ tự mới. Bài thêm sau khi người học hết lộ trình trở thành bài tiếp theo.
- Quản trị viên gỡ hoặc xoá bài hôm nay khỏi lộ trình khi người học đang học dở: người học vẫn học xong bài đó hôm nay; bài đã
  hoàn thành vẫn ở "Đã học".
- Bài hôm nay chưa có audio (bước Nghe không làm được): hiện thông báo của F4 và người học quay lại sau; không tự bỏ qua bước.
- Hết thẻ trong phiên Ôn nhưng có thẻ chọn Again (đến hạn lại sau vài phút): bước Ôn hoàn thành khi phiên ôn kết thúc, không
  chờ thẻ đó.
- Chủ đề bị quản trị viên xoá (chỉ khi không còn bài) hoặc đổi trình độ: mục tiêu theo chủ đề đó hiện theo tên và trình độ mới;
  chủ đề không còn thì người học được mời chọn chủ đề khác.
- Mở lại bài đã học ở chế độ xem lại khi bài hôm nay là chính bài đó (vừa xong hôm nay): không đổi gì.
- Màn hình 360px, bàn phím, sáng và tối: chọn mục tiêu, bài hôm nay, trang Bài học dùng được, không cuộn ngang.

## Requirements *(mandatory)*

### Functional Requirements

**Mục tiêu**

- **FR-001**: Người học PHẢI đặt được mục tiêu bằng cách chọn trình độ A1–C2 rồi chọn một chủ đề của trình độ đó; danh sách
  chủ đề hiện số bài trong lộ trình và tiến độ đã học (nếu có).
- **FR-002**: Mỗi người học có tối đa một mục tiêu đang học; mục tiêu là hoàn thành lộ trình của chủ đề (theo thứ tự quản trị
  viên xếp, gồm cả bài được thêm sau).
- **FR-003**: Tiến độ mỗi lộ trình PHẢI lưu riêng; quay lại chủ đề cũ thì tiếp tục từ bài chưa hoàn thành đầu tiên.
- **FR-004**: Đổi chủ đề hoặc trình độ PHẢI có hiệu lực ngay nếu bài hôm nay chưa có bước nào hoàn thành; ngược lại có hiệu lực
  từ ngày hôm sau, kèm thông báo.
- **FR-005**: Hoàn thành bài cuối của lộ trình PHẢI hiện lời chúc mừng và mời chọn chủ đề khác cùng trình độ hoặc trình độ kế
  tiếp.

**Bài hôm nay**

- **FR-006**: Bài hôm nay PHẢI là bài chưa hoàn thành đầu tiên trong lộ trình của mục tiêu đang có hiệu lực; chỉ gồm bài của trình
  độ đang chọn.
- **FR-007**: Mỗi ngày (theo múi giờ của người học, sang ngày lúc 0 giờ) có tối đa một bài mới; học xong thì bài tiếp theo chỉ
  mở vào ngày hôm sau. Nghỉ nhiều ngày không dồn bài.
- **FR-008**: Bài có ba bước theo thứ tự Ôn → Đọc → Nghe; bước sau chỉ mở khi bước trước xong, kể cả khi mở bằng địa chỉ trực
  tiếp; hệ thống PHẢI từ chối ghi nhận hoàn thành một bước khi bước trước chưa xong.
- **FR-009**: Bước Ôn gồm các thẻ đến hạn, quá hạn lâu nhất trước, tối đa 30 thẻ mỗi ngày (trừ số thẻ đã ôn ở bước Ôn hôm nay);
  không có thẻ nào thì bước tự hoàn thành; thẻ vượt giới hạn chờ tới hôm sau. Bước Ôn hoàn thành khi phiên ôn kết thúc.
- **FR-010**: Bước Đọc hoàn thành khi bấm "Đã đọc xong" (F3); bước Nghe hoàn thành khi mọi câu đã kiểm tra (F4).
- **FR-011**: Hệ thống PHẢI lưu bước hiện tại và vị trí câu (bước Đọc, bước Nghe) để vào lại tiếp tục đúng chỗ; lưu trong vòng
  vài giây sau mỗi lần chuyển câu.
- **FR-012**: Hoàn thành bước cuối PHẢI đánh dấu bài hôm nay xong: tiến độ mục tiêu +1 bài, chuỗi ngày học +1; ghi nhận hoàn thành
  một bước nhiều lần không cộng nhiều lần.
- **FR-013**: Khi lộ trình chưa có bài tiếp theo, trang bài hôm nay PHẢI hiện "Chưa có bài mới" kèm lối vào ôn tự do và chọn chủ
  đề khác.
- **FR-014**: Trang bài hôm nay PHẢI hiện thanh bước (Ôn · Đọc · Nghe) với trạng thái xong / hiện tại / khoá phân biệt được không
  chỉ bằng màu, và số thẻ cần ôn.

**Chuỗi ngày học**

- **FR-015**: Streak là số ngày liên tiếp có hoàn thành bài, tính tới hôm nay nếu hôm nay đã xong, hoặc tới hôm qua nếu hôm nay
  chưa xong; bỏ lỡ một ngày trọn vẹn thì streak về 0.
- **FR-016**: Đổi chủ đề, đổi trình độ, mở lại bài cũ hay ôn tự do KHÔNG được làm thay đổi streak.

**Trang Bài học**

- **FR-017**: Trang "Bài học" PHẢI hiện bài hôm nay, các bài đã học (mọi chủ đề, mới nhất trước) và các bài sắp tới của lộ trình
  đang học (chỉ tên, trạng thái khoá).
- **FR-018**: Bài đã học PHẢI mở lại được để đọc và nghe ở chế độ xem lại; chế độ xem lại KHÔNG ghi nhận hoàn thành bước, không
  đổi tiến độ, streak hay mục tiêu.
- **FR-019**: Người học KHÔNG được mở nội dung bài chưa tới lượt (không phải bài hôm nay và chưa học), kể cả bằng địa chỉ trực
  tiếp; quản trị viên không bị giới hạn này.

**Chung**

- **FR-020**: Mục tiêu, tiến độ, vị trí và streak PHẢI tách riêng theo người học (nguyên tắc V).
- **FR-021**: Trang chủ tạm thời có nút "Học hôm nay"; header có liên kết "Bài học" và "Sổ từ" (thay đầy đủ ở F6).
- **FR-022**: Mọi trang của L PHẢI dùng được ở 360px, bằng bàn phím, sáng và tối (nguyên tắc IV).

### Key Entities

- **Mục tiêu (Goal)**: của một người học; chủ đề (và trình độ của chủ đề), trạng thái đang học / tạm dừng / đã hoàn thành, thời
  điểm bắt đầu, ngày bắt đầu có hiệu lực, thời điểm hoàn thành.
- **Tiến độ bài (Lesson progress)**: của một người học với một bài; ngày học (theo múi giờ), trạng thái từng bước, bước hiện tại,
  vị trí câu, thời điểm hoàn thành.
- **Ngày học (Study day)**: của một người học cho một ngày; bài của ngày, số thẻ đã ôn ở bước Ôn, đã hoàn thành bài hay chưa. Dùng
  cho streak và giới hạn thẻ.
- **Lộ trình (Roadmap, F14)**: danh sách bài có thứ tự của chủ đề; đọc trực tiếp, không sao chép vào mục tiêu.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% bài hôm nay thuộc trình độ và chủ đề của mục tiêu đang có hiệu lực (0 bài của trình độ khác).
- **SC-002**: Đổi chủ đề rồi quay lại: bài hôm nay là đúng bài chưa hoàn thành đầu tiên ở 100% lần thử; streak không đổi.
- **SC-003**: Mở lại bài đã học làm thay đổi tiến độ, streak hay mục tiêu ở 0% lần thử.
- **SC-004**: Bài sắp tới mở được trước ngày của nó ở 0% lần thử (kể cả địa chỉ trực tiếp).
- **SC-005**: Ranh giới ngày đúng theo múi giờ người học ở 100% ca thử sát nửa đêm và khi nghỉ 1, 2, 7 ngày (một bài, streak 0).
- **SC-006**: Bước Ôn không bao giờ quá 30 thẻ trong một ngày; thẻ dư xuất hiện ở bước Ôn hôm sau.
- **SC-007**: Vào lại sau khi thoát giữa chừng: đúng bước và đúng câu ở 100% lần thử.
- **SC-008**: Từ lúc mở app, người học vào được bước hiện tại của bài hôm nay trong tối đa 2 lần bấm.
- **SC-009**: Ở 360px, các trang của L không cuộn ngang và dùng được hoàn toàn bằng bàn phím.

## Assumptions

- Giới hạn 30 thẻ/ngày chỉ áp cho bước Ôn; ôn tự do (F5) không giới hạn và không tính vào giới hạn. Chỉnh giới hạn thuộc F12.
- "Bài hôm nay đã bắt đầu" nghĩa là đã hoàn thành ít nhất một bước (kể cả bước Ôn tự hoàn thành vì không có thẻ).
- Bài hôm nay được cố định khi đã bắt đầu; trước đó tính lại mỗi lần mở theo lộ trình hiện tại.
- Streak chỉ tính ngày hoàn thành bài mới; ngày chỉ ôn tự do hoặc xem lại không tính.
- Xem lại bài: bước Nghe ở chế độ xem lại vẫn kiểm tra được từng câu nhưng không lưu kết quả; bước Đọc vẫn tra và lưu từ vào sổ
  được (sổ từ không thuộc tiến độ).
- Bài đã hoàn thành trước khi có L (chỉ có kết quả F4) không tính là đã học; tiến độ bắt đầu từ L.
- Múi giờ lấy từ tài khoản (F1); đổi múi giờ thuộc F12.
- Ngoài phạm vi: bước Viết và Nói (giai đoạn 2, 3), màn hình chính đầy đủ, thanh kỹ năng và thống kê (F6), chỉnh giới hạn thẻ và
  múi giờ (F12), nhắc học (thông báo).
