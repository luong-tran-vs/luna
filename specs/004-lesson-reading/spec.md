# Feature Specification: Đọc

**Feature Branch**: `004-lesson-reading`

**Created**: 2026-09-29

**Status**: Draft

**Mã tính năng**: F3 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f3-doc.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đọc bài và tra một từ (Priority: P1)

Người học mở bước Đọc của một bài và thấy toàn bộ bài. Chạm vào một từ thì hiện popup gồm: từ, phiên âm IPA,
nghĩa và nhãn cho biết nghĩa lấy từ đâu. Nghĩa lấy theo thứ tự: chú thích AI của bài ("AI · theo ngữ cảnh"), rồi
từ điển Anh–Việt trên máy ("Từ điển"). Tra dạng biến đổi (*went*, *studies*) vẫn ra nghĩa của dạng gốc.

**Why this priority**: Tra nghĩa ngay trong bài là giá trị cốt lõi của bước Đọc.

**Independent Test**: Mở một bài đã có chú thích, chạm vào một từ có chú thích và một từ không có; kiểm tra nghĩa,
nhãn nguồn và thời gian hiện popup.

**Acceptance Scenarios**:

1. **Given** bài có chú thích *went → go, "đã đi"*, **When** chạm vào *went*, **Then** popup hiện *went*, dạng gốc
   *go*, nghĩa "đã đi" và nhãn "AI · theo ngữ cảnh".
2. **Given** từ *park* không có trong chú thích nhưng có trong từ điển, **When** chạm vào *park*, **Then** popup hiện
   phiên âm, các nghĩa từ điển và nhãn "Từ điển".
3. **Given** từ *studies* không có trong chú thích, **When** chạm vào nó, **Then** popup hiện nghĩa của *study* từ từ
   điển.
4. **Given** bất kỳ từ nào, **When** chạm vào, **Then** kết quả hiện trong dưới 300ms và không có yêu cầu nào gửi tới
   AI.
5. **Given** popup đang mở trên điện thoại 360px, **When** nhìn màn hình, **Then** popup không che từ đang tra; chạm ra
   ngoài, bấm nút đóng hoặc phím Esc thì popup đóng.

---

### User Story 2 - Lưu từ vào sổ từ (Priority: P1)

Từ popup, người học bấm "Lưu vào sổ từ". Từ được lưu kèm câu chứa nó trong bài. Lưu lại một từ đã có thì không tạo
bản trùng. Từ đã có trong sổ được tô màu ở mọi bài. Nếu không tìm được nghĩa ở đâu, người học tự nhập nghĩa rồi lưu.

**Why this priority**: Sổ từ nối bước Đọc với bước Ôn (F5, R3); không lưu được thì tra xong là mất.

**Independent Test**: Lưu một từ, thấy từ được tô màu ở bài này và một bài khác có cùng từ; lưu lại thì báo đã có;
tra một từ không có nghĩa, tự nhập nghĩa rồi lưu.

**Acceptance Scenarios**:

1. **Given** popup của *went* trong câu "We went to the park.", **When** bấm "Lưu vào sổ từ", **Then** sổ từ có mục
   *go* (từ gặp: *went*) với nghĩa, phiên âm và câu "We went to the park.", và nút đổi thành "Đã có trong sổ".
2. **Given** *go* đã có trong sổ, **When** mở popup của *goes* hoặc *go* ở bất kỳ bài nào, **Then** popup hiện "Đã có
   trong sổ" và không tạo mục mới.
3. **Given** *go* đã có trong sổ, **When** mở một bài khác có *go* hoặc *went*, **Then** các từ đó được tô màu.
4. **Given** từ không có trong chú thích lẫn từ điển, **When** mở popup, **Then** popup báo "Chưa có nghĩa", có ô để
   người học nhập nghĩa; lưu khi ô trống thì bị chặn với lỗi.
5. **Given** chưa đăng nhập hoặc mất kết nối khi lưu, **When** bấm lưu, **Then** thấy thông báo lỗi và popup vẫn mở để
   thử lại.

---

### User Story 3 - Tra cả cụm từ (Priority: P2)

Người học bôi đen (trên máy tính) hoặc chạm giữ rồi kéo (trên điện thoại) nhiều từ liền nhau để tra cả cụm, ví dụ
*gave up*, *look forward to*. Cụm được tra và lưu như một từ.

**Why this priority**: Cụm động từ và thành ngữ là phần đáng học nhất, nhưng tra từng từ vẫn dùng được.

**Independent Test**: Chọn *gave up* trong một bài có chú thích *give up*; popup hiện nghĩa của cụm; lưu và thấy cả cụm
được tô màu.

**Acceptance Scenarios**:

1. **Given** bài có chú thích *gave up → give up*, **When** chọn hai từ *gave up*, **Then** popup hiện nghĩa của cả cụm
   với nhãn "AI · theo ngữ cảnh".
2. **Given** cụm không có trong chú thích nhưng có trong từ điển (ví dụ *look forward to*), **When** chọn cụm, **Then**
   popup hiện nghĩa từ điển.
3. **Given** chọn một đoạn cắt giữa từ (ví dụ "ave u"), **When** thả tay, **Then** vùng chọn được mở rộng thành các từ
   trọn vẹn (*gave up*).
4. **Given** chọn vắt qua hai câu, **When** thả tay, **Then** chỉ lấy phần trong câu đầu tiên được chọn.
5. **Given** cụm đã lưu, **When** mở bất kỳ bài nào có cụm đó, **Then** cả cụm được tô màu.

---

### User Story 4 - Nghe phát âm (Priority: P2)

Popup có nút nghe phát âm của từ hoặc cụm đang tra.

**Why this priority**: Cần cho việc học từ, nhưng tra nghĩa và lưu vẫn dùng được nếu không có âm thanh.

**Independent Test**: Bấm nút nghe trong popup của vài từ; nghe đúng từ; bấm lại lần hai phát ngay.

**Acceptance Scenarios**:

1. **Given** popup đang mở, **When** bấm nút nghe, **Then** nghe được cách đọc của từ hoặc cụm.
2. **Given** đã nghe một từ, **When** nghe lại (ở bài này hay bài khác), **Then** phát ngay, không phải chờ tạo lại.
3. **Given** dịch vụ đọc văn bản tạm lỗi, **When** bấm nghe, **Then** thấy "Chưa phát được âm thanh" và phần còn lại của
   popup vẫn dùng được.

---

### User Story 5 - Hoàn thành bước Đọc (Priority: P2)

Nút "Đã đọc xong" chỉ bật khi người học đã cuộn tới cuối bài. Bấm nút thì bước Đọc được đánh dấu hoàn thành.

**Why this priority**: Là điều kiện mở bước tiếp theo trong luồng một ngày học (L), nhưng L chưa làm ở F3.

**Independent Test**: Mở bài dài hơn màn hình; nút bị vô hiệu cho tới khi cuộn tới cuối; bấm thì thấy thông báo hoàn
thành.

**Acceptance Scenarios**:

1. **Given** bài dài hơn một màn hình, **When** mới mở, **Then** nút "Đã đọc xong" bị vô hiệu và có gợi ý "Đọc hết bài
   để hoàn thành".
2. **Given** đã cuộn tới câu cuối, **When** nhìn nút, **Then** nút bật; cuộn ngược lên đầu thì nút vẫn bật.
3. **Given** bài ngắn vừa một màn hình, **When** mở bài, **Then** nút bật ngay.
4. **Given** nút đang bật, **When** bấm, **Then** bước Đọc được báo hoàn thành và thấy "Đã hoàn thành bước Đọc".

---

### Edge Cases

- Từ có dấu nháy hoặc gạch nối (*don't*, *well-known*) là một từ; số, dấu câu và ký hiệu không bấm được.
- Cùng một từ xuất hiện nhiều lần: tra theo đúng câu chứa lần xuất hiện được chạm, để lấy chú thích và câu ngữ cảnh đúng.
- Chú thích có cho câu khác nhưng không có cho câu này: vẫn dùng chú thích cùng từ ở câu khác trước khi xuống từ điển.
- Bài đang chạy chú thích hoặc chú thích lỗi: vẫn đọc và tra được bằng từ điển.
- Từ điển chưa được cài trên máy: tra rơi xuống "Chưa có nghĩa" và người học tự nhập; app không lỗi.
- Dạng biến đổi không đoán được bằng quy tắc (*children*, *mice*) mà không có trong chú thích: báo "Chưa có nghĩa" nếu từ
  điển không có chính dạng đó.
- Chạm vào từ khác khi popup đang mở: popup chuyển sang từ mới.
- Chọn quá dài (hơn 6 từ): không tra, hiện gợi ý "Chọn tối đa 6 từ".
- Bài không tồn tại hoặc đã bị xoá: báo "Không tìm thấy bài học".
- Nút nghe khi âm thanh đang phát: phát lại từ đầu, không chồng tiếng.

## Requirements *(mandatory)*

### Functional Requirements

**Hiển thị bài**

- **FR-001**: Người đã đăng nhập PHẢI mở được bước Đọc của một bài và thấy toàn bộ nội dung, giữ nguyên đoạn và câu.
- **FR-002**: Mỗi từ trong bài PHẢI chạm (hoặc chọn bằng bàn phím) được để tra; dấu câu và số thì không.
- **FR-003**: Từ và cụm có trong sổ từ của người học PHẢI được tô màu ở mọi bài, gồm cả dạng biến đổi nhận ra được, và
  không chỉ dựa vào màu (có thêm dấu hiệu như gạch chân).
- **FR-004**: Bài đọc dùng cỡ chữ đọc (18px) và tối đa khoảng 65 ký tự mỗi dòng trên màn hình rộng.

**Tra từ**

- **FR-005**: Chạm một từ hoặc chọn cụm liền nhau (tối đa 6 từ, trong một câu) PHẢI mở popup tra cứu, đặt cạnh từ và không
  che từ đó.
- **FR-006**: Popup PHẢI gồm: từ/cụm như trong bài, dạng gốc (nếu khác), phiên âm IPA (nếu có), nghĩa tiếng Việt, nhãn
  nguồn, nút nghe phát âm, nút lưu hoặc trạng thái "Đã có trong sổ", và nút đóng.
- **FR-007**: Nghĩa PHẢI lấy theo thứ tự: (1) chú thích của bài khớp từ/cụm hoặc dạng gốc, ưu tiên câu đang đọc — nhãn
  "AI · theo ngữ cảnh"; (2) từ điển Anh–Việt trên máy, thử nguyên dạng rồi dạng gốc — nhãn "Từ điển"; (3) không có —
  "Chưa có nghĩa" và ô tự nhập.
- **FR-008**: Tra từ KHÔNG được gọi AI và PHẢI hiện kết quả trong dưới 300ms.
- **FR-009**: Dạng biến đổi thông dụng (số nhiều, ngôi thứ ba, quá khứ có quy tắc, -ing, động từ bất quy tắc thông dụng)
  PHẢI tra ra dạng gốc khi chú thích không có.
- **FR-010**: Popup PHẢI đóng khi chạm ra ngoài, bấm nút đóng hoặc phím Esc; dùng được bằng bàn phím và trình đọc màn hình.

**Phát âm**

- **FR-011**: Nút nghe PHẢI phát cách đọc tiếng Anh của từ/cụm bằng dịch vụ đọc văn bản chạy trên máy.
- **FR-012**: Âm thanh của một từ/cụm đã tạo PHẢI được giữ lại và dùng lại cho mọi người học và mọi bài.
- **FR-013**: Lỗi phát âm KHÔNG được làm hỏng popup; hiện thông báo ngắn.

**Sổ từ (tối thiểu, F5 mở rộng)**

- **FR-014**: Người học PHẢI lưu được từ/cụm vào sổ từ của mình kèm: từ như gặp trong bài, dạng gốc, phiên âm, nghĩa, câu
  chứa nó và bài học nguồn.
- **FR-015**: Mỗi người học có tối đa một mục cho mỗi dạng gốc (không phân biệt hoa thường); lưu lại PHẢI không tạo bản
  trùng và báo "Đã có trong sổ".
- **FR-016**: Khi nghĩa lấy từ từ điển có nhiều nghĩa, mục được lưu PHẢI chứa các nghĩa đang hiển thị (tối đa 3).
- **FR-017**: Khi tự nhập, nghĩa PHẢI khác rỗng và tối đa 200 ký tự.
- **FR-018**: Sổ từ PHẢI tách riêng theo người học: không ai thấy hay ghi vào sổ của người khác (nguyên tắc V).

**Hoàn thành bước**

- **FR-019**: Nút "Đã đọc xong" PHẢI bị vô hiệu cho tới khi phần cuối bài đã hiện trên màn hình ít nhất một lần.
- **FR-020**: Bấm "Đã đọc xong" PHẢI báo hoàn thành bước Đọc cho luồng học (nối vào L sau) và hiện xác nhận.

**Truy cập tạm thời**

- **FR-021**: Cho tới khi có luồng một ngày học (L), bước Đọc PHẢI mở được từ trang chi tiết bài của quản trị viên và bằng
  đường dẫn trực tiếp.

### Key Entities

- **Từ trong sổ (Card)**: thuộc một người học; từ như gặp, dạng gốc (duy nhất theo người học), phiên âm, nghĩa, câu ngữ
  cảnh, bài nguồn, thời điểm lưu. F5 sẽ thêm lịch ôn.
- **Kết quả tra cứu (Lookup result)**: nguồn (AI hoặc Từ điển), từ, dạng gốc, phiên âm, danh sách nghĩa. Không lưu.
- **Âm thanh phát âm (Word audio)**: âm thanh của một từ/cụm, dùng chung, tạo một lần.
- **Mục từ điển (Dictionary entry)**: từ, phiên âm, các nghĩa tiếng Việt; chỉ đọc, cài sẵn trên máy.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% lần tra (chạm từ hoặc chọn cụm) hiện kết quả trong dưới 300ms trên máy chủ tại nhà; 0 lần tra gọi AI.
- **SC-002**: Với bộ mẫu dạng biến đổi thông dụng (*went, gone, studies, running, stopped, children* có chú thích…), 100%
  ra đúng dạng gốc khi có chú thích hoặc có trong từ điển.
- **SC-003**: Lưu cùng một từ (hoặc dạng biến đổi của nó) nhiều lần chỉ tạo đúng 1 mục trong sổ.
- **SC-004**: Từ đã lưu được tô màu ở 100% các bài có chứa nó (cùng dạng gốc).
- **SC-005**: Nút "Đã đọc xong" không bật được trước khi cuộn tới cuối bài trong 100% lần thử với bài dài hơn màn hình.
- **SC-006**: Ở 360px, popup luôn nằm trọn trong màn hình và không che từ đang tra; không có cuộn ngang.
- **SC-007**: Nghe lại một từ đã nghe phát ngay (dưới 1 giây).

## Assumptions

- Bước Đọc chưa gắn với tiến độ: F3 chỉ báo "đã hoàn thành" và hiện xác nhận; lưu tiến độ, mở bước tiếp và nối vào
  thanh tiến trình thuộc L.
- Mọi người đã đăng nhập đều đọc được mọi bài (chưa giới hạn theo lộ trình); L sẽ chọn bài của ngày.
- Từ điển Anh–Việt là dữ liệu miễn phí, cài sẵn trên máy chủ (không tải khi tra); nếu chưa cài thì chỉ còn chú thích và
  tự nhập.
- Đưa về dạng gốc dùng quy tắc đơn giản và một bảng nhỏ động từ bất quy tắc; không dùng AI hay thư viện ngôn ngữ lớn.
- Khi lưu, nghĩa được lấy nguyên như popup hiển thị; sửa nghĩa trong sổ làm ở F5.
- Tô màu theo dạng gốc: từ đã lưu *go* thì *go, goes, went, going* trong bài đều được tô nếu nhận ra được dạng gốc.
- Âm thanh phát âm dùng chung giọng đọc của bài học (F2).
- Ngoài phạm vi: "Hỏi AI" cho từ chưa có chú thích (giai đoạn 2), ôn thẻ (F5), luồng một ngày học (L), dịch cả câu.
