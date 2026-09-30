# Contract: Dashboard API (F6)

Cả hai endpoint cần đăng nhập (cookie phiên như F1); thiếu phiên → `401 {"error":"unauthorized"}`. Số liệu luôn của người dùng trong
phiên, không nhận tham số. Lỗi máy chủ → `500 {"error":"internal"}` (như các API trước). Chỉ đọc: gọi bao nhiêu lần cũng không đổi
dữ liệu.

## GET /api/dashboard

`200 OK`

```json
{
  "kind": "studying",
  "goal": {
    "topicId": "66f0…",
    "topicName": "Gia đình",
    "level": "A1",
    "completedLessons": 2,
    "totalLessons": 12
  },
  "goalCompleted": false,
  "skills": { "read": 3, "listen": 2, "total": 12 },
  "lesson": { "id": "66f1…", "title": "At the café", "topicName": "Gia đình", "level": "A1" },
  "steps": { "review": "done", "read": "current", "listen": "locked" },
  "currentStep": "read",
  "action": { "kind": "continue", "step": "read" },
  "streak": 4,
  "tomorrowCards": 17
}
```

| Trường | Ghi chú |
| --- | --- |
| `kind` | `noGoal` \| `studying` \| `doneToday` \| `noNewLesson` |
| `goal` | `null` khi `noGoal` |
| `skills` | `null` khi `goal` là `null`; `total = goal.totalLessons` |
| `lesson` | `null` khi không có bài hôm nay (`noGoal`, `noNewLesson`); có khi `doneToday`. `title` là `""` khi bài đã bị xoá |
| `steps` | luôn đủ 3 khoá; `done` \| `current` \| `locked` |
| `currentStep` | `review` \| `read` \| `listen` \| `done`; `""` khi không có bài |
| `action` | chỉ khi `kind = studying`, ngược lại `null`. `kind`: `start` (chưa bước nào xong) \| `continue`. `step`: `review` \| `read` \| `listen` (`read` khi đang ở Ôn mà không có thẻ cần ôn) |
| `streak` | số ngày liên tiếp (L) |
| `tomorrowCards` | số thẻ có hạn trước 0 giờ ngày kia theo múi giờ tài khoản, gồm thẻ còn nợ; không giới hạn 30 |

Ví dụ các trạng thái khác:

```json
{ "kind": "noGoal", "goal": null, "goalCompleted": false, "skills": null, "lesson": null,
  "steps": { "review": "locked", "read": "locked", "listen": "locked" }, "currentStep": "", "action": null,
  "streak": 0, "tomorrowCards": 0 }
```

```json
{ "kind": "doneToday", "goal": { "…": "…" }, "skills": { "read": 4, "listen": 4, "total": 12 },
  "lesson": { "id": "…", "title": "At the café", "topicName": "Gia đình", "level": "A1" },
  "steps": { "review": "done", "read": "done", "listen": "done" }, "currentStep": "done", "action": null,
  "streak": 5, "tomorrowCards": 9 }
```

## GET /api/stats

`200 OK`

```json
{
  "cards": 25,
  "dictation": { "sentences": 40, "correctWords": 328, "totalWords": 400, "rate": 0.82 },
  "lessons": { "read": 6, "listen": 5, "completed": 5 }
}
```

| Trường | Ghi chú |
| --- | --- |
| `cards` | số thẻ trong sổ từ |
| `dictation.sentences` | số câu có kết quả chép chính tả (mỗi câu một lần, lần mới nhất), mọi bài |
| `dictation.rate` | `correctWords / totalWords` (0–1); `null` khi `totalWords = 0` (giao diện hiện "—") |
| `lessons` | số bài (mọi chủ đề) đã xong bước Đọc, bước Nghe, và hoàn thành |

Người học mới: `{"cards":0,"dictation":{"sentences":0,"correctWords":0,"totalWords":0,"rate":null},"lessons":{"read":0,"listen":0,"completed":0}}`.

## Frontend

- Route `/` (home, `authGuard`): gọi `GET /api/dashboard` mỗi lần vào trang.
- Route `/stats` (`authGuard`, title "Thống kê · Luna"): gọi `GET /api/stats`.
- Nút hành động → `/today`; "Đổi chủ đề" / "Chọn chủ đề" / "Chọn chủ đề khác" → `/goal`; "Ôn tự do" → `/vocabulary/review`;
  "Xem thống kê" → `/stats`.
- Chân trang (mọi trang): `GET /api/health` (F0) khi mở app và sau mỗi lần điều hướng; chỉ hiện khi lỗi.
