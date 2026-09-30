# Contract: Vocabulary & Review API

Mọi endpoint cần đăng nhập (`RequireAuth`, mọi vai trò); dữ liệu luôn của `userId` trong phiên, client không gửi `userId`
(trường lạ → 400 `invalid_body`). Lỗi dạng chung `{error, message, fields?}`. `401 unauthenticated` khi thiếu phiên.

## Card JSON

```json
{ "id": "…", "text": "went", "lemma": "go", "ipa": "/ɡəʊ/", "meaningVi": "đi", "contextSentence": "We went to the park.",
  "lessonId": "…" , "source": "ai", "createdAt": "2026-09-30T02:00:00Z",
  "due": "2026-09-30T17:00:00Z", "reps": 0, "state": "new" }
```

`lessonId` là `null` với thẻ tự thêm. `state`: `new | learning | review | relearning`.

## GET /api/vocab/cards?q=&lessonId=&page=

`page` từ 1 (mặc định 1), 30 thẻ/trang, `createdAt` giảm dần. `lessonId` = id bài hoặc `manual`. `q` ≤ 100 ký tự.

200: `{ "cards": [Card & {"day": "2026-09-30"}], "hasMore": true, "today": "2026-09-30", "yesterday": "2026-09-29" }`
(ngày theo múi giờ của người học). 400 `validation_failed` khi `page` < 1 hoặc `q` quá dài.

## GET /api/vocab/lessons

200: `{ "lessons": [{"id": "…", "title": "A day at the park", "count": 12}], "manualCount": 3 }` — chỉ bài còn tồn tại.

## POST /api/vocab/cards (F3, mở rộng)

Body như F3; `lessonId` **tuỳ chọn** (vắng/rỗng = tự thêm, `source` phải là `manual`). 201 `{card}` với `due` = 0 giờ hôm
sau theo múi giờ người học; 409 `card_exists` + `card`; 400 `validation_failed` (`text`, `meaningVi`, `lessonId` bài không
tồn tại, …).

## POST /api/vocab/cards/bulk

```json
{ "lessonId": "…", "lemmas": ["go", "give up"] }
```

1–200 lemma. Backend lấy dữ liệu từ mục Từ vựng của bài; lemma không có trong bài bị bỏ qua; lemma đã có trong sổ bị bỏ qua.
200 `{ "added": 1, "cards": [Card] }` (chỉ thẻ mới). 404 `not_found` bài không tồn tại; 400 `validation_failed`.

## PATCH /api/vocab/cards/{id}

Body chỉ gồm các trường cần sửa trong `meaningVi`, `ipa`, `contextSentence` (`text`, `lemma`, trường lịch → 400
`invalid_body`). 200 `{card}` (trường lịch không đổi); 400 `validation_failed`; 404 `not_found` (kể cả thẻ của người khác).

## DELETE /api/vocab/cards/{id}

204: đã xoá thẻ và mọi `review_logs` của thẻ. Idempotent với log còn sót. 404 `not_found` khi không có thẻ (của người này)
và không có log.

## GET /api/vocab/review/due?limit=

`limit` 1–200 (mặc định 50). 200:

```json
{ "cards": [Card & {"intervals": {"again": 60, "hard": 300, "good": 600, "easy": 172800}}],
  "total": 12, "nextDue": null }
```

`cards` sắp `due` tăng dần; `total` = tổng số thẻ đến hạn; `nextDue` = thời điểm sớm nhất của thẻ chưa đến hạn (`null` khi
không có).

## POST /api/vocab/cards/{id}/review

```json
{ "rating": 3, "mode": "flip", "reps": 0 }
```

200 `{ "card": Card & {"intervals": …} }` với lịch mới theo FSRS; ghi 1 `review_logs`. 409 `review_conflict` + `card` khi
`reps` không còn khớp (đã được đánh giá); 400 `validation_failed` (`rating` ngoài 1–4, `mode` lạ); 404 `not_found`.

## GET /api/lessons/{id}/vocabulary

```json
{ "available": true,
  "items": [{ "lemma": "go", "text": "went", "meaningVi": "đi", "ipa": "/ɡəʊ/", "sentenceIndex": 0,
              "sentence": "We went to the park." }] }
```

Không gọi AI. `available: false`, `items: []` khi bài chưa có chú thích xong. 404 bài không tồn tại.

## Test hợp đồng

- Danh sách: nhóm ngày sát nửa đêm theo múi giờ; `q` không phân biệt hoa thường, ký tự regex được thoát; lọc bài và `manual`;
  phân trang `hasMore`; người khác không thấy.
- Thêm tự do không `lessonId` → `due` = 0 giờ hôm sau (23:59 và 00:01); trùng → 409.
- Bulk hai lần → lần 2 `added: 0`; lemma lạ bị bỏ qua; bài chưa có chú thích → `added: 0`.
- PATCH nghĩa giữ `due`, `reps`, `state`; PATCH `text` → 400.
- DELETE xoá log; thẻ người khác → 404.
- Review từng mức → lịch khớp FSRS tham chiếu; gửi lại cùng `reps` → 409; thẻ F3 cũ (thiếu trường lịch) ôn được.
- Due: thẻ F3 cũ lưu hôm qua là đến hạn, lưu hôm nay thì chưa; `nextDue` đúng.
- Vocabulary: gộp theo lemma, câu đầu tiên, `available` theo trạng thái chú thích.
- 401 không cookie ở mọi endpoint.
