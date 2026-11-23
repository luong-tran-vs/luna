# API Contract: Luyện thi TOEIC (F27)

REST JSON, lỗi theo `httpx.ErrorBody` (`{"error": "<code>", "message": "…", "fields": {...}}`). Mọi route cần đăng nhập
(`requireAuth`). Mã lỗi chung: `401`; `400 validation_failed` (kèm `fields`, `httpx.WriteFieldErrors`); `403 members_only`
(khách vượt quyền); `403 forbidden` (không phải admin); `404 not_found`.

## Người học

### `GET /api/toeic` — thư viện đề

```json
{
  "tests": [
    { "key": "luna-test-01", "title": "Luna Test 01", "kind": "full", "questions": 200, "minutes": 120, "parts": 7,
      "locked": false,
      "state": { "status": "submitted", "attemptId": "…", "total": 610, "correct": null, "at": "2026-11-12T…" } },
    { "key": "luna-test-02", "kind": "full", "state": { "status": "in_progress", "attemptId": "…", "position": 147 } },
    { "key": "mini-05", "kind": "mini100", "questions": 100, "minutes": 60, "state": { "status": "submitted", "correct": 74 } }
  ],
  "goal": { "target": 750, "examDate": "2027-01-15", "daysLeft": 58 },
  "latest": { "total": 610, "listening": 330, "reading": 280, "attemptId": "…", "at": "…" },
  "recent": [ { "attemptId": "…", "title": "Part 5 · luyện tập", "correct": 21, "total": 30, "at": "…" } ],
  "mistakes": 23
}
```

`latest` chỉ lấy bài `counted` (FR-004a). Với khách: `locked` theo R10, `mistakes` = null.

### `GET /api/toeic/tests/{key}` — chi tiết đề

`{ "key", "title", "kind", "minutes", "parts": [ { "part": 5, "questions": 30, "tags": ["word-form","verb-tense", …] } ],
"locked": false, "attempts": [ { "id", "mode", "parts": [5,6], "correct", "total", "scaled": null, "retake": false,
"durationMs", "submittedAt" } ], "inProgress": null }`

### `POST /api/toeic/attempts` — bắt đầu một lượt

```json
{ "source": { "kind": "test", "testKey": "luna-test-03" }, "mode": "practice", "parts": [5, 6], "timeLimitMin": 20 }
```

`source.kind`: `test` (cần `testKey`; `mode` = `practice` + `parts`, hoặc `test` = cả đề) · `random` (`size`: 50 | 100,
chỉ thành viên) · `part` (`part`, `count`) · `tag` (`tag`, chỉ thành viên) · `retry` (`fromAttempt`: làm lại câu sai của
một bài). `timeLimitMin`: 0 = không giới hạn (chỉ luyện tập). Trả `201` + `AttemptView`. Đang có bài dở của cùng đề → `409
attempt_in_progress` `{ "attemptId" }`. Khách vượt quyền → `403 members_only`. Không đủ câu đã đăng → `409 not_enough_questions`.

### `GET /api/toeic/attempts/{id}` — mở lại bài làm

`AttemptView` (không có đáp án khi `in_progress`):

```json
{
  "id": "…", "mode": "test", "status": "in_progress", "phase": "reading", "parts": [1,2,3,4,5,6,7],
  "timeLimitMs": 4500000, "remainingMs": 2892000, "listeningSegment": 0, "position": "p7-garden-01#2",
  "answers": { "p5-lo-03#1": 1 }, "flags": ["p5-lo-03#3"],
  "paper": [ {
    "key": "p7-garden-01", "part": 7, "number": 147,
    "docs": [ { "kind": "notice", "title": "Greenfield Community Garden", "meta": [], "body": [ { "kind": "p", "text": "…" } ],
                "imageUrl": null } ],
    "questions": [ { "id": "p7-garden-01#1", "number": 147, "prompt": "What is the purpose of the notice?",
                     "options": ["…","…","…","…"] } ]
  }, {
    "key": "p1-warehouse-04", "part": 1, "number": 3, "imageUrl": "/api/toeic/images/p1-warehouse-04/photo",
    "questions": [ { "id": "p1-warehouse-04#1", "number": 3, "options": ["…","…","…","…"] } ]
  }, {
    "key": "p3-printshop-02", "part": 3, "number": 41,
    "script": [ { "speaker": "m", "accent": "us", "text": "…" } ], "graphic": null,
    "questions": [ { "id": "p3-printshop-02#1", "number": 41, "prompt": "Where most likely are the speakers?", "options": [ … ] } ]
  } ]
}
```

`number` = số câu trong lượt (đề đầy đủ dùng đúng số 1–200). `script` và `options` Part 1/2 có trong dữ liệu để đọc, giao
diện không hiện (R2). Không bao giờ có `answer`, `evidence`, `explainVi`, `translateVi`, `vocab` ở đây.

### `PUT /api/toeic/attempts/{id}` — lưu tiến độ

`{ "answers": {"p5-lo-03#2": 0}, "flags": ["…"], "position": "…", "remainingMs": 2887000, "phase": "reading",
"listeningSegment": 0 }` — mọi trường tuỳ chọn; `answers` được gộp (giá trị −1 = bỏ chọn). `remainingMs` lớn hơn giá trị
đã lưu thì bị bỏ qua. Full test ở `phase = listening`: chỉ nhận đáp án của câu thuộc cụm đang phát hoặc trước đó; không cho
`phase` quay lại `listening`. Bài đã nộp → `409 attempt_submitted`. Trả `204`.

### `POST /api/toeic/attempts/{id}/submit`

Body giống `PUT` (lần lưu cuối). Chấm, cập nhật sổ câu sai, gọi `MarkStudied`. Trả `200` + `ResultView`. Gọi lại khi đã nộp
→ trả `ResultView` cũ (idempotent).

### `GET /api/toeic/attempts/{id}/result` — trang kết quả

```json
{
  "id": "…", "title": "Luna Test 02 · Full test", "retake": false, "counted": true,
  "correct": 129, "wrong": 63, "skipped": 8, "accuracy": 64.5, "durationMs": 7092000,
  "listening": { "correct": 69, "total": 100, "scaled": 340 }, "reading": { "correct": 60, "total": 100, "scaled": 305 },
  "total": 645, "goal": 750,
  "byPart": [ { "part": 5, "tags": [ { "tag": "word-form", "labelVi": "Từ loại", "right": 5, "wrong": 4, "skipped": 0,
               "questions": [ { "number": 101, "id": "…", "state": "right" } ] } ] } ],
  "answers": [ { "number": 104, "id": "…", "part": 5, "correct": "B", "chosen": "A", "state": "wrong" } ]
}
```

`total`, `scaled` = null khi lượt không đủ 7 Part. Bài chưa nộp → `409 attempt_in_progress`.

### `GET /api/toeic/attempts/{id}/questions/{qid}` — đáp án chi tiết

Chỉ khi bài đã nộp. Trả cả cụm có đáp án:
`{ "group": { …như paper, thêm "translateVi" ở docs/turns… }, "question": { "id", "number", "prompt", "options",
"answer": 0, "chosen": 1, "evidence": [ {"doc":0,"block":0,"turn":-1,"text":"…"} ], "explainVi", "translateVi",
"tags": [ {"tag","labelVi","grammarPointId": "b1-word-forms"} ], "vocab": [ {"word","ipa","meaningVi"} ] } }`

### Sổ câu sai *(thành viên)*

- `GET /api/toeic/mistakes?part=5` → `{ "items": [ { "questionId", "groupKey", "part", "tags", "streak", "prompt" } ],
  "counts": { "1": 0, "5": 8 } }`
- `GET /api/toeic/mistakes/round?part=5&count=10` → `{ "items": [ { "group": PaperGroup, "questionId" } ] }` (sai lâu nhất
  trước; cụm hiện đủ, chỉ một câu được chấm).
- `POST /api/toeic/mistakes/{qid}/check` `{ "choice": 1 }` → chấm ngay: `{ "right": false, "answer": 0, "streak": 0,
  "removed": false, "explainVi", "evidence", "translateVi" }`. Một lượt sổ câu sai xong (client gọi `POST
  /api/toeic/mistakes/finish`) → `MarkStudied`.

### Luyện theo dạng câu, thống kê, mục tiêu

- `GET /api/toeic/tags` *(thành viên)* → `{ "tags": [ { "tag", "labelVi", "parts": [5], "accuracy": 54.5, "answered": 22,
  "available": 40, "grammarPointId": "…" } ] }` (accuracy tăng dần, chưa làm cuối).
- `GET /api/toeic/stats` *(thành viên)* → `{ "scores": [ ScorePoint ], "byPart": [ {part, accuracy, answered} ], "byTag": […],
  "goal": {…} }`
- `PUT /api/toeic/goal` `{ "target": 750, "examDate": "2027-01-15" }` → `200` Goal; `DELETE /api/toeic/goal` → `204`.
  `target` ngoài 450/600/750/900 hoặc ngày sai định dạng → `400`.

### Ảnh

- `GET /api/toeic/images/{groupKey}/{slot}` → ảnh JPEG (cache dài, `ETag` theo `createdAt`). Chỉ trả ảnh của cụm đã đăng
  (admin xem được cả nháp).

## Admin (`httpx.RequireAdmin`)

- `POST /api/admin/toeic/import?dryRun=true` body `{ "groups": [Group JSON], "replace": false }` (giới hạn 2 MB,
  `DecodeJSONDetailed`) →
  `{ "dryRun": true, "created": ["p5-lo-04"], "replaced": [], "duplicates": ["p5-lo-03"], "rescored": 0,
  "listeningMinutes": null, "reports": [ { "key", "part", "questions": 10, "status": "ok|duplicate|replace|error",
  "fields": {"questions[0].answer": "…"} } ] }`. Có lỗi ở bất kỳ cụm nào → nhập thật trả `400 validation_failed` với
  `fields` dạng `groups[i]...` và không lưu gì.
- `GET /api/admin/toeic/groups?part=7&status=draft` → danh sách (key, part, level, số câu, tags, status, source, ảnh có/thiếu,
  thuộc đề nào).
- `GET /api/admin/toeic/groups/{key}` → cụm đầy đủ (xem trước).
- `PATCH /api/admin/toeic/groups/{key}` `{ "status": "published" | "hidden" | "draft" }`.
- `POST /api/admin/toeic/tests` `{ "key", "title", "kind", "order", "groups": ["…"] }` / `PUT /api/admin/toeic/tests/{key}`
  (cùng body, khi nháp) / `PATCH /api/admin/toeic/tests/{key}` `{ "status" }` → đăng khi `check.ready`, nếu không `409
  test_not_ready` kèm `check`.
- `GET /api/admin/toeic/tests` · `GET /api/admin/toeic/tests/{key}` → đề + `check { parts: [{part, have, need}],
  draftGroups, missingPhotos: [{groupKey, number}], graphics, docs, listeningMinutes, ready }`.
- `DELETE /api/admin/toeic/tests/{key}` → `204` (cụm của đề vẫn ở kho); đề đang đăng → `400 fields.status`; đề đã có bài
  làm của người học → `400 fields.attempts`; mã lạ → `404`.
- `POST /api/admin/toeic/tests/compose` `{ kind, title?, key?, partial?, dryRun? }` → `plan.parts[]` có thêm
  `levels: {"450": n, …}` (số câu từng mức) và `note` (khi kho không đủ cụm cho cấu trúc đề thật, Part ghép theo số câu).
- `POST /api/admin/toeic/images/{groupKey}/{slot}/generate` `{ "engine": "gemini" | "codex" }` (bắt buộc) → `200`
  `{ imageUrl }`; lỗi vẽ theo `drawError` như lesson/wordbank.
- `POST /api/admin/toeic/images/{groupKey}/{slot}` (multipart, JPEG/PNG/GIF ≤ 5 MB, thu nhỏ ở server) → `200`.
- `DELETE /api/admin/toeic/images/{groupKey}/{slot}` → `204`.
- `POST /api/admin/toeic/tests/{key}/images/batch` `{ "engine" }` → `202` (vẽ ngầm ảnh Part 1 còn thiếu của đề, 1 lượt mỗi
  lúc, `409 batch_running`); `GET …/images/batch` → `{ running, done, failed, total }`; `POST …/images/batch/stop`.
- `POST /api/admin/toeic/images/batch` `{ part | testKey | keys, levels?, engine }` → `202`: `levels` (450/600/750/900,
  chỉ cho `part`) giới hạn lượt vẽ theo Part ở các mức điểm đã chọn, mức lạ → `400 levels`; bỏ trống = mọi mức.
  `GET /api/admin/toeic/images/missing` → `{ missing: {"1": n, ...}, byLevel: {"1": {"450": n, ...}, ...}, codex }`.

## Frontend routes

| Đường dẫn | Màn hình |
|---|---|
| `/toeic` | Thư viện đề (tab Đề thi) |
| `/toeic/practice` | Luyện theo Part |
| `/toeic/tags` | Luyện theo dạng câu |
| `/toeic/results` | Kết quả của tôi + thống kê |
| `/toeic/mistakes` | Sổ câu sai (và lượt làm lại) |
| `/toeic/goal` | Mục tiêu |
| `/toeic/tests/:key` | Chi tiết đề (tab Luyện tập / Làm full test / Kết quả) |
| `/toeic/attempts/:id` | Phòng thi (màn tập trung, ẩn thanh tab) |
| `/toeic/attempts/:id/result` | Trang kết quả |
| `/toeic/attempts/:id/q/:qid` | Đáp án chi tiết |
| `/admin/toeic` | Danh sách cụm câu + nhập JSON |
| `/admin/toeic/tests/:key` | Trang đề (ghép đề, kiểm tra, ảnh) |
