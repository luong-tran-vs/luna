# Contract: Admin lessons API

Tất cả `/api/admin/*` đi qua `RequireAuth` + `RequireAdmin` (401 chưa đăng nhập, 403 người học). Lỗi dùng dạng
chung của F1: `{"error": "<code>", "message": "<tiếng Việt>", "fields"?: {...}}`. POST/PUT nhận JSON (F1:
415 / 400 `invalid_body`).

## Đối tượng

`LessonSummary`:

```json
{ "id": "…", "title": "A day at the park", "level": "B1", "topic": "Daily life",
  "audioStatus": "running", "annotationStatus": "failed", "inRoadmap": true, "createdAt": "2026-09-29T08:00:00Z" }
```

`Lesson` = `LessonSummary` +

```json
{ "content": "…", "source": "VOA Learning English", "license": "Public domain", "revision": 2,
  "audioError": "", "annotationError": "AI chưa được cấu hình",
  "sentences": [ { "index": 0, "text": "Mr. Smith arrived.", "audioUrl": "/api/audio/…/2/0" } ],
  "annotations": [ { "text": "went", "lemma": "go", "meaningVi": "đã đi", "sentenceIndex": 1, "editedByAdmin": false } ] }
```

`audioUrl` là `null` khi audio câu đó chưa có.

## GET /api/admin/lessons?level=B1&topic=Daily%20life

200 `{ "lessons": [LessonSummary…] }`, mới nhất trước. `level` sai → 400 `validation_failed`.

## POST /api/admin/lessons

Body `{ title, content, level, topic?, source, license }` → 201 `{ "lesson": Lesson }` (`audioStatus` và
`annotationStatus` = `running`). 400 `validation_failed` + `fields` (`title`, `content`, `level`, `topic`,
`source`, `license`), ví dụ `"content": "Nội dung tối đa 10.000 ký tự"`.

## GET /api/admin/lessons/{id}

200 `{ "lesson": Lesson }`; 404 `not_found` ("Không tìm thấy bài học"); id sai định dạng → 404.

## PUT /api/admin/lessons/{id}

Body như POST (đủ trường). 200 `{ "lesson": Lesson }`. Nếu `content` đổi: `revision+1`, câu tách lại, chú thích
xoá, cả hai trạng thái `running`. 400 / 404 như trên.

## DELETE /api/admin/lessons/{id}

204. 409 `lesson_in_roadmap` ("Gỡ bài khỏi lộ trình trước khi xoá"). 404.

## PUT /api/admin/lessons/{id}/annotations

Body `{ "annotations": [ { "text", "lemma", "meaningVi" } … ] }` (danh sách đầy đủ sau khi sửa). Máy chủ tìm lại
`sentenceIndex`, đặt `editedByAdmin = true` cho mục mới hoặc đổi `lemma`/`meaningVi` so với bản đang lưu. 200
`{ "lesson": Lesson }`. 400 `validation_failed` với `fields["annotations.3.text"] = "Cụm từ không có trong bài"`.
409 `annotation_running` khi chú thích đang chạy. Tối đa 100 mục.

## POST /api/admin/lessons/{id}/retry?job=tts|annotate

202 `{ "lesson": Lesson }` (trạng thái việc đó → `running`). 400 `job` sai. 409 `not_failed` ("Chỉ chạy lại được
việc đang lỗi"). 404.

## GET /api/admin/roadmap

200 `{ "lessons": [LessonSummary…], "remaining": 2, "warning": true }` theo thứ tự lộ trình.

## PUT /api/admin/roadmap

Body `{ "lessonIds": ["…", "…"] }` → 200 như GET. 400 `validation_failed` khi có id trùng hoặc bài không tồn tại.

## GET /api/audio/{lessonId}/{revision}/{index}

`RequireAuth` (mọi vai trò). 200 `audio/mpeg`, `Cache-Control: private, max-age=31536000, immutable`. 404 khi
tham số sai định dạng hoặc file chưa có. 401 chưa đăng nhập.

## Test hợp đồng (httptest)

- Mỗi endpoint: thành công; lỗi đầu vào; learner 403; ẩn danh 401.
- DELETE bài trong lộ trình → 409; sau khi PUT roadmap gỡ bài → 204.
- PUT đổi content → revision tăng, trạng thái running, job mới; PUT chỉ đổi title → revision giữ.
- retry khi `done` → 409; khi `failed` → 202 và có job mới.
- audio: `../` hoặc id sai → 404; không cookie → 401.
