# API Contract: Trò chơi (F26)

REST JSON, cùng kiểu lỗi với các API khác (`httpx.ErrorBody`: `{"error": "<code>", "message": "…", "fields": {...}}`).
Mọi route cần đăng nhập (`requireAuth`). Mã lỗi chung: `401` chưa đăng nhập, `400 validation_failed` (kèm `fields`,
thông báo tiếng Việt, `httpx.WriteFieldErrors`), `403 members_only` (`httpx.RequireMember`) / `403` admin
(`httpx.RequireAdmin`), lỗi AI theo cách các route admin AI hiện có (chỉ route AI của admin).

## Người học

### `GET /api/games`

Danh sách trò cho trang `/games`.

```json
{
  "games": [
    { "id": "picture", "best": 2100, "bestAt": "2026-11-13T07:12:00Z", "locked": false },
    { "id": "verbs", "best": 27, "bestAt": "…", "locked": false },
    { "id": "grammar", "best": 0, "bestAt": null, "locked": true }
  ],
  "daily": { "day": "2026-11-13", "done": false, "streak": 12 }
}
```

`locked` = true với `grammar` khi tài khoản là khách. Chỉ liệt kê trò đã phát hành (danh sách cố định trong code, thêm dần
P1 → P6).

### `GET /api/games/picture/round`

```json
{ "questions": [ { "lemma": "apple", "imageUrl": "/api/words/apple/image", "meaningVi": "quả táo",
                   "options": ["apple", "orange", "pear", "peach"], "answer": 0 } ] }
```

40 câu (đủ cho 60 giây), không lặp lemma khi bể đủ lớn. `409 not_enough_words` khi bể có dưới 4 từ có ảnh.

### `GET /api/games/letters/round`

`{ "words": [ { "lemma": "house", "meaningVi": "ngôi nhà", "imageUrl": "/api/words/house/image" } ] }` — 25 từ một chữ,
3–8 ký tự. `409 not_enough_words` khi có dưới 5 từ.

### `GET /api/games/memory/round?lessonId=<id>`

`{ "pairs": [ { "lemma": "apple", "imageUrl": "…", "meaningVi": "quả táo" } ] }` — đúng 6 cặp. Có `lessonId`: từ của
bài đó (người học phải mở được bài, nếu không thì `403`); không có: từ đã học, đến hạn trước. `409 not_enough_words`.

### `GET /api/games/grammar/round` *(thành viên)*

```json
{ "questions": [ {
    "tokens": ["She", "go", "to", "school", "every", "day", "."],
    "wrong": { "from": 1, "to": 2 },
    "options": ["goes", "going", "went"], "answer": 0,
    "ruleVi": "Chủ ngữ số ít ở hiện tại đơn → động từ thêm -s/-es.",
    "topicVi": "Hiện tại đơn", "pointId": "a1-present-simple", "source": "choice" } ] }
```

10 câu, khoảng 3 câu `wrong: null` (câu đúng, `options` rỗng). `403 members_only` với khách. `409 not_enough_sentences`
khi cả hệ thống có dưới 10 câu.

### `POST /api/games/{game}/results`

`game` ∈ `picture | verbs | letters | memory | grammar`.

```json
{ "score": 2100, "completed": true,
  "wrongWords": ["bicycle"],
  "wrongVerbs": [ { "base": "run", "form": "past", "typed": "runned" } ] }
```

→ `200 { "best": 2100, "newBest": true, "rescheduled": 1 }`.

- `400` khi `score` < 0 hoặc > `MaxScore` của trò, `wrongWords` > 60, `wrongVerbs` > 50, `form` không phải
  `past|participle`.
- `completed: false` (bỏ giữa chừng): không đổi kỷ lục, vẫn đưa mục sai về hàng ôn (US1 #6).
- `403 members_only` với `grammar` khi là khách.

### `GET /api/games/daily`

```json
{
  "day": "2026-11-13", "seed": 1873301, "done": false,
  "streak": { "current": 12, "longest": 20 },
  "week": [true, true, true, true, false, false, false],
  "result": null,
  "compare": { "yesterday": 880, "avg7": 845.5, "best": 980 },
  "questions": [
    { "kind": "picture", "picture": { "lemma": "banana", "imageUrl": "…", "meaningVi": "quả chuối",
                                      "options": ["banana","orange","apple","pear"], "answer": 0 } },
    { "kind": "verb", "verb": { "base": "eat" } },
    { "kind": "letters", "letters": { "lemma": "bicycle", "meaningVi": "xe đạp", "imageUrl": "…" } },
    { "kind": "grammar", "grammar": { "tokens": ["…"], "wrong": { "from": 1, "to": 2 }, "options": ["…"], "answer": 1,
                                      "ruleVi": "…", "topicVi": "…", "pointId": "…", "source": "choice" } }
  ]
}
```

- Đã xong hôm nay: `done: true`, `result: { "score": 920, "answers": [{ "kind": "picture", "right": true }, …],
  "finishedAt": "…" }`, và `questions` vẫn trả (để chơi lại luyện tập).
- Khách không có câu `grammar`.

### `POST /api/games/daily/result`

```json
{ "day": "2026-11-13", "score": 920,
  "answers": [ { "kind": "picture", "right": true } ],
  "wrongWords": ["have"], "wrongVerbs": [] }
```

→ `200 { "streak": { "current": 13, "longest": 20 }, "compare": { … }, "rescheduled": 1 }`.

- `answers` đúng 10 phần tử, `kind` hợp lệ; `day` là hôm nay hoặc hôm qua theo múi giờ người học; `score` 0…3 000.
- `409 already_done` (kèm `result` đã lưu) khi ngày đó đã có kết quả — khi chơi lại để luyện, frontend không gọi route
  này mà gọi `POST /api/games/daily/practice` (chỉ xử lý `wrongWords`, `wrongVerbs`).
- Lưu thành công thì tính là một ngày học (`MarkStudied`).

### `POST /api/games/daily/practice`

`{ "wrongWords": [...], "wrongVerbs": [...] }` → `200 { "rescheduled": n }`. Không lưu gì khác.

## Đấu động từ

Lượt dựng ở frontend (danh sách động từ chỉ ở frontend). Kết quả gửi `POST /api/games/verbs/results` như trên; server
chuyển `wrongVerbs` sang `verbs.Service.Record` với `Game: true`.

## Admin — câu Bắn lỗi soạn thêm

| Route | Mô tả |
|---|---|
| `GET /api/admin/games/spot-counts` | `{ "points": [ { "pointId", "level", "choice", "fix", "spot", "draft", "total" } ] }` — mỗi điểm đã đăng, đếm câu dùng được theo nguồn (FR-026a). `total < 10` thì giao diện đánh dấu |
| `GET /api/admin/games/spot-items?pointId=` | danh sách câu (nháp + đã duyệt) của một điểm |
| `POST /api/admin/games/spot-items/generate` | `{ "pointId" }` → 1 request AI, `{ "created": [...], "dropped": 2 }`. `404` điểm không có bài đã đăng; lỗi AI (chưa cấu hình, sai khoá, hết hạn mức) trả cùng mã như `POST /api/admin/grammar-lessons/{pointId}/generate` |
| `PUT /api/admin/games/spot-items/{id}` | sửa `{ sentence, wrong, right, distractors, ruleVi }`; `400` khi không qua `ValidateSpot` |
| `POST /api/admin/games/spot-items/{id}/approve` | nháp → đã duyệt |
| `DELETE /api/admin/games/spot-items/{id}` | xoá |

Mọi route admin: `403` với member và khách.

## Frontend (Angular)

| Đường dẫn | Màn |
|---|---|
| `/games` | danh sách trò (thẻ Thử thách đứng đầu) |
| `/games/picture` · `/games/verbs` · `/games/letters` · `/games/memory` · `/games/grammar` | màn chơi, toàn màn hình |
| `/games/memory?lessonId=` | Lật thẻ với từ của một bài (từ màn xong bài) |
| `/games/daily` | Thử thách hằng ngày, toàn màn hình |

Trang chủ có thẻ Thử thách hằng ngày (đọc `GET /api/games/daily`, chỉ dùng `day`, `done`, `streak`).
