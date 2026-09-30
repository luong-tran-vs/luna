# Contract: Settings API (F12)

Cả hai endpoint cần đăng nhập (cookie phiên F1); thiếu phiên → `401`. Luôn là cài đặt của người dùng trong phiên.

## GET /api/settings

`200 OK`

```json
{ "theme": "system", "dailyReviewLimit": 30, "timezone": "Asia/Ho_Chi_Minh" }
```

Tài khoản chưa từng lưu cài đặt nhận giá trị mặc định (timezone = múi giờ lúc đăng ký).

## PUT /api/settings

Body: một hoặc nhiều trường; trường vắng giữ nguyên.

```json
{ "dailyReviewLimit": 20, "timezone": "Europe/London" }
```

```json
{ "theme": "dark" }
```

`200 OK` — cài đặt đầy đủ sau khi lưu:

```json
{ "theme": "dark", "dailyReviewLimit": 20, "timezone": "Europe/London" }
```

Lỗi:

| Trường hợp | Mã | Body |
| --- | --- | --- |
| `theme` không thuộc light/dark/system | 400 | `{"error":"validation_error","fields":{"theme":"Chế độ giao diện không hợp lệ"}}` |
| `dailyReviewLimit` < 5, > 200, không phải số nguyên | 400 | `{"error":"validation_error","fields":{"dailyReviewLimit":"Số thẻ từ 5 đến 200"}}` |
| `timezone` rỗng, `"Local"` hoặc không tải được | 400 | `{"error":"validation_error","fields":{"timezone":"Múi giờ không hợp lệ"}}` |
| JSON sai, trường lạ, sai kiểu | 400 | như `httpx.DecodeJSON` (F1) |
| thiếu phiên | 401 | `{"error":"unauthorized",…}` |

Có lỗi ở bất kỳ trường nào → không trường nào được lưu. Body `{}` → `200` với cài đặt hiện tại.

## Ảnh hưởng tới API khác

- `GET /api/auth/me`, `POST /api/auth/login|register`: `user.timezone` vẫn có, giờ lấy từ `settings.timezone`.
- `GET /api/today`, `GET /api/dashboard`, `GET /api/vocab/review/due`, sổ từ nhóm theo ngày: dùng múi giờ trong cài đặt từ request
  tiếp theo; `reviewCount` của `/api/today` dùng `dailyReviewLimit`.

## Frontend

- Route `/settings` (authGuard, title "Cài đặt · Luna"); link "Cài đặt" ở thanh trên.
- `SettingsApiService`: `get()` → `GET /api/settings`; `update(patch)` → `PUT /api/settings`.
- `ThemeService`: khi có người dùng đăng nhập → `get()` rồi áp `theme` (ghi localStorage); `set(pref)` → áp, ghi localStorage,
  và `update({theme})` nếu đã đăng nhập.
