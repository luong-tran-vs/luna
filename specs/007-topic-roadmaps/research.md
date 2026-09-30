# Research: Chủ đề và lộ trình theo trình độ (F14)

Không còn mục NEEDS CLARIFICATION.

## R1. Ranh giới gói `topic` và `lesson`

- **Decision**: Gói mới `internal/topic` sở hữu chủ đề và lộ trình (lộ trình là `lessonIds` trong document chủ đề). Hai gói
  **không import nhau**; mỗi bên khai báo port nhỏ, `main.go` nối bằng adapter (như `progress`, `vocab`):
  - `lesson.Topics` (hiện thực bởi `topic.Service`): `Get(id) (TopicRef{ID, Name, Level})`, `Names() map[id]TopicRef`,
    `RoadmapLessonIDs() set`, `MoveLesson(lessonID, from, to)`.
  - `topic.Lessons` (hiện thực trên `lesson.Repository`): `CountByTopic() map[topicID]int`, `TopicOf(ids) map[lessonID]topicID`,
    `SetLevelByTopic(topicID, level)`.
- **Rationale**: Tránh vòng import; mỗi gói test được với fake (nguyên tắc V, VI).
- **Alternatives considered**: đặt chủ đề trong gói `lesson` (gói phình to, trái đầu vào khối 2); `topic` import `lesson` và
  ngược lại (vòng import, Go không cho).

## R2. Bài học: `topicId` thay `topic`

- `Lesson.TopicID` (bắt buộc) thay `Lesson.Topic`; `Level` vẫn lưu trên bài (lọc nhanh) và **luôn lấy từ chủ đề** khi tạo/sửa
  (`Input` không còn `level`, `topic`; thêm `topicId`). Chủ đề không tồn tại → lỗi trường `topicId` "Chủ đề không tồn tại".
- Đổi chủ đề (FR-008): `lesson.Service.Update` gọi `Topics.MoveLesson(id, cũ, mới)` — `$pull` khỏi chủ đề cũ; nếu bài có ở đó
  thì `$push` vào cuối chủ đề mới. Hai document không có transaction: làm `$pull` trước, lỗi giữa chừng chỉ khiến bài rời lộ
  trình (an toàn, quản trị viên thêm lại).
- Sửa trình độ chủ đề (FR-005): `topic.Service.Update` gọi `Lessons.SetLevelByTopic` (`updateMany`).
- Xoá bài (FR-015): chặn khi `id ∈ Topics.RoadmapLessonIDs()`; danh sách bài đánh dấu `inRoadmap` cùng cách.
- Tên chủ đề hiện ở danh sách/chi tiết bài và trang Đọc (F3 tag): handler ghép qua `Topics.Names()`; `ReadingView.Topic` giữ
  kiểu chuỗi (tên chủ đề) để frontend F3 không đổi.
- Lọc: `GET /api/admin/lessons?level=&topicId=` (bỏ `topic` gõ tự do).

## R3. Chủ đề

- Document `topics {_id, name, nameKey, level, description, lessonIds[], createdAt, updatedAt}`; `nameKey` = tên viết thường,
  gộp khoảng trắng; **unique index `(level, nameKey)`** → trùng tên trả lỗi trường `name` (FR-002).
- Kiểm tra: tên 1–60, mô tả ≤ 200, trình độ A1–C2.
- Xoá (FR-003): `Lessons.CountByTopic()[id] > 0` → `*InUseError{Count}` → 409 `topic_in_use` "Chủ đề còn N bài…".
- Lộ trình (FR-011–013): `SetRoadmap(topicID, ids)` kiểm tra không trùng, mọi id tồn tại và `TopicOf(id) == topicID`; sai →
  lỗi trường `lessonIds`. Ghi bằng `$set lessonIds`.
- Cảnh báo (FR-014): `remaining = len(lessonIds)` (chưa có tiến độ, như F2), `warning = remaining < 3`; trả kèm mỗi chủ đề trong
  `GET /api/admin/topics` để trang Lộ trình liệt kê cảnh báo của mọi chủ đề.
- Người học (FR-020): `GET /api/topics?level=` (`RequireAuth`) trả `{id, name, level, description, lessonCount}` với
  `lessonCount = len(lessonIds)`.

## R4. Chuyển dữ liệu F2 → F14 (FR-016–018)

- **Decision**: tách **lập kế hoạch** (thuần, test được) khỏi **áp dụng** (Mongo):
  - `topic.PlanMigration(lessons []LegacyLesson, roadmap []string, existing []Topic) Plan` trong `internal/topic/migrate.go`:
    `LegacyLesson{ID, Level, TopicName, TopicID}`; gộp theo `(level, nameKey)`, tên rỗng → "Chung", giữ cách viết của bài
    xuất hiện đầu tiên (theo `createdAt`); dùng lại chủ đề đã có cùng khoá (chạy lại sau khi bị ngắt); bài đã có `topicId` giữ
    nguyên; lộ trình mỗi chủ đề = các id của lộ trình cũ thuộc chủ đề đó theo thứ tự cũ, id không còn bài bị bỏ.
  - `mongo.MigrateTopics(ctx, db)` trong `internal/storage/mongo/migrate.go`, gọi khi khởi động **trước** khi nhận request:
    1. có marker `migrations {_id: "f14-topics"}` → bỏ qua;
    2. đọc lessons (`topic`, `topicId`), roadmap `main` (nếu còn), topics đang có → `PlanMigration`;
    3. upsert chủ đề theo `(level, nameKey)`; chỉ ghi `lessonIds` khi còn document roadmap (tránh xoá lộ trình đã chia nếu lần
       trước đã xoá roadmap);
    4. `$set topicId, level` + `$unset topic` cho từng bài;
    5. ghi marker, rồi xoá collection `roadmap`.
- **Rationale**: Nguyên tắc VI (test không cần mạng/DB) cho phần khó (gộp tên, giữ thứ tự, chạy lại); phần Mongo chỉ là ghi
  theo kế hoạch, idempotent nhờ upsert theo khoá duy nhất và marker; kiểm tra bằng smoke test Docker (chạy 2 lần).
- **Alternatives considered**: lệnh migration chạy tay (dễ quên, trái "tự động"); migration trong Mongo shell script (không
  test được bằng Go); thư viện migration (thừa cho một lần, nguyên tắc VII).

## R5. Frontend

- `core/models/topic.ts`; `AdminApiService` thêm `topics()`, `createTopic`, `updateTopic`, `deleteTopic`, `topicRoadmap(id)`,
  `setTopicRoadmap(id, ids)`; bỏ `roadmap()`/`setRoadmap()`.
- `features/admin/topics/` (mới, route `/admin/topics`): danh sách nhóm theo trình độ, form thêm/sửa tại chỗ (reactive form),
  `ConfirmDialog` khi xoá, lỗi 409 hiện rõ số bài.
- `lesson-form`: một `<select>` chủ đề với `<optgroup label="A1">`; không có chủ đề → liên kết "Tạo chủ đề".
- `lesson-list`: hai `<select>` trình độ, chủ đề (chủ đề lọc theo trình độ đang chọn; đổi trình độ xoá chủ đề không khớp).
- `roadmap`: route `/admin/roadmap?topicId=`; ô chọn chủ đề (optgroup theo trình độ); phần CDK drag-drop hiện có giữ nguyên, gọi
  API theo `topicId`; khối cảnh báo liệt kê mọi chủ đề có `warning` (tên + số bài còn lại), bấm để chọn chủ đề đó.
- Điều hướng quản trị thêm "Chủ đề".
