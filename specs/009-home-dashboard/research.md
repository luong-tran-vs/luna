# Research: Màn hình chính và tiến độ (F6)

Không còn mục NEEDS CLARIFICATION. Các quyết định dưới đây giải quyết những điểm khối 2 để mở.

## R1. Dashboard dùng lại logic L, và chỉ đọc

- **Decision**: `StudyService.Dashboard` gọi `load` + `view` của L (bài hôm nay, các bước, streak, mục tiêu). Không gọi `Today` vì
  `Today` tự hoàn thành bước Ôn khi không có thẻ, việc đó "bắt đầu" bài hôm nay (khoá bài, đổi chủ đề thành từ mai). Mở màn hình
  chính không được thay đổi dữ liệu.
- **Rationale**: Một nguồn logic cho bài hôm nay (FR-003/004 khớp `/today`); màn hình chính an toàn khi tải lại nhiều lần.
- **Alternatives**: Gọi `Today` (có tác dụng phụ, sai quy tắc đổi chủ đề); tính lại bài hôm nay riêng (trùng logic, dễ lệch).

## R2. Nút hành động: "Bắt đầu" hay "Tiếp tục", bước nào

- **Decision**: Hàm thuần `Action(steps, currentStep, reviewCount)`: không bước nào xong → `start`, ngược lại `continue`. Bước hiển
  thị = `currentStep`, trừ khi `currentStep = review` và `reviewCount = 0` thì hiển thị `read` (khi mở `/today`, L tự hoàn thành Ôn
  rồi vào Đọc). Nút luôn dẫn tới `/today`; `/today` mở bước hiện tại ở vị trí đã lưu (L).
- **Rationale**: Khớp kịch bản 4 của US1; không cần route mới cho từng bước.
- **Alternatives**: Link thẳng `/lessons/:id/read` (bỏ qua kiểm tra thứ tự bước và vị trí đã lưu của L).

## R3. Thanh kỹ năng

- **Decision**: `ProgressRepository.StepCounts(ctx, userID, lessonIDs)` là aggregation trên `lesson_progress`: `$match {userId,
  lessonId: {$in: roadmap}}` → `$group` đếm `steps.read = true`, `steps.listen = true`, `completedAt` tồn tại. Dashboard truyền lộ
  trình hiện tại; stats truyền `nil` (mọi bài). Tổng của thanh = số bài của lộ trình.
- **Rationale**: Khối 2 yêu cầu đếm từ `lesson_progress.steps` bằng aggregation; một truy vấn cho cả ba số, dùng index `(userId,
  lessonId)`.
- **Alternatives**: Tải mọi `lesson_progress` rồi đếm trong Go (được, nhưng khối 2 chọn aggregation); bộ đếm lưu sẵn (vi phạm VII).

## R4. Số thẻ đến hạn tới hết ngày mai

- **Decision**: Mốc `cutoff` = 0 giờ ngày kia theo múi giờ tài khoản (hàm thuần `TomorrowCutoff(now, loc)`, dùng `time.Date(y, m,
  d+2, …, loc)` nên đúng qua đổi giờ mùa hè). Đếm thẻ `due < cutoff`, cộng thẻ cũ chưa có lịch (`due` không có) tạo trước 0 giờ ngày
  mai (chúng đến hạn 0 giờ ngày sau khi lưu, F5). `vocab.Repository.CountDue(ctx, userID, before, createdBefore)` dùng
  `CountDocuments` trên index `(userId, due)`.
- **Rationale**: Spec: gồm cả thẻ còn nợ hôm nay; không giới hạn 30. `Due(..., limit)` hiện có tải thẻ về, không hợp để chỉ đếm.
- **Alternatives**: Chỉ đếm thẻ có hạn đúng trong ngày mai (bỏ sót thẻ nợ, người học đánh giá sai khối lượng).

## R5. Thống kê chép chính tả

- **Decision**: `DictationRepository.Totals(ctx, userID)` aggregation `$match {userId}` → `$group {sentences: $sum 1, correct: $sum
  correctWords, total: $sum totalWords}`. Mỗi câu có một kết quả (unique `userId+lessonId+sentenceIndex`, kết quả mới ghi đè), nên
  "mỗi câu tính một lần, lần mới nhất" đúng tự nhiên. Tính cả kết quả của revision cũ (câu đã thật sự chép). Tỷ lệ = `correct /
  total`, `null` khi `total = 0` (giao diện hiện "—"), làm tròn phần trăm ở frontend.
- **Rationale**: Một truy vấn, tiền tố `userId` của index unique.
- **Alternatives**: Chỉ tính revision hiện tại (phải join `lessons`, phức tạp mà ít giá trị).

## R6. Số thẻ (từ đã học)

- **Decision**: `vocab.Repository.Count(ctx, userID)` = `CountDocuments {userId}` (index `(userId, lemma)`). Lộ ra qua port
  `Reviews.CardCount` của progress (adapter `dailyReviews` trong main).
- **Alternatives**: Cộng `LessonCounts` (aggregation thừa).

## R7. Index

- **Decision**: Không thêm index. `cards (userId, due)` có từ F5; `lesson_progress (userId, completedAt)` và unique `(userId,
  lessonId)` có từ L; `dictation_results` unique `(userId, lessonId, sentenceIndex)` có từ F4. Kiểm tra bằng `explain` trong
  quickstart.
- **Rationale**: Khối 2 yêu cầu các index này; chúng đã tồn tại, thêm nữa là trùng.

## R8. Tải lại số liệu khi quay về (FR-008)

- **Decision**: Trang home gọi `/api/dashboard` khi component được tạo. Router không tái sử dụng component giữa các route, nên mỗi lần
  quay về `/` (kể cả nút Back) là một lần tải mới. Không cache ở service. Khi đang tải hiện khung giữ chỗ cùng chiều cao để nút không
  nhảy.
- **Alternatives**: Store toàn cục + invalidation sau mỗi bước (phức tạp, không cần với 1 người học).

## R9. Trạng thái kết nối ở chân trang (FR-009)

- **Decision**: Tách phần kiểm tra `/api/health` của F0 thành `shared/components/connection-status`, đặt trong `<footer>` của `app.html`
  (mọi trang). Kiểm tra khi mở app và sau mỗi `NavigationEnd`; chỉ render khi `database-down` hoặc `server-unreachable`
  (`role="status"`, thông báo tiếng Việt như F0). Khi bình thường không có gì trên màn hình.
- **Rationale**: Không chiếm chỗ ở màn hình đầu 360px; vẫn báo lỗi như F0.
- **Alternatives**: Chỉ kiểm tra ở home (lỗi ở trang khác không thấy); polling định kỳ (thừa).

## R10. progress-bar và step-indicator

- **Decision**: `lu-progress-bar` inputs `label`, `value`, `max`, `tone: 'primary' | 'read' | 'listen'` (màu `--color-primary`,
  `--color-skill-read`, `--color-skill-listen`, nền `--color-track`), hiện chữ "value/max" + đơn vị do nơi dùng truyền (`unit`); div
  `role="progressbar"` với `aria-valuenow/min/max`, `aria-label`; `max = 0` → thanh rỗng, không chia cho 0. `step-indicator` (L) dùng
  `progress-bar` cho dòng "N/3 bước" → `/today` và home cùng hiển thị.
- **Alternatives**: `<progress>` gốc (khó đổi màu nhất quán giữa trình duyệt, nhất là chế độ tối).

## R11. Bố cục 360px

- **Decision**: Thứ tự: hàng đầu (tiêu đề "Hôm nay" + streak "🔥 N ngày"), thẻ mục tiêu (tên lộ trình, thanh mục tiêu, 2 thanh kỹ năng
  nhỏ, link "Đổi chủ đề"), thẻ bài hôm nay (tên bài, "trình độ · chủ đề", step-indicator), nút chính full-width ≥ 48px, rồi "Ngày mai: N
  thẻ cần ôn" và link "Xem thống kê". Tên dài xuống dòng (`overflow-wrap: anywhere`), giới hạn tên bài 2 dòng. Ước lượng chiều cao phần
  trên nút ≈ 420px < 640 − header − thanh trình duyệt.
- **Rationale**: Mockup 4.1; FR-007/SC-001. Kiểm tra thủ công ở 360 × 640 cả sáng và tối.
