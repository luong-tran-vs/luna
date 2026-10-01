# Contract: AI sinh bài học (F7)

Mọi route dưới `/api/admin` yêu cầu phiên đăng nhập (401 `unauthenticated`) và vai trò admin (403 `forbidden`).

## POST /api/admin/topics/{id}/generate

Sinh bản nháp cho chủ đề `{id}`. Đồng bộ, tối đa 60 giây. Không ghi gì vào DB.

Request:

```json
{ "count": 3, "words": 120, "kind": "reading", "idea": "a family picnic" }
```

| Trường | Bắt buộc | Quy tắc |
| --- | --- | --- |
| count | có | số nguyên 1–5 |
| words | có | số nguyên 50–800 |
| kind | có | `reading` \| `dialogue` |
| idea | không | chuỗi ≤ 500 ký tự (sau trim); rỗng = không có |

200:

```json
{
  "drafts": [
    { "title": "Sunday at Grandma's House", "content": "Every Sunday, my family ...", "words": 118 }
  ],
  "requested": 3,
  "dropped": 1
}
```

- `drafts`: 1..count bản, theo thứ tự AI trả về, đã lọc (không rỗng, không trùng tiêu đề trong lượt hoặc với bài của chủ đề, số
  từ trong ±20% `words`, trong giới hạn F2).
- `dropped`: số bản AI trả về bị loại; `requested - len(drafts)` có thể lớn hơn `dropped` khi AI trả thiếu bản.
- Hội thoại: `content` gồm các dòng `Tên: câu nói`, phân cách bằng `\n`.

Lỗi:

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 400 | `validation_failed` | đầu vào sai; `fields` có `count`, `words`, `kind`, `idea` |
| 400 | `invalid_body` | body không phải JSON hợp lệ (như các route khác) |
| 404 | `not_found` | chủ đề không tồn tại ("Không tìm thấy chủ đề") |
| 503 | `ai_not_configured` | chưa cấu hình AI hoặc khoá sai |
| 429 | `ai_quota` | hết lượt AI |
| 502 | `ai_unusable` | lọc xong không còn bản nào |
| 502 | `ai_failed` | AI lỗi khác hoặc quá 60 giây |
| 500 | `internal_error` | lỗi DB khi đọc chủ đề/bài |

Body lỗi theo dạng chung `{"error": "...", "message": "..."}` (tiếng Việt, hiển thị trực tiếp được).

## POST /api/admin/lessons (mở rộng của F2)

Thêm trường không bắt buộc `appendToRoadmap` (bool, mặc định `false`):

```json
{
  "title": "Sunday at Grandma's House",
  "content": "Every Sunday, my family ...",
  "topicId": "66f0...",
  "source": "AI sinh",
  "license": "Nội dung do AI tạo",
  "appendToRoadmap": true
}
```

- `true`: sau khi tạo, bài được nối vào cuối lộ trình của `topicId` trong cùng request. Nối lỗi → bài bị xoá, trả 500
  `internal_error`, không có job nào được tạo.
- 201 trả `{"lesson": {...}}` như F2; `inRoadmap: true` khi đã nối, `audioStatus` và `annotationStatus` = `running`.
- Lỗi kiểm tra như F2 (400 với `fields.title`, `fields.content`, `fields.topicId`...).
- `PUT /api/admin/lessons/{id}` bỏ qua trường này.

## Frontend

- `AdminApiService.generateLessons(topicId, input): Observable<GenerateResult>`.
- `AdminApiService.create(input: LessonInput)` với `LessonInput.appendToRoadmap?: boolean`.
- Route `admin/roadmap` có `canDeactivate: [unsavedChangesGuard]`.

## nginx

`location ~ ^/api/admin/topics/[^/]+/generate$` → cùng `proxy_pass` như `/api/`, `proxy_read_timeout 75s`.
