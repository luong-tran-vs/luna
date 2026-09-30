# Research: Sổ từ và ôn tập (F5)

Không còn mục NEEDS CLARIFICATION.

## R1. Thư viện FSRS

- **Decision**: `github.com/open-spaced-repetition/go-fsrs/v3` **v3.3.1** (MIT, không phụ thuộc gói nào, `go 1.21` nên hợp với
  Go 1.25.1 mà không phải nâng `go.mod`). Dùng `fsrs.NewFSRS(fsrs.DefaultParam())`:
  - `DefaultParam()`: `RequestRetention 0.9`, `MaximumInterval 36500`, `EnableShortTerm true`, **`EnableFuzz false`** → kết
    quả tất định, test so được với giá trị tham chiếu (SC-003).
  - `f.Repeat(card, now)` trả `RecordLog` (4 mức) → dùng để hiện khoảng cách trên từng nút (FR-012) và lấy mức đã chọn.
  - `fsrs.Card{Due, Stability, Difficulty, ElapsedDays, ScheduledDays, Reps, Lapses, State, LastReview}`; `State`: New,
    Learning, Review, Relearning.
  - Có học ngắn hạn: thẻ mới chọn Again/Hard/Good → đến hạn sau 1/5/10 phút (Learning); Easy → vài ngày (Review).
- **Rationale**: Đầu vào khối 2 chỉ định; thư viện chính chủ của nhóm FSRS, nhỏ, không cần mạng.
- **Alternatives considered**: tự cài FSRS (dễ sai công thức); SM-2 (đầu vào yêu cầu FSRS).

## R2. Thẻ mới đến hạn hôm sau theo múi giờ (FR-010, SC-002)

- `vocab.FirstDue(savedAt, tz)`: 0 giờ ngày hôm sau của `savedAt` trong `time.LoadLocation(users.timezone)` (lỗi →
  `Asia/Ho_Chi_Minh`, như F1). Tính bằng `time.Date(y, m, d+1, 0, 0, 0, 0, loc)` nên đúng cả ngày đổi giờ mùa hè.
- Thẻ tạo từ F5 trở đi lưu `due = FirstDue(createdAt)` và `state = New`.
- **Thẻ F3 cũ** (chưa có trường lịch): không chạy migration. Truy vấn đến hạn dùng
  `due ≤ now OR (due không tồn tại AND createdAt < 0 giờ hôm nay theo múi giờ)` — tương đương `FirstDue(createdAt) ≤ now`.
  Khi đọc, thẻ thiếu trường lịch được coi là `fsrs.NewCard()` với `Due = FirstDue(createdAt)`. Lần đánh giá đầu tiên ghi đủ
  trường.
- Múi giờ lấy qua port `vocab.Timezones` (adapter trên `auth` trong `main.go`), không nhận từ client.
- **Alternatives considered**: backfill lúc khởi động (phải đọc múi giờ từng người, thêm bước chạy một lần); tính theo giờ UTC
  (sai với người học ở Việt Nam sau 17h).

## R3. Đánh giá không bị tính hai lần

- `POST /api/vocab/cards/{id}/review {rating, mode, reps}`: `reps` là số lần ôn của thẻ mà client đang thấy. Backend tính lịch
  mới và ghi bằng cập nhật có điều kiện `{_id, userId, reps}` (thẻ cũ thiếu `reps` coi như 0). Không khớp → **409
  `review_conflict`** kèm thẻ hiện tại; client coi thẻ đó đã được đánh giá và đi tiếp.
- Log ôn chỉ ghi sau khi cập nhật thẻ thành công. Lưu thất bại (mạng) → client giữ thẻ, cho "Thử lại"; nếu lần trước thực ra đã
  ghi thì lần thử lại nhận 409 → không có đánh giá kép (edge case của spec).
- **Alternatives considered**: khoá idempotency theo request id (thêm collection); transaction Mongo (compose chạy
  standalone, không có replica set).

## R4. Sổ từ nhóm theo ngày và tải dần (FR-002–FR-004)

- `GET /api/vocab/cards?q=&lessonId=&page=`: sắp `createdAt` giảm dần, 30 thẻ/trang, trả `hasMore`. Mỗi thẻ có `day`
  (`YYYY-MM-DD` theo múi giờ của người học); phản hồi có `today` và `yesterday` cùng múi giờ → frontend gắn nhãn "Hôm nay",
  "Hôm qua", còn lại `dd/MM/yyyy`, gộp các thẻ liền nhau cùng `day` (kể cả khi nối trang sau).
- Đầu vào khối 2 có `groupBy=day`: backend **luôn** trả `day`, không cần tham số (nguyên tắc VII). Tính ngày ở backend để
  nhóm theo múi giờ tài khoản chứ không theo múi giờ của trình duyệt đang mở.
- `q`: chuỗi đã trim, ≤ 100 ký tự, `regexp.QuoteMeta`, không phân biệt hoa thường, khớp `text` hoặc `lemma`.
- `lessonId`: id bài, hoặc `manual` cho thẻ tự thêm. `GET /api/vocab/lessons` trả các bài có thẻ `{id, title, count}` + số thẻ
  tự thêm, để dựng bộ lọc (bài đã xoá không có tên → không liệt kê, thẻ vẫn thấy khi không lọc).
- Phân trang theo `page` (như đầu vào) là đủ cho một người học vài nghìn thẻ.

## R5. Thêm, sửa, xoá thẻ (FR-005–FR-008)

- Tự thêm: `POST /api/vocab/cards` (F3) nới `lessonId` thành tuỳ chọn; `source: manual`; `lemma` = từ viết thường gộp khoảng
  trắng. Trùng → 409 như F3 ("Từ này đã có trong sổ").
- Sửa: `PATCH /api/vocab/cards/{id} {meaningVi?, ipa?, contextSentence?}` chỉ `$set` ba trường này → không động tới trường
  lịch (FR-006, SC-004). `text`/`lemma` không sửa được (trường lạ bị `DecodeJSON` từ chối → 400).
- Xoá: `DELETE /api/vocab/cards/{id}` xoá thẻ `{_id, userId}` rồi xoá `review_logs {userId, cardId}`. Không có transaction
  nên làm idempotent: gọi lại khi thẻ đã mất vẫn dọn log; 404 chỉ khi không có thẻ và không có log.

## R6. Mục Từ vựng của bài và Lưu tất cả (FR-021–FR-025)

- `GET /api/lessons/{id}/vocabulary` (gói `lesson`, `Reader.Vocabulary`): từ `Annotations` của bài, gộp theo `lemma`, giữ
  chú thích đầu tiên theo thứ tự câu: `{lemma, text, meaningVi, ipa, sentenceIndex, sentence}`; IPA lấy từ từ điển offline
  (`dict.Resolve(lemma)`), như popup F3. Không gọi AI. `available: false` khi `annotationStatus ≠ done` hoặc không có chú
  thích → frontend hiện "Chưa có danh sách từ vựng".
- **Cờ `saved`**: đầu vào khối 2 đề xuất trả kèm `saved`. Trang Đọc đã tải `GET /api/vocab/words` (F3) để tô từ đã lưu, và
  cần cập nhật ✓ ngay khi lưu từ popup (FR-023) → frontend tính ✓ từ tập đó, endpoint không cần biết người học (không thêm
  port chéo `lesson → vocab`).
- `POST /api/vocab/cards/bulk {lessonId, lemmas[]}` (1–200 lemma): backend lấy dữ liệu từ `Reader.Vocabulary` qua port
  `vocab.LessonVocabulary` (không tin nghĩa do client gửi), tạo thẻ `source: ai` với `contextSentence` = câu chứa từ, bỏ qua
  lemma không có trong mục Từ vựng; ghi từng thẻ, lỗi trùng khoá (index unique `userId+lemma`) được bỏ qua → trả
  `{added, cards}`. Nút **Lưu** của một mục dùng cùng endpoint với 1 lemma; **Lưu tất cả** gửi các lemma chưa có ✓.
- **Alternatives considered**: `InsertMany` không thứ tự (lỗi trùng trộn lẫn lỗi khác, khó báo số đã thêm); client gửi đủ dữ
  liệu thẻ (tin client, lặp lại logic chọn câu ví dụ).

## R7. Phiên ôn tập dùng lại được (FR-013–FR-020)

- `GET /api/vocab/review/due?limit=` (mặc định 50, tối đa 200): thẻ đến hạn sắp `due` tăng dần (quá hạn lâu nhất trước; thẻ
  cũ thiếu `due` coi như đến hạn từ lâu), kèm `total` và `nextDue` (thời điểm sớm nhất của thẻ chưa đến hạn, `null` nếu sổ
  trống). Mỗi thẻ kèm `intervals {again, hard, good, easy}` (giây, tính bằng `Repeat` tại thời điểm trả).
- `shared/components/review-session`: input `cards`, `mode` (`flip` | `listen`); tự gọi API đánh giá (qua
  `core/services/vocab-api.service.ts`); output `finished` với `{reviewed, counts}`. Thẻ chọn Again được thêm vào cuối hàng
  đợi một lần (dùng lịch mới do phản hồi trả về để hiện khoảng cách). L sẽ dùng lại component này.
- Trang `/vocabulary/review`: tải thẻ đến hạn, chọn kiểu, chạy phiên; hết thẻ → "Không có thẻ nào đến hạn" + `nextDue`.
- Nhãn khoảng cách (`shared/utils/interval-label.ts`): < 1 giờ → "N phút", < 1 ngày → "N giờ", < 30 ngày → "N ngày", < 365 →
  "N tháng", còn lại "N năm".
- Bàn phím: Space/Enter lật; 1–4 chọn mức khi đã lật/kiểm tra và focus không ở ô gõ.

## R8. So khớp Nghe rồi gõ (FR-016, SC-005)

- `shared/utils/answer-match.ts`: `normalizeAnswer(s)` = `’‘` → `'`, chữ thường, trim, gộp khoảng trắng; đúng khi
  `normalizeAnswer(typed) === normalizeAnswer(card.text)`. Không bỏ dấu câu bên trong (từ vựng thường không có), không chấp
  nhận sai chính tả gần đúng (người học tự chọn mức đánh giá).
- Audio: `/api/tts/word?text=` (F3, có cache); phát bằng `AudioPlayer` (F4).

## R9. Điều hướng

- Header thêm link **Sổ từ** (`/vocabulary`) cho người đã đăng nhập; trang sổ từ có nút **Ôn tập** (`/vocabulary/review`) kèm
  số thẻ đến hạn. Route `vocabulary` lazy, `authGuard`.
