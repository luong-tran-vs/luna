# Feature Specification: Màn hình chính và tiến độ

**Feature Branch**: `009-home-dashboard`

**Created**: 2026-09-30

**Status**: Draft

**Mã tính năng**: F6 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`, `docs/mvp-features.md` mục 4.1)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f6-man-hinh-chinh.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Mở app là biết hôm nay làm gì (Priority: P1)

Người học mở app và thấy ngay trên màn hình đầu: thanh mục tiêu (tên lộ trình, ví dụ "A1 · Gia đình", và số bài đã xong trên
tổng số bài), bài hôm nay (tên bài, trình độ, chủ đề, thanh tiến trình các bước) và nút **Tiếp tục: <bước>** đưa thẳng tới bước
đang dở. Kèm theo là chuỗi ngày học (streak) và số thẻ đến hạn ngày mai.

**Why this priority**: Là mục tiêu của F6 và trải nghiệm đầu tiên mỗi lần mở app (mục 4.1 của `mvp-features.md`).

**Independent Test**: Người học đang học "A1 · Gia đình" (2/12 bài, streak 4), bài hôm nay đã xong bước Ôn: mở app trên màn
360px, không cuộn vẫn thấy mục tiêu "2/12 bài", bài hôm nay, thanh bước và nút "Tiếp tục: Đọc"; bấm nút vào thẳng bước Đọc.

**Acceptance Scenarios**:

1. **Given** người học có mục tiêu "A1 · Gia đình" đã xong 2/12 bài, **When** mở app, **Then** thấy thanh mục tiêu ghi "A1 · Gia
   đình" và "2/12 bài", có nút đổi chủ đề.
2. **Given** bài hôm nay "At the café" đã xong bước Ôn, **When** mở app, **Then** thấy tên bài, trình độ, chủ đề, thanh bước (Ôn
   xong, Đọc hiện tại, Nghe khoá) và nút "Tiếp tục: Đọc".
3. **Given** nút "Tiếp tục: Đọc", **When** bấm, **Then** mở đúng bước Đọc của bài hôm nay ở vị trí đang đọc dở.
4. **Given** bài hôm nay chưa bắt đầu, **When** mở app, **Then** nút là "Bắt đầu: Ôn" (hoặc "Bắt đầu: Đọc" khi không có thẻ cần ôn).
5. **Given** streak 4 ngày và 17 thẻ đến hạn ngày mai, **When** mở app, **Then** thấy "🔥 4 ngày" và "Ngày mai: 17 thẻ cần ôn".
6. **Given** điện thoại 360px ở cả chế độ sáng và tối, **When** mở app, **Then** thanh mục tiêu, bài hôm nay và nút Tiếp tục nằm
   trong màn hình đầu, không cần cuộn.

---

### User Story 2 - Trạng thái đặc biệt (Priority: P1)

Màn hình chính cho biết rõ khi người học chưa có mục tiêu, đã xong bài hôm nay, hoặc lộ trình chưa có bài mới, và chỉ ra việc nên
làm tiếp.

**Why this priority**: Không xử lý các trạng thái này thì người học mới hoặc người đã xong bài không biết làm gì.

**Independent Test**: Lần lượt với tài khoản mới, tài khoản vừa xong bài hôm nay, và tài khoản đã hết lộ trình: mở app thấy đúng
thông báo và nút tương ứng.

**Acceptance Scenarios**:

1. **Given** người học chưa chọn chủ đề, **When** mở app, **Then** thấy lời mời chọn mục tiêu và nút "Chọn chủ đề"; không có thanh
   mục tiêu hay bài hôm nay.
2. **Given** đã xong bài hôm nay, **When** mở app, **Then** thấy "Đã xong bài hôm nay, hẹn bạn ngày mai", thanh bước đầy đủ, nút
   "Ôn tự do" và số thẻ đến hạn ngày mai.
3. **Given** lộ trình chưa có bài tiếp theo, **When** mở app, **Then** thấy "Chưa có bài mới" kèm nút "Ôn tự do" và "Chọn chủ đề
   khác".
4. **Given** máy chủ hoặc cơ sở dữ liệu không kết nối được, **When** mở app, **Then** thấy thông báo lỗi kết nối ở chân trang; khi
   kết nối bình thường thì không có dòng trạng thái kết nối nào.

---

### User Story 3 - Tiến độ theo kỹ năng và số liệu cập nhật ngay (Priority: P2)

Dưới thanh mục tiêu có thanh kỹ năng Nghe và Đọc: mỗi thanh đếm số bài của lộ trình đã xong bước của kỹ năng đó. Mọi số liệu trên
màn hình chính cập nhật ngay khi người học hoàn thành một bước rồi quay về.

**Why this priority**: Giúp thấy kỹ năng nào đang bị bỏ lại; cần nhưng không chặn luồng học.

**Independent Test**: Hoàn thành bước Đọc của bài hôm nay rồi quay về màn hình chính: thanh Đọc tăng 1, nút đổi thành "Tiếp tục:
Nghe"; hoàn thành Nghe: thanh Nghe và thanh mục tiêu tăng 1, streak tăng 1.

**Acceptance Scenarios**:

1. **Given** lộ trình 12 bài, đã xong bước Đọc của 3 bài và bước Nghe của 2 bài, **When** mở app, **Then** thấy "Đọc 3/12" và
   "Nghe 2/12".
2. **Given** vừa hoàn thành một bước, **When** quay về màn hình chính, **Then** thanh kỹ năng, thanh mục tiêu, thanh bước, streak và
   nút Tiếp tục phản ánh ngay bước vừa xong, không cần tải lại trang.
3. **Given** giai đoạn 1, **When** xem thanh kỹ năng, **Then** chỉ có Nghe và Đọc (Viết, Nói chưa hiện).

---

### User Story 4 - Trang thống kê (Priority: P3)

Trang thống kê cho người học xem: số từ đã học, số câu đã chép chính tả, tỷ lệ đúng khi chép chính tả, số bài hoàn thành theo kỹ
năng.

**Why this priority**: Hữu ích để nhìn lại, không cần cho việc học hằng ngày.

**Independent Test**: Người học có 25 thẻ, đã chép 40 câu với tỷ lệ đúng 82%, xong 5 bài: mở trang thống kê thấy đúng các số đó.

**Acceptance Scenarios**:

1. **Given** sổ từ có 25 thẻ, **When** mở trang thống kê, **Then** thấy "25 từ đã học".
2. **Given** đã chép 40 câu với tổng 400 từ, 328 từ đúng, **When** mở trang thống kê, **Then** thấy "40 câu đã chép chính tả" và
   "Tỷ lệ đúng 82%".
3. **Given** đã xong bước Đọc của 6 bài, bước Nghe của 5 bài, 5 bài hoàn thành, **When** mở trang thống kê, **Then** thấy số bài
   theo từng kỹ năng và tổng số bài hoàn thành.
4. **Given** người học mới chưa học gì, **When** mở trang thống kê, **Then** thấy các số 0 và tỷ lệ "—", không có lỗi.

---

### Edge Cases

- Lộ trình rỗng (tổng 0 bài): thanh mục tiêu hiện "0/0 bài" không chia cho 0, thanh trống.
- Bài hôm nay chưa có tên (bài bị xoá sau khi học): hiện "Bài học" thay cho tên.
- Số liệu kỹ năng tính theo lộ trình hiện tại: bài đã học của chủ đề cũ không tính vào thanh của mục tiêu mới (vẫn có ở thống kê).
- Không có thẻ đến hạn ngày mai: hiện "Ngày mai: không có thẻ cần ôn".
- Quay về màn hình chính bằng nút Back của trình duyệt: số liệu vẫn được tải lại.
- Nhiều thẻ quá hạn: số thẻ ngày mai gồm cả thẻ còn nợ hôm nay (tối đa hiển thị số thật, bước Ôn vẫn giới hạn 30).
- Tên chủ đề hoặc tên bài dài: xuống dòng, không đẩy nút Tiếp tục ra khỏi màn hình đầu ở 360px.
- Quản trị viên mở màn hình chính: thấy như người học (tiến độ của chính mình).

## Requirements *(mandatory)*

### Functional Requirements

**Màn hình chính**

- **FR-001**: Màn hình chính PHẢI hiện thanh mục tiêu gồm tên lộ trình ("<trình độ> · <chủ đề>"), số bài đã xong / tổng số bài
  của lộ trình và nút đổi chủ đề.
- **FR-002**: Màn hình chính PHẢI hiện thanh kỹ năng Nghe và Đọc, mỗi thanh là số bài của lộ trình hiện tại đã xong bước của kỹ
  năng đó / tổng số bài; không hiện Viết và Nói ở giai đoạn 1.
- **FR-003**: Màn hình chính PHẢI hiện bài hôm nay: tên bài, trình độ, chủ đề, thanh tiến trình các bước (Ôn · Đọc · Nghe, trạng
  thái xong / hiện tại / khoá phân biệt được không chỉ bằng màu).
- **FR-004**: Nút **Tiếp tục: <bước>** (hoặc **Bắt đầu: <bước>** khi chưa có bước nào xong) PHẢI đưa thẳng tới bước đang dở của bài
  hôm nay, ở vị trí đã lưu.
- **FR-005**: Màn hình chính PHẢI hiện streak và số thẻ đến hạn tính tới hết ngày mai theo múi giờ người học.
- **FR-006**: Khi chưa có mục tiêu PHẢI hiện lời mời và nút "Chọn chủ đề"; khi đã xong bài hôm nay PHẢI hiện "Đã xong bài hôm nay,
  hẹn bạn ngày mai" và nút "Ôn tự do"; khi chưa có bài mới PHẢI hiện "Chưa có bài mới" với nút "Ôn tự do" và "Chọn chủ đề khác".
- **FR-007**: Trên màn hình 360px, thanh mục tiêu, bài hôm nay và nút Tiếp tục PHẢI nằm trong màn hình đầu (không cần cuộn), ở cả
  chế độ sáng và tối.
- **FR-008**: Số liệu trên màn hình chính PHẢI được tải lại mỗi lần người học quay về, nên phản ánh ngay bước vừa hoàn thành.
- **FR-009**: Trạng thái kết nối (F0) PHẢI chuyển xuống chân trang và chỉ hiện khi có lỗi kết nối.

**Thống kê**

- **FR-010**: Trang thống kê PHẢI hiện: số từ đã học (số thẻ trong sổ từ), số câu đã chép chính tả, tỷ lệ đúng (tổng số từ đúng /
  tổng số từ của các câu đã chép), số bài đã xong bước Đọc, số bài đã xong bước Nghe và số bài hoàn thành.
- **FR-011**: Trang thống kê PHẢI mở được từ màn hình chính; người học chưa có dữ liệu thấy số 0 và tỷ lệ "—".

**Chung**

- **FR-012**: Mọi số liệu là của người học đang đăng nhập (nguyên tắc V).
- **FR-013**: Các trang của F6 PHẢI dùng được bằng bàn phím và trình đọc màn hình: thanh tiến độ có nhãn và giá trị đọc được.

### Key Entities

- **Tổng quan màn hình chính (Dashboard)**: mục tiêu (tên lộ trình, số bài xong, tổng số bài), tiến độ theo kỹ năng (Đọc, Nghe),
  bài hôm nay và các bước (từ L), streak, số thẻ đến hạn tới hết ngày mai. Tính khi xem, không lưu riêng.
- **Thống kê (Stats)**: số thẻ, số câu đã chép chính tả, tỷ lệ đúng, số bài theo kỹ năng, số bài hoàn thành. Tính khi xem.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Ở màn hình 360 × 640, 100% trạng thái "đang học" hiện thanh mục tiêu, bài hôm nay và nút Tiếp tục mà không cần cuộn.
- **SC-002**: Bấm nút Tiếp tục mở đúng bước đang dở ở 100% lần thử (sau mỗi bước Ôn, Đọc, Nghe).
- **SC-003**: Sau khi hoàn thành một bước, quay về màn hình chính thấy số liệu mới ở 100% lần thử, không cần tải lại trang.
- **SC-004**: Màn hình chính hiện đầy đủ trong dưới 1 giây sau khi mở app (dữ liệu vài trăm bài, vài nghìn thẻ).
- **SC-005**: Số liệu thống kê khớp dữ liệu thật (đếm tay) ở 100% ca thử, kể cả người học mới.
- **SC-006**: 4 trạng thái (chưa có mục tiêu, đang học, xong hôm nay, chưa có bài mới) đều có thông báo và nút hành động đúng.

## Assumptions

- Thanh kỹ năng tính trong lộ trình hiện tại (để so với tổng số bài của lộ trình); thống kê tính trên mọi bài đã học.
- "Số thẻ đến hạn ngày mai" = số thẻ có hạn ôn trước 0 giờ ngày kia (gồm cả thẻ còn nợ hôm nay), để người học biết khối lượng ôn
  ngày mai; bước Ôn vẫn giới hạn 30 thẻ/ngày.
- "Số câu đã chép chính tả" đếm kết quả hiện có của các câu (mỗi câu tính một lần, lần kiểm tra mới nhất); tỷ lệ đúng tính trên
  các kết quả đó.
- Màn hình chính thay trang chủ tạm của F0; nút "Học hôm nay" tạm của L được thay bởi nút Tiếp tục.
- Thanh Viết và Nói thêm ở giai đoạn 2 và 3.
- Ngoài phạm vi: biểu đồ theo thời gian, bảng xếp hạng, mục tiêu hằng tuần, nhắc học.
