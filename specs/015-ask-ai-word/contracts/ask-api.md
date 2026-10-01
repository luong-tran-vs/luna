# Contract: Hỏi AI về từ (F9)

Các route dưới đây đều phải qua `requireAuth` và guard L của bài: chưa đăng nhập thì 401, bài chưa mở được thì 403 `lesson_locked`.

## POST /api/lessons/{id}/ask (MỚI)

Request:

```json
{ "text": "make up for", "sentenceIndex": 3 }
```

200:

```json
{
  "result": {
    "source": "ai",
    "text": "make up for",
    "lemma": "make up for",
    "ipa": "",
    "meanings": [{ "pos": "", "text": "bù lại (cho việc đã lỡ)" }],
    "note": "Trong câu này, 'make up for' nghĩa là làm điều gì đó để bù cho việc đến muộn."
  },
  "cached": false
}
```

- `cached` là `true` khi kết quả lấy từ `ai_lookups`. Khi đó server không gọi AI.

Lỗi:

| HTTP | `error` | Khi nào |
| --- | --- | --- |
| 400 | `validation_failed` | `fields.text`: rỗng, quá 100 ký tự / 6 từ, hoặc không có trong câu; `fields.sentenceIndex`: câu không tồn tại |
| 400 | `invalid_body` | JSON hỏng |
| 404 | `not_found` | bài không tồn tại |
| 503 | `ai_not_configured` | chưa cấu hình AI, hoặc khoá sai |
| 429 | `ai_quota` | hết lượt AI |
| 502 | `ai_failed` | AI lỗi, quá thời gian, hoặc trả kết quả thiếu nghĩa |

## GET /api/lessons/{id}/lookup (mở rộng F3)

- Thứ tự tra:
  1. Chú thích của bài.
  2. Kết quả trong `ai_lookups` của câu (`source: "ai"`, có `note`).
  3. Từ điển.
- Kết quả từ chú thích và từ điển có `note: ""`.
