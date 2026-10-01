# Feature Specification: Hỏi AI về từ

**Feature Branch**: `015-ask-ai-word`

**Created**: 2026-10-01

**Status**: Draft

**Mã tính năng**: F9 | **Giai đoạn**: Giai đoạn 2 (`docs/phases/giai-doan-2.md`)

**Input**: User description: "tạo spec theo khối 1 trong @..\docs\spec-inputs\f9-hoi-ai.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Hỏi AI nghĩa theo ngữ cảnh trong popup tra từ (Priority: P1)

Khi tra một từ hoặc cụm **không có** trong chú thích của bài, popup hiện nghĩa từ điển hoặc "Chưa có nghĩa". Lúc đó popup có thêm
nút **Hỏi AI**. Bấm nút thì popup hiện trạng thái đang hỏi. Sau đó popup hiện:

- nghĩa tiếng Việt theo đúng câu đang đọc;
- dạng gốc của từ;
- một câu giải thích ngắn;
- nhãn "AI · theo ngữ cảnh".

Người học lưu vào sổ từ với nghĩa AI vừa trả về.

**Why this priority**: Đây là toàn bộ giá trị của F9. Từ điển không cho nghĩa theo ngữ cảnh, còn tự nhập nghĩa thì người học phải
tự đoán.

**Independent Test**: Ở bước Đọc của một bài, tra một cụm không có trong chú thích (ví dụ "make up for"). Bấm **Hỏi AI**. Popup phải
hiện nghĩa theo câu, dạng gốc, giải thích và nhãn "AI · theo ngữ cảnh". Bấm **Lưu vào sổ từ**: sổ từ có thẻ với đúng nghĩa đó.

**Acceptance Scenarios**:

1. **Given** từ tra được bằng chú thích của bài (nhãn "AI"), **When** popup mở, **Then** không có nút Hỏi AI.
2. **Given** từ chỉ có nghĩa từ điển hoặc "Chưa có nghĩa", **When** popup mở, **Then** có nút **Hỏi AI**. Nghĩa từ điển và ô tự nhập
   nghĩa vẫn hiện như F3.
3. **Given** bấm Hỏi AI, **When** đang chờ, **Then** popup báo "Đang hỏi AI…" và nút bị khoá (bấm thêm không gửi yêu cầu nữa).
4. **Given** AI trả lời, **When** hiện kết quả, **Then** popup có nghĩa tiếng Việt hợp với câu đang đọc, dạng gốc, một câu giải thích
   ngắn và nhãn "AI · theo ngữ cảnh". Kết quả này thay cho nghĩa từ điển đang hiện.
5. **Given** đã có kết quả AI, **When** bấm Lưu vào sổ từ, **Then** thẻ được lưu với dạng gốc và nghĩa AI, kèm câu đang đọc như F3.
6. **Given** chưa bấm Hỏi AI, **When** tra từ, đóng popup hay lưu bằng nghĩa từ điển, **Then** không có yêu cầu AI nào được gửi.

---

### User Story 2 - Dùng lại kết quả đã hỏi (Priority: P1)

Kết quả AI được lưu theo bài và theo câu. Tra lại cùng từ trong cùng câu của cùng bài thì hiện ngay kết quả đã lưu, không gọi AI nữa.
Điều này đúng với cả tài khoản khác.

**Why this priority**: Đây là tiêu chí nghiệm thu của F9. Gói AI miễn phí có hạn mức thấp, nên mỗi từ trong một câu chỉ nên tốn một
lần hỏi.

**Independent Test**:

1. Người học A hỏi AI "make up for" ở câu 3 của bài X.
2. Người học B mở bài X, tra "make up for" ở câu 3: thấy ngay kết quả AI, không có nút đang hỏi, nhật ký không có lần gọi AI mới.
3. Tra cùng cụm ở câu 5: vẫn có nút Hỏi AI.

**Acceptance Scenarios**:

1. **Given** từ đã được hỏi AI trong câu này của bài, **When** bất kỳ người học nào tra lại đúng từ đó trong đúng câu đó, **Then** popup
   hiện ngay kết quả đã lưu với nhãn "AI · theo ngữ cảnh". Không có nút Hỏi AI và không gọi AI.
2. **Given** từ đã được hỏi ở câu khác của bài, **When** tra từ đó ở câu này, **Then** coi như chưa hỏi: vẫn có nút Hỏi AI.
3. **Given** quản trị viên sửa nội dung bài, **When** tra lại, **Then** kết quả AI cũ không còn dùng. Nghĩa có thể đã khác vì câu đã
   đổi, nên lại có nút Hỏi AI.
4. **Given** hai lần bấm Hỏi AI gần như cùng lúc (hai tab, hai người) cho cùng từ, cùng câu, **When** AI đang trả lời, **Then** chỉ
   có một lần gọi AI và cả hai đều nhận cùng kết quả.
5. **Given** cùng từ nhưng khác chữ hoa/thường hoặc có khoảng trắng thừa ("Make  up for" và "make up for"), **When** tra trong cùng
   câu, **Then** dùng chung một kết quả.

---

### User Story 3 - AI lỗi không làm hỏng popup (Priority: P2)

Khi AI chưa cấu hình, hết lượt, lỗi hoặc trả về nội dung không dùng được, popup báo rõ ràng. Người học vẫn dùng được nghĩa từ điển
hoặc tự nhập nghĩa để lưu.

**Why this priority**: Theo nguyên tắc "AI là phần bổ sung" của dự án. Chỉ quan trọng sau khi US1 chạy được.

**Independent Test**: Tắt AI rồi bấm Hỏi AI. Popup phải báo "AI chưa được cấu hình", vẫn có nghĩa từ điển và ô tự nhập, và lưu thẻ
được bình thường.

**Acceptance Scenarios**:

1. **Given** AI chưa cấu hình, **When** bấm Hỏi AI, **Then** hiện thông báo AI chưa được cấu hình ngay dưới nút. Nghĩa từ điển hoặc ô tự
   nhập vẫn dùng được.
2. **Given** AI hết lượt, **When** bấm Hỏi AI, **Then** báo đã hết lượt AI, nên thử lại sau. Nút bấm lại được.
3. **Given** AI lỗi khác hoặc trả về thiếu nghĩa, **When** bấm Hỏi AI, **Then** báo "Không hỏi được AI, vui lòng thử lại". Không lưu
   gì vào kết quả dùng chung.
4. **Given** mất mạng, **When** bấm Hỏi AI, **Then** báo lỗi kết nối và nút bấm lại được.

---

### Edge Cases

- Từ hoặc cụm dài quá giới hạn tra từ của F3 (100 ký tự, 6 từ): không có nút Hỏi AI, vì F3 đã chặn từ trước.
- Từ không có trong câu đang đọc (gửi yêu cầu sai): bị từ chối, không gọi AI.
- Bài không tồn tại hoặc người học không được mở bài (khoá theo L): bị từ chối như các yêu cầu khác của bài.
- Đóng popup trong lúc đang hỏi: yêu cầu vẫn chạy và kết quả vẫn được lưu cho lần sau. Popup đã đóng thì không hiện gì.
- Đổi sang từ khác trong lúc đang hỏi: kết quả của từ cũ không hiện cho từ mới.
- AI trả về dạng gốc rỗng: dùng chính từ đang tra làm dạng gốc.
- Người học đã có thẻ của từ đó: lưu xử lý như F3 (báo đã có trong sổ từ).
- Xem lại bài (chế độ xem lại) và trang Bài học: tra từ và Hỏi AI dùng được như ở bước Đọc.
- Ở 360px: nút Hỏi AI, kết quả và thông báo lỗi không cuộn ngang, nút ≥ 44px. Trạng thái đang hỏi và kết quả được thông báo cho
  trình đọc màn hình.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Popup tra từ MUST có nút **Hỏi AI** khi nghĩa đang hiện không đến từ AI, tức là đến từ từ điển hoặc chưa có nghĩa. Khi
  nghĩa lấy từ chú thích của bài hoặc từ kết quả AI đã lưu, popup MUST NOT có nút này.
- **FR-002**: Hệ thống MUST chỉ gọi AI khi người học bấm Hỏi AI. Mỗi lần bấm tối đa một lần gọi AI. Tra từ, mở hay đóng popup và lưu
  thẻ MUST NOT gọi AI.
- **FR-003**: Trong lúc hỏi, popup MUST hiện "Đang hỏi AI…" và khoá nút Hỏi AI.
- **FR-004**: Kết quả AI MUST gồm:
  - nghĩa tiếng Việt hợp với câu đang đọc;
  - dạng gốc của từ (rỗng thì dùng chính từ đang tra);
  - một câu giải thích ngắn tiếng Việt.

  Kết quả thiếu nghĩa MUST được coi là lỗi.
- **FR-005**: Popup MUST hiện kết quả AI với nhãn "AI · theo ngữ cảnh". Lưu vào sổ từ khi đang hiện kết quả này MUST dùng dạng gốc và
  nghĩa của kết quả AI.
- **FR-006**: Kết quả AI MUST được lưu dùng chung cho mọi người học, theo bộ khoá: bài, phiên bản nội dung bài, câu, từ (không phân biệt
  hoa thường, đã gộp khoảng trắng).
- **FR-007**: Khi tra một từ đã có kết quả AI lưu cho đúng bài, đúng phiên bản và đúng câu, popup MUST hiện ngay kết quả đã lưu, không
  gọi AI. Bấm Hỏi AI cho đúng bộ khoá đó (ví dụ từ tab cũ) MUST trả kết quả đã lưu.
- **FR-008**: Khi nhiều yêu cầu Hỏi AI cùng một bộ khoá đến gần như cùng lúc, hệ thống MUST chỉ gọi AI một lần và trả cùng kết quả cho
  tất cả.
- **FR-009**: Khi nội dung bài được sửa (phiên bản mới), kết quả AI của phiên bản cũ MUST NOT được dùng nữa.
- **FR-010**: Lỗi AI MUST được báo rõ bằng tiếng Việt, phân biệt các trường hợp:
  - chưa cấu hình;
  - hết lượt;
  - lỗi khác hoặc kết quả không dùng được.

  Lỗi MUST NOT xoá nghĩa từ điển hay ô tự nhập, và MUST NOT lưu kết quả lỗi.
- **FR-011**: Hỏi AI MUST chỉ dùng được cho từ hoặc cụm có trong câu đang đọc, trong giới hạn tra từ của F3. Câu đó phải thuộc một bài
  người học được mở (theo L).
- **FR-012**: Nút, trạng thái và kết quả MUST dùng được ở 360px và chỉ bằng bàn phím. Trạng thái đang hỏi và kết quả MUST được thông báo
  cho trình đọc màn hình.

### Key Entities

- **Kết quả hỏi AI** (dùng chung):
  - bài, phiên bản nội dung bài, số thứ tự câu;
  - từ hoặc cụm đã chuẩn hoá (chữ thường, gộp khoảng trắng);
  - dạng gốc, nghĩa tiếng Việt, câu giải thích, thời điểm hỏi.

  Mỗi bộ khoá có tối đa một kết quả.
- **Popup tra từ** (F3, mở rộng): có thêm nguồn nghĩa "AI theo ngữ cảnh", nút Hỏi AI và trạng thái đang hỏi.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 0 lần gọi AI khi người học không bấm Hỏi AI. Kiểm qua nhật ký: tra 20 từ, không bấm nút thì không có dòng gọi AI nào.
- **SC-002**: Tra lại một từ đã hỏi trong cùng câu, kể cả tài khoản khác, hiện kết quả trong dưới 1 giây và không có lần gọi AI mới.
- **SC-003**: 10 lần bấm Hỏi AI gần như cùng lúc cho cùng từ, cùng câu chỉ tạo đúng 1 lần gọi AI.
- **SC-004**: Với AI hoạt động, ít nhất 9/10 lần hỏi có kết quả trong dưới 10 giây.
- **SC-005**: Ở mọi loại lỗi AI, 100% lần người học vẫn lưu được thẻ bằng nghĩa từ điển hoặc nghĩa tự nhập.
- **SC-006**: Mọi thao tác của F9 làm được ở 360px và chỉ bằng bàn phím.

## Assumptions

- Dùng AI đã cấu hình cho chú thích ở F2 (Gemini gói miễn phí). Không thêm cấu hình mới.
- Kết quả dùng chung cho mọi người học là chấp nhận được. Kết quả chỉ chứa nghĩa của từ trong câu của bài, không chứa dữ liệu người học.
- Quản trị viên chưa cần xem hay sửa các kết quả hỏi AI. Kết quả của phiên bản bài cũ được giữ lại nhưng không dùng; không có bước dọn
  dữ liệu.
- Hỏi AI dùng được ở mọi nơi có popup tra từ của bài (bước Đọc, xem lại bài, bước Nghe nếu có popup).
- Kết quả hỏi AI không được thêm vào chú thích của bài. Chú thích vẫn do F2 sinh và quản trị viên sửa.
- Ngoài phạm vi: hỏi đáp tự do với AI, giải thích cả câu.
- Phụ thuộc: F3 (popup tra từ, giới hạn tra từ, lưu vào sổ từ), F2 (AI, phiên bản nội dung bài), L (quyền mở bài).
