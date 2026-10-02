# Contract: Từ vựng theo chủ đề (F18)

Mọi route dưới đây: `requireAuth` + `RequireAdmin` (401 `unauthenticated`, 403 `forbidden`). Chủ đề không tồn tại → 404 `not_found`.

## GET /api/admin/topics (mở rộng)

Mỗi `topic` thêm:

```json
{ "wordCount": 37, "usedWordCount": 12 }
```

## GET /api/admin/topics/{id}/words (MỚI)

200:

```json
{
  "words": [
    { "text": "Family", "used": true, "lessonCount": 3 },
    { "text": "take a shower", "used": false, "lessonCount": 0 }
  ]
}
```

Theo thứ tự trong danh sách.

## PUT /api/admin/topics/{id}/words (MỚI)

Request (thay toàn bộ danh sách, đặt `wordsSeeded=true`):

```json
{ "words": ["Family", "Parents", "cousin", "nephew"] }
```

200: như GET (đã chuẩn hoá: trim, gộp khoảng trắng, bỏ dòng rỗng).

400 `validation_failed` (không lưu gì):

```json
{
  "error": "validation_failed",
  "fields": {
    "words.3": "Từ bị trùng",
    "words.5": "Chỉ dùng chữ cái tiếng Anh, khoảng trắng, - ' /",
    "words.6": "Tối đa 40 ký tự",
    "words": "Tối đa 100 từ"
  }
}
```

`400 invalid_body` khi JSON hỏng.

## GET /api/admin/topics/{id}/word-plan?count=3&perLesson=8 (MỚI)

200:

```json
{ "groups": [["Family", "Parents", "…"], ["…"], ["…"]] }
```

- `count` 1–5, `perLesson` 0–15; sai → 400 `validation_failed` với `fields.count` / `fields.perLesson`.
- `perLesson=0` hoặc chủ đề không có từ → `count` nhóm rỗng.

## POST /api/admin/topics/{id}/generate (mở rộng)

Request thêm `targetWords` (tuỳ chọn; rỗng hoặc đúng `count` nhóm):

```json
{ "count": 3, "words": 150, "kind": "reading", "idea": "", "targetWords": [["Family", "Parents"], ["Uncle"], []] }
```

400 `validation_failed` thêm:

| Field | Khi nào |
| --- | --- |
| `targetWords` | số nhóm khác `count` |
| `targetWords.{i}` | nhóm quá 15 từ |
| `targetWords.{i}.{j}` | từ không có trong danh sách của chủ đề, hoặc trùng trong nhóm |

200: mỗi `draft` thêm:

```json
{ "title": "…", "content": "…", "words": 152, "targetWords": ["Family", "Parents"], "missingWords": ["Parents"] }
```

`targetWords` rỗng khi lượt sinh không có từ mục tiêu. Các lỗi AI giữ nguyên (503 `ai_not_configured`, 429 `ai_quota`, 502
`ai_failed` / `ai_unusable`).

## Không đổi

- `GET /api/topics` (người học): không có từ vựng.
- Các API chú thích bài: kết quả chú thích có thể có thêm từ của chủ đề.
