# Feature Specification: Trang quản trị bài học

**Feature Branch**: `003-lesson-admin`

**Created**: 2026-09-29

**Status**: Draft

**Mã tính năng**: F2 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f2-quan-tri-bai-hoc.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tạo bài học và xem trước (Priority: P1)

Quản trị viên mở trang quản trị, bấm "Thêm bài", dán văn bản tiếng Anh, nhập tiêu đề, chọn trình độ CEFR
(A1–C2), nhập chủ đề, nguồn và giấy phép rồi lưu. Bài xuất hiện trong danh sách; mở ra thấy bài đã được
tách thành từng câu.

**Why this priority**: Không có bài thì không có gì để học; mọi tính năng học (F3, F4, L) đều cần bài.

**Independent Test**: Tạo một bài 5 câu, kiểm tra bài có trong danh sách và trang xem trước hiện đúng 5 câu
theo thứ tự.

**Acceptance Scenarios**:

1. **Given** quản trị viên đang ở trang quản trị, **When** điền đủ thông tin hợp lệ và lưu, **Then** bài được
   lưu, xuất hiện đầu danh sách với tiêu đề, trình độ, chủ đề và trạng thái audio, chú thích.
2. **Given** vừa lưu bài "Mr. Smith arrived at 9.30 a.m. He was late! Why?", **When** mở xem trước, **Then** thấy
   đúng 3 câu: "Mr. Smith arrived at 9.30 a.m.", "He was late!", "Why?".
3. **Given** form thiếu tiêu đề, nội dung, trình độ, nguồn hoặc giấy phép, **When** bấm lưu, **Then** thấy lỗi
   tiếng Việt cạnh ô còn thiếu và bài không được lưu.
4. **Given** danh sách có bài ở nhiều trình độ và chủ đề, **When** lọc theo trình độ B1 hoặc theo một chủ đề,
   **Then** chỉ thấy các bài khớp bộ lọc.

---

### User Story 2 - Audio và chú thích tự động chạy nền (Priority: P1)

Sau khi lưu, hệ thống tự làm nền: tạo audio đọc từng câu và nhờ AI chú thích các từ, cụm từ đáng học (dạng
gốc, nghĩa tiếng Việt theo ngữ cảnh). Danh sách bài hiện trạng thái của từng việc (đang chạy, xong, lỗi) và tự
cập nhật; việc bị lỗi có nút "Chạy lại".

**Why this priority**: Audio cần cho bước Nghe (F4), chú thích cần cho bước Đọc (F3). Việc chạy nền và cho phép
chạy lại là yêu cầu của nguyên tắc III (AI là phần bổ sung).

**Independent Test**: Tạo bài khi AI hoạt động → cả hai trạng thái chuyển sang "xong", nghe được từng câu. Tắt
AI (hoặc không cấu hình) → tạo bài vẫn lưu được, chú thích "lỗi"; bật lại AI và bấm "Chạy lại" → "xong".

**Acceptance Scenarios**:

1. **Given** vừa lưu bài, **When** xem danh sách, **Then** trạng thái audio và chú thích là "đang chạy" và tự
   chuyển sang "xong" khi hoàn tất mà không cần tải lại trang.
2. **Given** audio đã xong, **When** mở xem trước, **Then** mỗi câu có nút nghe phát đúng câu đó.
3. **Given** audio của bài đã tạo, **When** mở bài nhiều lần, **Then** audio không bị tạo lại.
4. **Given** AI lỗi, hết hạn mức hoặc chưa được cấu hình, **When** lưu bài, **Then** bài vẫn được lưu, audio vẫn
   được tạo và trạng thái chú thích là "lỗi" kèm lý do ngắn.
5. **Given** chú thích hoặc audio đang "lỗi", **When** bấm "Chạy lại", **Then** trạng thái về "đang chạy" và
   việc đó được làm lại.
6. **Given** chú thích đã xong, **When** mở xem trước, **Then** thấy danh sách từ, cụm từ đáng học, mỗi mục có
   dạng gốc (ví dụ *went → go*) và nghĩa tiếng Việt theo ngữ cảnh.
7. **Given** lưu một bài mới, **When** hệ thống chú thích, **Then** chỉ gửi đúng một yêu cầu tới AI cho bài đó.

---

### User Story 3 - Xếp lộ trình (Priority: P2)

Quản trị viên thêm bài vào lộ trình, kéo thả để đổi thứ tự và gỡ bài khỏi lộ trình. Khi lộ trình còn dưới 3
bài chưa học, trang quản trị hiện cảnh báo để kịp thêm bài.

**Why this priority**: Lộ trình quyết định bài của mỗi ngày (R1, R2), nhưng chỉ cần khi đã có bài.

**Independent Test**: Thêm 4 bài vào lộ trình, đổi thứ tự bằng kéo thả, tải lại trang thấy giữ nguyên thứ tự;
gỡ 2 bài thì thấy cảnh báo "dưới 3 bài".

**Acceptance Scenarios**:

1. **Given** bài chưa có trong lộ trình, **When** bấm "Thêm vào lộ trình", **Then** bài được thêm vào cuối lộ
   trình.
2. **Given** lộ trình có nhiều bài, **When** kéo một bài lên vị trí đầu và thả, **Then** thứ tự mới được lưu và
   giữ nguyên sau khi tải lại trang.
3. **Given** dùng bàn phím hoặc màn hình cảm ứng nhỏ, **When** dùng nút "Lên" / "Xuống" của một bài, **Then** bài
   đổi chỗ như khi kéo thả.
4. **Given** bài đang trong lộ trình, **When** bấm "Gỡ khỏi lộ trình", **Then** bài biến mất khỏi lộ trình nhưng
   vẫn còn trong danh sách bài.
5. **Given** lộ trình còn 2 bài chưa học, **When** mở trang quản trị, **Then** thấy cảnh báo nêu số bài còn lại.
6. **Given** một bài đã có trong lộ trình, **When** thử thêm lần nữa, **Then** không bị trùng.

---

### User Story 4 - Xem và sửa chú thích (Priority: P2)

Quản trị viên xem bảng chú thích của bài, sửa dạng gốc hoặc nghĩa, thêm chú thích mới cho một từ/cụm từ trong
bài, hoặc xoá chú thích không cần.

**Why this priority**: AI có thể sai; chú thích đúng làm bước Đọc đáng tin, nhưng bài vẫn dùng được nếu chưa sửa.

**Independent Test**: Sửa nghĩa một chú thích, thêm một chú thích, xoá một chú thích; tải lại trang thấy đúng 3
thay đổi.

**Acceptance Scenarios**:

1. **Given** chú thích đã xong, **When** sửa nghĩa tiếng Việt của một mục và lưu, **Then** nghĩa mới được giữ và
   mục đó được đánh dấu "đã sửa tay".
2. **Given** đang xem chú thích, **When** thêm mục mới với cụm từ có trong bài, dạng gốc và nghĩa, **Then** mục
   được thêm và gắn với câu chứa cụm từ đó.
3. **Given** thêm mục với cụm từ không có trong bài, **When** lưu, **Then** thấy lỗi "Cụm từ không có trong bài".
4. **Given** một mục chú thích, **When** xoá, **Then** mục biến mất sau khi lưu.

---

### User Story 5 - Sửa và xoá bài (Priority: P3)

Quản trị viên sửa thông tin hoặc nội dung bài. Sửa nội dung thì tách câu, audio và chú thích được làm lại; nếu
bài có chú thích đã sửa tay thì được cảnh báo trước khi lưu. Bài không nằm trong lộ trình thì xoá được.

**Why this priority**: Cần để sửa lỗi gõ và dọn bài, nhưng không chặn luồng chính.

**Independent Test**: Sửa tiêu đề → audio, chú thích giữ nguyên; sửa nội dung → cả hai làm lại; xoá bài trong lộ
trình bị chặn, gỡ khỏi lộ trình rồi xoá được.

**Acceptance Scenarios**:

1. **Given** bài đã có audio và chú thích, **When** chỉ sửa tiêu đề, chủ đề, trình độ, nguồn hoặc giấy phép,
   **Then** câu, audio và chú thích giữ nguyên.
2. **Given** bài đã có audio và chú thích, **When** sửa nội dung và lưu, **Then** bài được tách câu lại, audio và
   chú thích về "đang chạy" và được tạo mới.
3. **Given** bài có chú thích đã sửa tay, **When** sửa nội dung và bấm lưu, **Then** thấy cảnh báo "Các chú thích
   đã sửa tay sẽ bị thay mới" và phải xác nhận mới lưu.
4. **Given** bài đang trong lộ trình, **When** bấm xoá, **Then** bị từ chối với thông báo "Gỡ bài khỏi lộ trình
   trước khi xoá".
5. **Given** bài không trong lộ trình, **When** xoá và xác nhận, **Then** bài, audio và chú thích của bài bị xoá.

---

### Edge Cases

- Sửa nội dung khi audio hoặc chú thích của nội dung cũ đang chạy: kết quả cũ bị bỏ, chỉ kết quả của nội dung
  mới được lưu.
- Máy chủ khởi động lại khi đang tạo audio hoặc chú thích: việc dang dở được làm tiếp sau khi khởi động.
- Dịch vụ tạo audio tạm lỗi: hệ thống tự thử lại vài lần có giãn cách; hết số lần thì trạng thái "lỗi".
- AI trả về kết quả sai định dạng hoặc cụm từ không có trong bài: bỏ các mục không hợp lệ; nếu không còn mục nào
  hợp lệ thì trạng thái "lỗi".
- Văn bản có dấu ba chấm, số thập phân (3.5), viết tắt (Mr., Dr., e.g., U.S.), câu trong ngoặc kép ("Stop!" she
  said.): tách câu không cắt sai ở các chỗ này.
- Văn bản có dòng trống, xuống dòng giữa câu, khoảng trắng thừa: được chuẩn hoá khi tách câu.
- Nội dung rỗng, chỉ khoảng trắng hoặc quá dài (trên 10.000 ký tự): bị từ chối với lỗi rõ ràng.
- Lộ trình trống: trang quản trị hiện cảnh báo "Lộ trình chưa có bài".
- Xoá một bài không tồn tại (đã bị xoá ở tab khác): báo không tìm thấy, danh sách được làm mới.
- Người học hoặc người chưa đăng nhập mở trang quản trị hoặc gọi thẳng dịch vụ quản trị: bị chặn như F1.
- Người chưa đăng nhập mở thẳng đường dẫn audio: bị từ chối.
- Màn hình 360px: danh sách, form, bảng chú thích và lộ trình dùng được, không cuộn ngang.

## Requirements *(mandatory)*

### Functional Requirements

**Bài học**

- **FR-001** *(ô trình độ và chủ đề gõ tự do thay bởi ô chọn chủ đề ở F14, `specs/007-topic-roadmaps`)*: Quản trị viên
  PHẢI tạo được bài với: tiêu đề (bắt buộc, ≤ 200 ký tự), nội dung tiếng Anh (bắt buộc,
  ≤ 10.000 ký tự), trình độ CEFR (bắt buộc, một trong A1, A2, B1, B2, C1, C2), chủ đề (tuỳ chọn, ≤ 60 ký tự), nguồn
  (bắt buộc) và giấy phép (bắt buộc).
- **FR-002**: Khi lưu, hệ thống PHẢI tách nội dung thành danh sách câu có thứ tự, không cắt sai ở viết tắt thông
  dụng, số thập phân, dấu ba chấm và dấu câu trong ngoặc kép.
- **FR-003**: Danh sách bài PHẢI hiện tiêu đề, trình độ, chủ đề, trạng thái audio, trạng thái chú thích, có trong
  lộ trình hay không; sắp xếp bài mới nhất trước; lọc được theo trình độ và chủ đề.
- **FR-004**: Trang xem trước PHẢI hiện thông tin bài, từng câu kèm nút nghe (khi audio xong) và bảng chú thích.
- **FR-005**: Sửa thông tin bài không đổi nội dung PHẢI giữ nguyên câu, audio và chú thích.
- **FR-006**: Sửa nội dung PHẢI tách câu lại và tạo lại audio và chú thích; nếu có chú thích đã sửa tay, giao
  diện PHẢI cảnh báo và yêu cầu xác nhận trước khi lưu.
- **FR-007**: Xoá bài PHẢI bị từ chối khi bài đang trong lộ trình; xoá thành công thì xoá luôn audio và chú thích
  của bài.

**Việc chạy nền**

- **FR-008**: Sau khi tạo bài hoặc sửa nội dung, hệ thống PHẢI tự tạo audio cho từng câu và chú thích bài ở nền;
  việc lưu bài không chờ hai việc này.
- **FR-009**: Mỗi bài PHẢI có trạng thái audio và trạng thái chú thích riêng: "đang chạy", "xong", "lỗi" (kèm lý do
  ngắn khi lỗi).
- **FR-010**: Hệ thống PHẢI tự thử lại việc lỗi tối đa 3 lần, có giãn cách tăng dần, trước khi chuyển "lỗi".
- **FR-011**: Quản trị viên PHẢI chạy lại được riêng audio hoặc riêng chú thích của một bài đang "lỗi".
- **FR-012**: Việc dang dở khi máy chủ tắt PHẢI được làm tiếp khi máy chủ khởi động lại.
- **FR-013**: Kết quả của nội dung cũ (trước khi sửa) KHÔNG được ghi đè lên bài đã sửa.
- **FR-014**: Trạng thái trên giao diện PHẢI tự cập nhật trong vòng 10 giây khi còn việc đang chạy, không cần tải
  lại trang.

**Audio**

- **FR-015**: Mỗi câu PHẢI có một file audio riêng, tạo bằng dịch vụ đọc văn bản chạy trên máy (không trả phí).
- **FR-016**: Audio đã tạo PHẢI được lưu và dùng lại; chỉ tạo lại khi nội dung bài đổi hoặc quản trị viên bấm
  "Chạy lại".
- **FR-017**: Audio chỉ tải được khi đã đăng nhập.

**Chú thích AI**

- **FR-018**: Mỗi lần tạo bài, sửa nội dung hoặc bấm "Chạy lại" chú thích, hệ thống PHẢI gửi đúng một yêu cầu tới
  AI cho cả bài.
- **FR-019**: Mỗi chú thích PHẢI gồm: từ hoặc cụm từ như trong bài, dạng gốc, nghĩa tiếng Việt theo ngữ cảnh, câu
  chứa nó, và cờ "đã sửa tay".
- **FR-020**: AI lỗi, chậm, hết hạn mức hoặc chưa cấu hình KHÔNG được làm hỏng việc lưu bài hay tạo audio; trạng
  thái chú thích là "lỗi".
- **FR-021**: Kết quả AI PHẢI được kiểm tra: bỏ mục thiếu trường hoặc có cụm từ không xuất hiện trong bài.
- **FR-022**: Quản trị viên PHẢI sửa được dạng gốc và nghĩa, thêm mục mới (cụm từ phải có trong bài), xoá mục; mục
  sửa hoặc thêm tay được đánh dấu "đã sửa tay".
- **FR-023**: Nhà cung cấp AI PHẢI đổi được bằng cấu hình, không sửa logic bài học (nguyên tắc III).

**Lộ trình**

- **FR-024**: Có đúng một lộ trình chung: danh sách bài có thứ tự, không trùng bài. *(Thay bởi F14: mỗi chủ đề một lộ
  trình, `specs/007-topic-roadmaps`.)*
- **FR-025**: Quản trị viên PHẢI thêm bài vào cuối lộ trình, gỡ bài, và đổi thứ tự bằng kéo thả hoặc nút Lên/Xuống.
- **FR-026**: Thứ tự lộ trình PHẢI được lưu ngay sau mỗi thay đổi.
- **FR-027**: Trang quản trị PHẢI cảnh báo khi lộ trình có dưới 3 bài chưa học (kể cả lộ trình trống).

**Quyền và giao diện**

- **FR-028**: Mọi trang và dịch vụ quản trị bài học chỉ dành cho quản trị viên, kiểm tra ở máy chủ (như F1).
- **FR-029**: Mọi trang quản trị PHẢI dùng tốt từ 360px, cả chế độ sáng và tối; trạng thái hiện bằng chữ và biểu
  tượng, không chỉ màu.

### Key Entities

- **Bài học (Lesson)**: tiêu đề, nội dung, trình độ CEFR, chủ đề, nguồn, giấy phép, danh sách câu, trạng thái audio,
  trạng thái chú thích, lý do lỗi, thời điểm tạo và sửa.
- **Câu (Sentence)**: thuộc một bài; thứ tự, văn bản, audio của câu.
- **Chú thích (Annotation)**: thuộc một bài; cụm từ trong bài, dạng gốc, nghĩa tiếng Việt, câu chứa nó, cờ đã sửa tay.
- **Việc nền (Background job)**: loại (audio hoặc chú thích), bài liên quan, trạng thái, số lần thử, lỗi gần nhất,
  thời điểm chạy tiếp.
- **Lộ trình (Roadmap)**: một danh sách có thứ tự các bài.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Quản trị viên tạo xong một bài (dán văn bản, điền thông tin, lưu) trong dưới 2 phút.
- **SC-002**: Với bài 20 câu, audio và chú thích xong trong vòng 5 phút trên máy không có GPU khi các dịch vụ hoạt
  động.
- **SC-003**: 100% lần AI lỗi hoặc chưa cấu hình, bài vẫn được lưu và có audio; chú thích "lỗi" và "Chạy lại" được.
- **SC-004**: Mỗi lần tạo, sửa nội dung hoặc chạy lại chú thích đúng 1 yêu cầu AI (đếm được trong log).
- **SC-005**: Mở lại bài nhiều lần không tạo thêm file audio nào.
- **SC-006**: Tách câu đúng 100% với bộ mẫu kiểm thử gồm viết tắt, số thập phân, dấu ba chấm và ngoặc kép.
- **SC-007**: 100% lần thử xoá bài đang trong lộ trình bị từ chối.
- **SC-008**: Trạng thái trên danh sách cập nhật trong vòng 10 giây sau khi việc nền đổi trạng thái.
- **SC-009**: Người học bị từ chối ở 100% lần thử vào trang hoặc dịch vụ quản trị bài học.
- **SC-010**: Mọi trang quản trị ở 360px không có cuộn ngang, ở cả hai chế độ.

## Assumptions

- Chưa có tiến độ học (làm ở L), nên ở F2 mọi bài trong lộ trình được tính là "chưa học"; khi có L, cảnh báo đếm
  theo tiến độ thật. Bộ lọc theo trạng thái học (chưa học, đang học, đã xong) trong `mvp-features.md` cũng thêm ở L.
- Quy tắc xoá dùng khối 1: chặn xoá mọi bài đang trong lộ trình (chặt hơn "đang học dở" trong `giai-doan-1.md`, nên
  vẫn thoả tiêu chí nghiệm thu).
- Một lộ trình chung cho mọi người học; nhiều lộ trình nằm ngoài phạm vi.
- Được thêm bài vào lộ trình khi audio hoặc chú thích chưa xong; danh sách lộ trình hiện trạng thái để quản trị viên
  thấy.
- Chỉ đổi nội dung mới làm lại câu, audio và chú thích; đổi trình độ không làm lại chú thích.
- Giọng đọc: một giọng tiếng Anh cố định cho mọi bài.
- Nút "Chạy lại" chỉ có khi việc đang "lỗi", để không ghi đè chú thích tốt (có thể đã sửa tay) và không tốn hạn
  mức AI.
- Hai quản trị viên sửa cùng lúc không được xử lý riêng (dự án một người): lần lưu sau thắng.
- Ngoài phạm vi: AI sinh bài (giai đoạn 2), nhập bài từ URL, nhiều lộ trình, tải file audio lên tay, đổi giọng đọc.
