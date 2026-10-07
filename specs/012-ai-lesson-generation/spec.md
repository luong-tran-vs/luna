# Feature Specification: AI sinh bài học

**Feature Branch**: `012-ai-lesson-generation`

**Created**: 2026-10-01

**Status**: Draft

**Mã tính năng**: F7 | **Giai đoạn**: Giai đoạn 2 (`docs/phases/giai-doan-2.md`)

**Input**: User description: "tạo spec theo khối 1 trong @docs/spec-inputs/f7-ai-sinh-bai.md"

**Sửa 2026-10-07 — chủ đề dùng chung cho mọi trình độ** (`specs/007-topic-roadmaps`): chủ đề không còn trình độ. Hộp sinh bài có ô
**Trình độ** (mặc định theo lộ trình đang mở, đổi được); AI sinh bài đúng chủ đề ở trình độ đã chọn, bài lưu vào lộ trình (chủ đề,
trình độ đó). Các chỗ thay đổi được đánh dấu *(sửa 2026-10-07)*.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sinh bản nháp cho lộ trình của một chủ đề (Priority: P1)

Ở trang lộ trình của một chủ đề và trình độ (ví dụ "A1 · Gia đình"), quản trị viên bấm **Sinh bài bằng AI**, chọn số bài (1–5),
độ dài (số từ, có gợi ý mặc định theo trình độ), dạng bài (bài đọc hoặc hội thoại) và ý chính nếu muốn. Chủ đề lấy theo lộ trình
đang mở; trình độ mặc định theo lộ trình đang mở và đổi được *(sửa 2026-10-07)*. Hệ thống sinh các bản nháp, mỗi bản có tiêu đề
và nội dung; trong lúc chờ có trạng thái đang sinh.

**Why this priority**: Là giá trị cốt lõi của F7: lấp đầy lộ trình nhanh mà không phải tự viết từng bài.

**Independent Test**: Mở lộ trình một chủ đề A1, sinh 3 bài đọc 120 từ: nhận 3 bản nháp hiển thị trên trang, mỗi bản có tiêu đề,
nội dung và số từ; danh sách bài và lộ trình chưa thay đổi.

**Acceptance Scenarios**:

1. **Given** quản trị viên đang ở lộ trình "A1 · Gia đình", **When** mở hộp sinh bài, **Then** trình độ A1 và chủ đề "Gia đình"
   được hiển thị sẵn, độ dài gợi ý theo A1, dạng bài mặc định là bài đọc, ý chính để trống. *(sửa 2026-10-07)* Đổi trình độ sang
   B1 thì độ dài gợi ý đổi theo B1 (nếu chưa sửa tay) và bài lưu vào lộ trình "B1 · Gia đình".
2. **Given** các lựa chọn hợp lệ, **When** bấm sinh, **Then** trang hiện trạng thái đang sinh, không cho bấm sinh lần nữa trong lúc
   chờ, và khi xong hiện đúng số bản nháp đã yêu cầu (hoặc ít hơn kèm thông báo nếu có bản bị loại).
3. **Given** độ dài yêu cầu N từ, **When** nhận bản nháp, **Then** mỗi bản có số từ nằm trong khoảng N ± 20% và đúng trình độ đã chọn.
4. **Given** sinh nhiều bài một lượt, **When** xem các bản nháp, **Then** các bài khác nội dung nhau và khác các bài đã có trong chủ
   đề (không trùng tiêu đề, không lặp cùng một câu chuyện).
5. **Given** dạng bài là hội thoại, **When** nhận bản nháp, **Then** nội dung là lời thoại giữa các nhân vật, mỗi lượt nói một dòng
   có tên người nói ở đầu.
6. **Given** số bài ngoài 1–5 hoặc độ dài ngoài khoảng cho phép, **When** bấm sinh, **Then** báo lỗi ngay ở ô tương ứng và không gửi
   yêu cầu.
7. **Given** người dùng không phải quản trị viên, **When** cố dùng chức năng sinh bài, **Then** bị từ chối.

---

### User Story 2 - Duyệt, sửa và lưu bản nháp (Priority: P1)

Quản trị viên xem từng bản nháp, sửa tiêu đề và nội dung, rồi bấm **Lưu** hoặc **Bỏ** từng bài, hoặc **Lưu tất cả**. Bài đã lưu có
nguồn "AI sinh", giấy phép "Nội dung do AI tạo", đi qua quy trình của F2 (tách câu, audio, chú thích) và được thêm vào cuối lộ
trình của chủ đề.

**Why this priority**: Quản trị viên phải giữ quyền duyệt từng bài; không có bước lưu thì bản nháp không có giá trị.

**Independent Test**: Từ 3 bản nháp, sửa tiêu đề bản 1 rồi lưu, bỏ bản 2, lưu bản 3: lộ trình có thêm đúng 2 bài ở cuối theo thứ
tự lưu, bài 1 mang tiêu đề đã sửa, cả hai đang chạy audio và chú thích; bản 2 không xuất hiện ở đâu.

**Acceptance Scenarios**:

1. **Given** một bản nháp, **When** sửa tiêu đề hoặc nội dung, **Then** số từ cập nhật theo nội dung đang sửa.
2. **Given** một bản nháp hợp lệ, **When** bấm Lưu, **Then** bài được tạo với nguồn "AI sinh", giấy phép "Nội dung do AI tạo",
   thuộc chủ đề đang mở, nằm ở cuối lộ trình; bản nháp biến khỏi danh sách nháp; lộ trình tải lại và bài mới hiện trạng thái audio
   và chú thích "Đang chạy".
3. **Given** một bản nháp, **When** bấm Bỏ, **Then** bản nháp biến mất và không có gì được lưu.
4. **Given** nhiều bản nháp, **When** bấm Lưu tất cả, **Then** mọi bản nháp hợp lệ được lưu theo thứ tự đang hiển thị, thêm vào cuối
   lộ trình theo đúng thứ tự đó.
5. **Given** bản nháp có tiêu đề hoặc nội dung trống hay quá dài, **When** bấm Lưu (hoặc Lưu tất cả), **Then** bản đó báo lỗi ngay
   ở ô tương ứng và vẫn là bản nháp; các bản hợp lệ khác vẫn được lưu khi dùng Lưu tất cả.
6. **Given** bản nháp chưa lưu, **When** mở danh sách bài, lộ trình (cả phía người học), **Then** không thấy bản nháp đó.
7. **Given** lưu một bản thất bại (mất mạng, lỗi máy chủ), **When** thao tác kết thúc, **Then** bản nháp đó vẫn còn nguyên nội dung
   đã sửa kèm thông báo lỗi, để thử lưu lại.

---

### User Story 3 - AI lỗi không làm mất công sức (Priority: P2)

Khi AI lỗi, hết lượt hoặc trả về nội dung không dùng được, quản trị viên thấy thông báo rõ ràng; các bản nháp đã sinh trước đó và
các lựa chọn đã nhập vẫn còn để thử lại. Rời trang khi còn bản nháp chưa lưu thì được hỏi xác nhận.

**Why this priority**: AI gói miễn phí hay hết lượt; mất bản nháp đã sửa hoặc phải nhập lại làm chức năng khó dùng. Là tiêu chí
nghiệm thu của F7 nhưng chỉ có ý nghĩa khi đã sinh và lưu được bài.

**Independent Test**: Sinh 2 bản nháp, sửa một bản; làm AI trả lỗi hết lượt rồi sinh tiếp: thấy thông báo hết lượt, 2 bản nháp và
các lựa chọn vẫn còn. Bấm sang trang khác: được hỏi xác nhận; chọn ở lại thì mọi thứ vẫn nguyên.

**Acceptance Scenarios**:

1. **Given** AI chưa được cấu hình, **When** bấm sinh, **Then** thông báo "AI chưa được cấu hình" (hoặc tương đương) và gợi ý liên
   hệ người vận hành; các chức năng khác của trang vẫn dùng được.
2. **Given** AI hết lượt, **When** bấm sinh, **Then** thông báo đã hết lượt AI, nên thử lại sau.
3. **Given** AI lỗi khác hoặc quá thời gian chờ, **When** bấm sinh, **Then** thông báo sinh bài thất bại, có thể thử lại.
4. **Given** AI trả về nội dung không dùng được (mọi bản rỗng hoặc trùng tiêu đề với bài đã có), **When** sinh xong, **Then** thông
   báo AI trả về nội dung không dùng được, không thêm bản nháp nào.
5. **Given** đã có bản nháp và lựa chọn đã nhập, **When** một lượt sinh mới thất bại vì bất kỳ lý do nào ở trên, **Then** bản nháp cũ
   (kể cả phần đã sửa) và các lựa chọn vẫn giữ nguyên.
6. **Given** đã có bản nháp, **When** sinh thêm một lượt thành công, **Then** bản nháp mới được thêm vào danh sách, không thay thế
   bản cũ.
7. **Given** còn bản nháp chưa lưu, **When** rời trang (chuyển trang trong app, tải lại hoặc đóng tab), **Then** được hỏi xác nhận;
   chọn ở lại thì giữ nguyên trang, chọn rời thì bản nháp mất.
8. **Given** không còn bản nháp nào (đã lưu hoặc bỏ hết), **When** rời trang, **Then** không bị hỏi.

---

### Edge Cases

- Chủ đề chưa có bài nào: vẫn sinh được; không có danh sách bài cũ để tránh lặp.
- Chủ đề đã có nhiều bài: AI vẫn được báo các tiêu đề đã có để tránh lặp; bản nháp trùng tiêu đề với bài đã có (không phân biệt
  hoa thường, bỏ khoảng trắng thừa) hoặc trùng nhau trong cùng lượt bị loại, kèm thông báo đã loại bao nhiêu bản.
- AI trả về ít bản hơn số yêu cầu: hiện các bản dùng được và báo số bản thiếu.
- Quản trị viên sửa tiêu đề bản nháp trùng với bài đã có: vẫn lưu được (F2 không cấm trùng tiêu đề), quản trị viên tự chịu.
- Ý chính rất dài: giới hạn độ dài, báo lỗi ngay ở ô.
- Ý chính không hợp với trình độ hoặc chủ đề: AI vẫn bám trình độ và chủ đề của lộ trình; ý chính chỉ là gợi ý.
- Chủ đề bị xoá bởi một phiên khác trong lúc đang có bản nháp: lưu báo lỗi rõ ràng, bản nháp vẫn còn.
- Bấm Lưu hai lần liên tiếp cho cùng bản: chỉ tạo một bài.
- Lưu tất cả khi mạng chập chờn: bài nào đã lưu thì biến khỏi danh sách nháp, bài nào lỗi thì ở lại với thông báo.
- Bài đã lưu nhưng tách câu, audio hoặc chú thích lỗi: xử lý như F2 (trạng thái lỗi, có nút chạy lại); bài vẫn nằm trong lộ trình.
- Ở màn hình 360px: hộp chọn, danh sách bản nháp và các nút dùng được, không cuộn ngang; thao tác được chỉ bằng bàn phím.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Trang lộ trình của một chủ đề MUST có nút **Sinh bài bằng AI**, chỉ quản trị viên thấy và dùng được; mọi yêu cầu sinh
  bài từ người không phải quản trị viên MUST bị từ chối.
- **FR-002** *(sửa 2026-10-07)*: Hộp sinh bài MUST lấy chủ đề theo lộ trình đang mở và cho chọn: trình độ (A1–C2, mặc định trình
  độ của lộ trình đang mở), số bài (số nguyên 1–5, mặc định 3), độ
  dài (số từ, 50–800, mặc định theo trình độ: A1 120, A2 160, B1 220, B2 300, C1 và C2 400), dạng bài (bài đọc hoặc hội thoại, mặc
  định bài đọc), ý chính (không bắt buộc, tối đa 500 ký tự).
- **FR-003**: Đầu vào sai MUST báo lỗi ngay ở ô tương ứng bằng tiếng Việt và không gửi yêu cầu sinh.
- **FR-004**: Trong lúc sinh, giao diện MUST hiện trạng thái đang sinh và chặn gửi thêm yêu cầu sinh cho tới khi xong; một lượt sinh
  MUST kết thúc (thành công hoặc báo lỗi) trong tối đa 60 giây.
- **FR-005**: Mỗi bản nháp MUST có tiêu đề và nội dung tiếng Anh đúng trình độ đã chọn *(sửa 2026-10-07)*, liên quan tới chủ đề, dài trong khoảng
  ±20% số từ yêu cầu, theo dạng bài đã chọn (hội thoại: mỗi lượt nói một dòng, bắt đầu bằng tên người nói).
- **FR-006**: Hệ thống MUST cung cấp cho AI tiêu đề các bài đã có trong chủ đề (mọi trình độ) để tránh lặp, và MUST loại bản nháp rỗng, trùng
  tiêu đề với bài đã có hoặc trùng nhau trong cùng lượt; nếu có bản bị loại MUST báo số bản đã loại.
- **FR-007**: Nếu sau khi loại không còn bản nháp nào, hệ thống MUST báo "AI trả về nội dung không dùng được".
- **FR-008**: Bản nháp MUST chỉ tồn tại trên trang đang mở; hệ thống MUST NOT lưu bản nháp vào dữ liệu bài học, danh sách bài hay lộ
  trình cho tới khi quản trị viên bấm Lưu.
- **FR-009**: Quản trị viên MUST sửa được tiêu đề và nội dung của từng bản nháp; mỗi bản nháp MUST hiển thị số từ của nội dung hiện
  tại.
- **FR-010**: Mỗi bản nháp MUST có nút **Lưu** và **Bỏ**; danh sách MUST có nút **Lưu tất cả** khi có từ hai bản nháp trở lên.
- **FR-011**: Bài được lưu MUST có nguồn "AI sinh", giấy phép "Nội dung do AI tạo", thuộc chủ đề đang mở và có trình độ đã chọn *(sửa 2026-10-07)*, theo cùng quy tắc kiểm tra
  của F2 (tiêu đề, nội dung, độ dài), và MUST đi qua quy trình của F2 (tách câu, audio, chú thích) như bài tạo tay.
- **FR-012**: Bài được lưu MUST được thêm vào cuối lộ trình (chủ đề, trình độ đã chọn) ngay trong cùng thao tác lưu (không có lúc bài đã tạo mà chưa
  vào lộ trình); Lưu tất cả MUST giữ thứ tự đang hiển thị.
- **FR-013**: Sau khi lưu, lộ trình MUST tải lại và hiện bài mới ở cuối kèm trạng thái xử lý (audio, chú thích) như F2.
- **FR-014**: Lưu thất bại MUST giữ nguyên bản nháp (kể cả phần đã sửa) kèm thông báo lỗi; một bản nháp MUST NOT tạo ra hai bài khi
  bấm Lưu nhiều lần.
- **FR-015**: Lỗi AI MUST được báo rõ ràng bằng tiếng Việt, phân biệt: AI chưa cấu hình, hết lượt AI, AI lỗi hoặc quá thời gian,
  nội dung không dùng được.
- **FR-016**: Khi một lượt sinh thất bại, các bản nháp đã có và các lựa chọn đã nhập MUST được giữ nguyên để thử lại; một lượt sinh
  thành công MUST thêm bản nháp mới vào danh sách, không thay bản cũ.
- **FR-017**: Rời trang (chuyển trang trong app, tải lại, đóng tab) khi còn bản nháp chưa lưu MUST hỏi xác nhận; không còn bản nháp
  thì không hỏi.
- **FR-018**: AI lỗi hoặc chưa cấu hình MUST NOT ảnh hưởng tới các chức năng khác của trang lộ trình và của app.
- **FR-019**: Hộp sinh bài và danh sách bản nháp MUST dùng được ở chiều rộng 360px không cuộn ngang, thao tác được chỉ bằng bàn phím,
  và các trạng thái (đang sinh, lỗi, đã lưu) được thông báo cho trình đọc màn hình.
- **FR-020**: Mỗi lượt sinh MUST tốn một yêu cầu AI cho cả lượt, không phải một yêu cầu cho mỗi bài.

### Key Entities

- **Yêu cầu sinh bài**: chủ đề, trình độ *(sửa 2026-10-07: chọn trong hộp, không lấy từ chủ đề)*, số bài, độ dài mục tiêu (số từ), dạng bài, ý chính; chỉ dùng trong một lượt, không
  lưu lại.
- **Bản nháp**: tiêu đề, nội dung, số từ, trạng thái trên trang (chưa lưu, đang lưu, lỗi lưu kèm thông báo); chỉ tồn tại trên trang
  của quản trị viên, mất khi rời trang.
- **Bài học** (F2, đã có): bản nháp sau khi lưu trở thành bài học với nguồn "AI sinh", giấy phép "Nội dung do AI tạo".
- **Lộ trình chủ đề** (F14, đã có): thứ tự bài của một cặp (chủ đề, trình độ); bài mới từ bản nháp được nối vào cuối.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Quản trị viên thêm được 5 bài mới vào cuối lộ trình một chủ đề (sinh, đọc lướt, lưu tất cả) trong dưới 5 phút, không
  phải nhập nguồn, giấy phép hay sắp lại thứ tự bằng tay.
- **SC-002**: 100% bản nháp nhận được có số từ trong khoảng ±20% độ dài yêu cầu, thử với mỗi trình độ A1–C2 và cả hai dạng bài.
- **SC-003**: Trong 10 lượt sinh 5 bài cho cùng một chủ đề, không có hai bài nào (trong lượt hoặc so với bài đã có) trùng tiêu đề, và
  quản trị viên đánh giá các bài khác nội dung nhau.
- **SC-004**: 0 bản nháp chưa lưu xuất hiện trong danh sách bài hoặc lộ trình (phía quản trị và phía người học).
- **SC-005**: Ở mọi loại lỗi AI (chưa cấu hình, hết lượt, lỗi, quá thời gian, nội dung không dùng được), 100% bản nháp và lựa chọn
  đã nhập vẫn còn sau thông báo lỗi, và thông báo cho biết đúng loại lỗi.
- **SC-006**: Một lượt sinh luôn kết thúc (có bản nháp hoặc thông báo lỗi) trong không quá 60 giây.
- **SC-007**: Mọi thao tác của F7 làm được ở chiều rộng 360px và chỉ bằng bàn phím.

## Assumptions

- Dùng AI đã cấu hình cho chú thích ở F2 (Gemini gói miễn phí, theo `docs/phases/giai-doan-2.md`); không thêm cấu hình mới bắt
  buộc. AI chưa cấu hình thì nút vẫn hiện nhưng sinh bài báo lỗi rõ ràng.
- "Đúng trình độ" được đảm bảo bằng yêu cầu gửi AI nêu rõ trình độ CEFR và được quản trị viên kiểm khi duyệt; hệ thống không tự chấm
  độ khó. Độ dài được kiểm bằng số từ (đếm theo khoảng trắng).
- "Khác nội dung" được hệ thống kiểm bằng tiêu đề (không trùng) và bằng cách báo AI các tiêu đề đã có; mức độ khác nhau về nội dung
  do quản trị viên đánh giá khi duyệt.
- Bản nháp mất khi rời trang (sau khi đã xác nhận) hoặc khi trình duyệt bị đóng đột ngột; không lưu bản nháp ở máy chủ hay trình
  duyệt.
- Khoảng độ dài 50–800 từ, mặc định 3 bài, giới hạn ý chính 500 ký tự là mặc định hợp lý, nằm trong giới hạn nội dung bài của F2.
- Ngoài phạm vi: sinh bài theo lịch tự động, sinh ảnh minh hoạ, sinh bài cho nhiều chủ đề một lượt.
- Phụ thuộc: F2 (tạo bài, tách câu, audio, chú thích), F14 (chủ đề, trình độ, trang lộ trình, thứ tự bài).
