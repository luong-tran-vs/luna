# Contract: Phần luyện tập (F17)

## GET /api/lessons/{id}/practice (MỚI, người học)

`requireAuth` + guard L (như `GET /api/lessons/{id}`).

200 — bài có phần luyện tập:

```json
{
  "status": "done",
  "lessonNumber": 3,
  "objectiveVi": "Bạn có thể chào hỏi và giới thiệu bản thân.",
  "examples": [
    { "lemma": "introduce", "sentence": "Let me introduce my friend.", "audioUrl": "/api/audio/66f…/2/practice/1/example/0" }
  ],
  "dialogue": {
    "speakers": ["Minh", "Anna"],
    "turns": [
      { "speaker": 0, "text": "Hi, I'm Minh. Nice to meet you.", "meaningVi": "Chào, mình là Minh. Rất vui được gặp bạn.",
        "audioUrl": "/api/audio/66f…/2/practice/1/turn/0" }
    ]
  },
  "fill": {
    "turns": [
      { "speaker": 0, "turnIndex": 0, "meaningVi": "Chào, mình là Minh. Rất vui được gặp bạn.",
        "parts": [{ "text": "Hi, I'm Minh. Nice to " }, { "blank": 0 }, { "text": " you." }] }
    ],
    "blanks": [{ "answer": "meet" }],
    "wordBank": ["meet", "introduce", "friend"]
  },
  "grammarTipVi": "Dùng \"Nice to meet you\" khi gặp ai lần đầu.",
  "translations": [
    { "vi": "Rất vui được gặp bạn.", "answer": ["Nice", "to", "meet", "you."], "tiles": ["Nice", "to", "meet", "you.", "see", "glad", "us"],
      "audioUrl": null }
  ]
}
```

200 — chưa có phần luyện tập (chưa sinh, đang sinh, lỗi):

```json
{ "status": "running", "lessonNumber": 3, "objectiveVi": "", "examples": [], "dialogue": null, "fill": null, "grammarTipVi": "",
  "translations": [] }
```

- `status`: `none | running | done | failed`. Nội dung có thể có cả khi `status` là `running`/`failed` (đang Tạo lại hoặc Tạo lại lỗi:
  vẫn trả bản cũ). Frontend coi là "có phần luyện tập" khi mảng/đối tượng tương ứng khác rỗng.
- `lessonNumber`: vị trí trong lộ trình chủ đề (từ 1); `0` khi bài không thuộc lộ trình.
- `audioUrl`: `null` khi file chưa có.
- `wordBank`, `tiles` theo thứ tự xác định; frontend xáo.
- Bài không có ví dụ cho một từ: từ đó không có trong `examples`.

Lỗi:

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 401 | `unauthenticated` | chưa đăng nhập |
| 403 | `lesson_locked` | bài sắp tới / không được mở |
| 404 | `not_found` | bài không tồn tại |

## GET /api/audio/{lessonId}/{revision}/practice/{version}/{kind}/{index} (MỚI)

- `requireAuth`. `kind ∈ {example, turn, answer}`; `lessonId` 24 hex; `revision`, `version`, `index` là số không âm.
- 200 `audio/mpeg`, `Cache-Control: private, max-age=31536000, immutable`.
- 404 khi tham số sai hoặc file không có.

## POST /api/admin/lessons/{id}/practice/regenerate (MỚI, quản trị)

`requireAuth` + `RequireAdmin`. Không có body.

202: `{"lesson": {...}}` (lesson quản trị, `practiceStatus: "running"`).

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 403 | `forbidden` | không phải quản trị viên |
| 404 | `not_found` | bài không tồn tại |
| 409 | `practice_running` | đang sinh phần luyện tập |
| 409 | `annotation_not_done` | chú thích chưa xong |

## GET /api/admin/lessons/{id} (mở rộng)

`lesson` thêm:

```json
{
  "practiceStatus": "done",
  "practiceError": "",
  "practice": {
    "objectiveVi": "…",
    "examples": [{ "lemma": "…", "sentence": "…" }],
    "dialogue": { "speakers": ["Minh", "Anna"], "turns": [{ "speaker": 0, "text": "…", "meaningVi": "…" }] },
    "grammarTipVi": "…",
    "translations": [{ "vi": "…", "en": "…", "distractors": ["…"] }]
  }
}
```

`practiceStatus` trả `"none"` khi chưa có; `practice` là `null` khi chưa có. Danh sách bài quản trị (`summary`) không đổi.

## Không đổi

- `GET /api/lessons/{id}`, `/vocabulary`, `GET /api/dashboard`, `GET /api/lessons/mine`: frontend dùng lại như hiện có.
- Không có endpoint nộp kết quả luyện tập.
