# Feature Specification: Sổ từ và ôn tập

**Feature Branch**: `006-vocabulary-review`

**Created**: 2026-09-30

**Status**: Draft

**Mã tính năng**: F5 | **Giai đoạn**: Giai đoạn 1 (`docs/phases/giai-doan-1.md`)

**Input**: User description: "Tạo spec theo khối 1 trong @docs/spec-inputs/f5-so-tu-on-tap.md"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ôn thẻ đến hạn (Priority: P1)

Người học vào ôn tự do bất cứ lúc nào. Hệ thống đưa ra các thẻ đã đến hạn ôn. Người học chọn một trong hai kiểu ôn: **Xem từ
đoán nghĩa** (hiện từ, chạm để lật xem IPA, nghĩa, câu ví dụ) hoặc **Nghe rồi gõ** (nghe audio của từ, gõ lại, hệ thống báo
đúng hay sai). Sau mỗi thẻ, người học chọn **Again**, **Hard**, **Good** hoặc **Easy**; ngày ôn tiếp theo của thẻ được tính
lại theo thuật toán FSRS.

**Why this priority**: Là mục tiêu của F5: ôn đúng lúc sắp quên để nhớ lâu. Thẻ đã có từ bước Đọc (F3), nên ôn tập dùng được
ngay.

**Independent Test**: Với tài khoản có vài thẻ đến hạn, mở ôn tự do, ôn một thẻ bằng mỗi kiểu, chọn đánh giá; kiểm tra thẻ
không còn đến hạn và ngày ôn tiếp theo khớp FSRS.

**Acceptance Scenarios**:

1. **Given** có 5 thẻ đến hạn, **When** mở ôn tự do, **Then** thấy số thẻ cần ôn ("5 thẻ"), chọn được kiểu ôn và bắt đầu.
2. **Given** kiểu Xem từ đoán nghĩa, **When** thẻ hiện ra, **Then** chỉ thấy từ (và nút nghe); **When** chạm lật thẻ, **Then**
   thấy IPA, nghĩa, câu ví dụ và 4 nút đánh giá.
3. **Given** kiểu Nghe rồi gõ, **When** thẻ hiện ra, **Then** audio của từ tự phát, có nút nghe lại và ô gõ, không hiện từ.
4. **Given** thẻ "went", **When** gõ "Went" và kiểm tra, **Then** báo đúng (không phân biệt hoa thường), hiện từ, IPA, nghĩa
   và 4 nút đánh giá; **When** gõ "want", **Then** báo sai và hiện từ đúng.
5. **Given** đã lật hoặc đã kiểm tra, **When** chọn Good, **Then** lịch của thẻ được cập nhật theo FSRS và thẻ tiếp theo hiện
   ra; mỗi nút đánh giá cho biết khoảng cách tới lần ôn tiếp theo (ví dụ "10 phút", "3 ngày").
6. **Given** chọn Again cho một thẻ, **When** các thẻ khác trong phiên đã ôn xong, **Then** thẻ đó hiện lại một lần ở cuối phiên.
7. **Given** đã ôn hết thẻ trong phiên, **When** phiên kết thúc, **Then** thấy tổng kết "Đã ôn N thẻ" kèm số lần chọn từng mức.
8. **Given** không có thẻ đến hạn, **When** mở ôn tự do, **Then** thấy "Không có thẻ nào đến hạn" và thời điểm thẻ sớm nhất
   đến hạn (nếu có).

---

### User Story 2 - Quản lý sổ từ (Priority: P1)

Người học xem danh sách thẻ trong sổ, nhóm theo ngày lưu ("Hôm nay", "Hôm qua", ngày cụ thể) theo múi giờ của mình, tìm theo
từ, lọc theo bài học. Người học sửa nghĩa, xoá thẻ, hoặc tự thêm thẻ mới.

**Why this priority**: Không xem và sửa được thẻ thì nghĩa sai sẽ bị ôn mãi; tự thêm thẻ cho phép học từ ngoài bài.

**Independent Test**: Lưu vài từ hôm nay và có thẻ lưu hôm qua; mở sổ từ, kiểm tra nhóm ngày, tìm một từ, lọc theo bài, sửa
nghĩa một thẻ (lịch ôn giữ nguyên), xoá một thẻ, tự thêm một thẻ.

**Acceptance Scenarios**:

1. **Given** có thẻ lưu hôm nay, hôm qua và ngày 20/09, **When** mở sổ từ, **Then** thẻ nằm dưới các nhóm "Hôm nay", "Hôm
   qua", "20/09/2026", mới nhất trước; mỗi thẻ hiện từ, IPA, nghĩa, câu ví dụ và nút nghe.
2. **Given** múi giờ của người học là `Asia/Ho_Chi_Minh`, **When** một thẻ được lưu lúc 23:30 giờ Việt Nam, **Then** thẻ nằm
   trong nhóm của ngày đó theo giờ Việt Nam (không theo giờ máy chủ).
3. **Given** sổ có "go" và "give up", **When** gõ "giv" vào ô tìm, **Then** chỉ thấy "give up" (tìm theo từ, không phân biệt
   hoa thường).
4. **Given** thẻ lưu từ hai bài, **When** lọc theo bài "A day at the park", **Then** chỉ thấy thẻ của bài đó; có lựa chọn
   "Thẻ tự thêm" cho thẻ không thuộc bài nào.
5. **Given** thẻ "went" đã ôn 2 lần, **When** sửa nghĩa thành "đã đi (quá khứ của go)", **Then** thẻ hiện nghĩa mới; ngày ôn
   tiếp theo và lịch sử ôn không đổi.
6. **Given** một thẻ, **When** bấm Xoá và xác nhận, **Then** thẻ và toàn bộ lịch sử ôn của thẻ bị xoá; từ đó lưu lại được như
   thẻ mới.
7. **Given** form Thêm thẻ, **When** nhập từ "serendipity", nghĩa "sự tình cờ may mắn" (IPA và câu ví dụ tuỳ chọn) và lưu,
   **Then** thẻ xuất hiện trong nhóm "Hôm nay", đến hạn ôn lần đầu vào ngày mai.
8. **Given** sổ đã có "go", **When** tự thêm "Go", **Then** thấy báo "Từ này đã có trong sổ" và không tạo thẻ trùng.

---

### User Story 3 - Mục Từ vựng của bài ở bước Đọc (Priority: P2)

Ở bước Đọc có mục "Từ vựng" liệt kê các từ và cụm từ đã được chú thích của bài (từ, IPA, nghĩa, nút nghe). Mỗi từ có nút
**Lưu**; cả mục có nút **Lưu tất cả**; từ đã có trong sổ hiện dấu ✓.

**Why this priority**: Giúp lưu nhanh từ của bài mà không phải tra từng từ; bước Đọc (F3) vẫn lưu được từng từ nếu chưa có mục
này.

**Independent Test**: Mở bước Đọc của bài đã có chú thích, mở mục Từ vựng, lưu một từ, rồi Lưu tất cả; kiểm tra sổ từ không có
thẻ trùng. Mở bài chưa có chú thích thấy "Chưa có danh sách từ vựng".

**Acceptance Scenarios**:

1. **Given** bài có chú thích cho "went → go" và "gave up → give up", **When** mở mục Từ vựng, **Then** thấy từng mục với từ
   gốc, IPA, nghĩa, nút nghe và nút Lưu.
2. **Given** "go" đã có trong sổ, **When** mở mục Từ vựng, **Then** "go" hiện dấu ✓ "Đã lưu" thay cho nút Lưu.
3. **Given** 10 mục, 3 đã lưu, **When** bấm Lưu tất cả, **Then** chỉ 7 thẻ mới được tạo, mọi mục hiện ✓, và thấy "Đã lưu 7 từ".
4. **Given** bấm Lưu tất cả hai lần liên tiếp, **When** lần thứ hai chạy, **Then** không tạo thẻ nào (không trùng).
5. **Given** bài chưa có chú thích (hoặc chú thích đang được tạo), **When** mở mục Từ vựng, **Then** thấy "Chưa có danh sách từ
   vựng" và không có nút Lưu tất cả.
6. **Given** một từ được lưu từ mục Từ vựng, **When** xem thẻ trong sổ, **Then** câu ví dụ là câu đầu tiên của bài chứa từ đó
   và thẻ gắn với bài.

---

### Edge Cases

- Thẻ lưu lúc 23:59 và 00:01 giờ của người học: thuộc hai ngày khác nhau (nhóm ngày và ngày đến hạn đầu tiên đều theo múi giờ
  của người học).
- Thẻ đã lưu ở F3 (trước khi có lịch ôn): coi là thẻ mới, đến hạn lần đầu vào ngày sau ngày lưu, nên thẻ lưu từ hôm qua trở về
  trước đã đến hạn.
- Nghe rồi gõ: bỏ khoảng trắng đầu/cuối và khoảng trắng thừa giữa các từ; dấu nháy cong và thẳng coi như nhau (*don’t* =
  *don't*); cụm từ ("give up") gõ đủ cả cụm.
- Không phát được audio của một thẻ ở kiểu Nghe rồi gõ: báo "Chưa phát được âm thanh" và cho phép hiện từ để tiếp tục ôn thẻ
  đó như kiểu Xem từ đoán nghĩa.
- Ô gõ trống khi kiểm tra: nhắc "Hãy gõ từ bạn nghe được", không tính là sai.
- Rời trang giữa phiên ôn: các thẻ đã đánh giá được lưu; thẻ chưa đánh giá vẫn đến hạn.
- Lưu đánh giá thất bại (mất mạng): báo lỗi, không chuyển thẻ, cho thử lại; không có thẻ nào bị đánh giá hai lần.
- Một thẻ bị xoá ở thiết bị khác khi đang ôn: bỏ qua thẻ đó và tiếp tục phiên.
- Tìm kiếm không có kết quả: "Không tìm thấy từ nào"; sổ trống: hướng dẫn lưu từ ở bước Đọc hoặc tự thêm thẻ.
- Sửa nghĩa thành rỗng hoặc quá dài: báo lỗi ở trường nghĩa, không lưu.
- Bài bị xoá: thẻ của bài vẫn còn trong sổ; bộ lọc theo bài không còn liệt kê bài đó, thẻ vẫn thấy khi không lọc.
- Chú thích của bài trùng nhau (hai dạng "went" và "goes" cùng gốc "go"): mục Từ vựng hiện một mục cho mỗi từ gốc.
- Sổ nhiều thẻ (hàng trăm): danh sách tải dần, không chậm.

## Requirements *(mandatory)*

### Functional Requirements

**Thẻ và sổ từ**

- **FR-001**: Mỗi thẻ PHẢI có: từ (hoặc cụm từ), từ gốc dùng để chống trùng, IPA (có thể trống), nghĩa tiếng Việt, câu ví dụ
  (có thể trống), bài học nguồn (có thể không có), ngày lưu; audio của từ phát được từ thẻ.
  *(Sửa 2026-10-07)* Khi lấy thẻ ra ôn, thẻ trống IPA mà từ điển offline có đúng từ gốc đó thì được điền IPA từ từ điển
  (cụm từ: ghép IPA của từng từ, chỉ khi từ điển có đủ mọi từ) và lưu lại (lịch ôn không đổi); tra hoặc lưu lỗi thì thẻ vẫn ôn bình thường, không có IPA.
- **FR-002**: Sổ từ PHẢI liệt kê thẻ của người học, nhóm theo ngày lưu tính theo múi giờ của người học: "Hôm nay", "Hôm qua",
  các ngày khác dạng ngày/tháng/năm; mới nhất trước.
- **FR-003**: Sổ từ PHẢI tìm được theo từ hoặc từ gốc (chứa chuỗi tìm, không phân biệt hoa thường) và lọc được theo bài học
  (kể cả lựa chọn "Thẻ tự thêm"); tìm và lọc dùng được cùng lúc.
- **FR-004**: Danh sách PHẢI tải dần khi sổ lớn (từng phần, có "Xem thêm" hoặc tải khi cuộn), không tải toàn bộ một lần.
- **FR-005**: Người học PHẢI tự thêm được thẻ: từ và nghĩa bắt buộc; IPA và câu ví dụ tuỳ chọn; thẻ tự thêm không gắn bài học.
- **FR-006**: Người học PHẢI sửa được nghĩa, IPA và câu ví dụ của thẻ; từ không sửa được (muốn đổi thì xoá và thêm lại). Sửa
  thẻ KHÔNG ĐƯỢC thay đổi lịch ôn hay lịch sử ôn.
- **FR-007**: Người học PHẢI xoá được thẻ sau khi xác nhận; xoá thẻ xoá cả lịch sử ôn của thẻ.
- **FR-008**: Không có hai thẻ cùng từ gốc trong sổ của một người học (so không phân biệt hoa thường và khoảng trắng thừa);
  thêm trùng thì báo "Từ này đã có trong sổ".

**Lịch ôn**

- **FR-009**: Lịch ôn PHẢI theo thuật toán FSRS với tham số mặc định.
- **FR-010**: Thẻ mới lưu PHẢI đến hạn lần đầu vào đầu ngày hôm sau (0 giờ) theo múi giờ của người học; thẻ đã lưu trước khi
  có F5 được coi là thẻ mới theo cùng quy tắc với ngày lưu của nó.
- **FR-011**: Sau mỗi đánh giá (Again, Hard, Good, Easy), ngày ôn tiếp theo và trạng thái của thẻ PHẢI được cập nhật theo FSRS
  và lưu lại cùng một bản ghi lịch sử (mức đánh giá, kiểu ôn, thời điểm).
- **FR-012**: Mỗi nút đánh giá PHẢI hiện khoảng cách tới lần ôn tiếp theo nếu chọn nút đó.

**Ôn tập**

- **FR-013**: Người học PHẢI vào ôn tự do được bất cứ lúc nào; phiên ôn gồm các thẻ đã đến hạn tại lúc mở, thẻ quá hạn lâu
  nhất trước.
- **FR-014**: Người học PHẢI chọn được kiểu ôn cho phiên: Xem từ đoán nghĩa hoặc Nghe rồi gõ.
- **FR-015**: Kiểu Xem từ đoán nghĩa: hiện từ và nút nghe; lật thẻ (chạm, Enter hoặc Space) hiện IPA, nghĩa, câu ví dụ và 4
  nút đánh giá.
- **FR-016**: Kiểu Nghe rồi gõ: tự phát audio của từ, có nút nghe lại và ô gõ; kiểm tra so với từ của thẻ không phân biệt hoa
  thường, bỏ khoảng trắng thừa, coi dấu nháy cong và thẳng như nhau; báo đúng/sai (không chỉ bằng màu) rồi hiện từ, IPA, nghĩa
  và 4 nút đánh giá.
- **FR-017**: Người học chọn đánh giá sau mỗi thẻ; thẻ chọn Again PHẢI hiện lại một lần ở cuối phiên (lần hiện lại không tạo
  thêm đánh giá nếu người học không chọn).
- **FR-018**: Kết thúc phiên PHẢI hiện tổng kết (số thẻ đã ôn, số lần chọn từng mức) và báo "phiên đã xong" để luồng học (L)
  dùng lại được.
- **FR-019**: Phiên ôn PHẢI dùng được bằng bàn phím (lật thẻ, gõ, chọn đánh giá bằng phím 1–4) và trên điện thoại 360px (nút
  đánh giá ≥ 44px, ô gõ không bị bàn phím ảo che).
- **FR-020**: Không có thẻ đến hạn PHẢI hiện "Không có thẻ nào đến hạn" và thời điểm thẻ sớm nhất đến hạn (nếu có thẻ).

**Mục Từ vựng của bài**

- **FR-021**: Bước Đọc PHẢI có mục "Từ vựng" (thu gọn hoặc tab, không che bài đọc) liệt kê các từ và cụm từ đã chú thích của
  bài, mỗi từ gốc một mục: từ gốc, IPA, nghĩa, nút nghe, nút Lưu.
- **FR-022**: Mục Từ vựng KHÔNG ĐƯỢC gọi AI; chỉ dùng chú thích đã có. Bài chưa có chú thích (hoặc đang tạo) hiện "Chưa có
  danh sách từ vựng".
- **FR-023**: Từ đã có trong sổ PHẢI hiện dấu ✓ "Đã lưu"; lưu một từ từ mục này (hoặc từ popup tra từ của F3) cập nhật dấu ✓
  ngay.
- **FR-024**: "Lưu tất cả" PHẢI chỉ thêm các từ chưa có trong sổ, không tạo thẻ trùng kể cả khi bấm nhiều lần, và báo số từ
  đã thêm.
- **FR-025**: Thẻ lưu từ mục Từ vựng PHẢI gắn với bài và có câu ví dụ là câu đầu tiên của bài chứa từ hoặc cụm từ đó.

**Dữ liệu và quyền**

- **FR-026**: Thẻ, lịch ôn và lịch sử ôn PHẢI tách riêng theo người học: không ai đọc, sửa, xoá hay ôn thẻ của người khác
  (nguyên tắc V).
- **FR-027**: Sổ từ và ôn tự do PHẢI mở được từ thanh điều hướng của ứng dụng và bằng đường dẫn trực tiếp.

### Key Entities

- **Thẻ (Card)**: thuộc một người học; từ, từ gốc (duy nhất trong sổ), IPA, nghĩa, câu ví dụ, bài nguồn (tuỳ chọn), nguồn nghĩa
  (AI, từ điển, tự nhập), ngày lưu, và trạng thái lịch ôn (ngày đến hạn, độ ổn định, độ khó, số lần ôn, số lần quên, giai
  đoạn, lần ôn gần nhất).
- **Lịch sử ôn (Review log)**: thuộc một thẻ của một người học; mức đánh giá, kiểu ôn, thời điểm, trạng thái thẻ trước khi ôn.
  Xoá cùng thẻ.
- **Mục từ vựng của bài (Lesson vocabulary item)**: lấy từ chú thích của bài; từ gốc, dạng xuất hiện trong bài, IPA, nghĩa, câu
  chứa từ, đã lưu hay chưa (theo sổ của người đang xem). Không lưu riêng.
- **Phiên ôn (Review session)**: danh sách thẻ đến hạn, kiểu ôn, thẻ hiện tại, các đánh giá đã chọn. Chỉ tồn tại trên máy người
  học.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Bấm "Lưu tất cả" bao nhiêu lần cũng tạo 0 thẻ trùng ở 100% lần thử; số thẻ mới bằng đúng số từ chưa có trong sổ.
- **SC-002**: Thẻ lưu lúc 23:59 giờ của người học đến hạn lúc 0 giờ hôm sau; thẻ lưu lúc 00:01 đến hạn lúc 0 giờ ngày kế tiếp;
  đúng ở 100% trường hợp thử với các múi giờ khác nhau.
- **SC-003**: Ngày ôn tiếp theo sau mỗi mức đánh giá khớp kết quả tham chiếu của FSRS (tham số mặc định) ở 100% ca thử.
- **SC-004**: Sửa nghĩa, IPA hoặc câu ví dụ giữ nguyên 100% lịch ôn và lịch sử ôn của thẻ; xoá thẻ xoá 100% lịch sử ôn của thẻ.
- **SC-005**: Kiểu Nghe rồi gõ chấp nhận mọi cách viết hoa khác nhau của đúng từ ở 100% ca thử.
- **SC-006**: Nhóm theo ngày đúng theo múi giờ của người học ở 100% ca thử, kể cả thẻ lưu sát nửa đêm.
- **SC-007**: Mở mục Từ vựng của bài không phát sinh lần gọi AI nào; bài chưa có chú thích luôn hiện "Chưa có danh sách từ
  vựng".
- **SC-008**: Người học ôn xong 20 thẻ kiểu Xem từ đoán nghĩa trong dưới 5 phút; thẻ tiếp theo hiện ngay sau khi chọn đánh giá
  (dưới 1 giây).
- **SC-009**: Người dùng khác không đọc, sửa, xoá hay ôn được thẻ của người học ở 100% lần thử.
- **SC-010**: Ở 360px, sổ từ và phiên ôn không cuộn ngang, nút đánh giá bấm được bằng ngón cái; dùng được hoàn toàn bằng bàn
  phím.

## Assumptions

- Thẻ đã lưu từ F3 dùng lại nguyên (từ, từ gốc, IPA, nghĩa, câu ngữ cảnh, bài nguồn); F5 thêm trạng thái lịch ôn.
- Audio của thẻ là giọng đọc từ/cụm từ tạo tự động như popup tra từ ở F3; không lưu file ghi âm riêng cho từng thẻ.
- Từ gốc của thẻ tự thêm là chính từ đó (viết thường, gộp khoảng trắng); không tự đoán dạng gốc.
- Ôn tự do chỉ gồm thẻ đến hạn; không có giới hạn số thẻ mỗi ngày (giới hạn và bước Ôn bắt buộc đầu bài thuộc L).
- Kiểu ôn chọn cho cả phiên; nhớ lựa chọn lâu dài thuộc F12 (Cài đặt).
- Ở kiểu Nghe rồi gõ, đúng hay sai chỉ để người học tự đánh giá; hệ thống không tự chọn mức đánh giá.
- Thẻ chọn Again được hiện lại một lần ở cuối phiên để củng cố; lịch chính thức vẫn là kết quả FSRS của lần đánh giá.
- Mục Từ vựng hiện từ gốc của chú thích (ví dụ "go" cho "went"), vì thẻ chống trùng theo từ gốc.
- Ngoài phạm vi: giới hạn số thẻ mỗi ngày và bước Ôn bắt buộc đầu bài (L), thống kê ôn tập (F6), nhập/xuất sổ từ (F13), tối
  ưu tham số FSRS theo người học.
