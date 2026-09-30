# Feature Specification: Khung dự án Luna

**Feature Branch**: `001-project-skeleton`

**Created**: 2026-09-29

**Amended**: 2026-09-30: chỉ còn hai lựa chọn giao diện Sáng / Tối trong một menu thả xuống; khi chưa chọn thì theo thiết bị (US3, FR-005, FR-007).

**Status**: Draft

**Mã tính năng**: F0 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f0-khung-du-an.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Chạy toàn bộ app bằng một lệnh (Priority: P1)

Người phát triển có một máy mới, chỉ cài công cụ chạy container. Họ lấy mã nguồn về, làm theo
README, chạy một lệnh duy nhất và mở được Luna trên trình duyệt: thấy thanh trên cùng có tên
"Luna" và một trang chủ tạm.

**Why this priority**: Đây là nền cho mọi tính năng sau (F1 trở đi). Không chạy được toàn bộ app
thì không có gì để thêm vào.

**Independent Test**: Trên máy chỉ cài công cụ chạy container, làm theo README, chạy một lệnh, mở
trình duyệt và thấy trang chủ tạm có chữ "Luna".

**Acceptance Scenarios**:

1. **Given** máy mới chỉ cài công cụ chạy container và có mã nguồn, **When** người phát triển chạy
   lệnh được ghi trong README, **Then** giao diện, máy chủ và cơ sở dữ liệu cùng khởi động và
   trang chủ tạm mở được trên trình duyệt.
2. **Given** app đang chạy, **When** mở trang chủ, **Then** thấy thanh trên cùng có tên "Luna" và
   toàn bộ chữ trên giao diện bằng tiếng Việt.

---

### User Story 2 - Xem tình trạng kết nối trên trang chủ (Priority: P1)

Người dùng mở trang chủ tạm và biết ngay app có kết nối được tới máy chủ và cơ sở dữ liệu hay
không, qua một trong ba thông báo: "Đã kết nối", "Mất kết nối cơ sở dữ liệu", "Không kết nối
được máy chủ".

**Why this priority**: Chứng minh ba phần (giao diện, máy chủ, cơ sở dữ liệu) thực sự nối với
nhau từ đầu đến cuối, và app không trắng trang khi một phần hỏng.

**Independent Test**: Lần lượt để cả hệ thống chạy, tắt cơ sở dữ liệu, tắt máy chủ; mỗi lần tải
lại trang chủ và kiểm tra thông báo tương ứng.

**Acceptance Scenarios**:

1. **Given** máy chủ và cơ sở dữ liệu đều chạy, **When** mở trang chủ, **Then** hiện "Đã kết nối".
2. **Given** máy chủ chạy nhưng cơ sở dữ liệu đã tắt, **When** mở trang chủ, **Then** hiện "Mất kết
   nối cơ sở dữ liệu" và phần còn lại của trang vẫn hiển thị bình thường.
3. **Given** máy chủ không chạy, **When** mở trang chủ, **Then** hiện "Không kết nối được máy chủ"
   và trang không bị trắng.
4. **Given** đang chờ kết quả kiểm tra, **When** trang vừa mở, **Then** hiện trạng thái đang kiểm
   tra thay vì để trống.

---

### User Story 3 - Chọn giao diện Sáng / Tối (Priority: P2)

Nút giao diện trên thanh trên cùng hiện chế độ đang dùng; bấm vào mở menu gồm Sáng và Tối. Giao diện đổi ngay, và
lần sau mở lại trên cùng trình duyệt vẫn giữ lựa chọn đó.

**Why this priority**: Là yêu cầu bắt buộc của constitution (nguyên tắc IV) và là nền cho mọi màn
hình sau, nhưng app vẫn dùng được nếu chưa có.

**Independent Test**: Chọn từng chế độ, quan sát màu đổi ngay; tải lại trang và kiểm tra lựa chọn
được giữ; khi chưa chọn lần nào, đổi cài đặt sáng/tối của thiết bị và quan sát app đổi theo.

**Acceptance Scenarios**:

1. **Given** lần đầu mở app trên một trình duyệt, **When** trang hiện ra, **Then** nút giao diện hiện
   đúng chế độ của thiết bị (Sáng hoặc Tối) và giao diện khớp với cài đặt đó.
2. **Given** đang ở bất kỳ chế độ nào, **When** chọn Sáng hoặc Tối, **Then** giao diện đổi ngay mà
   không cần tải lại trang.
3. **Given** đã chọn Tối, **When** tải lại trang, **Then** giao diện vẫn ở chế độ Tối.
4. **Given** chưa chọn chế độ lần nào, **When** thiết bị đổi từ sáng sang tối, **Then** app đổi theo
   mà không cần tải lại trang.

---

### User Story 4 - Công cụ cho người phát triển (Priority: P3)

Người phát triển chạy riêng từng phần khi viết code (giao diện tự tải lại khi sửa), cấu hình qua
biến môi trường theo file mẫu, chạy test và lint cho cả giao diện lẫn máy chủ mà không cần mạng,
và dừng máy chủ an toàn.

**Why this priority**: Giúp các tính năng sau làm nhanh và đúng constitution (nguyên tắc V, VI),
nhưng không ảnh hưởng trực tiếp tới người dùng cuối.

**Independent Test**: Ngắt mạng rồi chạy lệnh test và lint của cả hai phía; khởi động máy chủ khi
thiếu cấu hình bắt buộc; dừng máy chủ khi đang có request.

**Acceptance Scenarios**:

1. **Given** máy đã cài đủ công cụ phát triển và đã tải phụ thuộc, **When** ngắt mạng và chạy lệnh
   test, lint của giao diện và máy chủ, **Then** tất cả chạy xong và báo kết quả.
2. **Given** thiếu một biến môi trường bắt buộc, **When** khởi động máy chủ, **Then** máy chủ in ra
   thông báo nêu tên biến bị thiếu và dừng với mã lỗi khác 0.
3. **Given** máy chủ đang xử lý một request, **When** gửi tín hiệu dừng (Ctrl+C hoặc dừng
   container), **Then** request đó được trả kết quả đầy đủ trước khi máy chủ tắt.
4. **Given** đang chạy giao diện ở chế độ phát triển, **When** sửa mã nguồn giao diện, **Then**
   trình duyệt tự cập nhật thay đổi.
5. **Given** repo vừa được lấy về, **When** tìm trong toàn bộ file đã commit, **Then** không có mật
   khẩu, khóa hay chuỗi kết nối thật nào; chỉ có file cấu hình mẫu.

---

### Edge Cases

- Cơ sở dữ liệu phản hồi rất chậm: kiểm tra tình trạng có giới hạn thời gian chờ; quá hạn thì coi
  là "Mất kết nối cơ sở dữ liệu", trang không bị treo.
- Trình duyệt chặn lưu dữ liệu cục bộ (chế độ riêng tư, bị tắt): chọn giao diện vẫn có hiệu lực
  trong phiên hiện tại, chỉ không được nhớ sau khi tải lại; app không lỗi.
- Giá trị lưu trên trình duyệt bị hỏng hoặc không hợp lệ: coi như chưa chọn, app theo thiết bị.
- Máy không có mạng: chữ tiếng Việt có dấu vẫn hiển thị đúng font vì font đi kèm app.
- Màn hình hẹp 360px: thanh trên cùng, nút chọn giao diện và thông báo tình trạng vừa màn hình,
  không cuộn ngang.
- Cơ sở dữ liệu có lại sau khi mất: tải lại trang chủ thì hiện lại "Đã kết nối" mà không cần khởi
  động lại máy chủ.
- Dừng máy chủ khi request chưa xong quá thời gian chờ tối đa: máy chủ vẫn tắt sau khi hết thời
  gian chờ.

## Requirements *(mandatory)*

### Functional Requirements

**Giao diện**

- **FR-001**: App PHẢI có thanh trên cùng hiển thị tên "Luna" trên mọi trang.
- **FR-002**: App PHẢI có một trang chủ tạm hiển thị tình trạng kết nối với đúng một trong các
  thông báo: "Đang kiểm tra kết nối…", "Đã kết nối", "Mất kết nối cơ sở dữ liệu", "Không kết nối
  được máy chủ".
- **FR-003**: Trang chủ PHẢI kiểm tra tình trạng mỗi lần được mở; lỗi kết nối KHÔNG được làm trắng
  trang hoặc chặn phần còn lại của giao diện.
- **FR-004**: Thông báo tình trạng KHÔNG được truyền đạt chỉ bằng màu; mỗi trạng thái có chữ và dấu
  hiệu đi kèm (biểu tượng) theo `docs/design-system.md`.
- **FR-005**: App PHẢI có nút chọn giao diện hiện chế độ đang dùng; bấm vào mở menu gồm hai lựa chọn Sáng và
  Tối. Khi chưa chọn, app theo chế độ của thiết bị.
- **FR-006**: Lựa chọn giao diện PHẢI có hiệu lực ngay và PHẢI được nhớ trên trình duyệt đó sau khi
  tải lại trang.
- **FR-007**: Khi người dùng chưa chọn chế độ, app PHẢI đổi theo khi cài đặt sáng/tối của thiết bị thay đổi.
- **FR-008**: Toàn bộ chữ trên giao diện PHẢI bằng tiếng Việt; font PHẢI đi kèm app để chữ có dấu
  hiển thị đúng khi không có mạng.
- **FR-009**: Màu, font, cỡ chữ, bo góc, khoảng cách PHẢI lấy từ token trong
  `docs/design-system.md`; độ tương phản chữ đạt WCAG AA ở cả hai chế độ.
- **FR-010**: Giao diện PHẢI dùng tốt từ chiều rộng 360px, không cuộn ngang.

**Máy chủ**

- **FR-011**: Máy chủ PHẢI cung cấp một điểm kiểm tra tình trạng cho biết máy chủ đang chạy và cơ sở
  dữ liệu có phản hồi hay không; kiểm tra cơ sở dữ liệu có thời gian chờ tối đa 2 giây.
- **FR-012**: Máy chủ PHẢI đọc cấu hình từ biến môi trường; thiếu cấu hình bắt buộc thì báo lỗi nêu
  rõ tên cấu hình bị thiếu và dừng.
- **FR-013**: Khi nhận tín hiệu dừng, máy chủ PHẢI ngừng nhận request mới, hoàn tất request đang xử
  lý (chờ tối đa 10 giây), đóng kết nối cơ sở dữ liệu rồi mới tắt.
- **FR-014**: Máy chủ PHẢI ghi log có cấu trúc cho mỗi request, kèm mã định danh request.

**Vận hành và phát triển**

- **FR-015**: Toàn bộ app (giao diện, máy chủ, cơ sở dữ liệu) PHẢI khởi động được bằng một lệnh trên
  máy chỉ cài công cụ chạy container; dữ liệu cơ sở dữ liệu được giữ giữa các lần khởi động.
- **FR-016**: README ở gốc repo PHẢI hướng dẫn: chạy toàn bộ bằng một lệnh, chạy riêng từng phần khi
  phát triển, cấu hình biến môi trường, chạy test và lint.
- **FR-017**: PHẢI có cách chạy riêng giao diện ở chế độ phát triển, tự cập nhật khi sửa mã nguồn.
- **FR-018**: Repo PHẢI có file cấu hình mẫu liệt kê mọi biến môi trường; file cấu hình thật bị loại
  khỏi quản lý phiên bản; repo không chứa bí mật.
- **FR-019**: Giao diện và máy chủ PHẢI mỗi bên có ít nhất một bộ test mẫu và lệnh lint; test và lint
  chạy được không cần mạng (sau khi đã tải phụ thuộc).
- **FR-020**: Test PHẢI bao phủ: ba trạng thái kết nối của trang chủ, lưu và khôi phục lựa chọn giao
  diện, điểm kiểm tra tình trạng khi cơ sở dữ liệu lên và xuống, máy chủ dừng khi thiếu cấu hình bắt
  buộc.

### Key Entities

- **Tình trạng hệ thống**: kết quả kiểm tra gồm tình trạng chung (bình thường / suy giảm) và tình
  trạng cơ sở dữ liệu (hoạt động / không hoạt động). Không lưu lại, chỉ tính khi được hỏi.
- **Lựa chọn giao diện**: Sáng, Tối, hoặc chưa chọn (theo thiết bị); lưu trên trình duyệt cho tới
  khi có F12 (cài đặt theo tài khoản).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Trên máy mới chỉ cài công cụ chạy container, người phát triển mở được app trên trình
  duyệt bằng đúng 1 lệnh, không cần bước cài đặt nào khác ngoài những gì README ghi.
- **SC-002**: Trang chủ hiện đúng trạng thái kết nối trong 100% các lần thử ở cả ba tình huống (đủ
  cả, tắt cơ sở dữ liệu, tắt máy chủ), và kết quả hiện ra trong vòng 3 giây sau khi mở trang.
- **SC-003**: Đổi chế độ giao diện thấy thay đổi ngay (dưới 1 giây, không tải lại trang); lựa chọn
  được giữ đúng trong 100% các lần tải lại trên cùng trình duyệt.
- **SC-004**: Ở chiều rộng 360px, cả chế độ sáng và tối, không có cuộn ngang và mọi cặp màu chữ/nền
  đạt tỷ lệ tương phản WCAG AA.
- **SC-005**: Khi ngắt mạng, chữ tiếng Việt có dấu hiển thị đúng font trên mọi màn hình; lệnh test và
  lint của cả hai phía chạy xong và báo kết quả.
- **SC-006**: Khi dừng máy chủ trong lúc có request đang xử lý, 100% request đó nhận được phản hồi đầy
  đủ (nếu xong trong 10 giây).
- **SC-007**: Kiểm tra toàn bộ repo không phát hiện bí mật nào; thiếu biến môi trường bắt buộc thì máy
  chủ dừng và thông báo nêu đúng tên biến.

## Assumptions

- Trang chủ chỉ kiểm tra tình trạng khi được mở hoặc tải lại; không tự kiểm tra lại định kỳ (đơn giản
  trước, nguyên tắc VII).
- Nút chọn giao diện đặt trên thanh trên cùng vì chưa có trang cài đặt; khi có F12, lựa chọn sẽ chuyển
  sang lưu theo tài khoản.
- "Chạy được không cần mạng" nghĩa là sau khi đã tải phụ thuộc và image một lần khi có mạng.
- Chưa có tài khoản nên mọi người mở app đều thấy cùng trang chủ tạm; điểm kiểm tra tình trạng không
  trả dữ liệu người dùng nên không cần kiểm tra quyền.
- Ngoài phạm vi: tài khoản và đăng nhập (F1), mọi tính năng học, dịch vụ AI, chuyển văn bản thành
  giọng nói, nhận dạng giọng nói, từ điển, sao lưu.
- Tiêu chí nghiệm thu gốc: mục F0 trong `docs/phases/giai-doan-1.md`; cấu trúc theo
  `docs/architecture.md`; token theo `docs/design-system.md`.
