# Contract: Vocab API (tối thiểu, F5 mở rộng)

Tất cả cần đăng nhập; mọi dữ liệu theo `userId` của phiên.

## POST /api/vocab/cards

```json
{ "text": "went", "lemma": "go", "ipa": "/ˈɡəʊ/", "meaningVi": "đã đi",
  "contextSentence": "We went to the park.", "lessonId": "…", "source": "ai" }
```

| Status | Body | Khi nào |
|---|---|---|
| 201 | `{ "card": Card }` | tạo mới |
| 409 | `{ "error": "card_exists", "message": "Từ này đã có trong sổ", "card": Card }` | đã có cùng `lemma` (không phân biệt hoa thường, khoảng trắng) |
| 400 | `validation_failed` + `fields` | thiếu/dài quá, `source` sai, `lessonId` sai hoặc bài không tồn tại |
| 401 | `unauthenticated` | |

`Card` = `{ id, text, lemma, ipa, meaningVi, contextSentence, lessonId, source, createdAt }` (không có `userId`).

## GET /api/vocab/words

200 `{ "words": [ { "lemma": "go", "text": "went" } ] }` — chỉ của người gọi.

## Test hợp đồng

- 201 rồi 409 cho cùng lemma khác hoa thường (`Go`, ` go `); 400 từng trường.
- Người dùng B không thấy từ của A trong `/words`; B lưu `go` vẫn được 201 (sổ riêng).
- 401 khi không có cookie.
