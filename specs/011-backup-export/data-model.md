# Data Model: Sao lưu và xuất dữ liệu (F13)

F13 không thêm collection. Hai "thực thể" là file.

## Bản sao lưu (file trên máy chủ)

| Thuộc tính | Giá trị |
| --- | --- |
| Thư mục | `deploy/backups/` (bind mount `/backups` trong container `backup`) |
| Tên | `luna-YYYYMMDD.archive.gz` (ngày theo `TZ` của service backup); đang ghi: `…archive.gz.tmp` |
| Nội dung | `mongodump --archive --gzip` của database `luna` (mọi collection) |
| Giữ lại | `BACKUP_KEEP` (mặc định 7) file khớp mẫu mới nhất theo tên; file khác không bị xoá |

Vòng đời: `(chờ 03:00) → ghi .tmp → thành công: đổi tên, xoay vòng | lỗi: xoá .tmp, log ERROR → (chờ lần sau)`.

## File xuất (trả về trình duyệt, không lưu)

```json
{
  "version": 1,
  "exportedAt": "2026-09-30T14:05:00+07:00",
  "account": { "id": "…", "email": "hoc@example.com", "role": "learner", "createdAt": "2026-09-28T02:00:00Z" },
  "settings": { "theme": "system", "dailyReviewLimit": 30, "timezone": "Asia/Ho_Chi_Minh" },
  "cards": [ { "_id": "…", "lemma": "go", "text": "go", "meaningVi": "đi", "due": "…", "…": "…" } ],
  "reviewLogs": [ { "_id": "…", "cardId": "…", "rating": 3, "reviewedAt": "…", "context": "daily" } ],
  "goals": [ { "_id": "…", "topicId": "…", "level": "A1", "status": "active" } ],
  "lessonProgress": [ { "lessonId": "…", "steps": { "review": true, "read": true, "listen": false } } ],
  "studyDays": [ { "dayKey": "2026-09-30", "lessonId": "…", "completed": false } ],
  "dictationResults": [ { "lessonId": "…", "sentenceIndex": 0, "correctWords": 4, "totalWords": 5 } ],
  "lessons": [ { "_id": "…", "title": "At the café", "level": "A1", "sentences": ["…"] }, { "id": "…", "deleted": true } ]
}
```

### Quy tắc

- Mọi mảng lấy từ collection tương ứng với `{userId: <người dùng của phiên>}`; bỏ trường `userId`; `ObjectID` → hex, ngày → RFC 3339;
  mảng rỗng là `[]` (không `null`).
- `account` chỉ có 4 trường trên; `passwordHash` không bao giờ được đọc; `sessions` không bao giờ được đọc.
- `settings` = cài đặt đã điền mặc định (như `GET /api/settings`).
- `lessons` = các `lessonId` khác nhau trong `lessonProgress`; bài không còn → `{id, deleted: true}`.

### Go (`internal/export`)

```go
type Account struct {
	ID, Email, Role string
	CreatedAt       time.Time
}

// Doc is a stored document converted to JSON-friendly values.
type Doc = map[string]any

type Repository interface {
	Account(ctx context.Context, userID string) (Account, error) // ErrNotFound
	// UserDocs returns the user's documents of an allowed collection, oldest first.
	UserDocs(ctx context.Context, collection, userID string) ([]Doc, error)
	// Lessons returns the lessons among ids that still exist.
	Lessons(ctx context.Context, ids []string) ([]Doc, error)
}

// Settings gives the learner's settings with defaults (settings.Service in main).
type Settings interface {
	Get(ctx context.Context, userID string) (settings.Settings, error) // via a small adapter to avoid importing settings
	Location(ctx context.Context, userID string) (*time.Location, error)
}
```

Danh sách collection cho phép: `cards`, `review_logs`, `goals`, `lesson_progress`, `study_days`, `dictation_results`.

## Frontend

```ts
export interface ExportFile {
  blob: Blob;
  filename: string; // from Content-Disposition, fallback "luna-export.json"
}
```
