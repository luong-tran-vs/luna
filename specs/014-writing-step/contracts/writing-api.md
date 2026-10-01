# Contract: Bước Viết (F8)

Mọi route yêu cầu phiên đăng nhập (401 `unauthenticated`). Body lỗi chung có dạng `{"error", "message"[, "fields"]}`, `message` bằng tiếng Việt.

## Writing JSON

```json
{
  "id": "…", "lessonId": "…", "lessonTitle": "My family", "prompt": "Write 5 sentences about your family.",
  "text": "My family has four people…", "status": "submitted", "submittedAt": "2026-10-01T03:00:00Z",
  "grade": {
    "status": "done", "error": "",
    "criteria": [
      { "name": "task", "score": 4, "commentVi": "…" },
      { "name": "grammar", "score": 3, "commentVi": "…" },
      { "name": "vocabulary", "score": 4, "commentVi": "…" },
      { "name": "coherence", "score": 4, "commentVi": "…" }
    ],
    "average": 3.8, "overallVi": "…", "correctedText": "My family has four members…", "gradedAt": "…"
  }
}
```

- `grade` là `null` khi còn nháp.
- Khi đang chấm hoặc chấm lỗi: `criteria: []`, `average: null`.
- `submittedAt` là `null` khi còn nháp.

## Bài học (sau guard L: 403 `lesson_locked`)

### GET /api/lessons/{id}/writing

200:

```json
{ "prompt": "…", "level": "A1", "canWrite": true, "writing": { …Writing… } }
```

- `writing` là `null` khi chưa có nháp.
- `prompt` là đề của bài, hoặc `Tóm tắt bài bằng 3–5 câu.` khi bài chưa có đề. Khi đã nộp, đây là đề đã lưu lúc nộp.
- `canWrite` là `true` khi đây là bài hôm nay và đang ở bước Viết.

Lỗi: 404 `not_found` khi bài không tồn tại.

### PUT /api/lessons/{id}/writing

Body `{ "text": "…" }`, tối đa 4000 ký tự. Lưu nháp. 200 trả `{writing}`.

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 400 | `validation_failed` | `fields.text`: quá 4000 ký tự |
| 409 | `already_submitted` | đã nộp |
| 409 | `write_locked` | không phải bài hôm nay ở bước Viết |
| 404 | `not_found` | bài không tồn tại |

### POST /api/lessons/{id}/writing/submit

Body `{ "text": "…" }`. Lưu bài viết, đặt `status: submitted`, `grade.status: pending`, xếp job chấm. 200 trả `{writing}`.

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 400 | `validation_failed` | `fields.text`: "Bài viết cần ít nhất 5 từ" / "Bài viết tối đa 400 từ" |
| 409 | `already_submitted` | đã nộp |
| 409 | `write_locked` | không phải bài hôm nay ở bước Viết |

### POST /api/today/steps/write/complete (mở rộng của L)

409 `write_incomplete` ("Hãy nộp bài viết") khi chưa nộp. Các trường hợp khác như L.

## Bài viết của tôi

### GET /api/writings

200:

```json
{
  "writings": [
    { "id": "…", "lessonId": "…", "lessonTitle": "…", "submittedAt": "…",
      "gradeStatus": "done", "average": 3.8, "seen": true }
  ]
}
```

- Chỉ gồm bài đã nộp của người dùng hiện tại, mới nhất trước.

### GET /api/writings/{id}

- 200 trả `{writing}`.
- 404 khi không có, hoặc là bài của người khác.

### POST /api/writings/{id}/seen

- 204.
- 404 khi không có hoặc của người khác.

### POST /api/writings/{id}/regrade

- 202 trả `{writing}`, với `grade.status = pending`.
- 409 `not_failed` khi bài không ở trạng thái chấm lỗi.
- 404 khi không có hoặc của người khác.

### GET /api/writings/unseen-count

200:

```json
{ "unseen": 1, "pending": 0, "latest": { "id": "…", "status": "done" } }
```

- `latest`: bài có kết quả chưa xem gần nhất; `null` khi không có.

## Màn hình chính và thống kê (mở rộng)

- `GET /api/dashboard`: `skills.write`.
- `GET /api/stats`: `lessons.write` và `writing: { "submitted": 2, "averageScore": 3.8 }`. `averageScore` là `null` khi chưa có bài chấm xong.

## Xuất dữ liệu (mở rộng F13)

`writings`: mọi bài viết của người học (cả nháp), không có `userId`.
