# Research: Cài đặt (F12)

Không còn mục NEEDS CLARIFICATION.

## R1. Nơi lưu cài đặt

- **Decision**: Tài liệu con `settings {theme, dailyReviewLimit, timezone}` trong `users` (khối 2). Trường thiếu = mặc định (`system`,
  30, múi giờ lúc đăng ký). `Register` (F1) ghi `settings.timezone` (+ theme, limit mặc định) thay cho `timezone` gốc.
- **Rationale**: Một tài khoản có đúng một bộ cài đặt; đọc cùng tài khoản; không cần collection hay index mới.
- **Alternatives**: Collection `settings` riêng (thêm join/khoá, không lợi gì với 3 trường).

## R2. Chuyển `users.timezone` vào `settings`

- **Decision**: Migration `users-settings-timezone` trong framework migration của F14 (`migrate.go`, bảng `migrations`, chạy trong
  `PrepareInBackground`): với mỗi user có `timezone` mà chưa có `settings.timezone` → `$set settings.timezone`, `$unset timezone`
  (một `updateMany` với pipeline). Trong lúc migration chưa chạy, repository đọc `settings.timezone` và dự phòng `timezone` cũ.
- **Rationale**: Khối 2 yêu cầu chuyển; đọc dự phòng giữ app chạy đúng cả trước khi migration xong (khởi động không bị chặn như F0).
- **Alternatives**: Giữ trùng hai trường (dễ lệch nhau).

## R3. API cập nhật từng phần

- **Decision**: `PUT /api/settings` nhận các trường tuỳ chọn; trường có mặt được kiểm tra và ghi bằng một `$set`, trường vắng giữ
  nguyên; trả về cài đặt đầy đủ sau khi lưu. Trường lạ → 400 (như `httpx.DecodeJSON`). Body rỗng `{}` → 200, không đổi gì.
- **Rationale**: Nút giao diện nhanh ở thanh trên chỉ biết theme; trang Cài đặt lưu limit + múi giờ cùng lúc. Một endpoint cho cả hai.
- **Alternatives**: PUT toàn bộ (thanh trên phải đọc trước rồi gửi lại cả ba — thừa và dễ ghi đè); PATCH (khối 2 chọn PUT).

## R4. Kiểm tra đầu vào

- **Decision**: `theme ∈ {light, dark, system}`; `dailyReviewLimit` số nguyên 5–200 (JSON số thực như 10.5 → lỗi); `timezone` không
  rỗng, khác `"Local"`, `time.LoadLocation` thành công (có `time/tzdata` nhúng nên không phụ thuộc máy chủ). Lỗi trả 400
  `validation_error` với `fields` tiếng Việt: `theme` "Chế độ giao diện không hợp lệ", `dailyReviewLimit` "Số thẻ từ 5 đến 200",
  `timezone` "Múi giờ không hợp lệ". Có lỗi thì không ghi trường nào.
- **Alternatives**: Danh sách múi giờ cứng ở server (phải cập nhật tay theo tzdata).

## R5. progress và vocab dùng cài đặt

- **Decision**: `main.go`: `userTimezones.Location` gọi `settings.Service.Location` (thay `auth.UserByID`); `StudyDeps.ReviewLimit`
  gọi `settings.Service.ReviewLimit` (lỗi đọc → log và dùng 30, không chặn bài hôm nay). Không đổi interface của `progress`/`vocab`.
- **Rationale**: Port đã có từ L; "hiệu lực từ lần mở Bài hôm nay tiếp theo" tự đúng vì mỗi request đọc cài đặt mới (không cache).

## R6. Đổi múi giờ không làm mất tiến độ

- **Decision**: Hàm thuần `EffectiveDayKey(localKey, latestKey string) string` = `max(localKey, latestKey)` (so sánh chuỗi
  `YYYY-MM-DD`). `StudyService.load` tính `today = EffectiveDayKey(DayKey(now, loc), Days.LatestKey(user))`. Hệ quả:
  - Cùng ngày theo múi giờ mới: ngày học, bài, bước, vị trí giữ nguyên (khoá theo `dayKey` + `lessonId`).
  - Sang ngày sau: ngày mới; bài đang dở (chưa hoàn thành) vẫn là bài đầu tiên chưa xong của lộ trình nên vẫn là bài hôm nay, bước
    đã xong giữ (tiến độ lưu theo bài).
  - Lùi về ngày trước (vd. từ châu Á sang châu Mỹ sau nửa đêm): ngày học giữ ở ngày mới nhất đã có → không sinh thêm bài trong cùng
    một ngày thực, ngày đã xong vẫn "đã xong".
  `DayRepository.LatestKey(ctx, userID)` = `dayKey` lớn nhất của người học (index unique `userId+dayKey`, sort −1, limit 1).
- **Rationale**: Một quy tắc nhỏ, thuần, test được; không phải sửa dữ liệu cũ khi đổi múi giờ.
- **Alternatives**: Ghi lại `dayKey` của mọi ngày theo múi giờ mới (phức tạp, có thể gộp/đứt streak sai).

## R7. Giới hạn thẻ đổi giữa ngày

- **Decision**: Không đổi quy tắc L: `reviewCount = min(due, max(0, limit − reviewedToday))` với `reviewedToday` đếm từ `review_logs`
  context `daily` từ 0 giờ hôm nay theo múi giờ hiện tại. Hạ limit dưới số đã ôn → 0 thẻ (bước Ôn tự xong khi mở, như L); nâng lên →
  còn phần chênh.

## R8. ThemeService đồng bộ với tài khoản

- **Decision**:
  - Trước đăng nhập: như F0 (localStorage `luna.theme`, script inline trong `index.html` đặt `data-theme` trước khi Angular chạy).
  - Khi `AuthService.currentUser()` chuyển sang một user (đăng nhập, hoặc `/api/auth/me` lúc mở app): `ThemeService` gọi
    `GET /api/settings`, áp theme của tài khoản và ghi localStorage (không PUT lại).
  - `ThemeService.set(pref)` (trang Cài đặt hoặc nút nhanh ở header): áp ngay + ghi localStorage; nếu đã đăng nhập thì
    `PUT /api/settings {theme}`; lỗi mạng không hoàn tác giao diện, trang Cài đặt báo lỗi lưu.
  - Đăng xuất: giữ localStorage (màn hình đăng nhập dùng lựa chọn cuối).
- **Rationale**: Khối 2; đổi giao diện không chờ server nên < 0,5 s.
- **Alternatives**: Trả theme trong `/api/auth/me` (đổi contract F1; vẫn cần PUT riêng).

## R9. Danh sách múi giờ

- **Decision**: `Intl.supportedValuesOf('timeZone')` (dự phòng: chỉ múi giờ hiện tại nếu trình duyệt không hỗ trợ), mỗi mục hiện
  "Asia/Ho_Chi_Minh (GMT+7)" bằng `Intl.DateTimeFormat(..., {timeZoneName: 'shortOffset'})`. Ô tìm (`input type=search`) lọc
  không phân biệt hoa thường theo tên và "/"→" "; kết quả trong `<select size>` có nhãn; múi giờ đang lưu luôn có trong danh sách
  kể cả khi không khớp bộ lọc. Server vẫn kiểm tra (R4) vì danh sách trình duyệt có thể khác tzdata.
- **Alternatives**: Combobox tự viết (nhiều việc a11y hơn); `<datalist>` (hiển thị khác nhau giữa trình duyệt, khó dùng bằng trình đọc
  màn hình).

## R10. Trang Cài đặt

- **Decision**: `/settings` (authGuard, title "Cài đặt · Luna"), một cột:
  - **Giao diện** (sửa 2026-09-30: còn 2 lựa chọn Sáng, Tối; radio dùng `click` để chọn lại Sáng khi đang "theo thiết bị" vẫn lưu được): radiogroup 3 lựa chọn, đổi là áp + lưu ngay (không cần nút Lưu); dòng trạng thái "Đã lưu" / lỗi.
  - **Học tập** + **Múi giờ**: một form với ô số (min 5, max 200, bước 1) và danh sách múi giờ, nút **Lưu**; lỗi theo trường từ server
    hiện dưới ô (`aria-describedby`, `aria-invalid`); thành công hiện "Đã lưu" (`role="status"`).
  - Tải lỗi: thông báo + "Thử lại"; nhóm Giao diện vẫn dùng được (lưu cục bộ).
  Link "Cài đặt" ở thanh trên (header F0/L). Nút giao diện nhanh ở header giữ nguyên, đi qua `ThemeService.set`.
