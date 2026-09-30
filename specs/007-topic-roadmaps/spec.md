# Feature Specification: Chủ đề và lộ trình theo trình độ

**Feature Branch**: `007-topic-roadmaps`

**Created**: 2026-09-30

**Status**: Draft

**Mã tính năng**: F14 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`) | **Thay đổi**: phần quản trị của F2
(`specs/003-lesson-admin`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f14-chu-de.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Quản lý danh mục chủ đề (Priority: P1)

Quản trị viên mở trang Chủ đề, thấy các chủ đề nhóm theo trình độ (A1 → C2), mỗi chủ đề có tên, mô tả ngắn, số bài và số bài
trong lộ trình. Quản trị viên thêm, sửa, xoá chủ đề. Ví dụ "A1 · Gia đình", "A1 · Mua sắm", "B1 · Công việc".

**Why this priority**: Không có danh mục chủ đề thì không gán được bài và không có lộ trình theo chủ đề.

**Independent Test**: Thêm "A1 · Gia đình" và "B1 · Công việc", sửa mô tả, thử thêm "A1 · gia đình" (bị chặn), xoá một chủ đề
chưa có bài, thử xoá chủ đề còn bài (bị chặn).

**Acceptance Scenarios**:

1. **Given** trang Chủ đề, **When** thêm chủ đề tên "Gia đình", trình độ A1, mô tả "Người thân, nhà cửa", **Then** chủ đề xuất
   hiện trong nhóm A1 với 0 bài.
2. **Given** đã có "A1 · Gia đình", **When** thêm "gia đình" cùng trình độ A1, **Then** thấy báo "Chủ đề này đã có ở trình độ
   A1" và không tạo thêm; thêm "Gia đình" ở A2 thì được.
3. **Given** một chủ đề, **When** sửa tên hoặc mô tả, **Then** thay đổi hiện ngay ở danh sách, ô chọn chủ đề của form bài và
   bộ lọc.
4. **Given** chủ đề chưa có bài, **When** bấm Xoá và xác nhận, **Then** chủ đề bị xoá.
5. **Given** chủ đề còn 3 bài, **When** bấm Xoá, **Then** thấy báo "Chủ đề còn 3 bài, hãy chuyển hoặc xoá bài trước" và chủ đề
   vẫn còn.
6. **Given** tên trống hoặc quá dài, **When** lưu, **Then** thấy lỗi ở ô tên, không lưu.

---

### User Story 2 - Gán chủ đề cho bài và lọc danh sách bài (Priority: P1)

Khi tạo hoặc sửa bài, quản trị viên chọn chủ đề từ danh sách (bắt buộc, nhóm theo trình độ); trình độ của bài lấy theo chủ đề,
không chọn riêng. Danh sách bài lọc được theo trình độ và chủ đề bằng ô chọn.

**Why this priority**: Mỗi bài phải thuộc đúng một chủ đề thì lộ trình theo chủ đề mới đúng.

**Independent Test**: Tạo bài chọn "A1 · Gia đình" → bài có trình độ A1; sửa bài sang "B1 · Công việc" → trình độ thành B1;
lọc danh sách theo A1 rồi theo "A1 · Gia đình".

**Acceptance Scenarios**:

1. **Given** form Thêm bài, **When** mở ô chủ đề, **Then** thấy các chủ đề nhóm theo trình độ ("A1 · Gia đình", …); không còn ô
   trình độ và ô chủ đề gõ tự do.
2. **Given** chưa chọn chủ đề, **When** lưu, **Then** thấy "Vui lòng chọn chủ đề" và bài không được lưu.
3. **Given** bài được lưu với chủ đề "A1 · Gia đình", **When** xem bài ở danh sách hoặc trang chi tiết, **Then** trình độ là A1
   và chủ đề là "Gia đình".
4. **Given** bài đang ở lộ trình "A1 · Gia đình", **When** sửa bài sang "A2 · Du lịch", **Then** bài biến khỏi lộ trình cũ và
   nằm ở cuối lộ trình "A2 · Du lịch"; trình độ của bài thành A2.
5. **Given** danh sách bài nhiều trình độ, **When** chọn trình độ A1, **Then** chỉ thấy bài A1 và ô chủ đề chỉ còn các chủ đề
   A1; **When** chọn thêm "Gia đình", **Then** chỉ thấy bài của chủ đề đó.
6. **Given** chưa có chủ đề nào, **When** mở form Thêm bài, **Then** thấy hướng dẫn và đường dẫn tới trang Chủ đề.

---

### User Story 3 - Lộ trình riêng cho từng chủ đề (Priority: P1)

Trang Lộ trình cho chọn một chủ đề rồi xếp lộ trình của chủ đề đó: thêm bài của chủ đề, kéo thả đổi thứ tự, gỡ bài. Trang hiện
cảnh báo cho từng chủ đề còn dưới 3 bài chưa học.

**Why this priority**: Là mục tiêu của F14; người học (L) sẽ học theo lộ trình của chủ đề đã chọn.

**Independent Test**: Chọn "A1 · Gia đình", thêm 3 bài của chủ đề, kéo bài cuối lên đầu, tải lại thấy giữ thứ tự; gỡ một bài
thấy cảnh báo "còn 2 bài"; bài của chủ đề khác không thêm được.

**Acceptance Scenarios**:

1. **Given** trang Lộ trình, **When** chọn chủ đề "A1 · Gia đình", **Then** thấy lộ trình của chủ đề đó và danh sách các bài
   của chủ đề chưa có trong lộ trình để thêm.
2. **Given** lộ trình có nhiều bài, **When** kéo một bài lên đầu (hoặc dùng nút Lên/Xuống), **Then** thứ tự mới được lưu ngay;
   tải lại vẫn giữ.
3. **Given** bài trong lộ trình, **When** bấm Gỡ, **Then** bài rời lộ trình nhưng vẫn thuộc chủ đề và vẫn trong danh sách bài.
4. **Given** bài của chủ đề "B1 · Công việc", **When** tìm cách thêm vào lộ trình "A1 · Gia đình", **Then** bị từ chối; lộ
   trình của một chủ đề chỉ chứa bài của chủ đề đó.
5. **Given** "A1 · Gia đình" có 2 bài trong lộ trình và "A1 · Mua sắm" có 5, **When** mở trang Lộ trình, **Then** chỉ "A1 · Gia
   đình" có cảnh báo "còn 2 bài chưa học"; chủ đề có lộ trình trống báo "Lộ trình chưa có bài".
6. **Given** bài đã có trong lộ trình, **When** thêm lần nữa, **Then** không bị trùng.

---

### User Story 4 - Chuyển dữ liệu cũ sang chủ đề (Priority: P1)

Khi cập nhật lên phiên bản có F14, dữ liệu F2 được chuyển tự động, không cần thao tác: chủ đề được tạo từ các cặp trình độ và
chủ đề của bài hiện có; bài chưa có chủ đề vào chủ đề "Chung" cùng trình độ; lộ trình chung cũ được chia theo chủ đề, giữ thứ
tự tương đối.

**Why this priority**: Không chuyển được dữ liệu thì bài và lộ trình đã soạn bị mất hoặc không dùng được.

**Independent Test**: Chuẩn bị dữ liệu F2 (bài A1 "Gia đình", bài A1 không chủ đề, bài B1 "Công việc", lộ trình chung xen kẽ
các bài), khởi động phiên bản mới, kiểm tra chủ đề, gán bài và thứ tự; khởi động lại lần nữa không sinh dữ liệu trùng.

**Acceptance Scenarios**:

1. **Given** bài A1 chủ đề "Gia đình", bài A1 chủ đề "gia đình " và bài B1 chủ đề "Công việc", **When** chuyển dữ liệu,
   **Then** có đúng 2 chủ đề "A1 · Gia đình" (gộp khác hoa thường, khoảng trắng) và "B1 · Công việc", mỗi bài thuộc đúng chủ đề
   của nó.
2. **Given** bài A2 không có chủ đề, **When** chuyển dữ liệu, **Then** bài thuộc chủ đề "A2 · Chung".
3. **Given** lộ trình chung cũ theo thứ tự X(A1·Gia đình), Y(B1·Công việc), Z(A1·Gia đình), **When** chuyển dữ liệu, **Then**
   lộ trình "A1 · Gia đình" là X, Z và lộ trình "B1 · Công việc" là Y.
4. **Given** bài không nằm trong lộ trình cũ, **When** chuyển dữ liệu, **Then** bài thuộc chủ đề nhưng không vào lộ trình.
5. **Given** đã chuyển dữ liệu, **When** khởi động lại máy chủ, **Then** không tạo chủ đề trùng, không đổi lộ trình.
6. **Given** chuyển dữ liệu xong, **When** đếm bài, **Then** số bài không đổi và mọi bài đều có chủ đề.

---

### Edge Cases

- Sửa trình độ của một chủ đề đã có bài: mọi bài của chủ đề đổi theo trình độ mới; tên phải không trùng ở trình độ mới.
- Đổi chủ đề của bài không nằm trong lộ trình cũ: bài chỉ đổi chủ đề, không tự vào lộ trình mới.
- Đổi chủ đề của bài sang chính chủ đề đang có: không thay đổi lộ trình.
- Chủ đề bị xoá bởi người khác khi đang mở form bài: lưu bài báo "Chủ đề không tồn tại", yêu cầu chọn lại.
- Xoá bài đang ở lộ trình vẫn bị chặn như F2 ("Gỡ bài khỏi lộ trình trước"); gỡ rồi xoá được.
- Lộ trình trống: cảnh báo "Lộ trình chưa có bài".
- Tên chủ đề có khoảng trắng thừa: được cắt; so trùng không phân biệt hoa thường và khoảng trắng thừa.
- Chuyển dữ liệu bị ngắt giữa chừng (tắt máy): lần khởi động sau làm tiếp và cho kết quả như chạy một lần.
- Lộ trình cũ chứa id bài đã bị xoá: bỏ qua, không tạo mục rỗng.
- Màn hình 360px: danh sách chủ đề, form chủ đề, form bài, bộ lọc và trang lộ trình dùng được, không cuộn ngang.

## Requirements *(mandatory)*

### Functional Requirements

**Chủ đề**

- **FR-001**: Quản trị viên PHẢI thêm, sửa, xoá được chủ đề; mỗi chủ đề có tên (bắt buộc, 1–60 ký tự), trình độ (A1–C2, bắt
  buộc) và mô tả ngắn (không bắt buộc, ≤ 200 ký tự).
- **FR-002**: Tên chủ đề KHÔNG được trùng trong cùng trình độ (so không phân biệt hoa thường, bỏ khoảng trắng đầu cuối và
  thừa); cùng tên ở trình độ khác thì được.
- **FR-003**: Xoá chủ đề còn bài PHẢI bị từ chối, kèm số bài còn lại; chủ đề không còn bài thì xoá được sau khi xác nhận.
- **FR-004**: Trang Chủ đề PHẢI liệt kê chủ đề nhóm theo trình độ A1 → C2, trong mỗi trình độ theo tên, kèm số bài và số bài
  trong lộ trình.
- **FR-005**: Sửa trình độ của chủ đề PHẢI đổi trình độ của mọi bài thuộc chủ đề đó.

**Bài học**

- **FR-006**: Mỗi bài PHẢI thuộc đúng một chủ đề. Form tạo/sửa bài PHẢI có ô chọn chủ đề (bắt buộc, nhóm theo trình độ), thay
  cho ô trình độ và ô chủ đề gõ tự do của F2.
- **FR-007**: Trình độ của bài PHẢI luôn bằng trình độ của chủ đề của bài.
- **FR-008**: Đổi chủ đề của bài đang ở lộ trình PHẢI gỡ bài khỏi lộ trình cũ và thêm vào cuối lộ trình của chủ đề mới; bài
  không ở lộ trình thì chỉ đổi chủ đề.
- **FR-009**: Danh sách bài PHẢI lọc được theo trình độ và theo chủ đề bằng ô chọn; chọn trình độ thì ô chủ đề chỉ còn chủ đề
  của trình độ đó.
- **FR-010**: Danh sách bài và trang chi tiết bài PHẢI hiện trình độ và tên chủ đề của bài.

**Lộ trình**

- **FR-011**: Mỗi chủ đề PHẢI có một lộ trình riêng: danh sách bài có thứ tự, không trùng, chỉ gồm bài của chủ đề đó. Lộ trình
  chung duy nhất của F2 không còn.
- **FR-012**: Trang Lộ trình PHẢI cho chọn chủ đề (nhóm theo trình độ), rồi thêm bài của chủ đề vào cuối, gỡ bài, đổi thứ tự
  bằng kéo thả hoặc nút Lên/Xuống; thứ tự PHẢI được lưu ngay sau mỗi thay đổi.
- **FR-013**: Thêm vào lộ trình một bài không thuộc chủ đề PHẢI bị từ chối.
- **FR-014**: Trang Lộ trình PHẢI cảnh báo riêng cho từng chủ đề có dưới 3 bài chưa học trong lộ trình (kể cả lộ trình trống),
  nêu tên chủ đề và số bài còn lại.
- **FR-015**: Xoá bài đang ở lộ trình của chủ đề PHẢI bị từ chối như F2.

**Chuyển dữ liệu**

- **FR-016**: Khi khởi động phiên bản có F14, hệ thống PHẢI tự chuyển dữ liệu F2: tạo chủ đề từ các cặp (trình độ, chủ đề) của
  bài (gộp tên khác hoa thường và khoảng trắng thừa, giữ cách viết của bài đầu tiên), bài không có chủ đề vào chủ đề "Chung"
  cùng trình độ, gán chủ đề cho mọi bài.
- **FR-017**: Lộ trình chung cũ PHẢI được chia vào lộ trình của từng chủ đề, giữ thứ tự tương đối; bài không ở lộ trình cũ thì
  không vào lộ trình mới; id bài không còn tồn tại bị bỏ qua.
- **FR-018**: Việc chuyển dữ liệu PHẢI chạy đúng một lần về kết quả: chạy lại (kể cả sau khi bị ngắt) không tạo chủ đề trùng,
  không đổi lộ trình, không mất bài.

**Quyền và giao diện**

- **FR-019**: Mọi thao tác quản lý chủ đề, lộ trình và gán chủ đề PHẢI chỉ dành cho quản trị viên.
- **FR-020**: Người đã đăng nhập PHẢI đọc được danh sách chủ đề của một trình độ kèm số bài trong lộ trình (để L cho người học
  chọn), không thấy mô tả nội bộ nào khác.
- **FR-021**: Mọi trang của F14 PHẢI dùng được ở 360px, bằng bàn phím, sáng và tối (nguyên tắc IV).

### Key Entities

- **Chủ đề (Topic)**: tên, trình độ, mô tả ngắn, lộ trình (danh sách bài có thứ tự), thời điểm tạo. Tên duy nhất trong trình độ.
- **Bài học (Lesson, F2)**: thay chủ đề gõ tự do bằng liên kết tới một chủ đề; trình độ luôn theo chủ đề.
- **Lộ trình (Roadmap)**: thuộc một chủ đề; danh sách bài của chủ đề, có thứ tự, không trùng. Thay lộ trình chung của F2.
- **Lần chuyển dữ liệu (Migration)**: đánh dấu việc chuyển dữ liệu F2 → F14 đã xong.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sau chuyển dữ liệu, 100% bài có đúng một chủ đề, số bài không đổi, và thứ tự tương đối của các bài trong mỗi lộ
  trình mới giống thứ tự trong lộ trình chung cũ.
- **SC-002**: Chạy chuyển dữ liệu 2 lần cho kết quả giống hệt chạy 1 lần (0 chủ đề trùng, 0 thay đổi lộ trình).
- **SC-003**: 100% lần thử thêm bài của chủ đề khác vào một lộ trình bị từ chối.
- **SC-004**: 100% lần thử xoá chủ đề còn bài bị từ chối; 100% lần thử tạo tên trùng trong cùng trình độ bị từ chối.
- **SC-005**: Cảnh báo "dưới 3 bài chưa học" hiện đúng cho từng chủ đề ở 100% trường hợp thử (0, 1, 2, 3 bài).
- **SC-006**: Quản trị viên tạo một chủ đề mới, gán 3 bài và xếp lộ trình trong dưới 3 phút.
- **SC-007**: Trình độ của bài khớp trình độ chủ đề ở 100% bài, kể cả sau khi đổi chủ đề hoặc sửa trình độ của chủ đề.
- **SC-008**: Ở 360px, mọi trang của F14 không cuộn ngang, dùng được hoàn toàn bằng bàn phím.

## Assumptions

- Chưa có tiến độ học (làm ở L), nên như F2 mọi bài trong lộ trình được tính là "chưa học"; khi có L, cảnh báo đếm theo tiến độ
  thật.
- Thứ tự giữa các chủ đề không quan trọng ở F14; người học tự chọn chủ đề ở L.
- Người học chọn trình độ và chủ đề, bài hôm nay theo chủ đề, lời chào khi hết lộ trình thuộc L; F14 chỉ cung cấp danh sách
  chủ đề theo trình độ.
- Sửa trình độ của chủ đề được phép cả khi đã có bài (bài đổi trình độ theo); không có lịch sử thay đổi.
- Tên "Chung" không đặc biệt sau khi chuyển dữ liệu: quản trị viên đổi tên hoặc chuyển bài như chủ đề thường.
- Ngoài phạm vi: người học chọn trình độ và chủ đề, bài hôm nay theo chủ đề (L); chủ đề nhiều cấp; ảnh đại diện cho chủ đề; sắp
  xếp thủ công thứ tự các chủ đề.
