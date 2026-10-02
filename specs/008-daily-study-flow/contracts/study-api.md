# Contract: Study API (L)

Mọi endpoint cần đăng nhập (`RequireAuth`); dữ liệu luôn của người trong phiên. Ngày tính theo múi giờ tài khoản. Lỗi dạng chung
`{error, message, fields?}`; trường lạ trong body → 400 `invalid_body`; 401 khi thiếu phiên.

## GET /api/goals

```json
{ "active": { "topicId": "…", "topicName": "Gia đình", "level": "A1", "completedLessons": 2, "totalLessons": 12,
              "status": "active", "effectiveFrom": "2026-10-01" } | null,
  "others": [ { "topicId": "…", "topicName": "Mua sắm", "level": "A1", "completedLessons": 1, "totalLessons": 5,
                "status": "paused", "effectiveFrom": "…" } ] }
```

Chủ đề đã bị xoá không hiện.

## POST /api/goals

`{ "topicId": "…" }` → 200 `{ "active": Goal, "effectiveFrom": "2026-10-01", "startsTomorrow": true }`.
`startsTomorrow` = bài hôm nay đã bắt đầu (FR-004). 400 `topicId` thiếu; 404 `not_found` chủ đề không tồn tại.

## GET /api/today

```json
{ "kind": "studying", "goal": Goal | null,
  "lesson": { "id": "…", "title": "At the café" } | null,
  "steps": { "review": "done", "read": "current", "listen": "locked" },
  "currentStep": "read", "sentenceIndex": 10, "reviewCount": 0,
  "streak": 5, "goalCompleted": false }
```

`kind`: `noGoal` | `studying` | `doneToday` | `noNewLesson`. Mở khi `kind = studying` mà bước Ôn có `reviewCount = 0` → server tự
đánh dấu Ôn xong (tính là "đã bắt đầu"). `reviewCount = min(thẻ đến hạn, 30 − thẻ đã ôn ở bước Ôn hôm nay)`.

## GET /api/today/review-cards

200 như `GET /api/vocab/review/due` (F5) nhưng `limit` = quota còn lại hôm nay; 409 `not_review_step` khi bước hiện tại không
phải Ôn.

## POST /api/today/steps/{step}/complete

`step` ∈ `review`, `read`, `listen`. 200 = trạng thái như `GET /api/today` (sau khi ghi), thêm `goalCompleted` khi bài cuối của lộ
trình vừa xong. Lỗi: 400 `step` lạ; 409 `step_locked` "Hoàn thành bước trước" (bước trước chưa xong); 409 `listen_incomplete`
(chưa kiểm tra hết các câu, F4); 409 `no_lesson` (không có bài hôm nay). Gọi lại khi bước đã xong → 200, không cộng lần nữa.

## PUT /api/today/position

`{ "step": "listen", "sentenceIndex": 3 }` → 204. 400 `sentenceIndex` < 0 hoặc ≥ số câu; 409 `not_current_step`.

## GET /api/lessons/mine

```json
{ "today": { "id": "…", "title": "…" } | null,
  "completed": [ { "id": "…", "title": "…", "topicName": "Gia đình", "completedAt": "…" } ],
  "upcoming": [ { "id": "…", "title": "…" } ] }
```

`upcoming` = bài còn lại của lộ trình đang học sau bài hôm nay, chưa học, theo thứ tự; chỉ `id`, `title`.

`completed` (sửa 2026-10-02) = chỉ bài đã học thuộc lộ trình đang học, mới nhất trước. Bài đã học ở chủ đề khác hiện lại khi
người học chọn lại chủ đề đó; vẫn mở được bằng đường dẫn `/lessons/:id/read` (không bị chặn).

## Chặn bài chưa tới lượt (thay đổi F3, F4, F5)

`GET /api/lessons/{id}`, `/lookup`, `/vocabulary`, `/dictation`, `/dictation/summary`: người học chỉ mở được bài hôm nay và bài đã
có tiến độ; còn lại 403 `lesson_locked` "Bài này sẽ mở khi tới lượt". Quản trị viên không bị chặn.

## Thay đổi F5

`POST /api/vocab/cards/{id}/review` nhận thêm `context` (`daily` | `free`, mặc định `free`); giá trị khác → 400.

## Test hợp đồng

- Chưa có goal → `kind: noGoal`; chọn chủ đề → `studying` với bài đầu tiên của lộ trình.
- Không có thẻ đến hạn → bước Ôn tự xong; có 45 thẻ → `reviewCount` 30; review-cards trả 30.
- Hoàn thành `read` trước `review` → 409; `listen` khi F4 chưa xong → 409; gọi lại `complete` không cộng streak hai lần.
- Xong `listen` → `doneToday`, streak +1; cùng ngày `GET /api/today` vẫn `doneToday`; hôm sau bài tiếp theo.
- Đổi chủ đề sau khi đã xong bước Ôn → `startsTomorrow: true`; hôm nay bài không đổi.
- `GET /api/lessons/{id}` của bài sắp tới → 403 cho người học, 200 cho quản trị viên.
- Người khác không thấy goal, tiến độ của người học.
