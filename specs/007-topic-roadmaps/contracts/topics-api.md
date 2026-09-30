# Contract: Topics API và thay đổi Admin Lessons API

Endpoint `/api/admin/*` cần `RequireAdmin` (401 chưa đăng nhập, 403 không phải quản trị viên). Lỗi dạng chung
`{error, message, fields?}`. Trường lạ trong body → 400 `invalid_body`.

## Topic JSON (quản trị)

```json
{ "id": "…", "name": "Gia đình", "level": "A1", "description": "Người thân, nhà cửa",
  "lessonCount": 4, "roadmapCount": 2, "remaining": 2, "warning": true, "createdAt": "…" }
```

`remaining` = số bài chưa học trong lộ trình (F14: bằng `roadmapCount`); `warning` = `remaining < 3`.

## GET /api/admin/topics?level=

200 `{ "topics": [Topic] }` sắp theo trình độ A1 → C2 rồi theo tên. `level` tuỳ chọn; sai → 400.

## POST /api/admin/topics

`{ "name": "Gia đình", "level": "A1", "description": "" }` → 201 `{topic}`. 400 `validation_failed`: `name` (trống, > 60,
**trùng trong cùng trình độ**: "Chủ đề này đã có ở trình độ A1"), `level`, `description` (> 200).

## PUT /api/admin/topics/{id}

Body như POST. 200 `{topic}`; đổi `level` thì mọi bài của chủ đề đổi trình độ theo. 400 như POST; 404 `not_found`.

## DELETE /api/admin/topics/{id}

204. 409 `topic_in_use` `{error, message: "Chủ đề còn 3 bài, hãy chuyển hoặc xoá bài trước", count: 3}`; 404.

## GET /api/admin/topics/{id}/roadmap

200 `{ "topic": Topic, "lessons": [LessonSummary], "remaining": 2, "warning": true }` (bài đã xoá bị bỏ qua); 404.

## PUT /api/admin/topics/{id}/roadmap

`{ "lessonIds": ["…", "…"] }` → 200 như GET. 400 `lessonIds`: trùng ("Lộ trình có bài bị trùng"), không tồn tại, **thuộc chủ
đề khác** ("Chỉ thêm được bài của chủ đề này"); 404 chủ đề.

## GET /api/topics?level= (người học, `RequireAuth`)

200 `{ "topics": [{ "id", "name", "level", "description", "lessonCount" }] }`, `lessonCount` = số bài trong lộ trình. 401.

## Bỏ

`GET /api/admin/roadmap`, `PUT /api/admin/roadmap` → 404 (route không còn).

## Thay đổi Admin Lessons API (F2)

- `LessonSummary`/`Lesson`: bỏ `topic`; thêm `topicId`, `topicName`; `level` lấy từ chủ đề; `inRoadmap` = có trong lộ trình của
  chủ đề.
- `POST /api/admin/lessons`, `PUT /api/admin/lessons/{id}`: body `{title, content, topicId, source, license}` (bỏ `level`,
  `topic` → 400 `invalid_body`). `topicId` trống → "Vui lòng chọn chủ đề"; không tồn tại → "Chủ đề không tồn tại".
  Đổi `topicId` khi bài đang ở lộ trình cũ → bài ở cuối lộ trình chủ đề mới.
- `GET /api/admin/lessons?level=&topicId=`: lọc theo trình độ và/hoặc chủ đề (bỏ `topic`).
- `DELETE /api/admin/lessons/{id}`: 409 `lesson_in_roadmap` khi bài ở lộ trình của chủ đề.
- `GET /api/lessons/{id}` (trang Đọc): `topic` = tên chủ đề (không đổi kiểu).

## Test hợp đồng

- Tạo chủ đề; trùng tên khác hoa thường cùng trình độ → 400; khác trình độ → 201.
- Sửa trình độ chủ đề → bài của chủ đề đổi `level`.
- Xoá chủ đề còn bài → 409 + `count`; chủ đề trống → 204.
- Lộ trình: bài chủ đề khác → 400; trùng → 400; thứ tự lưu đúng; cảnh báo 0/1/2/3 bài.
- Bài: thiếu `topicId` → 400; đổi chủ đề khi ở lộ trình → chuyển cuối lộ trình mới; khi không ở lộ trình → không vào.
- Lọc `level`, `topicId`.
- Learner 403 trên `/api/admin/topics`; learner 200 trên `/api/topics`; 401 không cookie.
