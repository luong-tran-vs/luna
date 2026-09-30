# Feature Specification: Tài khoản

**Feature Branch**: `002-user-accounts`

**Created**: 2026-09-29

**Amended**: 2026-09-30: nút Đăng xuất nằm cạnh nút giao diện ở góc phải; hàng liên kết cuộn ngang khi không đủ chỗ (FR-017).

**Status**: Draft

**Mã tính năng**: F1 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f1-tai-khoan.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đăng ký tài khoản (Priority: P1)

Người mới mở Luna, chưa có tài khoản. Họ được đưa tới trang đăng nhập, bấm liên kết sang trang đăng
ký, nhập email và mật khẩu. Đăng ký thành công thì họ được đăng nhập luôn và vào trang chủ. Tài khoản
đầu tiên của hệ thống là quản trị viên; các tài khoản sau là người học. Múi giờ của trình duyệt được
lưu vào tài khoản.

**Why this priority**: Không có tài khoản thì không có gì để đăng nhập hay phân quyền; đây cũng là cách
duy nhất để tạo quản trị viên.

**Independent Test**: Trên hệ thống trống, đăng ký một tài khoản và xác nhận đã vào trang chủ với vai
trò quản trị viên; đăng ký tài khoản thứ hai và xác nhận vai trò người học.

**Acceptance Scenarios**:

1. **Given** hệ thống chưa có tài khoản nào, **When** đăng ký bằng email và mật khẩu hợp lệ, **Then**
   tài khoản được tạo với vai trò quản trị viên, người dùng được đăng nhập và thấy trang chủ.
2. **Given** hệ thống đã có ít nhất một tài khoản, **When** một người khác đăng ký, **Then** tài khoản
   mới có vai trò người học.
3. **Given** email `An@Example.com` đã đăng ký, **When** đăng ký với `an@example.com`, **Then** bị từ
   chối với thông báo email đã được dùng.
4. **Given** đang ở trang đăng ký, **When** nhập mật khẩu dưới 8 ký tự hoặc email sai định dạng,
   **Then** thấy thông báo lỗi tiếng Việt ngay cạnh ô nhập và tài khoản không được tạo.
5. **Given** trình duyệt đặt múi giờ `Asia/Ho_Chi_Minh`, **When** đăng ký, **Then** tài khoản lưu múi
   giờ `Asia/Ho_Chi_Minh`.

---

### User Story 2 - Đăng nhập, giữ đăng nhập và đăng xuất (Priority: P1)

Người đã có tài khoản đăng nhập bằng email và mật khẩu, và được giữ đăng nhập trên thiết bị đó trong 30
ngày. Họ đăng xuất được bất cứ lúc nào. Nếu phiên hết hạn giữa chừng, họ được đưa về trang đăng nhập
và sau khi đăng nhập lại thì quay về đúng trang đang xem.

**Why this priority**: Là lối vào hằng ngày của người học; cùng với US1 tạo thành luồng tài khoản tối
thiểu.

**Independent Test**: Đăng nhập, đóng và mở lại trình duyệt vẫn còn đăng nhập; đăng xuất thì về trang
đăng nhập; làm hết hạn phiên rồi thao tác và kiểm tra luồng quay lại.

**Acceptance Scenarios**:

1. **Given** có tài khoản, **When** đăng nhập đúng email (không phân biệt hoa thường) và mật khẩu,
   **Then** vào trang chủ; thanh trên cùng hiện email của người dùng và nút Đăng xuất.
2. **Given** đã đăng nhập, **When** đóng trình duyệt rồi mở lại trong vòng 30 ngày, **Then** vẫn đang
   đăng nhập.
3. **Given** đã đăng nhập, **When** bấm Đăng xuất, **Then** về trang đăng nhập và phiên đó không dùng
   lại được nữa (kể cả trên tab khác).
4. **Given** chưa đăng nhập, **When** mở bất kỳ trang nào ngoài đăng nhập và đăng ký, **Then** được đưa
   tới trang đăng nhập.
5. **Given** đang xem một trang và phiên vừa hết hạn, **When** thao tác cần máy chủ, **Then** được đưa
   tới trang đăng nhập; đăng nhập lại thì quay về đúng trang đó.
6. **Given** đã đăng nhập, **When** mở trang đăng nhập hoặc đăng ký, **Then** được đưa về trang chủ.

---

### User Story 3 - Chống đoán mật khẩu (Priority: P2)

Đăng nhập sai không tiết lộ sai email hay sai mật khẩu. Sai 5 lần liên tiếp cho cùng một email thì tạm
khoá đăng nhập email đó 15 phút.

**Why this priority**: Bảo vệ tài khoản (nguyên tắc V) nhưng không chặn luồng chính nếu làm sau US1–US2.

**Independent Test**: Đăng nhập sai email, sai mật khẩu và so sánh thông báo; sai liên tiếp 5 lần rồi
thử đúng mật khẩu và xác nhận vẫn bị khoá cho tới khi hết 15 phút.

**Acceptance Scenarios**:

1. **Given** email không tồn tại, hoặc email đúng nhưng mật khẩu sai, **When** đăng nhập, **Then** cả
   hai trường hợp đều nhận cùng một thông báo "Email hoặc mật khẩu không đúng".
2. **Given** đã sai 4 lần liên tiếp, **When** sai lần thứ 5, **Then** email đó bị tạm khoá đăng nhập 15
   phút và người dùng thấy thông báo tạm khoá kèm thời gian chờ.
3. **Given** email đang bị tạm khoá, **When** nhập đúng mật khẩu, **Then** vẫn bị từ chối với thông báo
   tạm khoá.
4. **Given** đã hết 15 phút, **When** đăng nhập đúng, **Then** thành công.
5. **Given** đã sai 3 lần, **When** đăng nhập đúng, **Then** thành công và bộ đếm lần sai về 0.

---

### User Story 4 - Chỉ quản trị viên vào trang quản trị (Priority: P2)

Quản trị viên thấy liên kết "Quản trị" và vào được trang quản trị (F1 chỉ có trang tạm; nội dung thật
làm ở F2). Người học không thấy liên kết; nếu tự gõ địa chỉ hoặc gọi thẳng máy chủ thì bị chặn và thấy
thông báo không có quyền.

**Why this priority**: Là tiêu chí nghiệm thu F1 và là nền cho F2, nhưng chưa có nội dung quản trị thật.

**Independent Test**: Đăng nhập bằng người học, mở trực tiếp địa chỉ trang quản trị và gọi thẳng dịch
vụ quản trị ở máy chủ; cả hai đều bị từ chối. Đăng nhập bằng quản trị viên thì vào được.

**Acceptance Scenarios**:

1. **Given** đăng nhập là quản trị viên, **When** bấm "Quản trị", **Then** vào trang quản trị tạm.
2. **Given** đăng nhập là người học, **When** xem thanh trên cùng, **Then** không có liên kết "Quản trị".
3. **Given** đăng nhập là người học, **When** gõ địa chỉ trang quản trị, **Then** không vào được và
   thấy thông báo "Bạn không có quyền truy cập trang này".
4. **Given** đăng nhập là người học, **When** gọi thẳng dịch vụ quản trị ở máy chủ (bỏ qua giao diện),
   **Then** máy chủ từ chối vì không có quyền.
5. **Given** chưa đăng nhập, **When** gọi thẳng dịch vụ quản trị, **Then** máy chủ từ chối vì chưa đăng
   nhập.

---

### Edge Cases

- Email có khoảng trắng đầu/cuối hoặc chữ hoa: được chuẩn hoá (bỏ khoảng trắng, chữ thường) trước khi
  lưu và so khớp.
- Hai người đăng ký cùng email gần như đồng thời: chỉ một tài khoản được tạo.
- Mật khẩu rất dài: chấp nhận tới 128 ký tự; dài hơn thì báo lỗi. Mật khẩu được phép chứa khoảng trắng
  và ký tự tiếng Việt; không bị cắt bớt.
- Trình duyệt không cung cấp múi giờ hợp lệ: dùng `Asia/Ho_Chi_Minh`.
- Tạm khoá áp dụng cả với email chưa đăng ký, để thông báo không tiết lộ email nào tồn tại.
- Máy chủ khởi động lại: các phiên đăng nhập vẫn còn hiệu lực; bộ đếm lần sai có thể bị xoá (chấp nhận
  được với dự án một người dùng).
- Mất kết nối máy chủ khi đang đăng nhập hoặc đăng ký: hiện thông báo lỗi kết nối, giữ nguyên dữ liệu
  đã nhập (trừ mật khẩu không cần giữ).
- Địa chỉ quay về sau đăng nhập trỏ ra ngoài app: bỏ qua, về trang chủ.
- Người dùng bị đưa về đăng nhập khi đang ở trang đăng nhập: không tạo vòng lặp chuyển trang.

## Requirements *(mandatory)*

### Functional Requirements

**Đăng ký**

- **FR-001**: Người chưa đăng nhập PHẢI đăng ký được bằng email và mật khẩu.
- **FR-002**: Email PHẢI được chuẩn hoá (bỏ khoảng trắng đầu/cuối, chữ thường), đúng định dạng email và
  không trùng với tài khoản đã có.
- **FR-003**: Mật khẩu PHẢI dài 8–128 ký tự; lỗi kiểm tra hiện bằng tiếng Việt ở cả giao diện và máy chủ.
- **FR-004**: Mật khẩu PHẢI được lưu dạng băm một chiều có muối; không lưu và không ghi log mật khẩu gốc.
- **FR-005**: Tài khoản đầu tiên của hệ thống PHẢI có vai trò quản trị viên; mọi tài khoản sau có vai trò
  người học.
- **FR-006**: Khi đăng ký, hệ thống PHẢI lưu múi giờ của trình duyệt (tên múi giờ chuẩn IANA) vào tài
  khoản; thiếu hoặc không hợp lệ thì lưu `Asia/Ho_Chi_Minh`.
- **FR-007**: Đăng ký thành công PHẢI đăng nhập luôn và đưa người dùng tới trang chủ.

**Đăng nhập và phiên**

- **FR-008**: Người dùng PHẢI đăng nhập được bằng email (không phân biệt hoa thường) và mật khẩu.
- **FR-009**: Sai email hoặc sai mật khẩu PHẢI cho cùng một thông báo chung; thời gian phản hồi hai trường
  hợp không được chênh lệch đủ để đoán email có tồn tại.
- **FR-010**: Phiên đăng nhập PHẢI có hiệu lực 30 ngày kể từ lúc đăng nhập trên thiết bị đó, và vẫn còn
  sau khi đóng trình duyệt hoặc máy chủ khởi động lại.
- **FR-011**: Mã phiên PHẢI không đọc được bởi mã chạy trong trang và không bị gửi kèm từ trang web khác.
- **FR-012**: Đăng xuất PHẢI huỷ phiên ở phía máy chủ; phiên đã huỷ hoặc hết hạn không dùng được nữa.
- **FR-013**: Sai mật khẩu 5 lần liên tiếp cho cùng một email PHẢI tạm khoá đăng nhập email đó 15 phút
  (tính từ lần sai thứ 5); trong thời gian khoá, mọi lần đăng nhập đều bị từ chối với thông báo tạm khoá.
  Đăng nhập thành công đặt lại bộ đếm.

**Giao diện và điều hướng**

- **FR-014**: Người chưa đăng nhập mở bất kỳ trang nào ngoài Đăng nhập và Đăng ký PHẢI được đưa tới trang
  Đăng nhập; trang Đăng nhập có liên kết sang Đăng ký và ngược lại.
- **FR-015**: Khi máy chủ báo phiên không còn hiệu lực, app PHẢI đưa người dùng tới trang Đăng nhập kèm
  địa chỉ trang đang xem; đăng nhập lại thành công thì quay về địa chỉ đó (chỉ chấp nhận địa chỉ trong
  app).
- **FR-016**: Người đã đăng nhập mở trang Đăng nhập hoặc Đăng ký PHẢI được đưa về trang chủ.
- **FR-017**: Thanh trên cùng PHẢI hiện email người dùng và nút Đăng xuất (cạnh nút giao diện) khi đã đăng nhập; hiện liên kết
  "Quản trị" chỉ với quản trị viên. Ở 360px, hàng liên kết cuộn ngang được và không làm trang tràn ngang.
- **FR-018**: Form đăng nhập và đăng ký PHẢI dùng được bằng bàn phím và trình đọc màn hình (nhãn, thông
  báo lỗi gắn với ô nhập, nút gửi bị khoá khi đang xử lý).

**Phân quyền**

- **FR-019**: Có một trang quản trị tạm và một dịch vụ quản trị ở máy chủ chỉ quản trị viên truy cập được.
- **FR-020**: Máy chủ PHẢI kiểm tra đăng nhập và vai trò cho mọi dịch vụ cần quyền, độc lập với giao diện:
  chưa đăng nhập → từ chối "chưa đăng nhập"; không đủ vai trò → từ chối "không có quyền".
- **FR-021**: Người học vào trang quản trị PHẢI thấy thông báo "Bạn không có quyền truy cập trang này".
- **FR-022**: Dịch vụ trả thông tin tài khoản chỉ trả dữ liệu của người đang đăng nhập (email, vai trò, múi
  giờ); không bao giờ trả mật khẩu băm hay mã phiên.
- **FR-023**: Mọi dữ liệu cá nhân thêm ở các tính năng sau PHẢI gắn với tài khoản sở hữu; F1 cung cấp danh
  tính người đang đăng nhập cho mọi dịch vụ ở máy chủ.

### Key Entities

- **Tài khoản (User)**: email (chuẩn hoá, duy nhất), mật khẩu đã băm, vai trò (quản trị viên / người học),
  múi giờ (IANA), thời điểm tạo.
- **Phiên đăng nhập (Session)**: thuộc một tài khoản, thời điểm hết hạn (30 ngày sau khi đăng nhập). Chỉ
  lưu dạng không suy ngược được mã phiên gốc.
- **Theo dõi đăng nhập sai (Login attempts)**: theo email: số lần sai liên tiếp, thời điểm hết khoá.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Người mới hoàn tất đăng ký và thấy trang chủ trong dưới 1 phút.
- **SC-002**: 100% mật khẩu trong cơ sở dữ liệu ở dạng băm; không tìm thấy mật khẩu gốc trong cơ sở dữ
  liệu hay log.
- **SC-003**: Thông báo khi sai email và khi sai mật khẩu giống hệt nhau trong 100% lần thử.
- **SC-004**: Sau 5 lần sai liên tiếp, 100% lần đăng nhập tiếp theo trong 15 phút bị từ chối, kể cả đúng
  mật khẩu; sau 15 phút đăng nhập đúng thành công.
- **SC-005**: Người học bị từ chối ở 100% lần thử vào trang quản trị, cả qua giao diện và gọi thẳng máy chủ.
- **SC-006**: Mỗi tài khoản chỉ nhận được thông tin của chính mình trong 100% lần gọi.
- **SC-007**: Đóng và mở lại trình duyệt trong 30 ngày vẫn còn đăng nhập; sau đăng xuất, phiên cũ bị từ
  chối ngay lập tức.
- **SC-008**: Hết phiên giữa chừng, đăng nhập lại đưa về đúng trang đang xem trong 100% lần thử.

## Assumptions

- Thời hạn 30 ngày tính cố định từ lúc đăng nhập, không tự gia hạn khi dùng (đơn giản trước, nguyên tắc VII).
- Đăng ký trùng email báo "Email này đã được dùng": chấp nhận việc lộ email đã đăng ký ở trang đăng ký, vì
  dự án một người dùng và không có luồng khôi phục mật khẩu.
- Bộ đếm lần sai được giữ trong bộ nhớ máy chủ; khởi động lại máy chủ thì bộ đếm về 0.
- Không giới hạn số phiên đồng thời; mỗi thiết bị có phiên riêng; đăng xuất chỉ huỷ phiên của thiết bị đó.
- Ai truy cập được app cũng đăng ký được (chưa có mời hay duyệt tài khoản).
- Trang chủ F0 (tình trạng kết nối) trở thành trang cần đăng nhập; điểm kiểm tra tình trạng ở máy chủ vẫn
  công khai.
- Trang quản trị ở F1 chỉ là trang tạm có tiêu đề và lời chào; nội dung thật làm ở F2.
- Tiêu chí "hai tài khoản không thấy dữ liệu của nhau": ở F1 dữ liệu cá nhân duy nhất là thông tin tài khoản;
  các tính năng sau phải giữ quy tắc này cho dữ liệu của chúng.
- Ngoài phạm vi: quên mật khẩu, đổi mật khẩu, xác minh email, đăng nhập bằng Google, xoá tài khoản, trang
  quản lý người dùng.
