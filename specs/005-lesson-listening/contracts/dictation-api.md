# Contract: Dictation API

Cần đăng nhập (`RequireAuth`, mọi vai trò). Dữ liệu luôn của `userId` trong phiên. Lỗi dạng chung `{error, message, fields?}`.

## POST /api/lessons/{id}/dictation

```json
{ "sentenceIndex": 2, "typed": "i dont like green apples", "correctWords": 4, "totalWords": 5 }
```

| Status | Body | Khi nào |
|---|---|---|
| 200 | `{ "summary": Summary }` | lưu (thay kết quả cũ của câu) |
| 400 | `validation_failed` + `fields` | `sentenceIndex` ngoài `[0, số câu)`, `typed` rỗng hoặc > 1000 ký tự, `totalWords` < 1 hoặc > 1000, `correctWords` < 0 hoặc > `totalWords`, trường lạ (vd. `userId`) |
| 404 | `not_found` | bài không tồn tại |
| 401 | `unauthenticated` | |

## GET /api/lessons/{id}/dictation/summary

200:

```json
{ "summary": {
  "sentenceCount": 4, "checkedCount": 2, "correctWords": 9, "totalWords": 11, "rate": 0.818, "completed": false,
  "results": [ { "sentenceIndex": 0, "typed": "we went to the park", "correctWords": 5, "totalWords": 5, "checkedAt": "…" } ]
} }
```

Chỉ gồm kết quả của revision hiện tại của bài. 404 khi bài không tồn tại; 401.

## Test hợp đồng

- POST 200 rồi POST lại cùng câu → `checkedCount` không tăng, số liệu là lần mới.
- Kiểm tra hết các câu (có câu sai) → `completed: true`, `rate` đúng.
- 400 từng lỗi đầu vào; `userId` trong body → 400.
- Người dùng B gọi summary cùng bài → không có kết quả của A.
- Sửa nội dung bài (revision tăng) → summary trả `checkedCount: 0`.
- 401 không cookie; 404 bài lạ.
