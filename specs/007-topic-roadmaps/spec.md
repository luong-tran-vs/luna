# Feature Specification: Chủ đề và lộ trình theo trình độ

**Feature Branch**: `007-topic-roadmaps`

**Created**: 2026-09-30

**Status**: Draft

**Mã tính năng**: F14 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`) | **Thay đổi**: phần quản trị của F2
(`specs/003-lesson-admin`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f14-chu-de.md"

**Sửa 2026-10-07 — chủ đề dùng chung cho mọi trình độ**: chủ đề không còn gắn với một trình độ. Danh mục chỉ có một "Gia đình",
một "Mua sắm"…; trình độ thuộc về bài. Mỗi cặp (chủ đề, trình độ) vẫn là một lộ trình riêng, nên người học vẫn chọn trình độ rồi
chủ đề như trước. Khi sinh bài, quản trị viên chọn trình độ, AI sinh bài hợp với trình độ đó (F7, `specs/012-ai-lesson-generation`).
Dữ liệu cũ được gộp tự động (US5). Các chỗ thay đổi trong spec này được đánh dấu *(sửa 2026-10-07)*.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Quản lý danh mục chủ đề (Priority: P1) *(sửa 2026-10-07)*

Quản trị viên mở trang Chủ đề, thấy mọi chủ đề xếp theo tên, mỗi chủ đề có tên, mô tả ngắn, tổng số bài và số bài của từng trình
độ đang có bài (ví dụ "A1: 4 bài · 2 trong lộ trình", "A2: 3 bài · 3 trong lộ trình"). Quản trị viên thêm, sửa, xoá chủ đề. Chủ đề
không có ô trình độ.

**Why this priority**: Không có danh mục chủ đề thì không gán được bài và không có lộ trình theo chủ đề.

**Independent Test**: Thêm "Gia đình" và "Công việc", sửa mô tả, thử thêm "gia đình" (bị chặn), xoá một chủ đề chưa có bài, thử
xoá chủ đề còn bài (bị chặn).

**Acceptance Scenarios**:

1. **Given** trang Chủ đề, **When** thêm chủ đề tên "Gia đình", mô tả "Người thân, nhà cửa", **Then** chủ đề xuất hiện trong danh
   sách với 0 bài.
2. **Given** đã có "Gia đình", **When** thêm "gia đình", **Then** thấy báo "Chủ đề này đã có" và không tạo thêm.
3. **Given** một chủ đề, **When** sửa tên hoặc mô tả, **Then** thay đổi hiện ngay ở danh sách, ô chọn chủ đề của form bài và bộ lọc.
4. **Given** chủ đề chưa có bài, **When** bấm Xoá và xác nhận, **Then** chủ đề bị xoá.
5. **Given** chủ đề còn 3 bài (ở bất kỳ trình độ nào), **When** bấm Xoá, **Then** thấy báo "Chủ đề còn 3 bài, hãy chuyển hoặc xoá
   bài trước" và chủ đề vẫn còn.
6. **Given** tên trống hoặc quá dài, **When** lưu, **Then** thấy lỗi ở ô tên, không lưu.
7. **Given** "Gia đình" có bài A1 và A2, **When** xem trang Chủ đề, **Then** chủ đề hiện số bài và số bài trong lộ trình của A1 và
   của A2; trình độ chưa có bài nào không hiện.

---

### User Story 2 - Gán chủ đề, trình độ cho bài và lọc danh sách bài (Priority: P1) *(sửa 2026-10-07)*

Khi tạo hoặc sửa bài, quản trị viên chọn chủ đề (bắt buộc) và trình độ (bắt buộc) bằng hai ô chọn. Danh sách bài lọc được theo
trình độ và chủ đề bằng ô chọn.

**Why this priority**: Mỗi bài phải thuộc đúng một chủ đề và một trình độ thì lộ trình (chủ đề, trình độ) mới đúng.

**Independent Test**: Tạo bài chọn "Gia đình", A1; sửa bài sang A2 → bài chuyển sang lộ trình "A2 · Gia đình" nếu đang ở lộ trình;
lọc danh sách theo A1 rồi theo "Gia đình".

**Acceptance Scenarios**:

1. **Given** form Thêm bài, **When** mở ô chủ đề, **Then** thấy mọi chủ đề xếp theo tên; ô trình độ cho chọn A1–C2.
2. **Given** chưa chọn chủ đề hoặc trình độ, **When** lưu, **Then** thấy "Vui lòng chọn chủ đề" hoặc "Vui lòng chọn trình độ" và bài
   không được lưu.
3. **Given** bài được lưu với "Gia đình", A1, **When** xem bài ở danh sách hoặc trang chi tiết, **Then** trình độ là A1 và chủ đề là
   "Gia đình".
4. **Given** bài đang ở lộ trình "A1 · Gia đình", **When** sửa bài sang chủ đề "Du lịch" hoặc sang trình độ A2, **Then** bài biến
   khỏi lộ trình cũ và nằm ở cuối lộ trình của cặp mới ("A1 · Du lịch" hoặc "A2 · Gia đình").
5. **Given** danh sách bài nhiều trình độ, **When** chọn trình độ A1, **Then** chỉ thấy bài A1; **When** chọn thêm "Gia đình",
   **Then** chỉ thấy bài A1 của chủ đề đó. Ô chủ đề luôn có mọi chủ đề.
6. **Given** chưa có chủ đề nào, **When** mở form Thêm bài, **Then** thấy hướng dẫn và đường dẫn tới trang Chủ đề.

---

### User Story 3 - Lộ trình riêng cho từng cặp chủ đề và trình độ (Priority: P1) *(sửa 2026-10-07)*

Trang Lộ trình cho chọn một chủ đề và một trình độ rồi xếp lộ trình của cặp đó: thêm bài của chủ đề ở trình độ đó, kéo thả đổi thứ
tự, gỡ bài. Trang hiện cảnh báo cho từng lộ trình còn dưới 3 bài chưa học.

**Why this priority**: Là mục tiêu của F14; người học (L) học theo lộ trình của trình độ và chủ đề đã chọn.

**Independent Test**: Chọn "Gia đình", A1, thêm 3 bài A1 của chủ đề, kéo bài cuối lên đầu, tải lại thấy giữ thứ tự; gỡ một bài
thấy cảnh báo "còn 2 bài"; bài A2 của "Gia đình" hay bài của chủ đề khác không thêm được.

**Acceptance Scenarios**:

1. **Given** trang Lộ trình, **When** chọn "Gia đình" và A1, **Then** thấy lộ trình "A1 · Gia đình" và danh sách các bài A1 của chủ
   đề chưa có trong lộ trình để thêm.
2. **Given** lộ trình có nhiều bài, **When** kéo một bài lên đầu (hoặc dùng nút Lên/Xuống), **Then** thứ tự mới được lưu ngay; tải
   lại vẫn giữ.
3. **Given** bài trong lộ trình, **When** bấm Gỡ, **Then** bài rời lộ trình nhưng vẫn thuộc chủ đề và vẫn trong danh sách bài.
4. **Given** bài "B1 · Công việc" hoặc bài "A2 · Gia đình", **When** tìm cách thêm vào lộ trình "A1 · Gia đình", **Then** bị từ chối;
   lộ trình chỉ chứa bài đúng chủ đề và đúng trình độ.
5. **Given** "A1 · Gia đình" có 2 bài trong lộ trình và "A1 · Mua sắm" có 5, **When** mở trang Lộ trình, **Then** chỉ "A1 · Gia
   đình" có cảnh báo "còn 2 bài chưa học". Cặp có bài nhưng lộ trình trống báo "Lộ trình chưa có bài"; cặp chưa có bài nào không
   bị cảnh báo.
6. **Given** bài đã có trong lộ trình, **When** thêm lần nữa, **Then** không bị trùng.

---

### User Story 4 - Chuyển dữ liệu F2 sang chủ đề (Priority: P1)

Khi cập nhật lên phiên bản có F14, dữ liệu F2 được chuyển tự động, không cần thao tác: chủ đề được tạo từ các cặp trình độ và
chủ đề của bài hiện có; bài chưa có chủ đề vào chủ đề "Chung" cùng trình độ; lộ trình chung cũ được chia theo chủ đề, giữ thứ
tự tương đối. (Đã chạy trước 2026-10-07; sau đó US5 gộp các chủ đề này lại.)

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

### User Story 5 - Gộp chủ đề trùng tên ở các trình độ (Priority: P1) *(mới 2026-10-07)*

Khi cập nhật lên phiên bản có chủ đề dùng chung, các chủ đề cùng tên ở nhiều trình độ (ví dụ "A1 · Gia đình" và "A2 · Gia đình")
được gộp tự động thành một chủ đề. Bài, lộ trình, từ vựng của chủ đề (F18) và mục tiêu của người học (L) được giữ nguyên.

**Why this priority**: Không gộp được thì mất lộ trình đã xếp, mất tiến độ của người học.

**Independent Test**: Chuẩn bị "A1 · Gia đình" (lộ trình X, Y), "A2 · Gia đình" (lộ trình Z), "B1 · Công việc", một người học đang
học "A2 · Gia đình" ở bài thứ 1. Khởi động phiên bản mới: còn 2 chủ đề "Gia đình", "Công việc"; lộ trình "A1 · Gia đình" là X, Y,
"A2 · Gia đình" là Z; người học vẫn ở "A2 · Gia đình", bài thứ 1. Khởi động lại không đổi gì.

**Acceptance Scenarios**:

1. **Given** "A1 · Gia đình" và "A2 · gia đình", **When** gộp, **Then** còn một chủ đề "Gia đình" (so tên không phân biệt hoa thường
   và khoảng trắng thừa; giữ cách viết và mô tả của chủ đề ở trình độ thấp nhất, mô tả trống thì lấy mô tả đầu tiên không trống).
2. **Given** bài của hai chủ đề cũ, **When** gộp, **Then** mọi bài thuộc chủ đề mới và giữ nguyên trình độ của mình.
3. **Given** lộ trình của hai chủ đề cũ, **When** gộp, **Then** lộ trình "A1 · Gia đình" và "A2 · Gia đình" giống hệt lộ trình cũ,
   cùng thứ tự.
4. **Given** người học có mục tiêu "A2 · Gia đình" và tiến độ ở đó, **When** gộp, **Then** mục tiêu, tiến độ và bài hôm nay không đổi.
5. **Given** từ vựng của hai chủ đề cũ (F18), **When** gộp, **Then** chủ đề mới có mọi từ; mỗi từ mang trình độ của chủ đề cũ chứa nó,
   từ có ở nhiều chủ đề cũ giữ một lần với trình độ thấp nhất.
6. **Given** đã gộp, **When** khởi động lại máy chủ (kể cả sau khi bị ngắt giữa chừng), **Then** kết quả như chạy một lần.

---

### Edge Cases

- *(sửa 2026-10-07)* Không còn "sửa trình độ của chủ đề"; đổi trình độ là đổi ở từng bài.
- Đổi chủ đề hoặc trình độ của bài không nằm trong lộ trình: bài chỉ đổi chủ đề/trình độ, không tự vào lộ trình mới.
- Đổi sang đúng chủ đề và trình độ đang có: không thay đổi lộ trình.
- Chủ đề bị xoá bởi người khác khi đang mở form bài: lưu bài báo "Chủ đề không tồn tại", yêu cầu chọn lại.
- Xoá bài đang ở lộ trình vẫn bị chặn như F2 ("Gỡ bài khỏi lộ trình trước"); gỡ rồi xoá được.
- Lộ trình trống của cặp có bài: cảnh báo "Lộ trình chưa có bài".
- Tên chủ đề có khoảng trắng thừa: được cắt; so trùng không phân biệt hoa thường và khoảng trắng thừa.
- Chuyển hoặc gộp dữ liệu bị ngắt giữa chừng (tắt máy): lần khởi động sau làm tiếp và cho kết quả như chạy một lần.
- Lộ trình cũ chứa id bài đã bị xoá: bỏ qua, không tạo mục rỗng.
- Màn hình 360px: danh sách chủ đề, form chủ đề, form bài, bộ lọc và trang lộ trình dùng được, không cuộn ngang.

## Requirements *(mandatory)*

### Functional Requirements

**Chủ đề** *(sửa 2026-10-07)*

- **FR-001**: Quản trị viên PHẢI thêm, sửa, xoá được chủ đề; mỗi chủ đề có tên (bắt buộc, 1–60 ký tự) và mô tả ngắn (không bắt buộc,
  ≤ 200 ký tự). Chủ đề không có trình độ.
- **FR-002**: Tên chủ đề KHÔNG được trùng trong toàn danh mục (so không phân biệt hoa thường, bỏ khoảng trắng đầu cuối và thừa).
- **FR-003**: Xoá chủ đề còn bài (ở bất kỳ trình độ nào) PHẢI bị từ chối, kèm số bài còn lại; chủ đề không còn bài thì xoá được sau
  khi xác nhận.
- **FR-004**: Trang Chủ đề PHẢI liệt kê chủ đề theo tên, kèm tổng số bài và, với mỗi trình độ có bài, số bài và số bài trong lộ
  trình của trình độ đó.
- **FR-005**: *(bỏ 2026-10-07: chủ đề không còn trình độ để sửa.)*

**Bài học** *(sửa 2026-10-07)*

- **FR-006**: Mỗi bài PHẢI thuộc đúng một chủ đề và có đúng một trình độ (A1–C2). Form tạo/sửa bài PHẢI có ô chọn chủ đề và ô chọn
  trình độ, cả hai bắt buộc.
- **FR-007**: Trình độ là của bài, chọn độc lập với chủ đề; một chủ đề có bài ở nhiều trình độ.
- **FR-008**: Đổi chủ đề hoặc trình độ của bài đang ở lộ trình PHẢI gỡ bài khỏi lộ trình cũ và thêm vào cuối lộ trình của cặp
  (chủ đề, trình độ) mới; bài không ở lộ trình thì chỉ đổi.
- **FR-009**: Danh sách bài PHẢI lọc được theo trình độ và theo chủ đề bằng ô chọn, độc lập với nhau.
- **FR-010**: Danh sách bài và trang chi tiết bài PHẢI hiện trình độ và tên chủ đề của bài.

**Lộ trình** *(sửa 2026-10-07)*

- **FR-011**: Mỗi cặp (chủ đề, trình độ) PHẢI có một lộ trình riêng: danh sách bài có thứ tự, không trùng, chỉ gồm bài của chủ đề
  đó ở trình độ đó. Lộ trình chung duy nhất của F2 không còn.
- **FR-012**: Trang Lộ trình PHẢI cho chọn chủ đề và trình độ, rồi thêm bài của cặp đó vào cuối, gỡ bài, đổi thứ tự bằng kéo thả hoặc
  nút Lên/Xuống; thứ tự PHẢI được lưu ngay sau mỗi thay đổi.
- **FR-013**: Thêm vào lộ trình một bài khác chủ đề hoặc khác trình độ PHẢI bị từ chối.
- **FR-014**: Trang Lộ trình PHẢI cảnh báo riêng cho từng cặp (chủ đề, trình độ) có ít nhất một bài mà lộ trình còn dưới 3 bài chưa
  học (kể cả lộ trình trống), nêu tên cặp và số bài còn lại.
- **FR-015**: Xoá bài đang ở lộ trình PHẢI bị từ chối như F2.

**Chuyển dữ liệu**

- **FR-016**: Khi khởi động phiên bản có F14, hệ thống PHẢI tự chuyển dữ liệu F2: tạo chủ đề từ các cặp (trình độ, chủ đề) của
  bài (gộp tên khác hoa thường và khoảng trắng thừa, giữ cách viết của bài đầu tiên), bài không có chủ đề vào chủ đề "Chung"
  cùng trình độ, gán chủ đề cho mọi bài.
- **FR-017**: Lộ trình chung cũ PHẢI được chia vào lộ trình của từng chủ đề, giữ thứ tự tương đối; bài không ở lộ trình cũ thì
  không vào lộ trình mới; id bài không còn tồn tại bị bỏ qua.
- **FR-018**: Việc chuyển dữ liệu PHẢI chạy đúng một lần về kết quả: chạy lại (kể cả sau khi bị ngắt) không tạo chủ đề trùng,
  không đổi lộ trình, không mất bài.
- **FR-022** *(mới 2026-10-07)*: Khi khởi động phiên bản có chủ đề dùng chung, hệ thống PHẢI gộp các chủ đề cùng tên (so như FR-002)
  thành một: giữ id, cách viết và mô tả của chủ đề ở trình độ thấp nhất (mô tả trống thì lấy mô tả đầu tiên không trống theo trình
  độ); lộ trình của mỗi chủ đề cũ thành lộ trình (chủ đề mới, trình độ cũ), giữ thứ tự; bài giữ trình độ.
- **FR-023** *(mới 2026-10-07)*: Việc gộp PHẢI chuyển mọi tham chiếu tới chủ đề cũ sang chủ đề mới mà không đổi nghĩa: mục tiêu và
  tiến độ của người học (L) giữ cặp (chủ đề, trình độ); từ vựng của chủ đề (F18) gộp theo FR của F18; bài ngữ pháp, thống kê không
  đổi. Chạy lại (kể cả sau khi bị ngắt) cho kết quả như chạy một lần.

**Quyền và giao diện**

- **FR-019**: Mọi thao tác quản lý chủ đề, lộ trình và gán chủ đề PHẢI chỉ dành cho quản trị viên.
- **FR-020** *(sửa 2026-10-07)*: Người đã đăng nhập PHẢI đọc được, với một trình độ, danh sách chủ đề có lộ trình ở trình độ đó kèm
  số bài trong lộ trình (để L cho người học chọn), không thấy mô tả nội bộ nào khác.
- **FR-021**: Mọi trang của F14 PHẢI dùng được ở 360px, bằng bàn phím, sáng và tối (nguyên tắc IV).

### Key Entities

- **Chủ đề (Topic)** *(sửa 2026-10-07)*: tên (duy nhất trong danh mục), mô tả ngắn, các lộ trình theo trình độ, thời điểm tạo.
- **Bài học (Lesson, F2)**: liên kết tới một chủ đề; có trình độ riêng.
- **Lộ trình (Roadmap)** *(sửa 2026-10-07)*: thuộc một cặp (chủ đề, trình độ); danh sách bài của cặp đó, có thứ tự, không trùng.
- **Lần chuyển dữ liệu (Migration)**: đánh dấu việc chuyển dữ liệu F2 → F14 đã xong, và việc gộp chủ đề (2026-10-07) đã xong.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sau chuyển dữ liệu, 100% bài có đúng một chủ đề, số bài không đổi, và thứ tự tương đối của các bài trong mỗi lộ
  trình mới giống thứ tự trong lộ trình chung cũ.
- **SC-002**: Chạy chuyển hoặc gộp dữ liệu 2 lần cho kết quả giống hệt chạy 1 lần (0 chủ đề trùng, 0 thay đổi lộ trình).
- **SC-003**: 100% lần thử thêm bài khác chủ đề hoặc khác trình độ vào một lộ trình bị từ chối.
- **SC-004**: 100% lần thử xoá chủ đề còn bài bị từ chối; 100% lần thử tạo tên trùng bị từ chối.
- **SC-005**: Cảnh báo "dưới 3 bài chưa học" hiện đúng cho từng lộ trình ở 100% trường hợp thử (0, 1, 2, 3 bài).
- **SC-006**: Quản trị viên tạo một chủ đề mới, gán 3 bài và xếp lộ trình trong dưới 3 phút.
- **SC-007** *(sửa 2026-10-07)*: Sau khi gộp, danh mục không còn hai chủ đề trùng tên; 100% lộ trình, mục tiêu và tiến độ của người
  học giống trước khi gộp.
- **SC-008**: Ở 360px, mọi trang của F14 không cuộn ngang, dùng được hoàn toàn bằng bàn phím.

## Assumptions

- Chưa có tiến độ học (làm ở L), nên như F2 mọi bài trong lộ trình được tính là "chưa học"; khi có L, cảnh báo đếm theo tiến độ
  thật.
- Thứ tự giữa các chủ đề không quan trọng ở F14; người học tự chọn chủ đề ở L.
- Người học chọn trình độ và chủ đề, bài hôm nay theo lộ trình (chủ đề, trình độ), lời chào khi hết lộ trình thuộc L; F14 chỉ cung
  cấp danh sách chủ đề có lộ trình ở một trình độ.
- *(sửa 2026-10-07)* Một chủ đề không cần có bài ở mọi trình độ; người học chỉ thấy chủ đề ở những trình độ có lộ trình.
- Tên "Chung" không đặc biệt sau khi chuyển dữ liệu: quản trị viên đổi tên hoặc chuyển bài như chủ đề thường.
- Ngoài phạm vi: người học chọn trình độ và chủ đề, bài hôm nay theo chủ đề (L); chủ đề nhiều cấp; ảnh đại diện cho chủ đề; sắp
  xếp thủ công thứ tự các chủ đề.
