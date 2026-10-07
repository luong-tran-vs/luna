# Feature Specification: Kho từ vựng dùng chung

**Feature Branch**: `018-word-bank`

**Created**: 2026-10-07

**Status**: Implemented

**Mã tính năng**: F24 | **Giai đoạn**: tính năng quản trị

**Input**: User description: "Phía trang admin, tôi muốn có thêm một tab về từ vựng, để có thể quản lý từ vựng (thêm ảnh,...)".
Đã chốt: kho từ dùng chung (mỗi từ một mục cho mọi bài và chủ đề); mỗi từ quản lý ảnh, nghĩa và phiên âm; bài dùng ảnh của
kho khi từ chưa có ảnh riêng trong bài (ảnh riêng của bài F23 vẫn giữ và được ưu tiên).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Quản lý kho từ (Priority: P1)

Quản trị viên mở mục "Từ vựng" trên thanh bên (`/admin/words`), xem các từ của kho theo thứ tự chữ cái, tìm theo từ hoặc
nghĩa, lọc từ thiếu ảnh / thiếu phiên âm / thiếu nghĩa, thêm từ, sửa nghĩa và phiên âm, xoá từ.

**Acceptance Scenarios**:

1. **Given** kho có "house" và "table", **When** mở trang, **Then** thấy "2 từ", mỗi từ có ảnh thu nhỏ (hoặc "Chưa có ảnh"),
   phiên âm (hoặc "Chưa có phiên âm") và nghĩa (hoặc "Chưa có nghĩa"); mỗi trang 50 từ, nút "Xem thêm" tải trang sau.
2. **Given** ô tìm kiếm, **When** gõ "nhà" rồi dừng gõ, **Then** chỉ còn các từ có "nhà" trong từ hoặc nghĩa (không phân biệt
   hoa thường); chọn lọc "Thiếu ảnh" thì chỉ còn từ chưa có ảnh.
3. **Given** form Thêm từ, **When** nhập "House" và để trống nghĩa, phiên âm, **Then** từ được lưu là "house" với phiên âm và
   nghĩa lấy từ từ điển offline (2 nghĩa đầu, cách nhau "; "); cụm từ được ghép phiên âm từng từ nếu từ điển có đủ mọi từ.
4. **Given** "house" đã có, **When** thêm "HOUSE", **Then** báo "Từ này đã có trong kho" dưới form.
5. **Given** một từ, **When** bấm "Sửa", đổi nghĩa và phiên âm rồi "Lưu", **Then** dòng đó hiện giá trị mới; từ không sửa được.
6. **Given** một từ, **When** bấm "Xoá từ" và xác nhận, **Then** từ và ảnh của nó bị xoá khỏi kho; các bài vẫn giữ nghĩa và
   ảnh riêng của bài.
7. **Given** các chủ đề có từ vựng (F18), **When** bấm "Nhập từ các chủ đề", **Then** mọi từ của chủ đề chưa có trong kho được
   thêm (phiên âm và nghĩa từ từ điển), báo "Đã thêm N từ từ các chủ đề."; từ đã có giữ nguyên; bấm lại thì không thêm gì.

### User Story 2 - Ảnh của từ (Priority: P1)

Mỗi từ có tối đa một ảnh: tải lên, lấy từ link, sinh bằng AI, đổi hoặc xoá.

**Acceptance Scenarios**:

1. **Given** một từ, **When** tải lên ảnh JPEG/PNG/GIF ≤ 5 MB, **Then** máy chủ thu nhỏ (≤ 360 px, JPEG) và dòng đó hiện ảnh mới.
   File khác loại hoặc quá lớn bị từ chối ngay ở trình duyệt.
2. **Given** một từ, **When** bấm "Dán link ảnh" và dán link http(s), **Then** máy chủ tải ảnh về như ảnh tải lên (không lấy
   từ địa chỉ nội bộ, giống F23).
3. **Given** một từ, **When** bấm "Sinh ảnh AI", **Then** một request tới model ảnh (có tính phí, như F23) vẽ ảnh theo từ và
   nghĩa, thay ảnh cũ; lỗi AI (chưa cấu hình, hết lượt, sinh hỏng) hiện thông báo tiếng Việt dưới dòng đó.
4. **Given** từ có ảnh, **When** bấm "Xoá ảnh", **Then** từ trở về "Chưa có ảnh".

### User Story 3 - Bài học và ôn tập dùng kho từ (Priority: P1)

**Acceptance Scenarios**:

1. **Given** bài có từ "give up" chưa có ảnh riêng và kho có ảnh "give up", **When** người học mở bước Từ vựng, **Then** từ đó
   hiện ảnh của kho. Từ có ảnh riêng trong bài vẫn hiện ảnh của bài.
2. **Given** kho có phiên âm của một từ, **When** người học xem từ vựng của bài, **Then** phiên âm của kho được dùng thay cho
   từ điển; thẻ lưu từ bài mang phiên âm đó.
3. **Given** thẻ ôn tập trống phiên âm (F5, sửa 2026-10-07), **When** lấy ra ôn, **Then** phiên âm được điền từ kho trước, rồi
   mới đến từ điển.

### Edge Cases

- Từ được chuẩn hoá: chữ thường, dấu nháy cong thành thẳng, gộp khoảng trắng (giống từ gốc của bài).
- Từ tối đa 100 ký tự, nghĩa 300, phiên âm 100; từ khoá tìm kiếm 100.
- Sinh ảnh có thể lâu: request đó được phép kéo dài tới 90 giây.
- Xoá từ không xoá ảnh riêng của bài, không đổi thẻ ôn tập đã lưu.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Kho từ PHẢI có đúng một mục cho mỗi từ (đã chuẩn hoá), gồm nghĩa tiếng Việt, phiên âm (đều có thể trống), ảnh
  (tuỳ chọn), thời điểm tạo và sửa.
- **FR-002**: Chỉ quản trị viên dùng được API `/api/admin/words` (liệt kê, thêm, nhập từ chủ đề, sửa, xoá, ảnh).
- **FR-003**: Thêm từ hoặc nhập từ chủ đề PHẢI điền phiên âm và nghĩa còn trống từ từ điển offline, chỉ khi từ điển có đúng dạng
  từ đó.
- **FR-004**: Ảnh lưu trong DB như F23 (Mongo `word_bank_images`, MySQL migration 021 `word_bank_images`); đường dẫn ảnh đổi
  khi ảnh đổi.
- **FR-005**: Bước Từ vựng và ảnh từ của bài PHẢI dùng ảnh của kho khi bài không có ảnh riêng cho từ đó; phiên âm của kho được
  ưu tiên hơn từ điển ở từ vựng của bài và khi bổ sung phiên âm cho thẻ ôn tập.
- **FR-006**: AI không tự sinh ảnh cho kho; ảnh AI chỉ sinh khi quản trị viên bấm cho từng từ.

### Key Entities

- **Từ trong kho (Bank word)**: từ (khoá), nghĩa, phiên âm, thời điểm có ảnh, thời điểm tạo, sửa.
- **Ảnh của từ (Bank image)**: từ, loại (image/jpeg), dữ liệu, thời điểm lưu.

## Assumptions

- Kho chỉ nhập từ các chủ đề; từ trong bài mà không thuộc chủ đề nào được thêm tay.
- Nghĩa trong kho là nghĩa chung để quản lý; bài học vẫn dùng nghĩa theo ngữ cảnh do AI chú thích.
