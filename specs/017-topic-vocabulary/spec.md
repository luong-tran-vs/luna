# Feature Specification: Từ vựng theo chủ đề

**Feature Branch**: `017-topic-vocabulary`

**Created**: 2026-10-02

**Status**: Draft

**Mã tính năng**: F18 | **Giai đoạn**: Giai đoạn 3 (`docs/phases/giai-doan-3.md`, tính năng quản trị)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f18-tu-vung-chu-de.md"

**Sửa 2026-10-07 — chủ đề dùng chung cho mọi trình độ** (`specs/007-topic-roadmaps`): một chủ đề có bài ở nhiều trình độ, nên mỗi
từ của chủ đề có thêm **trình độ** (A1–C2) hoặc để trống (dùng cho mọi trình độ). Khi sinh bài trình độ X, app chỉ giao những từ có
trình độ X trở xuống hoặc để trống. Từ của các chủ đề cũ cùng tên được gộp. Các chỗ thay đổi được đánh dấu *(sửa 2026-10-07)*.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Danh sách từ vựng của chủ đề và độ phủ (Priority: P1)

Mỗi chủ đề có một danh sách từ vựng tiếng Anh cốt lõi (ví dụ Gia đình: family, parents, father, mother…), mỗi từ có thể gắn một
trình độ *(sửa 2026-10-07)*. Lần khởi động đầu, các chủ đề có tên trùng với file dữ liệu ban đầu (42 chủ đề, 1.257 từ) tự nhận
danh sách. Ở trang **Chủ đề**, mỗi chủ đề hiện
"Từ vựng: đã dùng X/Y". Quản trị viên mở một chủ đề để xem danh sách, biết từ nào đã có trong bài, thêm nhiều từ một lần và xoá từ.

**Why this priority**: Là nền cho mọi phần còn lại: không có danh sách từ thì không chia được từ khi sinh bài, cũng không biết lộ trình
đã phủ được bao nhiêu từ.

**Independent Test**: Khởi động app lần đầu với chủ đề "Gia đình": chủ đề có danh sách từ. Trang Chủ đề hiện "Từ vựng: đã dùng X/Y"
đúng với các bài đang có. Thêm "cousin, nephew" và một từ trùng: hai từ mới được lưu, từ trùng bị báo lỗi ngay tại từ đó. Khởi động
lại: danh sách giữ nguyên như đã sửa.

**Acceptance Scenarios**:

1. **Given** lần khởi động đầu tiên sau khi có F18, **When** app chạy xong, **Then** mỗi chủ đề có tên trùng với một chủ đề trong file
   dữ liệu (không phân biệt hoa/thường, "khoẻ" = "khỏe") nhận đúng danh sách từ đó; chủ đề không trùng có danh sách rỗng.
2. **Given** quản trị viên đã sửa hoặc xoá hết danh sách của một chủ đề, **When** khởi động lại app, **Then** danh sách giữ nguyên như
   đã sửa, không bị nạp lại.
3. **Given** trang Chủ đề, **When** xem một chủ đề có danh sách từ, **Then** thấy "Từ vựng: đã dùng X/Y" với Y là số từ trong danh sách
   và X là số từ có trong ít nhất một bài của chủ đề.
4. **Given** chủ đề có bài chứa "Parents", "grandmothers" và cụm "take a shower", **When** xem danh sách, **Then** "parents",
   "grandmother", "take a shower" có nhãn "Đã dùng"; từ không có trong bài nào có nhãn "Chưa dùng" (có chữ, không chỉ màu).
5. **Given** ô thêm từ, **When** dán "cousin, nephew" hoặc mỗi dòng một từ rồi bấm Lưu, **Then** các từ hợp lệ được thêm, khoảng trắng
   thừa được bỏ.
6. **Given** từ thêm vào trùng với từ đã có (không phân biệt hoa/thường), dài quá 40 ký tự hoặc có ký tự không cho phép, **When** bấm
   Lưu, **Then** không lưu gì và mỗi từ lỗi được báo lỗi riêng.
7. **Given** danh sách đã có 300 từ, **When** thêm từ mới, **Then** báo lỗi "Tối đa 300 từ" (trước 2026-10-02 là 100).
8. **Given** một từ trong danh sách, **When** xoá rồi Lưu, **Then** từ biến mất khỏi danh sách; các bài đã có không thay đổi.
9. *(sửa 2026-10-07)* **Given** ô thêm từ có ô chọn trình độ (mặc định "Mọi trình độ"), **When** chọn A2 rồi thêm "cousin, nephew",
   **Then** hai từ được lưu với trình độ A2; danh sách hiện nhãn trình độ cạnh mỗi từ ("Mọi trình độ" khi để trống) và đổi được
   trình độ của từng từ.
10. *(sửa 2026-10-07)* **Given** danh sách có từ ở nhiều trình độ, **When** chọn lọc "A2", **Then** chỉ thấy từ A2; số "đã dùng X/Y"
    của bộ lọc tính trên các từ đang thấy.

---

### User Story 2 - Sinh bài bằng AI theo từ mục tiêu (Priority: P1)

Khi quản trị viên sinh bài bằng AI cho một chủ đề có từ vựng, hộp thoại có thêm mục "Từ mục tiêu mỗi bài". App chia sẵn cho từng bài
một nhóm từ trong các từ hợp trình độ đang sinh *(sửa 2026-10-07)*, ưu tiên từ chưa dùng, rồi đến từ dùng ít nhất; các bài trong
một lượt không trùng từ khi còn đủ từ. Quản trị viên bỏ hoặc
thêm từ cho từng bài rồi bấm Sinh. AI phải dùng các từ được giao. Mỗi bản nháp báo đã dùng bao nhiêu từ mục tiêu và còn thiếu từ nào.

**Why this priority**: Đây là mục tiêu chính của F18: học hết lộ trình của chủ đề là gặp đủ từ vựng của chủ đề.

**Independent Test**: Chủ đề có 40 từ, 10 từ đã dùng. Mở Sinh bài bằng AI, chọn 3 bài, 8 từ mỗi bài: thấy 3 nhóm 8 từ, không trùng
nhau, toàn từ chưa dùng. Bỏ một từ ở bài 1, thêm một từ khác. Bấm Sinh: chỉ 1 request AI; mỗi bản nháp hiện "Dùng a/b từ mục tiêu"
và "Còn thiếu: …" nếu có.

**Acceptance Scenarios**:

1. **Given** chủ đề có danh sách từ, **When** mở hộp thoại sinh bài, **Then** có trường "Từ mục tiêu mỗi bài" với mặc định 8 (A1–A2),
   10 (B1–B2), 12 (C1–C2), chọn được từ 0 đến 15.
2. **Given** số bài và số từ mỗi bài, **When** hộp thoại hiện nhóm từ, **Then** mỗi bài có một nhóm; từ chưa dùng được chọn trước, rồi đến
   từ dùng ít nhất; các nhóm không trùng nhau khi danh sách còn đủ từ.
3. **Given** danh sách không đủ từ cho mọi bài, **When** chia từ, **Then** mỗi bài vẫn có đủ số từ nếu danh sách có đủ (được lặp lại từ
   dùng ít nhất giữa các bài), hoặc nhận toàn bộ danh sách nếu danh sách ít hơn số từ mỗi bài.
4. **Given** nhóm từ của một bài, **When** quản trị viên bỏ một từ hoặc thêm một từ của chủ đề (có gợi ý từ danh sách), **Then** nhóm
   cập nhật; không thêm được từ ngoài danh sách của chủ đề hay từ đã có trong nhóm.
5. **Given** đổi số bài, số từ mỗi bài hoặc trình độ, **When** hộp thoại tính lại, **Then** các nhóm được chia lại.
9. *(sửa 2026-10-07)* **Given** chủ đề có từ A1, từ B2 và từ để trống trình độ, **When** sinh bài A2, **Then** nhóm từ chỉ gồm từ A1,
   A2 và từ để trống; từ B2 không được đề xuất và không thêm tay được. Từ đúng trình độ A2 chưa dùng được chọn trước từ trình độ thấp
   hơn hoặc để trống.
10. *(sửa 2026-10-07)* **Given** bấm "Bổ sung bằng AI" khi đang sinh bài A2, **When** AI trả từ, **Then** các từ mới được lưu với
    trình độ A2.
6. **Given** bấm Sinh, **When** AI trả về, **Then** cả lượt chỉ tốn 1 request AI, và mỗi bản nháp hiện "Dùng a/b từ mục tiêu" kèm
   "Còn thiếu: …" khi có từ không xuất hiện trong bài. Bản nháp thiếu từ không bị loại.
7. **Given** chọn 0 từ mục tiêu, hoặc chủ đề chưa có từ vựng, **When** sinh bài, **Then** sinh như hiện nay, không có nhóm từ và
   không có dòng "từ mục tiêu" trên bản nháp.
8. **Given** AI lỗi, **When** sinh bài, **Then** báo lỗi như hiện nay; hộp thoại giữ nguyên số bài, các lựa chọn và nhóm từ để thử lại.

---

### User Story 3 - Chú thích đưa từ của chủ đề vào từ vựng của bài (Priority: P2)

Khi chú thích một bài thuộc chủ đề có từ vựng, các từ của chủ đề xuất hiện trong bài luôn có mặt trong danh sách từ vựng của bài
(cùng các từ khác như hiện nay). Nhờ vậy bước Đọc, sổ từ và trang chi tiết bài (F17) xoay quanh đúng các từ đó.

**Why this priority**: Làm tăng giá trị của US1, US2 cho người học, nhưng bài vẫn dùng được nếu thiếu.

**Independent Test**: Tạo bài trong chủ đề Gia đình có "father", "grandmother" và "take a shower" (đều có trong danh sách). Chú thích
xong: danh sách từ vựng của bài có cả ba từ, kèm các từ khác AI chọn. Vẫn 1 request AI.

**Acceptance Scenarios**:

1. **Given** bài thuộc chủ đề có từ vựng, **When** chú thích, **Then** AI được gửi kèm các từ của chủ đề có trong bài và vẫn chỉ 1
   request AI cho mỗi lần chú thích.
2. **Given** chú thích xong, **When** xem danh sách từ vựng của bài, **Then** mọi từ của chủ đề có trong bài đều có mặt, kèm nghĩa tiếng
   Việt.
3. **Given** số từ của chủ đề trong bài vượt giới hạn số chú thích, **When** chú thích, **Then** từ của chủ đề được ưu tiên giữ.
4. **Given** bài thuộc chủ đề chưa có từ vựng, **When** chú thích, **Then** như hiện nay.

---

### Edge Cases

- Từ có dấu nháy, gạch nối hoặc "/" ("o'clock", "T-shirt", "and/or"): lưu được; khớp trong bài theo nguyên chuỗi đó.
- Từ trong danh sách viết hoa ("Family"): so khớp và kiểm tra trùng không phân biệt hoa/thường; hiển thị như đã lưu.
- Từ chỉ khớp một phần từ khác ("son" trong "season"): không tính là đã dùng.
- Dạng biến đổi đơn giản ("grandmothers", "watched", "watching") tính là đã dùng; dạng bất quy tắc ("went" cho "go") chỉ tính khi chú
  thích của bài cho dạng gốc đó.
- Bài đã chuyển sang chủ đề khác: chỉ tính vào chủ đề hiện tại của bài.
- *(sửa 2026-10-07)* Từ dùng trong bài ở bất kỳ trình độ nào của chủ đề đều tính là đã dùng, kể cả bài thấp hơn trình độ của từ.
- *(sửa 2026-10-07)* Gộp hai chủ đề cũ cùng tên mà cùng có một từ ở hai trình độ: giữ một từ, trình độ thấp hơn.
- *(sửa 2026-10-07)* Sinh bài A1 khi chỉ có từ B1: không có nhóm từ, hộp thoại báo "Chưa có từ A1" và mời bổ sung bằng AI.
- Bài chưa có trong lộ trình nhưng thuộc chủ đề: vẫn tính vào độ phủ.
- Đổi tên chủ đề sau khi đã nạp: danh sách không đổi. Chủ đề tạo mới mà trùng tên trong file dữ liệu: nhận danh sách ở lần khởi động
  kế tiếp nếu chưa từng được nạp hay sửa.
- Từ mục tiêu bị xoá khỏi danh sách trong lúc hộp thoại đang mở: khi Sinh bị báo lỗi tại từ đó, nhóm từ được tính lại.
- AI dùng từ mục tiêu ở dạng biến đổi ("brothers" cho "brother"): tính là đã dùng.
- Ở 360px, sáng và tối: danh sách từ, nhóm từ và nút xoá không cuộn ngang; mọi thao tác dùng được bằng bàn phím.

## Requirements *(mandatory)*

### Functional Requirements

**Danh sách từ**

- **FR-001**: Mỗi chủ đề MUST có một danh sách từ tiếng Anh (từ đơn hoặc cụm), tối đa 300 từ (đổi từ 100 ngày 2026-10-02); mỗi từ tối đa 40 ký tự, chỉ gồm chữ
  cái tiếng Anh, khoảng trắng, dấu gạch nối, dấu nháy đơn và "/"; không trùng trong một chủ đề (không phân biệt hoa/thường). Từ được
  bỏ khoảng trắng đầu cuối và gộp khoảng trắng giữa. *(Sửa 2026-10-07)* Mỗi từ có trình độ A1–C2 hoặc để trống (= mọi trình độ);
  quản trị viên chọn trình độ khi thêm và đổi được cho từng từ; danh sách lọc được theo trình độ.
- **FR-002**: Danh sách MUST chỉ lưu từ tiếng Anh; nghĩa và phiên âm vẫn lấy từ chú thích bài và từ điển của app.
- **FR-003** *(sửa 2026-10-07: từ nạp từ file dữ liệu để trống trình độ)*: Khi khởi động, mỗi chủ đề chưa từng được nạp mà tên trùng một chủ đề trong file dữ liệu ban đầu (so tên đã chuẩn hoá,
  không phân biệt hoa/thường và cách đặt dấu thanh) MUST nhận danh sách từ đó, đúng một lần. Danh sách đã được nạp hoặc đã được quản trị
  viên sửa (kể cả xoá hết) MUST NOT bị ghi đè.
- **FR-004**: Quản trị viên MUST xem, thêm (nhiều từ một lần: mỗi dòng một từ hoặc cách nhau bằng dấu phẩy) và xoá từ của một chủ đề.
  Lưu không hợp lệ MUST không thay đổi gì và báo lỗi theo từng từ.
- **FR-005**: Xoá hoặc đổi từ trong danh sách MUST NOT thay đổi bài đã có.

**Độ phủ**

- **FR-006**: Một từ MUST được coi là "đã dùng" khi xuất hiện trong nội dung của ít nhất một bài thuộc chủ đề: khớp nguyên từ hoặc
  nguyên cụm, không phân biệt hoa/thường, tính cả dạng số nhiều và chia động từ đơn giản (thêm s, es, ed, ing) và dạng có trong chú
  thích của bài với cùng dạng gốc.
- **FR-007**: Trang Chủ đề MUST hiện "Từ vựng: đã dùng X/Y" cho mỗi chủ đề có danh sách từ; danh sách từ của chủ đề MUST đánh dấu
  "Đã dùng"/"Chưa dùng" bằng chữ, không chỉ bằng màu.
- **FR-008**: Hệ thống MUST biết mỗi từ được dùng ở bao nhiêu bài của chủ đề, để chia từ ưu tiên từ dùng ít nhất.

**Sinh bài**

- **FR-009**: Hộp thoại Sinh bài bằng AI của chủ đề có từ vựng MUST có trường "Từ mục tiêu mỗi bài" (0–15), mặc định 8 (A1–A2),
  10 (B1–B2), 12 (C1–C2).
- **FR-010** *(sửa 2026-10-07)*: Khi sinh bài trình độ X, hệ thống MUST chỉ xét các từ có trình độ ≤ X hoặc để trống. Hệ thống MUST
  đề xuất nhóm từ cho từng bài: theo thứ tự từ chưa dùng (từ đúng trình độ X trước, rồi từ trình độ thấp hơn hoặc để trống), rồi
  số bài đã dùng tăng dần, rồi thứ tự trong danh sách; các bài trong một lượt không trùng từ khi danh sách đủ từ; mỗi nhóm có tối đa số từ đã chọn.
- **FR-011**: Quản trị viên MUST bỏ hoặc thêm từ cho từng nhóm trước khi sinh; chỉ được thêm từ thuộc danh sách của chủ đề và hợp trình
  độ đang sinh (theo FR-010), không trùng trong nhóm, tối đa 15 từ mỗi nhóm. Hệ thống MUST từ chối nhóm có từ không thuộc danh sách (báo lỗi theo từ).
- **FR-012**: AI MUST được yêu cầu dùng mọi từ của nhóm trong bài tương ứng; cả lượt sinh vẫn chỉ 1 request AI.
- **FR-013**: Mỗi bản nháp MUST hiện "Dùng a/b từ mục tiêu" (theo cách khớp của FR-006) và liệt kê từ còn thiếu; bản nháp MUST NOT bị
  loại vì thiếu từ.
- **FR-014**: Với 0 từ mục tiêu hoặc chủ đề chưa có từ vựng, sinh bài MUST như hiện nay.
- **FR-015**: AI lỗi khi sinh bài MUST báo lỗi như hiện nay và giữ nguyên các lựa chọn, nhóm từ trong hộp thoại.
- **FR-024** *(mới 2026-10-07)*: Từ do "Bổ sung bằng AI" thêm MUST mang trình độ đang sinh; AI MUST được yêu cầu gợi ý từ hợp trình độ đó.

**Gộp chủ đề** *(mới 2026-10-07)*

- **FR-025** *(mới 2026-10-07)*: Khi gộp các chủ đề cũ cùng tên (F14, FR-022), danh sách từ MUST được gộp: mỗi từ mang trình độ của
  chủ đề cũ chứa nó; từ có ở nhiều chủ đề cũ giữ một lần, trình độ thấp nhất, theo thứ tự của chủ đề ở trình độ thấp nhất rồi các
  chủ đề kế tiếp; cờ "đã nạp dữ liệu ban đầu" bật nếu một chủ đề cũ đã bật. Danh sách gộp vượt 300 từ thì giữ đủ, chỉ chặn thêm từ
  mới.

**Chú thích**

- **FR-016**: Khi chú thích bài thuộc chủ đề có từ vựng, AI MUST được gửi kèm các từ của chủ đề có trong bài (theo FR-006) và được yêu
  cầu đưa các từ đó vào danh sách từ vựng của bài; vẫn 1 request AI mỗi lần chú thích.
- **FR-017**: Khi số chú thích vượt giới hạn hiện có, từ của chủ đề MUST được ưu tiên giữ.
- **FR-018**: Từ của chủ đề có trong bài mà AI bỏ sót MUST được thêm vào danh sách từ vựng của bài với nghĩa từ từ điển của app; từ điển
  không có nghĩa thì bỏ qua từ đó.

**Chung**

- **FR-019**: F18 MUST NOT thay đổi luồng học, tiến độ, streak, thống kê; người học không thấy danh sách từ của chủ đề.
- **FR-020**: Mọi thao tác quản trị của F18 MUST dùng được bằng bàn phím, ở 360px, chế độ sáng và tối.

### Key Entities

- **Danh sách từ của chủ đề**: thuộc một chủ đề; các từ tiếng Anh theo thứ tự quản trị viên nhập, mỗi từ có trình độ hoặc để trống
  *(sửa 2026-10-07)*; cờ "đã nạp dữ liệu ban đầu" để không nạp lại.
- **Độ phủ từ**: với mỗi từ của chủ đề: đã dùng hay chưa, số bài của chủ đề có dùng từ đó (tính khi xem, không lưu).
- **Nhóm từ mục tiêu**: các từ giao cho một bài trong một lượt sinh; đi kèm bản nháp với các từ đã dùng và còn thiếu.
- **Dữ liệu ban đầu**: 42 chủ đề, 1.257 từ tiếng Anh (`docs/spec-inputs/f18-topic-words.json`), đóng gói cùng app.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sau lần khởi động đầu, 100% chủ đề trùng tên trong file dữ liệu có danh sách từ; sau khi khởi động lại, 0 danh sách đã sửa
  bị thay đổi.
- **SC-002**: Độ phủ khớp kiểm tra thủ công trên một chủ đề mẫu (cụm từ, hoa/thường, số nhiều) ở 100% từ.
- **SC-003**: Sinh 3 bài với 8 từ mỗi bài trên chủ đề còn ≥ 24 từ chưa dùng: 3 nhóm không trùng nhau, toàn từ chưa dùng, đúng 1 request AI.
- **SC-004**: Với AI hoạt động, trung bình mỗi bản nháp dùng ít nhất 80% từ mục tiêu.
- **SC-005**: 100% bài mới chú thích xong có trong danh sách từ vựng mọi từ của chủ đề xuất hiện trong bài mà từ điển hoặc AI cho được
  nghĩa.
- **SC-006**: Quản trị viên thêm 20 từ (dán một lần) và lưu trong dưới 1 phút.
- **SC-007**: Tiến độ, streak, thống kê của người học không đổi sau khi bật F18.

## Assumptions

- File dữ liệu ban đầu chỉ chứa từ tiếng Anh lấy tham khảo từ danh sách công khai của Langmaster; không chép phiên âm hay nghĩa.
- "Bài thuộc chủ đề" là mọi bài có chủ đề đó, kể cả bài chưa đưa vào lộ trình.
- Chủ đề tạo mới chưa từng được nạp hay sửa danh sách thì cũng được xét nạp ở lần khởi động kế tiếp; quản trị viên xoá được nếu không
  muốn.
- Từ đã thêm giữ nguyên cách viết hoa của quản trị viên; trùng được xét không phân biệt hoa/thường.
- Độ phủ và số bài dùng mỗi từ được tính khi xem, từ nội dung và chú thích hiện tại của các bài.
- Nhóm từ đề xuất có thể khác nhau giữa các lần mở hộp thoại nếu bài trong chủ đề thay đổi.
- Dùng AI, từ điển đã cấu hình (F2, F3, F7); không thêm cấu hình.
- Đổi 2026-10-02: chủ đề thiếu từ chưa dùng thì hộp thoại sinh bài có nút "Bổ sung bằng AI" (1 request AI riêng, tối đa 50 từ mỗi
  lần, chỉ thêm từ hợp lệ và chưa có); danh sách tối đa 300 từ; sinh tối đa 10 bài mỗi lượt.
- Thêm 2026-10-07: 24 chủ đề bổ sung (chào hỏi, thời gian, nhà cửa, cơ thể, sở thích, thành phố, nhà hàng, âm nhạc, công nghệ,
  giao tiếp, các mối quan hệ, lễ kỷ niệm, khách sạn và sân bay, thiên nhiên, sách, nấu ăn, báo chí, khoa học, nghệ thuật, lịch sử,
  giáo dục đại học, kinh doanh, pháp luật, xã hội) kèm từ có trình độ (`docs/spec-inputs/f18-extra-topics.json`), được tạo đúng một
  lần khi khởi động nếu chưa có chủ đề cùng tên; xoá sau đó thì không tạo lại. Danh sách từ do Claude soạn, không lấy từ nguồn ngoài.
- Ngoài phạm vi: thẻ ôn tập theo chủ đề cho người học, người học xem danh sách từ của chủ đề, nhập phiên âm/nghĩa cho từ của chủ đề,
  nhập/xuất danh sách từ bằng file, tự động sinh bài cho đủ từ.
- Phụ thuộc: F14 (chủ đề, lộ trình), F7 (sinh bài), F2 (chú thích), F3 (từ điển); F17 dùng ngay danh sách từ vựng của bài.
