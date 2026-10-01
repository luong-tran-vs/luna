# Contract: Câu hỏi hiểu bài, ngữ pháp, đề viết (F15)

Body lỗi dùng dạng chung `{"error", "message"[, "fields"]}`, `message` bằng tiếng Việt.

## Người học

Các route dưới đây chạy sau `requireAuth` rồi guard L của bài (`lesson_locked` 403 khi chưa mở được bài).

### GET /api/lessons/{id} (mở rộng)

Thêm hai trường vào `lesson`:

```json
{
  "lesson": {
    "id": "…", "title": "…", "level": "A1", "topic": "…", "sentences": [], "paragraphs": [], "lemmas": {}, "phrases": [],
    "quiz": {
      "version": 3,
      "questions": [
        { "prompt": "Where does Tom live?", "options": ["In Hanoi", "In Hue", "In Da Nang", "In Can Tho"] }
      ],
      "answers": [
        { "questionIndex": 0, "choice": 1, "correct": false, "answerIndex": 0, "explanationVi": "Bài viết: Tom sống ở Hà Nội." }
      ]
    },
    "grammarNote": { "title": "Thì hiện tại đơn", "bodyVi": "…", "examples": ["Tom lives in Hanoi."] }
  }
}
```

- `quiz` là `null` khi bài không có câu hỏi.
- `questions` không bao giờ có `answerIndex` hay `explanationVi`.
- `answers` chỉ gồm câu trả lời của người dùng hiện tại, ở đúng `version` hiện tại, sắp theo `questionIndex`.
- `grammarNote` là `null` khi bài không có ghi chú.
- Không có `writingPrompt`.

### POST /api/lessons/{id}/answers (MỚI)

Request:

```json
{ "version": 3, "questionIndex": 1, "choice": 2 }
```

200:

```json
{
  "answer": { "questionIndex": 1, "choice": 2, "correct": true, "answerIndex": 2, "explanationVi": "…" },
  "answered": 2,
  "total": 4,
  "correct": 1
}
```

Lỗi:

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 400 | `validation_failed` | `questionIndex` ngoài 0..total-1 hoặc `choice` ngoài 0–3 (`fields.questionIndex` / `fields.choice`) |
| 400 | `invalid_body` | JSON hỏng |
| 404 | `not_found` | bài không tồn tại |
| 409 | `quiz_changed` | `version` khác phiên bản hiện tại ("Câu hỏi vừa được cập nhật, vui lòng tải lại") |
| 409 | `already_answered` | câu đã trả lời; body thêm `"answer": {…}` là câu trả lời đã ghi |
| 409 | `no_quiz` | bài không có câu hỏi |

### POST /api/today/steps/read/complete (mở rộng)

Bài hôm nay có câu hỏi mà chưa trả lời hết ở phiên bản hiện tại thì trả 409 `read_incomplete` ("Hãy trả lời hết câu hỏi hiểu
bài"). Mọi trường hợp khác giữ như L.

### GET /api/stats (mở rộng)

Thêm:

```json
"reading": { "answered": 12, "correct": 9, "rate": 0.75 }
```

`rate` là `null` khi `answered` = 0.

## Quản trị

Các route dưới đây chạy sau `requireAuth` và yêu cầu vai trò admin.

### GET /api/admin/lessons/{id} (mở rộng)

`lesson` có thêm `questions` (đầy đủ, gồm cả `answerIndex` và `explanationVi`), `grammarNote`, `writingPrompt`,
`extrasEditedByAdmin`, `quizVersion`.

### PUT /api/admin/lessons/{id}/extras (MỚI)

```json
{
  "questions": [
    { "prompt": "Where does Tom live?", "options": ["In Hanoi", "In Hue", "In Da Nang", "In Can Tho"], "answerIndex": 0,
      "explanationVi": "…" }
  ],
  "grammarNote": { "title": "Thì hiện tại đơn", "bodyVi": "…", "examples": ["Tom lives in Hanoi."] },
  "writingPrompt": "Write 5 sentences about your family."
}
```

- `grammarNote: null` xoá ghi chú.
- `questions: []` xoá mọi câu hỏi; bước Đọc khi đó quay về nút "Đã đọc xong".
- 200 trả `{lesson}` như GET.

Lỗi `fields` theo đường dẫn:

| Trường | Thông báo |
| --- | --- |
| `questions` | Tối đa 5 câu hỏi |
| `questions.{i}.prompt` | Vui lòng nhập câu hỏi / Tối đa 300 ký tự / Câu hỏi bị trùng |
| `questions.{i}.options.{j}` | Vui lòng nhập lựa chọn / Tối đa 150 ký tự / Lựa chọn bị trùng |
| `questions.{i}.answerIndex` | Chọn đáp án đúng |
| `questions.{i}.explanationVi` | Vui lòng nhập giải thích / Tối đa 500 ký tự |
| `grammarNote.title` | Vui lòng nhập tiêu đề / Tối đa 100 ký tự |
| `grammarNote.bodyVi` | Vui lòng nhập giải thích / Tối đa 1500 ký tự |
| `grammarNote.examples` | Cần 1–3 ví dụ |
| `grammarNote.examples.{k}` | Ví dụ phải có trong bài |
| `writingPrompt` | Tối đa 500 ký tự |

Các lỗi khác:

- 404 `not_found`: bài không tồn tại.
- 409 `annotation_running`: chú thích đang chạy.

### POST /api/admin/lessons/{id}/retry?job=annotate (đổi quy tắc)

- Chạy được khi chú thích đang ở trạng thái `failed` **hoặc `done`**.
- Đang ở trạng thái `running` thì trả 409 `annotation_running`.
- `job=tts` giữ nguyên: chỉ chạy khi `failed`, nếu không thì trả 409 `not_failed`.

## Frontend

- `ReadingApiService.answer(lessonId, {version, questionIndex, choice}): Observable<AnswerResult>`.
- `AdminApiService.updateExtras(id, input: ExtrasInput): Observable<Lesson>`.
