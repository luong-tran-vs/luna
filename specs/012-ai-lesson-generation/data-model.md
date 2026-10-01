# Data Model: AI sinh bài học (F7)

Không collection, index hay migration mới. Bản nháp không bao giờ được lưu ở server.

## Kiểu mới (Go)

### `ai.LessonKind`

`"reading"` (bài đọc, đoạn văn liền mạch) | `"dialogue"` (hội thoại, mỗi lượt một dòng `Tên: câu nói`).

### `ai.GenerateRequest`

| Trường | Kiểu | Ghi chú |
| --- | --- | --- |
| Level | string | CEFR của chủ đề (A1–C2) |
| TopicName | string | tên chủ đề |
| Count | int | 1–5 |
| Words | int | 50–800, độ dài mục tiêu mỗi bài |
| Kind | LessonKind | |
| Idea | string | có thể rỗng, ≤ 500 ký tự |
| ExistingTitles | []string | tiêu đề mọi bài của chủ đề |

### `ai.LessonDraft`

`{Title, Content string}` như AI trả về (chưa lọc).

### `lesson.GenerateInput` (từ JSON quản trị viên gửi)

`{Count, Words int; Kind string; Idea string}`. Kiểm tra (`ValidateGenerate`), lỗi theo trường:

| Trường | Quy tắc | Thông báo |
| --- | --- | --- |
| count | số nguyên 1–5 | Số bài từ 1 đến 5 |
| words | số nguyên 50–800 | Độ dài từ 50 đến 800 từ |
| kind | reading \| dialogue | Dạng bài không hợp lệ |
| idea | ≤ 500 ký tự sau trim | Ý chính tối đa 500 ký tự |

### `lesson.Draft` và `lesson.GenerateResult`

- `Draft {Title, Content string; Words int}`: bản nháp đã qua lọc (R4), `Words` = số từ theo khoảng trắng.
- `GenerateResult {Drafts []Draft; Requested, Dropped int}`.
- Lỗi mới `lesson.ErrUnusableDraft` khi lọc xong không còn bản nào.

### Thay đổi kiểu có sẵn

- `ai.Provider` thêm `GenerateLessons`; `ai.Disabled.GenerateLessons` → `ErrNotConfigured`.
- `lesson.Input` thêm `AppendToRoadmap bool` (chỉ `Create` dùng; `Update` bỏ qua).
- `lesson.Topics` thêm `AppendLesson(ctx, topicID, lessonID string) error`.
- `topic.Service` thêm `AppendLesson` (bọc `Repository.AppendLesson`, đã có từ F14).

## Bài học được lưu từ bản nháp (dùng lại `lessons` của F2)

| Trường | Giá trị |
| --- | --- |
| title, content | do quản trị viên duyệt/sửa, cùng kiểm tra như F2 |
| topicId, level | chủ đề đang mở; level lấy theo chủ đề |
| source | `AI sinh` |
| license | `Nội dung do AI tạo` |
| audioStatus, annotationStatus | `running` ngay khi lưu (job TTS + chú thích như F2) |

Chủ đề (`topics.lessonIds`): id bài mới được `$push` vào cuối trong cùng request tạo bài.

## Trạng thái phía client (frontend)

### `GenerateOptions` (signal của trang, giữ qua các lượt)

`{count: number; words: number; kind: 'reading' | 'dialogue'; idea: string}`; mặc định `{3, DEFAULT_WORDS[level], 'reading', ''}`.
Đổi sang chủ đề khác trình độ thì `words` về mặc định của trình độ mới (nếu quản trị viên chưa tự sửa trong lượt đó thì vẫn đổi).

### `DraftState`

| Trường | Ghi chú |
| --- | --- |
| key | số tăng dần phía client, dùng cho `track` |
| title, content | đang sửa |
| saving | đang gửi POST |
| error | thông báo lỗi lưu chung (mất mạng, 500) hoặc null |
| fields | lỗi theo trường từ server (`title`, `content`) |

Chuyển trạng thái: `mới sinh → (sửa)* → saving → [thành công: xoá khỏi danh sách] | [lỗi: error/fields, quay lại sửa được]`;
`Bỏ` xoá ngay. Lượt sinh thành công nối thêm vào cuối danh sách; lượt lỗi không đổi danh sách.

### Trạng thái lượt sinh (trong dialog)

`idle → generating → (success: đóng dialog, thêm bản nháp, báo "đã loại k bản" nếu có) | (error: thông báo theo `error` code, giữ form)`.
