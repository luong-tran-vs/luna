# Feature Specification: Sao lưu và xuất dữ liệu

**Feature Branch**: `011-backup-export`

**Created**: 2026-09-30

**Status**: Draft

**Mã tính năng**: F13 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f13-sao-luu.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tự sao lưu mỗi ngày (Priority: P1)

Hệ thống tự sao lưu toàn bộ cơ sở dữ liệu mỗi ngày lúc 3 giờ sáng và chỉ giữ 7 bản gần nhất. Người vận hành không phải làm gì;
dữ liệu học tập tích luỹ nhiều tháng không mất khi máy hỏng hoặc dữ liệu bị xoá nhầm.

**Why this priority**: Là mục tiêu chính của F13; mất dữ liệu nhiều tháng học là rủi ro lớn nhất của app tự vận hành.

**Independent Test**: Cho hệ thống chạy qua 3 giờ sáng (hoặc kích hoạt sao lưu ngay theo hướng dẫn): thư mục sao lưu có thêm một
bản mang ngày hôm đó; khi đã có 7 bản, bản mới làm bản cũ nhất biến mất.

**Acceptance Scenarios**:

1. **Given** app đang chạy, **When** tới 3 giờ sáng (theo múi giờ cấu hình của máy chủ), **Then** có một bản sao lưu mới ghi rõ ngày
   trong tên.
2. **Given** đã có 7 bản sao lưu, **When** bản thứ 8 được tạo xong, **Then** bản cũ nhất tự bị xoá, còn đúng 7 bản.
3. **Given** sao lưu lỗi (ví dụ đầy đĩa, cơ sở dữ liệu không trả lời), **When** tới giờ sao lưu, **Then** lỗi được ghi log rõ ràng
   (thời điểm, nguyên nhân), không xoá bản cũ nào, người học vẫn dùng app bình thường, và lần sau vẫn thử lại.
4. **Given** người vận hành muốn thử, **When** kích hoạt sao lưu ngay theo hướng dẫn trong README, **Then** có bản sao lưu mới mà
   không cần chờ tới 3 giờ sáng.

---

### User Story 2 - Khôi phục bằng một lệnh (Priority: P1)

Người vận hành chọn một bản sao lưu và khôi phục toàn bộ dữ liệu bằng một lệnh, theo hướng dẫn trong README.

**Why this priority**: Sao lưu chỉ có giá trị khi khôi phục được; là tiêu chí nghiệm thu của F13.

**Independent Test**: Sao lưu, rồi xoá hết dữ liệu (hoặc dùng cơ sở dữ liệu trống), chạy lệnh khôi phục với bản vừa tạo: số tài
khoản, thẻ, bài học, tiến độ… khớp trước khi xoá; đăng nhập lại thấy đúng sổ từ và tiến độ.

**Acceptance Scenarios**:

1. **Given** một bản sao lưu, **When** chạy lệnh khôi phục với tên bản đó, **Then** dữ liệu trở về đúng như lúc sao lưu (số bản ghi
   mỗi loại khớp).
2. **Given** cơ sở dữ liệu đang có dữ liệu mới hơn, **When** khôi phục, **Then** dữ liệu hiện tại được thay bằng dữ liệu của bản sao
   lưu (không trộn lẫn), và README cảnh báo điều này trước.
3. **Given** tên bản sao lưu sai hoặc file hỏng, **When** chạy lệnh khôi phục, **Then** lệnh báo lỗi rõ ràng và không xoá dữ liệu
   hiện tại.
4. **Given** README, **When** người vận hành đọc mục khôi phục, **Then** có đủ: xem danh sách bản sao lưu, lệnh khôi phục, cảnh báo
   ghi đè, cách kiểm tra sau khi khôi phục.

---

### User Story 3 - Xuất dữ liệu của mình (Priority: P2)

Người học bấm **Xuất dữ liệu** trong trang Cài đặt và tải về một file JSON chứa toàn bộ dữ liệu học tập của mình: sổ từ, lịch sử
ôn, tiến độ, cài đặt và các bài học đã học.

**Why this priority**: Người học giữ được bản dữ liệu của riêng mình; cần nhưng không chặn việc học hằng ngày.

**Independent Test**: Tài khoản A có 25 thẻ và đã học 3 bài; tài khoản B có dữ liệu khác. A bấm Xuất dữ liệu: file tải về có tên kèm
ngày, chứa đúng 25 thẻ, tiến độ 3 bài và nội dung 3 bài đó, không có dữ liệu của B, không có mật khẩu hay phiên đăng nhập.

**Acceptance Scenarios**:

1. **Given** đã đăng nhập, **When** bấm **Xuất dữ liệu** trong Cài đặt, **Then** trình duyệt tải về file JSON tên dạng
   `luna-export-YYYYMMDD.json`.
2. **Given** file xuất, **When** mở ra, **Then** có các phần: tài khoản (email, vai trò), cài đặt, sổ từ, lịch sử ôn, mục tiêu, tiến độ
   bài học, ngày học, kết quả chép chính tả và nội dung các bài đã học.
3. **Given** hai tài khoản, **When** mỗi người xuất dữ liệu, **Then** mỗi file chỉ có dữ liệu của chính người đó.
4. **Given** file xuất, **When** tìm mật khẩu (kể cả dạng mã hoá) hoặc mã phiên đăng nhập, **Then** không có.
5. **Given** người học mới chưa có dữ liệu, **When** xuất, **Then** file hợp lệ với các danh sách rỗng.
6. **Given** xuất lỗi (mất kết nối), **When** bấm nút, **Then** thấy thông báo lỗi tiếng Việt và nút vẫn bấm lại được.

---

### Edge Cases

- Máy chủ khởi động lại gần 3 giờ sáng: không sao lưu hai lần trong cùng một ngày, không bỏ lỡ nếu khởi động trước 3 giờ.
- Sao lưu bị ngắt giữa chừng: bản dở dang không được tính là một trong 7 bản và không làm xoá bản cũ.
- Thư mục sao lưu có file lạ không phải bản sao lưu: không bị xoá khi dọn bản cũ.
- Kích hoạt sao lưu ngay hai lần trong một ngày: bản sau thay bản trước cùng ngày (vẫn giữ 7 ngày gần nhất).
- Khôi phục khi app đang chạy: README hướng dẫn thứ tự an toàn; người học có thể cần đăng nhập lại.
- Sổ từ rất lớn (vài nghìn thẻ, vài chục nghìn lượt ôn): xuất vẫn xong trong thời gian chấp nhận được, file hợp lệ.
- Bài học đã học nhưng sau đó bị quản trị viên xoá: tiến độ vẫn có trong file, nội dung bài không còn thì ghi rõ là đã xoá.
- Quản trị viên xuất dữ liệu: chỉ dữ liệu học tập của chính mình, không kèm toàn bộ bài học của hệ thống.

## Requirements *(mandatory)*

### Functional Requirements

**Sao lưu**

- **FR-001**: Hệ thống PHẢI tự sao lưu toàn bộ cơ sở dữ liệu mỗi ngày lúc 03:00 theo múi giờ cấu hình của máy chủ, mỗi ngày tối đa
  một bản (bản sau cùng ngày thay bản trước).
- **FR-002**: Mỗi bản sao lưu PHẢI mang ngày tạo trong tên; hệ thống PHẢI chỉ giữ 7 bản gần nhất, xoá bản cũ nhất sau khi bản mới
  tạo xong thành công.
- **FR-003**: Sao lưu lỗi PHẢI được ghi log rõ ràng (thời điểm, nguyên nhân), KHÔNG ĐƯỢC xoá bản cũ, KHÔNG ĐƯỢC ảnh hưởng app đang
  chạy; lần hẹn sau vẫn chạy.
- **FR-004**: Người vận hành PHẢI kích hoạt được một lần sao lưu ngay (không chờ 03:00) theo hướng dẫn.
- **FR-005**: File âm thanh và từ điển không nằm trong bản sao lưu (tạo lại được); bản sao lưu chứa mọi dữ liệu trong cơ sở dữ liệu.

**Khôi phục**

- **FR-006**: Người vận hành PHẢI khôi phục được toàn bộ dữ liệu từ một bản sao lưu bằng một lệnh; dữ liệu sau khôi phục thay hẳn
  dữ liệu hiện tại.
- **FR-007**: Lệnh khôi phục PHẢI báo lỗi và giữ nguyên dữ liệu hiện tại khi bản sao lưu không tồn tại hoặc không đọc được.
- **FR-008**: README PHẢI có mục sao lưu và khôi phục: nơi lưu, xem danh sách bản, sao lưu ngay, lệnh khôi phục, cảnh báo ghi đè,
  cách kiểm tra.

**Xuất dữ liệu**

- **FR-009**: Trang Cài đặt PHẢI có nút **Xuất dữ liệu** tải về một file JSON tên `luna-export-YYYYMMDD.json` (ngày theo múi giờ của
  người học).
- **FR-010**: File xuất PHẢI gồm: thông tin tài khoản (email, vai trò, ngày tạo), cài đặt, sổ từ (thẻ và lịch ôn), lịch sử ôn, mục
  tiêu, tiến độ từng bài, ngày học, kết quả chép chính tả, và nội dung các bài người học đã học (bài đã bắt đầu).
- **FR-011**: File xuất PHẢI chỉ chứa dữ liệu của tài khoản đang đăng nhập; KHÔNG ĐƯỢC chứa mật khẩu (kể cả dạng mã hoá), phiên
  đăng nhập hay dữ liệu của người khác.
- **FR-012**: Chỉ người đã đăng nhập mới xuất được; lỗi xuất PHẢI hiện thông báo tiếng Việt.
- **FR-013**: Nút xuất PHẢI dùng được bằng bàn phím và trình đọc màn hình, trong lúc xuất cho biết đang xử lý.

### Key Entities

- **Bản sao lưu (Backup)**: một file chứa toàn bộ cơ sở dữ liệu tại một thời điểm, tên có ngày tạo; tối đa 7 bản, nằm trên máy chủ,
  chỉ người vận hành truy cập.
- **File xuất (Export)**: một file JSON của một tài khoản, tạo khi người học yêu cầu, không lưu trên máy chủ; gồm các phần ở FR-010 và
  thời điểm xuất.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Mỗi ngày có đúng một bản sao lưu mới sau 03:00; sau 8 ngày liên tục chạy, thư mục có đúng 7 bản.
- **SC-002**: Khôi phục từ một bản sao lưu vào cơ sở dữ liệu trống cho số bản ghi của mọi loại dữ liệu khớp 100% với lúc sao lưu.
- **SC-003**: Người vận hành làm theo README khôi phục xong trong dưới 5 phút, bằng một lệnh.
- **SC-004**: Sao lưu lỗi không làm app gián đoạn: người học không thấy lỗi nào trong lúc sao lưu chạy hoặc hỏng.
- **SC-005**: 100% file xuất chỉ chứa dữ liệu của người yêu cầu và không chứa mật khẩu hay phiên đăng nhập.
- **SC-006**: Xuất dữ liệu của tài khoản có vài nghìn thẻ xong trong dưới 10 giây.

## Assumptions

- Múi giờ của giờ sao lưu (03:00) là múi giờ cấu hình cho máy chủ, mặc định giờ Việt Nam; không theo múi giờ từng người học.
- Bản sao lưu nằm trên cùng máy chủ (thư mục riêng, gắn ra ngoài để người vận hành tự chép đi nơi khác nếu muốn); sao lưu lên cloud
  ngoài phạm vi.
- "Bài học của mình" là các bài người học đã bắt đầu học (có tiến độ); gồm nội dung chữ và chú thích, không kèm file âm thanh.
- File xuất tạo theo yêu cầu, trả thẳng về trình duyệt, không lưu trên máy chủ.
- Ngoài phạm vi: nhập lại dữ liệu từ file JSON, sao lưu lên cloud, sao lưu/khôi phục qua giao diện web, mã hoá bản sao lưu.
