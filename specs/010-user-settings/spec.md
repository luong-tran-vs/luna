# Feature Specification: Cài đặt

**Feature Branch**: `010-user-settings`

**Created**: 2026-09-30

**Amended**: 2026-09-30: chỉ còn hai chế độ Sáng / Tối; khi chưa chọn thì theo thiết bị (US1, FR-001, FR-002). Giá trị `system` vẫn hợp lệ ở API, nghĩa là "chưa chọn".

**Status**: Draft

**Mã tính năng**: F12 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`, `docs/design-system.md` mục 2)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f12-cai-dat.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Chọn giao diện Sáng hoặc Tối (Priority: P1)

Người học mở trang Cài đặt (hoặc menu giao diện trên thanh trên cùng), chọn Sáng hoặc Tối. Giao diện đổi ngay, không tải lại trang. Lựa chọn được lưu vào
tài khoản, nên đăng nhập trên thiết bị khác cũng thấy đúng chế độ đã chọn.

**Why this priority**: Là tiêu chí nghiệm thu đầu tiên của F12 và ảnh hưởng mọi màn hình (đọc lâu buổi tối cần chế độ tối).

**Independent Test**: Trên máy A chọn "Tối": trang đổi sang tối ngay. Đăng nhập cùng tài khoản trên máy B (trình duyệt đang
để "Sáng"): sau khi đăng nhập giao diện là tối.

**Acceptance Scenarios**:

1. **Given** giao diện đang Sáng, **When** chọn "Tối" trong Cài đặt, **Then** toàn bộ app chuyển sang tối ngay, không tải lại trang
   và không mất nội dung đang nhập.
2. **Given** chưa chọn chế độ lần nào và thiết bị đang ở chế độ tối, **When** thiết bị chuyển sang sáng, **Then** app chuyển sang sáng
   theo, không cần tải lại.
3. **Given** đã chọn "Tối" trên máy A, **When** đăng nhập cùng tài khoản trên máy B, **Then** app ở chế độ tối.
4. **Given** chưa đăng nhập, **When** mở trang đăng nhập, **Then** app dùng lựa chọn giao diện đã lưu trên trình duyệt đó (như F0),
   không nhấp nháy sang chế độ khác khi tải trang.
5. **Given** ở cả hai chế độ, **When** xem các trang, **Then** màu và font đúng bảng màu và font của design system.

---

### User Story 2 - Đặt số thẻ ôn tối đa mỗi ngày (Priority: P2)

Người học đặt số thẻ ôn tối đa mỗi ngày trong bước Ôn của bài hôm nay, từ 5 đến 200 (mặc định 30).

**Why this priority**: Giúp điều chỉnh nhịp học; 30 thẻ có thể quá nhiều hoặc quá ít với từng người.

**Independent Test**: Đặt giới hạn 10 khi có 25 thẻ đến hạn và chưa ôn thẻ nào hôm nay: mở Bài hôm nay, bước Ôn có 10 thẻ.

**Acceptance Scenarios**:

1. **Given** giới hạn mặc định, **When** mở Cài đặt, **Then** thấy 30.
2. **Given** nhập 10 và lưu, **When** mở Bài hôm nay với 25 thẻ đến hạn, **Then** bước Ôn có 10 thẻ.
3. **Given** nhập 4, 201, ô trống hoặc chữ, **When** lưu, **Then** thấy lỗi "Số thẻ từ 5 đến 200" ngay dưới ô, giá trị cũ giữ nguyên.
4. **Given** hôm nay đã ôn 20 thẻ ở bước Ôn, **When** hạ giới hạn xuống 10 rồi mở Bài hôm nay, **Then** bước Ôn không còn thẻ phải
   ôn hôm nay; thẻ đã ôn vẫn giữ kết quả.
5. **Given** hôm nay đã ôn 30 thẻ, **When** nâng giới hạn lên 50 rồi mở Bài hôm nay, **Then** còn tối đa 20 thẻ nữa để ôn.

---

### User Story 3 - Đổi múi giờ tính ngày học (Priority: P2)

Người học chọn múi giờ dùng để tính ngày học (lúc nào sang ngày mới, streak, thẻ đến hạn ngày mai) từ danh sách múi giờ. Mặc định
là múi giờ lưu lúc đăng ký.

**Why this priority**: Người học đi nước khác hoặc đăng ký nhầm múi giờ sẽ bị sang ngày sai giờ.

**Independent Test**: Tài khoản đang ở Việt Nam, 23:30 giờ Việt Nam đã xong bài hôm nay. Đổi sang "Europe/London" (17:30 cùng
ngày): màn hình chính vẫn "Đã xong bài hôm nay", streak không đổi.

**Acceptance Scenarios**:

1. **Given** đăng ký với múi giờ Asia/Ho_Chi_Minh, **When** mở Cài đặt, **Then** múi giờ đang chọn là Asia/Ho_Chi_Minh.
2. **Given** danh sách múi giờ, **When** gõ "London" vào ô tìm, **Then** thấy Europe/London; chọn và lưu thì từ lần mở Bài hôm nay
   tiếp theo, ngày học tính theo giờ London.
3. **Given** đang học dở bài hôm nay ở bước Đọc, **When** đổi múi giờ mà theo múi giờ mới vẫn là cùng ngày, **Then** Bài hôm nay
   vẫn là bài đó, bước Ôn vẫn xong, Đọc mở ở đúng câu đang đọc.
4. **Given** đổi sang múi giờ mà theo đó đã sang ngày mới, **When** mở Bài hôm nay, **Then** bước đã xong của bài đang dở vẫn giữ;
   bài đó tiếp tục là bài hôm nay (chưa xong thì chưa có bài mới), các ngày đã học trước đó không mất.
5. **Given** gửi múi giờ không có trong danh sách chuẩn, **When** lưu, **Then** bị từ chối với lỗi "Múi giờ không hợp lệ", múi giờ cũ
   giữ nguyên.

---

### Edge Cases

- Mở Cài đặt khi mất kết nối: thấy lỗi tải và nút thử lại; giao diện vẫn đổi được trên trình duyệt (lưu vào tài khoản khi lưu lại
  được).
- Lưu thất bại (mất mạng): báo lỗi, không báo "Đã lưu"; giá trị trên màn hình giữ như người học vừa chọn để lưu lại.
- Trình duyệt chặn lưu trữ cục bộ (chế độ riêng tư): chọn giao diện vẫn có hiệu lực trong phiên và vẫn lưu vào tài khoản.
- Đăng xuất: trình duyệt giữ lựa chọn giao diện cuối cùng cho màn hình đăng nhập.
- Máy B đang có lựa chọn cục bộ khác tài khoản: sau khi đăng nhập, lựa chọn của tài khoản thắng và ghi đè lựa chọn cục bộ.
- Đổi múi giờ nhiều lần trong ngày: mỗi lần chỉ đổi cách tính ngày từ lúc đó; không tạo thêm bài mới trong cùng một ngày thực.
- Đổi múi giờ sang ngày trước (ví dụ từ châu Á sang châu Mỹ sau nửa đêm): ngày đã xong theo múi giờ cũ vẫn tính là đã xong, không
  phải học lại bài đó.
- Giới hạn thẻ hạ xuống thấp hơn số thẻ đã ôn hôm nay: bước Ôn coi như đủ, không báo lỗi.
- Nút chọn giao diện nhanh đã có ở thanh trên (F0) và trang Cài đặt luôn khớp nhau; đổi ở đâu cũng lưu vào tài khoản.
- Quản trị viên có trang Cài đặt như người học.

## Requirements *(mandatory)*

### Functional Requirements

**Giao diện**

- **FR-001**: Người học PHẢI chọn được một trong 2 chế độ giao diện: **Sáng**, **Tối**. Khi chưa chọn, app theo chế độ của thiết bị.
- **FR-002**: Đổi chế độ PHẢI có hiệu lực ngay trên mọi phần của app, không tải lại trang; khi chưa chọn, app PHẢI đổi theo thiết bị
  khi thiết bị đổi chế độ.
- **FR-003**: Lựa chọn giao diện PHẢI lưu vào tài khoản và vào trình duyệt; sau khi đăng nhập, lựa chọn của tài khoản được dùng và
  ghi đè lựa chọn cục bộ.
- **FR-004**: Trước khi đăng nhập, app PHẢI dùng lựa chọn đã lưu trên trình duyệt, áp dụng ngay khi tải trang (không nhấp nháy).
- **FR-005**: Màu và font PHẢI đúng design system ở cả chế độ sáng và tối.

**Học tập**

- **FR-006**: Người học PHẢI đặt được số thẻ ôn tối đa mỗi ngày của bước Ôn, số nguyên từ 5 đến 200, mặc định 30; giá trị ngoài
  khoảng bị từ chối với thông báo tiếng Việt.
- **FR-007**: Giới hạn mới PHẢI có hiệu lực từ lần mở Bài hôm nay tiếp theo, tính cả số thẻ đã ôn ở bước Ôn trong ngày (còn lại =
  giới hạn − đã ôn, không âm); thẻ đã ôn giữ nguyên kết quả.

**Múi giờ**

- **FR-008**: Người học PHẢI chọn được múi giờ tính ngày học từ danh sách múi giờ chuẩn, có ô tìm; mặc định là múi giờ lưu lúc đăng
  ký.
- **FR-009**: Múi giờ không có trong danh sách chuẩn PHẢI bị từ chối với thông báo tiếng Việt.
- **FR-010**: Múi giờ mới PHẢI có hiệu lực từ lần mở Bài hôm nay (và màn hình chính) tiếp theo, dùng cho: lúc sang ngày mới, streak,
  thẻ đến hạn, giới hạn thẻ trong ngày.
- **FR-011**: Đổi múi giờ KHÔNG ĐƯỢC làm mất tiến độ: bước đã xong, vị trí đang đọc/nghe, ngày đã học và thẻ đã ôn giữ nguyên; bài
  đang học dở tiếp tục là bài hôm nay cho tới khi xong.

**Chung**

- **FR-012**: Cài đặt PHẢI lưu theo tài khoản; đăng nhập trên thiết bị khác thấy đúng cả ba cài đặt.
- **FR-013**: Trang Cài đặt PHẢI mở được từ thanh trên, chia nhóm Giao diện, Học tập, Múi giờ; lưu thành công báo "Đã lưu".
- **FR-014**: Mỗi người học chỉ đọc và sửa được cài đặt của chính mình (nguyên tắc V).
- **FR-015**: Trang Cài đặt PHẢI dùng được ở màn hình 360px, bằng bàn phím và trình đọc màn hình (nhãn cho mọi ô, lỗi đọc được).

### Key Entities

- **Cài đặt (Settings)**: thuộc một tài khoản; gồm chế độ giao diện (sáng | tối | chưa chọn = theo thiết bị), số thẻ ôn tối đa mỗi ngày (5–200)
  và múi giờ (tên múi giờ chuẩn). Tài khoản cũ chưa có cài đặt dùng giá trị mặc định và múi giờ lúc đăng ký.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Đổi chế độ giao diện thấy hiệu lực trong dưới 0,5 giây, 0 lần tải lại trang.
- **SC-002**: 100% lần đăng nhập trên thiết bị khác thấy đúng cả ba cài đặt đã lưu.
- **SC-003**: 100% giá trị không hợp lệ (giới hạn ngoài 5–200, múi giờ lạ) bị từ chối với thông báo tiếng Việt, cài đặt cũ giữ
  nguyên.
- **SC-004**: Đổi múi giờ giữa ngày không làm mất bước đã xong, vị trí đang học hay ngày đã học trong 100% ca thử (cùng ngày, sang
  ngày sau, lùi ngày trước).
- **SC-005**: Người học đổi được một cài đặt trong không quá 3 thao tác từ màn hình chính.
- **SC-006**: Mọi trang đạt tương phản chữ WCAG AA ở cả hai chế độ, font đúng design system.

## Assumptions

- Giới hạn thẻ chỉ áp dụng cho bước Ôn của Bài hôm nay; Ôn tự do không giới hạn (như F5).
- "Có hiệu lực từ lần mở Bài hôm nay tiếp theo": trang đang mở không tự đổi; mở lại (hoặc quay về màn hình chính) thì dùng cài đặt
  mới.
- Nút chọn giao diện nhanh ở thanh trên (F0) được giữ; khi đã đăng nhập, đổi ở đó cũng lưu vào tài khoản.
- Danh sách múi giờ là danh sách múi giờ chuẩn mà trình duyệt cung cấp; tên hiển thị kèm giờ lệch so với UTC để dễ chọn.
- Tài khoản đã có trước F12 được coi như có cài đặt mặc định (chưa chọn giao diện, 30 thẻ) và múi giờ đã lưu lúc đăng ký.
- Ngoài phạm vi: đổi email, đổi mật khẩu, xuất dữ liệu (F13), cài đặt nhắc học, ngôn ngữ giao diện.
