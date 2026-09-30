# Research: Nghe (F4)

Không còn mục NEEDS CLARIFICATION.

## R1. Chuẩn hoá và so sánh chép chính tả (FR-008, FR-009)

- **Chuẩn hoá** (áp cho cả câu đúng và câu gõ):
  1. `’` `‘` → `'`; chữ thường.
  2. Gạch nối, gạch ngang (`-`, `–`, `—`) → khoảng trắng.
  3. Giữ dấu chấm/phẩy **giữa hai chữ số** (`9.30`, `1,000`); giữ `'` **giữa hai chữ cái** (`don't`, `it's`,
     `rock'n'roll`); mọi ký tự không phải chữ, số, khoảng trắng còn lại → khoảng trắng.
  4. Tách theo khoảng trắng, bỏ phần rỗng.
- **Căn chỉnh**: khoảng cách chỉnh sửa (Levenshtein) trên mảng từ, chi phí khớp 0, thay 1, xoá 1, chèn 1; truy vết từ cuối,
  ưu tiên khớp → thay → thiếu → thừa khi hoà. Kết quả theo thứ tự câu đúng:
  - khớp → `{status: 'ok', word, expected}`
  - thay → `{status: 'wrong', word, expected}`
  - từ đúng không được gõ → `{status: 'missing', expected}`
  - từ gõ thừa → `{status: 'wrong', word}` (không có `expected`)
- **Đếm**: `correctWords` = số `ok`; `totalWords` = số từ của câu đúng + số từ thừa (spec Assumptions).
- **Hiệu năng**: câu 40 từ × 40 từ = 1 600 ô → < 1ms.
- **Bộ mẫu test** (SC-001): khớp hoàn toàn; thiếu đầu/giữa/cuối; thừa đầu/giữa/cuối; sai một từ; gõ trống; hoa thường; dấu
  câu; `don't`/`dont`; `’`; gạch nối; số `9.30`; câu một từ; gõ toàn sai; lặp từ ("the the").
- **Alternatives considered**: LCS thuần (không phân biệt được sai và thiếu+thừa); so sánh theo vị trí (lệch toàn bộ khi thiếu
  một từ); thư viện diff (thừa, không theo từ).

## R2. So sánh ở frontend, backend chỉ lưu

- **Decision**: So sánh chỉ chạy ở trình duyệt (kết quả tức thì, không cần mạng). Client gửi `{sentenceIndex, typed,
  correctWords, totalWords}`; backend kiểm tra phạm vi (`0 ≤ correct ≤ total ≤ 1000`, `total ≥ 1`, `typed` 1–1000 ký tự,
  `sentenceIndex` < số câu hiện tại) rồi lưu.
- **Rationale**: Người học tự luyện, số liệu chỉ dùng cho thống kê của chính họ; viết lại thuật toán bằng Go để tự tính lại
  là trùng lặp (nguyên tắc VII). `typed` được lưu để mở lại bài hiện lại kết quả từng từ (FR-017).
- **Alternatives considered**: backend tự so sánh (hai bản thuật toán phải giữ đồng bộ).

## R3. Lưu kết quả và tổng kết (FR-015–FR-017)

- `dictation_results`: `{userId, lessonId, lessonRevision, sentenceIndex, typed, correctWords, totalWords, checkedAt}`, unique
  `(userId, lessonId, sentenceIndex)`; ghi bằng upsert (lần mới nhất thay lần cũ).
- `lessonRevision` lấy từ bài lúc lưu (client không gửi). Tổng kết chỉ tính kết quả có `lessonRevision` bằng revision hiện
  tại và `sentenceIndex < số câu` → sửa nội dung bài thì kết quả cũ tự bị bỏ (edge case).
- Tổng kết: `sentenceCount`, `checkedCount`, `correctWords`, `totalWords`, `rate = correct/total` (0 khi chưa có),
  `completed = checkedCount == sentenceCount`, `results[]` theo thứ tự câu.
- `POST` trả luôn tổng kết mới để frontend cập nhật tiến độ và biết đã hoàn thành.

## R4. Hiển thị kết quả đạt WCAG và không cần màu (FR-010)

| Trạng thái | Hiển thị | Màu (chỉ ở đường gạch / biểu tượng) | Nhãn trình đọc màn hình |
|---|---|---|---|
| đúng | chữ bình thường | — | (không) |
| sai | chữ gõ **gạch ngang**, rồi `→` và từ đúng in đậm | đường gạch `--color-bad` | "sai, đúng là …" |
| thừa | chữ gõ gạch ngang, không có từ đúng | `--color-bad` | "thừa" |
| thiếu | từ đúng với **gạch chân chấm** | `--color-warn` | "thiếu" |

Chữ luôn dùng `--color-text` (tương phản ≥ 13:1); màu trạng thái chỉ tô `text-decoration-color` và viền (≥ 3:1 như đã tính ở
F2). Nhãn trình đọc màn hình bằng phần tử ẩn `.visually-hidden`.

## R5. Phát audio từng câu (FR-001–FR-006)

- `AudioPlayer` (`shared/components/audio-player`): bọc một `HTMLAudioElement`; input `src`, `rate`; phương thức `play()`,
  `replay()` (đặt `currentTime = 0` rồi phát), `stop()`; phát lại `playbackRate` sau mỗi lần đổi `src` (trình duyệt đặt lại
  về 1); output `playing`, `ended`, `failed`. Đổi `src` thì dừng câu cũ (FR-006).
- Tốc độ: nhóm 4 nút chọn một (`role="radiogroup"`), giữ trong signal của trang.
- Phím tắt khi không gõ trong ô: không thêm (đơn giản); Enter trong ô gõ = Kiểm tra.

## R6. Bàn phím ảo trên điện thoại (FR-013)

- `index.html` viewport thêm `interactive-widget=resizes-content` (Chrome Android thu nhỏ khung nhìn thay vì phủ lên).
- Thanh trả lời (ô gõ + Kiểm tra) `position: sticky; bottom: 0` với nền `--color-surface`; khi ô gõ nhận focus gọi
  `scrollIntoView({block: 'nearest'})`.
- Ô gõ: `<input type="text">` với `autocapitalize="off" autocorrect="off" autocomplete="off" spellcheck="false"`
  `enterkeyhint="done"` để bàn phím không tự sửa chữ.

## R7. Lưu thất bại (edge case)

- Kết quả vẫn hiện ngay; lần gửi lỗi được giữ trong hàng đợi của trang (theo `sentenceIndex`, chỉ giữ lần mới nhất) và gửi lại
  trước lần kiểm tra tiếp theo hoặc khi bấm "Thử lưu lại". Hiện "Chưa lưu được, sẽ thử lại".

## R8. Route và truy cập tạm

- Route `lessons/:id/listen` (`authGuard`). Trang dùng `GET /api/lessons/{id}` (F3) cho câu và `audioUrl`; không câu nào có
  audio → "Audio của bài chưa sẵn sàng" và không hiện ô chép chính tả (US1-6); một số câu thiếu audio → câu đó báo "Chưa có
  audio", vẫn gõ theo transcript được.
- Trang chi tiết bài admin thêm nút "Mở bước Nghe".
