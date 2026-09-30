# Data Model: Cài đặt (F12)

## Settings (tài liệu con `users.settings`)

| Trường | Kiểu | Mặc định | Ràng buộc |
| --- | --- | --- | --- |
| `theme` | `"light"` \| `"dark"` \| `"system"` | `"system"` | một trong 3 giá trị |
| `dailyReviewLimit` | int | `30` | 5 ≤ x ≤ 200 |
| `timezone` | string (tên IANA) | múi giờ lúc đăng ký (F1, dự phòng `Asia/Ho_Chi_Minh`) | không rỗng, khác `"Local"`, `time.LoadLocation` được |

Ví dụ tài liệu `users` sau F12:

```json
{
  "_id": "…", "email": "hoc@example.com", "passwordHash": "…", "role": "learner", "createdAt": "…",
  "settings": { "theme": "dark", "dailyReviewLimit": 20, "timezone": "Europe/London" }
}
```

- Trường thiếu (tài khoản cũ, hoặc migration chưa chạy) đọc ra giá trị mặc định; `timezone` thiếu trong `settings` thì đọc trường
  gốc `users.timezone` của F1, rồi mới tới `Asia/Ho_Chi_Minh`.
- Cập nhật: `$set` từng trường có trong yêu cầu (`settings.theme`, …); không bao giờ ghi đè cả tài liệu con.

### Go (`internal/settings`)

```go
type Theme string // ThemeLight, ThemeDark, ThemeSystem

type Settings struct {
	Theme            Theme
	DailyReviewLimit int
	Timezone         string
}

// Patch holds the fields to change; nil = unchanged.
type Patch struct {
	Theme            *Theme
	DailyReviewLimit *int
	Timezone         *string
}

type Repository interface {
	// Get returns the stored fields; zero values for missing ones. ErrNotFound for an unknown user.
	Get(ctx context.Context, userID string) (Settings, error)
	// Update sets the non-nil fields of p and returns the stored settings.
	Update(ctx context.Context, userID string, p Patch) (Settings, error)
}
```

`Service.Get` điền mặc định cho trường rỗng; `Service.Update` kiểm tra mọi trường có mặt trước (lỗi → `*ValidationError{Fields}`,
không ghi gì), rồi gọi `Repository.Update`. `Service.Location(ctx, userID) (*time.Location, error)` và
`Service.ReviewLimit(ctx, userID) int` phục vụ adapter trong `main.go`.

## Migration `users-settings-timezone`

```js
db.users.updateMany(
  { timezone: { $exists: true }, "settings.timezone": { $exists: false } },
  [{ $set: { "settings.timezone": "$timezone" } }, { $unset: "timezone" }]
)
```

Ghi vào `migrations` sau khi chạy xong (như F14); chạy lại không đổi gì.

## Thay đổi trong progress (L)

| Thành phần | Thay đổi |
| --- | --- |
| `rules.go` | `EffectiveDayKey(localKey, latestKey string) string` = chuỗi lớn hơn (rỗng coi như không có) |
| `DayRepository` | `LatestKey(ctx, userID) (string, error)`: `dayKey` lớn nhất, `""` khi chưa có ngày nào |
| `StudyService.load` | `today = EffectiveDayKey(DayKey(now, loc), LatestKey)` |

Không đổi dữ liệu `study_days`, `lesson_progress`, `review_logs`.

## Frontend

```ts
export type ThemePreference = 'light' | 'dark' | 'system'; // đã có (theme.service.ts)

export interface Settings {
  theme: ThemePreference;
  dailyReviewLimit: number;
  timezone: string;
}

export type SettingsPatch = Partial<Settings>;
```
